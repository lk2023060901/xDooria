package client

import "errors"

var (
	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("grpc/client: invalid config")

	// ErrClientNotConnected Client 未连接
	ErrClientNotConnected = errors.New("grpc/client: client not connected")

	// ErrClientAlreadyConnected Client 已连接
	ErrClientAlreadyConnected = errors.New("grpc/client: client already connected")

	// ErrDialFailed 连接失败
	ErrDialFailed = errors.New("grpc/client: dial failed")
)
