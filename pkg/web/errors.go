package web

import "errors"

var (
	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("web: invalid config")

	// ErrServerNotStarted Server 未启动
	ErrServerNotStarted = errors.New("web: server not started")

	// ErrServerAlreadyStarted Server 已启动
	ErrServerAlreadyStarted = errors.New("web: server already started")

	// ErrServiceRegistration 服务注册失败
	ErrServiceRegistration = errors.New("web: service registration failed")
)
