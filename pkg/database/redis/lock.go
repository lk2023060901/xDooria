package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

const (
	// 默认锁过期时间
	defaultLockTTL = 10 * time.Second
)

// Lock 分布式锁（单节点实现）
type Lock struct {
	client *Client
	key    string        // 锁的键
	value  string        // 锁的值（用于验证锁持有者）
	ttl    time.Duration // 锁的过期时间
}

// NewLock 创建分布式锁（单节点）
func NewLock(client *Client, key string, ttl time.Duration) *Lock {
	if ttl <= 0 {
		ttl = defaultLockTTL
	}

	return &Lock{
		client: client,
		key:    key,
		value:  uuid.New().String(), // 使用 UUID 作为锁的唯一标识
		ttl:    ttl,
	}
}

// Lock 获取锁（阻塞方式，直到成功获取锁或超时）
func (l *Lock) Lock(ctx context.Context) error {
	// 使用 SET NX EX 原子操作获取锁
	ok, err := l.client.getMaster().SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	if !ok {
		return ErrLockFailed
	}

	return nil
}

// TryLock 尝试获取锁（非阻塞方式，立即返回）
func (l *Lock) TryLock(ctx context.Context) (bool, error) {
	ok, err := l.client.getMaster().SetNX(ctx, l.key, l.value, l.ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to try lock: %w", err)
	}

	return ok, nil
}

// LockWithRetry 获取锁（带重试机制）
func (l *Lock) LockWithRetry(ctx context.Context, retryInterval time.Duration, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		ok, err := l.TryLock(ctx)
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			// 继续重试
		}
	}

	return ErrLockFailed
}

// Unlock 释放锁（使用 Lua 脚本保证原子性：只有锁持有者才能释放锁）
func (l *Lock) Unlock(ctx context.Context) error {
	// Lua 脚本：检查锁的值是否匹配，匹配则删除
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.getMaster().Eval(ctx, script, []string{l.key}, l.value).Result()
	if err != nil {
		return fmt.Errorf("failed to unlock: %w", err)
	}

	// 检查返回值（0 表示锁不存在或不是当前持有者）
	if result.(int64) == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// Refresh 刷新锁的过期时间（延长锁的持有时间）
func (l *Lock) Refresh(ctx context.Context) error {
	// Lua 脚本：检查锁的值是否匹配，匹配则更新过期时间
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.getMaster().Eval(ctx, script, []string{l.key}, l.value, l.ttl.Milliseconds()).Result()
	if err != nil {
		return fmt.Errorf("failed to refresh lock: %w", err)
	}

	// 检查返回值（0 表示锁不存在或不是当前持有者）
	if result.(int64) == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// WithLock 在锁的保护下执行函数（自动加锁、解锁）
func (c *Client) WithLock(ctx context.Context, key string, ttl time.Duration, fn func() error) error {
	lock := NewLock(c, key, ttl)

	// 获取锁
	if err := lock.Lock(ctx); err != nil {
		return err
	}

	// 确保函数返回时释放锁
	defer func() {
		if err := lock.Unlock(context.Background()); err != nil {
			// 记录日志但不返回错误（避免覆盖函数的错误）
			// TODO: 使用 zap 记录日志
		}
	}()

	// 执行函数
	return fn()
}

// WithLockRetry 在锁的保护下执行函数（带重试机制）
func (c *Client) WithLockRetry(ctx context.Context, key string, ttl time.Duration, retryInterval time.Duration, maxRetries int, fn func() error) error {
	lock := NewLock(c, key, ttl)

	// 获取锁（带重试）
	if err := lock.LockWithRetry(ctx, retryInterval, maxRetries); err != nil {
		return err
	}

	// 确保函数返回时释放锁
	defer func() {
		if err := lock.Unlock(context.Background()); err != nil {
			// 记录日志但不返回错误（避免覆盖函数的错误）
			// TODO: 使用 zap 记录日志
		}
	}()

	// 执行函数
	return fn()
}

// IsLocked 检查锁是否被持有（从从库读取）
func (c *Client) IsLocked(ctx context.Context, key string) (bool, error) {
	val, err := c.getSlave().Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check lock: %w", err)
	}

	return val != "", nil
}
