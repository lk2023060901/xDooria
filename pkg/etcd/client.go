package etcd

import (
	"context"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Client etcd 客户端封装
type Client struct {
	client *clientv3.Client
	config *Config

	// 子客户端
	kv       *KV
	lease    *Lease
	watcher  *Watcher
	locker   *Locker
	election *Election
	txn      *Transaction

	mu     sync.RWMutex
	closed bool
}

// New 创建 etcd 客户端
func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	clientCfg, err := cfg.ToClientConfig()
	if err != nil {
		return nil, err
	}

	cli, err := clientv3.New(*clientCfg)
	if err != nil {
		return nil, err
	}

	c := &Client{
		client: cli,
		config: cfg,
	}

	// 初始化子客户端
	c.kv = newKV(c)
	c.lease = newLease(c)
	c.watcher = newWatcher(c)
	c.locker = newLocker(c)
	c.election = newElection(c)
	c.txn = newTransaction(c)

	return c, nil
}

// KV 返回 KV 操作客户端
func (c *Client) KV() *KV {
	return c.kv
}

// Lease 返回租约管理客户端
func (c *Client) Lease() *Lease {
	return c.lease
}

// Watcher 返回监听客户端
func (c *Client) Watcher() *Watcher {
	return c.watcher
}

// Locker 返回分布式锁客户端
func (c *Client) Locker() *Locker {
	return c.locker
}

// Election 返回选举客户端
func (c *Client) Election() *Election {
	return c.election
}

// Transaction 返回事务客户端
func (c *Client) Transaction() *Transaction {
	return c.txn
}

// HealthCheck 健康检查
func (c *Client) HealthCheck(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	c.mu.RUnlock()

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	_, err := c.client.Status(ctx, c.config.Endpoints[0])
	return err
}

// Endpoints 获取当前端点列表
func (c *Client) Endpoints() []string {
	return c.client.Endpoints()
}

// Sync 同步端点列表
func (c *Client) Sync(ctx context.Context) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return ErrClientClosed
	}
	c.mu.RUnlock()

	return c.client.Sync(ctx)
}

// Close 关闭客户端
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	c.closed = true
	return c.client.Close()
}

// IsClosed 是否已关闭
func (c *Client) IsClosed() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.closed
}

// rawClient 获取原始 etcd 客户端（内部使用）
func (c *Client) rawClient() *clientv3.Client {
	return c.client
}

// withRetry 带重试执行
func (c *Client) withRetry(ctx context.Context, fn func() error) error {
	if !c.config.EnableRetry {
		return fn()
	}

	var lastErr error
	for i := 0; i <= c.config.MaxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.config.RetryInterval):
			}
		}

		if err := fn(); err != nil {
			lastErr = err
			continue
		}

		return nil
	}

	return lastErr
}
