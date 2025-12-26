package tcp

import "errors"

var (
	// 配置错误
	ErrInvalidConfig = errors.New("tcp: invalid config")
	ErrInvalidAddr   = errors.New("tcp: invalid address")

	// 连接错误
	ErrConnectionClosed  = errors.New("tcp: connection closed")
	ErrConnectionTimeout = errors.New("tcp: connection timeout")
	ErrConnectionFailed  = errors.New("tcp: connection failed")
	ErrAlreadyConnected  = errors.New("tcp: already connected")
	ErrNotConnected      = errors.New("tcp: not connected")

	// 发送错误
	ErrSendQueueFull = errors.New("tcp: send queue full")
	ErrSendTimeout   = errors.New("tcp: send timeout")
	ErrMessageTooBig = errors.New("tcp: message too big")

	// 服务器错误
	ErrServerClosed         = errors.New("tcp: server closed")
	ErrServerAlreadyStarted = errors.New("tcp: server already started")
	ErrServerNotStarted     = errors.New("tcp: server not started")

	// 编解码错误
	ErrInvalidPacket   = errors.New("tcp: invalid packet")
	ErrPacketTooLarge  = errors.New("tcp: packet too large")
	ErrIncompleteFrame = errors.New("tcp: incomplete frame")
)
