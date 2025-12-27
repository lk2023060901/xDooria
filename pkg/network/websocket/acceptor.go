package websocket

import (
	"net/http"

	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// Acceptor 实现 session.Acceptor 接口。
type Acceptor struct {
	server        *Server
	sessionConfig *session.Config
	handler       session.SessionHandler
	addr          string
}

func NewAcceptor(server *Server, sessCfg *session.Config, addr string, handler session.SessionHandler) *Acceptor {
	if handler == nil {
		handler = &session.NopSessionHandler{}
	}
	return &Acceptor{
		server:        server,
		sessionConfig: sessCfg,
		handler:       handler,
		addr:          addr,
	}
}

func (a *Acceptor) Start() error {
	// 这里的 Start 可能需要一个 http server。
	// 简单实现：
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := a.server.Upgrade(w, r)
		if err != nil {
			return
		}
		s := NewWebSocketSession(conn, a.sessionConfig)
		a.handler.OnOpened(s)
		
		// 驱动读取循环
		conc.Go(func() (struct{}, error) {
			conn.ReadLoop(func(c *Connection, m *Message) error {
				env, err := framer.Unmarshal(m.Data)
				if err != nil {
					a.handler.OnError(s, err)
					return nil
				}
				// 验证签名并解密/解压
				op, payload, err := s.Framer().Decode(env)
				if err != nil {
					a.handler.OnError(s, err)
					return nil
				}
				// 构建解码后的 Envelope
				decodedEnv := &common.Envelope{
					Header:  &common.MessageHeader{Op: op},
					Payload: payload,
				}
				if err := s.PushRecv(decodedEnv); err != nil {
					a.handler.OnError(s, err)
					return nil
				}
				a.handler.OnMessage(s, decodedEnv)
				return nil
			})
			a.handler.OnClosed(s, conn.CloseError())
			return struct{}{}, nil
		})
	})
	
	return http.ListenAndServe(a.addr, mux)
}

func (a *Acceptor) Stop() error {
	return a.server.Close()
}