package tcp

import (
	"context"

	"github.com/google/uuid"
	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/panjf2000/gnet/v2"
)

// TCPSession 基于 gnet 连接的会话实现，服务端和客户端通用。
type TCPSession struct {
	*session.BaseSession
	conn gnet.Conn
}

// NewTCPSession 创建一个新的 gnet TCP 会话。
func NewTCPSession(conn gnet.Conn, cfg *session.Config) *TCPSession {
	id := uuid.New().String()
	s := &TCPSession{
		BaseSession: session.NewBaseSession(id, conn.RemoteAddr().String(), cfg),
		conn:        conn,
	}
	conc.Go(func() (struct{}, error) {
		s.writeLoop()
		return struct{}{}, nil
	})
	return s
}

// Send 发送消息信封，压入发送队列。
func (s *TCPSession) Send(ctx context.Context, env *common.Envelope) error {
	select {
	case s.SendChan() <- env:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.Context().Done():
		return session.ErrConnectionClosed
	}
}

func (s *TCPSession) writeLoop() {
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
			_ = s.conn.AsyncWrite(data, nil)
		case <-ctx.Done():
			return
		}
	}
}

// Close 关闭会话。
func (s *TCPSession) Close() error {
	_ = s.BaseSession.Close()
	return s.conn.Close()
}

// Conn 返回底层 gnet 连接。
func (s *TCPSession) Conn() gnet.Conn {
	return s.conn
}
