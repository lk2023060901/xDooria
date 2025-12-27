package websocket

import (
	"context"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
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
	conc.Go(func() (struct{}, error) {
		s.writeLoop()
		return struct{}{}, nil
	})
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
	f := s.Framer()
	for {
		select {
		case env := <-s.SendChan():
			signedEnv, err := f.Encode(env.Header.Op, env.Payload)
			if err != nil {
				continue
			}
			data, err := framer.Marshal(signedEnv)
			if err != nil {
				continue
			}
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
