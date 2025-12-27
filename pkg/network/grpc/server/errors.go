package server

import "errors"

var (
	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("grpc/server: invalid config")

	// ErrServerNotStarted Server 未启动
	ErrServerNotStarted = errors.New("grpc/server: server not started")

	// ErrServerAlreadyStarted Server 已启动
	ErrServerAlreadyStarted = errors.New("grpc/server: server already started")

	// ErrServiceRegistration 服务注册失败
	ErrServiceRegistration = errors.New("grpc/server: service registration failed")
)
