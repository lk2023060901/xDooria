package tcp

import (
	"context"
	"net"

	"github.com/lk2023060901/xdooria/pkg/framer"
	"github.com/lk2023060901/xdooria/pkg/session"
)

// Connector 实现 session.Connector 接口。
type Connector struct {
	config        *ClientConfig
	sessionConfig *session.Config
	handler       session.SessionHandler
}

func NewConnector(cfg *ClientConfig, sessCfg *session.Config, handler session.SessionHandler) *Connector {
	if handler == nil {
		handler = &session.NopSessionHandler{}
	}
	return &Connector{
		config:        cfg,
		sessionConfig: sessCfg,
		handler:       handler,
	}
}

func (c *Connector) Connect(ctx context.Context, addr string) (session.Session, error) {
	d := net.Dialer{
		Timeout:   c.config.DialTimeout,
		KeepAlive: c.config.TCPKeepAlive,
	}
	conn, err := d.DialContext(ctx, c.config.Network, addr)
	if err != nil {
		return nil, err
	}

	if tcpConn, ok := conn.(*net.TCPConn); ok {
		_ = tcpConn.SetNoDelay(c.config.TCPNoDelay)
		if c.config.ReadBufferSize > 0 {
			_ = tcpConn.SetReadBuffer(c.config.ReadBufferSize)
		}
		if c.config.WriteBufferSize > 0 {
			_ = tcpConn.SetWriteBuffer(c.config.WriteBufferSize)
		}
	}

	s := NewNetTCPSession(conn, c.sessionConfig)
	
	c.handler.OnOpened(s)

	// 启动后台读取循环驱动 handler
	go c.readLoop(s, conn)

	return s, nil
}

func (c *Connector) readLoop(s *TCPSession, conn net.Conn) {
	defer func() {
		_ = s.Close()
	}()

	buf := make([]byte, c.config.ReadBufferSize)
	if len(buf) == 0 {
		buf = make([]byte, 4096)
	}

	for {
		n, err := conn.Read(buf)
		if err != nil {
			c.handler.OnClosed(s, err)
			return
		}
		if n > 0 {
			data := make([]byte, n)
			copy(data, buf[:n])
			env, err := framer.Unmarshal(data)
			if err == nil {
				_ = s.PushRecv(env)
				c.handler.OnMessage(s, env)
			}
		}
	}
}
