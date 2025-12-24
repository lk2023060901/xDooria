// pkg/websocket/errors.go
package websocket

import "errors"

var (
	// 配置错误
	ErrInvalidConfig = errors.New("websocket: invalid config")
	ErrInvalidURL    = errors.New("websocket: invalid url")

	// 连接错误
	ErrConnectionClosed  = errors.New("websocket: connection closed")
	ErrConnectionTimeout = errors.New("websocket: connection timeout")
	ErrConnectionFailed  = errors.New("websocket: connection failed")
	ErrAlreadyConnected  = errors.New("websocket: already connected")
	ErrNotConnected      = errors.New("websocket: not connected")

	// 发送错误
	ErrSendQueueFull = errors.New("websocket: send queue full")
	ErrSendTimeout   = errors.New("websocket: send timeout")
	ErrMessageTooBig = errors.New("websocket: message too big")

	// 连接池错误
	ErrPoolFull            = errors.New("websocket: connection pool full")
	ErrPoolClosed          = errors.New("websocket: connection pool closed")
	ErrMaxConnectionsPerIP = errors.New("websocket: max connections per ip exceeded")

	// 心跳错误
	ErrHeartbeatTimeout = errors.New("websocket: heartbeat timeout")

	// 重连错误
	ErrReconnectFailed    = errors.New("websocket: reconnect failed")
	ErrMaxRetriesExceeded = errors.New("websocket: max retries exceeded")

	// 升级错误
	ErrUpgradeFailed = errors.New("websocket: upgrade failed")

	// TLS 错误
	ErrTLSConfigInvalid = errors.New("websocket: tls config invalid")

	// 序列化错误
	ErrSerializeFailed   = errors.New("websocket: serialize failed")
	ErrDeserializeFailed = errors.New("websocket: deserialize failed")
)
