package websocket

import (
	"context"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/framer"
	"github.com/lk2023060901/xdooria/pkg/session"
)

// WebSocketSession 基于 WebSocket 连接的会话实现。
type WebSocketSession struct {
	*session.BaseSession
	conn *Connection
}

// NewWebSocketSession 创建一个新的 WebSocket 会话。
func NewWebSocketSession(conn *Connection, cfg *session.Config) *WebSocketSession {
	s := &WebSocketSession{
		BaseSession: session.NewBaseSession(conn.ID(), conn.RemoteAddr(), cfg),
		conn:        conn,
	}
	go s.writeLoop()
	return s
}

// Send 发送消息信封，压入发送队列。
func (s *WebSocketSession) Send(ctx context.Context, env *common.Envelope) error {
	select {
	case s.SendChan() <- env:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.Context().Done():
		return session.ErrConnectionClosed
	}
}

func (s *WebSocketSession) writeLoop() {
	ctx := s.Context()
	for {
		select {
		case env := <-s.SendChan():
			data, err := framer.Marshal(env)
			if err != nil {
				continue
			}
			// 使用 Connection 原有的异步发送
			_ = s.conn.SendAsync(NewMessage(data))
		case <-ctx.Done():
			return
		}
	}
}

// Close 关闭会话。
func (s *WebSocketSession) Close() error {
	_ = s.BaseSession.Close()
	return s.conn.Close()
}

// UnderlyingConn 返回底层连接。
func (s *WebSocketSession) UnderlyingConn() *Connection {
	return s.conn
}
