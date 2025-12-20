package sentry

import "errors"

var (
	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("sentry: invalid config")

	// ErrInvalidDSN 无效的 DSN
	ErrInvalidDSN = errors.New("sentry: invalid DSN")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("sentry: client closed")

	// ErrNilConfig 配置为空
	ErrNilConfig = errors.New("sentry: nil config")
)
