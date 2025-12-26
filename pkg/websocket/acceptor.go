package websocket

import (
	"net/http"

	"github.com/lk2023060901/xdooria/pkg/framer"
	"github.com/lk2023060901/xdooria/pkg/session"
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
		go func() {
			conn.ReadLoop(func(c *Connection, m *Message) error {
				env, err := framer.Unmarshal(m.Data)
				if err == nil {
					_ = s.PushRecv(env)
					a.handler.OnMessage(s, env)
				}
				return nil
			})
			a.handler.OnClosed(s, conn.CloseError())
		}()
	})
	
	return http.ListenAndServe(a.addr, mux)
}

func (a *Acceptor) Stop() error {
	return a.server.Close()
}