package etcd

import (
	"context"
	"time"

	"go.etcd.io/etcd/client/v3/concurrency"
)

// Locker 分布式锁客户端
type Locker struct {
	client *Client
}

// newLocker 创建 Locker 客户端
func newLocker(client *Client) *Locker {
	return &Locker{
		client: client,
	}
}

// Lock 分布式锁
type Lock struct {
	session *concurrency.Session
	mutex   *concurrency.Mutex
	key     string
}

// NewLock 创建分布式锁
func (l *Locker) NewLock(key string, opts ...LockOption) (*Lock, error) {
	options := &lockOptions{
		ttl:     60, // 默认 60 秒
		timeout: 0,  // 默认不超时
	}

	for _, opt := range opts {
		opt(options)
	}

	// 创建 session
	session, err := concurrency.NewSession(
		l.client.rawClient(),
		concurrency.WithTTL(int(options.ttl)),
	)
	if err != nil {
		return nil, err
	}

	// 创建 mutex
	mutex := concurrency.NewMutex(session, key)

	return &Lock{
		session: session,
		mutex:   mutex,
		key:     key,
	}, nil
}

// Lock 获取锁（阻塞直到获取成功）
func (lock *Lock) Lock(ctx context.Context) error {
	return lock.mutex.Lock(ctx)
}

// TryLock 尝试获取锁（非阻塞）
func (lock *Lock) TryLock(ctx context.Context) error {
	// 创建一个立即超时的 context
	tryCtx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
	defer cancel()

	if err := lock.mutex.Lock(tryCtx); err != nil {
		if err == context.DeadlineExceeded {
			return ErrLockTimeout
		}
		return err
	}

	return nil
}

// LockWithTimeout 带超时的获取锁
func (lock *Lock) LockWithTimeout(ctx context.Context, timeout time.Duration) error {
	lockCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := lock.mutex.Lock(lockCtx); err != nil {
		if err == context.DeadlineExceeded {
			return ErrLockTimeout
		}
		return err
	}

	return nil
}

// Unlock 释放锁
func (lock *Lock) Unlock(ctx context.Context) error {
	return lock.mutex.Unlock(ctx)
}

// Key 获取锁的键
func (lock *Lock) Key() string {
	return lock.key
}

// Close 关闭锁（释放 session）
func (lock *Lock) Close() error {
	return lock.session.Close()
}

// WithLockDo 在锁保护下执行函数
func (l *Locker) WithLockDo(ctx context.Context, key string, fn func() error, opts ...LockOption) error {
	lock, err := l.NewLock(key, opts...)
	if err != nil {
		return err
	}
	defer lock.Close()

	if err := lock.Lock(ctx); err != nil {
		return err
	}
	defer lock.Unlock(context.Background())

	return fn()
}

// WithLockDoWithTimeout 在锁保护下执行函数（带超时）
func (l *Locker) WithLockDoWithTimeout(ctx context.Context, key string, timeout time.Duration, fn func() error, opts ...LockOption) error {
	lock, err := l.NewLock(key, opts...)
	if err != nil {
		return err
	}
	defer lock.Close()

	if err := lock.LockWithTimeout(ctx, timeout); err != nil {
		return err
	}
	defer lock.Unlock(context.Background())

	return fn()
}

// --- Lock Options ---

// WithLockTTL 设置锁的 TTL
func WithLockTTL(ttl int64) LockOption {
	return func(o *lockOptions) {
		o.ttl = ttl
	}
}

// WithLockTimeout 设置锁超时
func WithLockTimeout(timeout time.Duration) LockOption {
	return func(o *lockOptions) {
		o.timeout = timeout
	}
}
