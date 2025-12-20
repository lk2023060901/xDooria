package etcd

import "errors"

var (
	// ErrKeyNotFound 键不存在
	ErrKeyNotFound = errors.New("etcd: key not found")

	// ErrLeaseNotFound 租约不存在
	ErrLeaseNotFound = errors.New("etcd: lease not found")

	// ErrLockTimeout 获取锁超时
	ErrLockTimeout = errors.New("etcd: lock timeout")

	// ErrLockFailed 获取锁失败
	ErrLockFailed = errors.New("etcd: lock failed")

	// ErrAlreadyLocked 已经持有锁
	ErrAlreadyLocked = errors.New("etcd: already locked")

	// ErrNotLocked 未持有锁
	ErrNotLocked = errors.New("etcd: not locked")

	// ErrElectionFailed 选举失败
	ErrElectionFailed = errors.New("etcd: election failed")

	// ErrNotLeader 不是 Leader
	ErrNotLeader = errors.New("etcd: not leader")

	// ErrWatchClosed 监听已关闭
	ErrWatchClosed = errors.New("etcd: watch closed")

	// ErrContextCanceled 上下文已取消
	ErrContextCanceled = errors.New("etcd: context canceled")

	// ErrTimeout 操作超时
	ErrTimeout = errors.New("etcd: operation timeout")

	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("etcd: invalid config")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("etcd: client closed")

	// ErrTxnFailed 事务执行失败
	ErrTxnFailed = errors.New("etcd: transaction failed")
)
