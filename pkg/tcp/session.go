package tcp

import (
	"context"
	"net"

	"github.com/google/uuid"
	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/framer"
	"github.com/lk2023060901/xdooria/pkg/session"
	"github.com/panjf2000/gnet/v2"
)

// TCPSession 基于 gnet 或 net 连接的会话实现。
type TCPSession struct {
	*session.BaseSession
	gnetConn gnet.Conn
	netConn  net.Conn
}

// NewTCPSession 创建一个新的 gnet TCP 会话。
func NewTCPSession(conn gnet.Conn, cfg *session.Config) *TCPSession {
	id := uuid.New().String()
	s := &TCPSession{
		BaseSession: session.NewBaseSession(id, conn.RemoteAddr().String(), cfg),
		gnetConn:    conn,
	}
	go s.writeLoop()
	return s
}

// NewNetTCPSession 创建一个新的 net.Conn TCP 会话。
func NewNetTCPSession(conn net.Conn, cfg *session.Config) *TCPSession {
	id := uuid.New().String()
	s := &TCPSession{
		BaseSession: session.NewBaseSession(id, conn.RemoteAddr().String(), cfg),
		netConn:     conn,
	}
	go s.writeLoop()
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
	for {
		select {
		case env := <-s.SendChan():
			data, err := framer.Marshal(env)
			if err != nil {
				continue
			}
			if s.gnetConn != nil {
				_ = s.gnetConn.AsyncWrite(data, nil)
			} else if s.netConn != nil {
				_, _ = s.netConn.Write(data)
			}
		case <-ctx.Done():
			return
		}
	}
}

// Close 关闭会话。
func (s *TCPSession) Close() error {
	_ = s.BaseSession.Close()
	if s.gnetConn != nil {
		return s.gnetConn.Close()
	}
	if s.netConn != nil {
		return s.netConn.Close()
	}
	return nil
}

// UnderlyingConn 返回底层连接。
func (s *TCPSession) UnderlyingConn() any {
	if s.gnetConn != nil {
		return s.gnetConn
	}
	return s.netConn
}
