package session

import (
	"github.com/lk2023060901/xdooria/pkg/websocket"
)

// WebSocketSession 基于 WebSocket 连接的会话实现。
// 嵌入 websocket.Connection，直接复用其 ID() 方法。
type WebSocketSession struct {
	*websocket.Connection
}

// NewWebSocketSession 创建一个新的 WebSocket 会话。
func NewWebSocketSession(conn *websocket.Connection) *WebSocketSession {
	return &WebSocketSession{Connection: conn}
}
