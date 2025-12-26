package websocket

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
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
	// 使用项目现有的 Client 或直接使用 gorilla
	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: c.config.DialTimeout,
		ReadBufferSize:   c.config.ReadBufferSize,
		WriteBufferSize:  c.config.WriteBufferSize,
	}

	header := make(http.Header)
	for k, v := range c.config.Headers {
		header.Set(k, v)
	}

	wsConn, _, err := dialer.DialContext(ctx, addr, header)
	if err != nil {
		return nil, err
	}

	// 封装成项目内部的 Connection
	conn := NewConnection(wsConn)
	s := NewWebSocketSession(conn, c.sessionConfig)

	c.handler.OnOpened(s)

	// 驱动读取
	go func() {
		conn.ReadLoop(func(c_ *Connection, m *Message) error {
			env, err := framer.Unmarshal(m.Data)
			if err == nil {
				_ = s.PushRecv(env)
				c.handler.OnMessage(s, env)
			}
			return nil
		})
		c.handler.OnClosed(s, conn.CloseError())
	}()

	return s, nil
}