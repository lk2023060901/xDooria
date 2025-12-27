package websocket

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
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
	conc.Go(func() (struct{}, error) {
		conn.ReadLoop(func(c_ *Connection, m *Message) error {
			env, err := framer.Unmarshal(m.Data)
			if err != nil {
				c.handler.OnError(s, err)
				return nil
			}
			// 验证签名并解密/解压
			op, payload, err := s.Framer().Decode(env)
			if err != nil {
				c.handler.OnError(s, err)
				return nil
			}
			// 构建解码后的 Envelope
			decodedEnv := &common.Envelope{
				Header:  &common.MessageHeader{Op: op},
				Payload: payload,
			}
			if err := s.PushRecv(decodedEnv); err != nil {
				c.handler.OnError(s, err)
				return nil
			}
			c.handler.OnMessage(s, decodedEnv)
			return nil
		})
		c.handler.OnClosed(s, conn.CloseError())
		return struct{}{}, nil
	})

	return s, nil
}