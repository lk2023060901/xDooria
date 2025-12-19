package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const (
	// Redlock 默认配置
	defaultRedlockTTL         = 10 * time.Second
	defaultRedlockRetries     = 3
	defaultRedlockRetryDelay  = 200 * time.Millisecond
	defaultRedlockClockDrift  = 0.01 // 1% 时钟漂移
	defaultRedlockQuorumRatio = 0.5  // 需要超过 50% 的节点同意
)

// Redlock 分布式锁（多节点实现，基于 Redlock 算法）
// 参考: https://redis.io/docs/manual/patterns/distributed-locks/
type Redlock struct {
	clients []*Client     // Redis 客户端列表（每个客户端代表一个独立的 Redis 实例）
	quorum  int           // 需要获取锁的最小节点数（超过半数）
	ttl     time.Duration // 锁的过期时间
}

// NewRedlock 创建 Redlock 实例
// clients: 多个独立的 Redis 客户端（建议 3 或 5 个节点）
// ttl: 锁的过期时间
func NewRedlock(clients []*Client, ttl time.Duration) (*Redlock, error) {
	if len(clients) == 0 {
		return nil, fmt.Errorf("redlock: no clients provided")
	}

	if ttl <= 0 {
		ttl = defaultRedlockTTL
	}

	// 计算 quorum（需要超过半数节点同意）
	quorum := len(clients)/2 + 1

	return &Redlock{
		clients: clients,
		quorum:  quorum,
		ttl:     ttl,
	}, nil
}

// RedlockInstance 单个 Redlock 锁实例
type RedlockInstance struct {
	redlock *Redlock
	key     string        // 锁的键
	value   string        // 锁的值（用于验证锁持有者）
	ttl     time.Duration // 锁的过期时间
}

// NewRedlockInstance 创建 Redlock 锁实例
func (r *Redlock) NewInstance(key string) *RedlockInstance {
	return &RedlockInstance{
		redlock: r,
		key:     key,
		value:   uuid.New().String(), // 使用 UUID 作为锁的唯一标识
		ttl:     r.ttl,
	}
}

// Lock 获取 Redlock 锁（阻塞方式，使用默认重试策略）
func (ri *RedlockInstance) Lock(ctx context.Context) error {
	return ri.LockWithRetry(ctx, defaultRedlockRetries, defaultRedlockRetryDelay)
}

// LockWithRetry 获取 Redlock 锁（带重试机制）
func (ri *RedlockInstance) LockWithRetry(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		ok, err := ri.TryLock(ctx)
		if err != nil {
			return err
		}

		if ok {
			return nil
		}

		// 等待一段时间后重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			// 继续重试
		}
	}

	return ErrRedlockFailed
}

// TryLock 尝试获取 Redlock 锁（非阻塞方式，立即返回）
func (ri *RedlockInstance) TryLock(ctx context.Context) (bool, error) {
	startTime := time.Now()

	// 尝试在所有节点上获取锁
	successCount := 0
	for _, client := range ri.redlock.clients {
		// 检查上下文是否已取消
		select {
		case <-ctx.Done():
			// 如果上下文已取消，释放已获取的锁
			_ = ri.unlockAll(context.Background())
			return false, ctx.Err()
		default:
		}

		// 尝试在单个节点上获取锁
		ok, err := client.getMaster().SetNX(ctx, ri.key, ri.value, ri.ttl).Result()
		if err != nil {
			// 获取锁失败，继续尝试其他节点
			continue
		}

		if ok {
			successCount++
		}
	}

	// 计算有效时间（考虑时钟漂移）
	elapsed := time.Since(startTime)
	drift := time.Duration(float64(ri.ttl) * defaultRedlockClockDrift)
	validityTime := ri.ttl - elapsed - drift

	// 检查是否获取了足够多的锁，且有效时间大于 0
	if successCount >= ri.redlock.quorum && validityTime > 0 {
		return true, nil
	}

	// 如果未能获取足够多的锁，释放已获取的锁
	_ = ri.unlockAll(context.Background())
	return false, nil
}

// Unlock 释放 Redlock 锁
func (ri *RedlockInstance) Unlock(ctx context.Context) error {
	return ri.unlockAll(ctx)
}

// unlockAll 在所有节点上释放锁
func (ri *RedlockInstance) unlockAll(ctx context.Context) error {
	// Lua 脚本：检查锁的值是否匹配，匹配则删除
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	var lastErr error
	successCount := 0

	for _, client := range ri.redlock.clients {
		result, err := client.getMaster().Eval(ctx, script, []string{ri.key}, ri.value).Result()
		if err != nil {
			lastErr = err
			continue
		}

		if result.(int64) == 1 {
			successCount++
		}
	}

	// 如果所有节点都释放失败，返回最后一个错误
	if successCount == 0 && lastErr != nil {
		return fmt.Errorf("failed to unlock on all nodes: %w", lastErr)
	}

	return nil
}

// Refresh 刷新 Redlock 锁的过期时间
func (ri *RedlockInstance) Refresh(ctx context.Context) error {
	// Lua 脚本：检查锁的值是否匹配，匹配则更新过期时间
	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	successCount := 0
	for _, client := range ri.redlock.clients {
		result, err := client.getMaster().Eval(ctx, script, []string{ri.key}, ri.value, ri.ttl.Milliseconds()).Result()
		if err != nil {
			continue
		}

		if result.(int64) == 1 {
			successCount++
		}
	}

	// 检查是否在足够多的节点上刷新成功
	if successCount < ri.redlock.quorum {
		return ErrLockNotHeld
	}

	return nil
}

// WithRedlock 在 Redlock 锁的保护下执行函数（自动加锁、解锁）
func (r *Redlock) WithRedlock(ctx context.Context, key string, fn func() error) error {
	instance := r.NewInstance(key)

	// 获取锁
	if err := instance.Lock(ctx); err != nil {
		return err
	}

	// 确保函数返回时释放锁
	defer func() {
		if err := instance.Unlock(context.Background()); err != nil {
			// 记录日志但不返回错误（避免覆盖函数的错误）
			// TODO: 使用 zap 记录日志
		}
	}()

	// 执行函数
	return fn()
}

// WithRedlockRetry 在 Redlock 锁的保护下执行函数（带重试机制）
func (r *Redlock) WithRedlockRetry(ctx context.Context, key string, maxRetries int, retryDelay time.Duration, fn func() error) error {
	instance := r.NewInstance(key)

	// 获取锁（带重试）
	if err := instance.LockWithRetry(ctx, maxRetries, retryDelay); err != nil {
		return err
	}

	// 确保函数返回时释放锁
	defer func() {
		if err := instance.Unlock(context.Background()); err != nil {
			// 记录日志但不返回错误（避免覆盖函数的错误）
			// TODO: 使用 zap 记录日志
		}
	}()

	// 执行函数
	return fn()
}
