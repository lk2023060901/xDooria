package postgres

import "errors"

var (
	// ErrNilConfig 配置为空
	ErrNilConfig = errors.New("postgres: config is nil")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("postgres: invalid config")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("postgres: client is closed")

	// ErrTxAlreadyCommitted 事务已提交
	ErrTxAlreadyCommitted = errors.New("postgres: transaction already committed")

	// ErrTxAlreadyRolledBack 事务已回滚
	ErrTxAlreadyRolledBack = errors.New("postgres: transaction already rolled back")

	// ErrNoRows 没有查询到数据
	ErrNoRows = errors.New("postgres: no rows in result set")
)
