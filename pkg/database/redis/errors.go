package redis

import "errors"

var (
	// ErrNilConfig 配置为空
	ErrNilConfig = errors.New("redis config is nil")

	// ErrInvalidConfig 配置无效（Standalone/Master-Slave/Cluster 必须且只能配置一种）
	ErrInvalidConfig = errors.New("invalid redis config: must specify exactly one of standalone, master-slave, or cluster mode")

	// ErrNil Redis 返回 nil（键不存在）
	ErrNil = errors.New("redis: nil")

	// ErrLockFailed 获取锁失败
	ErrLockFailed = errors.New("redis: failed to acquire lock")

	// ErrLockNotHeld 锁未持有（解锁时发现锁不存在或已被其他持有者占用）
	ErrLockNotHeld = errors.New("redis: lock not held")

	// ErrInvalidSlaveLoadBalance 无效的从库负载均衡策略
	ErrInvalidSlaveLoadBalance = errors.New("invalid slave load balance strategy: must be 'random' or 'round_robin'")

	// ErrNoSlaves 没有配置从库
	ErrNoSlaves = errors.New("no slave nodes configured")

	// ErrRedlockFailed Redlock 获取锁失败（未能在多数节点上获取锁）
	ErrRedlockFailed = errors.New("redlock: failed to acquire lock on majority of nodes")
)
