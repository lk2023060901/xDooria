package etcd

import (
	"context"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Lease 租约管理客户端
type Lease struct {
	client *Client
	lease  clientv3.Lease
}

// newLease 创建 Lease 客户端
func newLease(client *Client) *Lease {
	return &Lease{
		client: client,
		lease:  clientv3.NewLease(client.rawClient()),
	}
}

// Grant 创建租约
func (l *Lease) Grant(ctx context.Context, ttl int64) (LeaseID, error) {
	resp, err := l.lease.Grant(ctx, ttl)
	if err != nil {
		return 0, err
	}
	return LeaseID(resp.ID), nil
}

// Revoke 撤销租约
func (l *Lease) Revoke(ctx context.Context, id LeaseID) error {
	_, err := l.lease.Revoke(ctx, clientv3.LeaseID(id))
	return err
}

// KeepAlive 持续续约（返回续约响应通道）
func (l *Lease) KeepAlive(ctx context.Context, id LeaseID) (<-chan *clientv3.LeaseKeepAliveResponse, error) {
	return l.lease.KeepAlive(ctx, clientv3.LeaseID(id))
}

// KeepAliveOnce 单次续约
func (l *Lease) KeepAliveOnce(ctx context.Context, id LeaseID) (*clientv3.LeaseKeepAliveResponse, error) {
	return l.lease.KeepAliveOnce(ctx, clientv3.LeaseID(id))
}

// TTL 获取租约剩余时间
func (l *Lease) TTL(ctx context.Context, id LeaseID) (int64, error) {
	resp, err := l.lease.TimeToLive(ctx, clientv3.LeaseID(id))
	if err != nil {
		return 0, err
	}

	if resp.TTL == -1 {
		return 0, ErrLeaseNotFound
	}

	return resp.TTL, nil
}

// Leases 获取所有租约
func (l *Lease) Leases(ctx context.Context) ([]LeaseID, error) {
	resp, err := l.lease.Leases(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]LeaseID, 0, len(resp.Leases))
	for _, lease := range resp.Leases {
		result = append(result, LeaseID(lease.ID))
	}

	return result, nil
}

// GrantWithKeepAlive 创建租约并自动续约
// 返回租约 ID 和停止续约的函数
func (l *Lease) GrantWithKeepAlive(ctx context.Context, ttl int64) (LeaseID, func(), error) {
	// 创建租约
	leaseID, err := l.Grant(ctx, ttl)
	if err != nil {
		return 0, nil, err
	}

	// 启动自动续约
	keepAliveCh, err := l.KeepAlive(ctx, leaseID)
	if err != nil {
		// 创建失败，撤销租约
		_ = l.Revoke(context.Background(), leaseID)
		return 0, nil, err
	}

	// 启动协程处理续约响应
	stopCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopCh:
				return
			case _, ok := <-keepAliveCh:
				if !ok {
					// 续约通道关闭，租约可能已失效
					return
				}
			}
		}
	}()

	// 返回停止函数
	stop := func() {
		close(stopCh)
		_ = l.Revoke(context.Background(), leaseID)
	}

	return leaseID, stop, nil
}

// GrantWithTimeout 创建租约并在超时后自动撤销
func (l *Lease) GrantWithTimeout(ctx context.Context, ttl int64, timeout time.Duration) (LeaseID, error) {
	leaseID, err := l.Grant(ctx, ttl)
	if err != nil {
		return 0, err
	}

	// 启动定时器
	time.AfterFunc(timeout, func() {
		_ = l.Revoke(context.Background(), leaseID)
	})

	return leaseID, nil
}
