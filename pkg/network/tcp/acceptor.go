package tcp

import (
	"context"
	"fmt"
	"time"

	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/panjf2000/gnet/v2"
)

// Acceptor 实现 session.Acceptor 接口。
type Acceptor struct {
	gnet.BuiltinEventEngine
	config        *ServerConfig
	sessionConfig *session.Config
	handler       session.SessionHandler
	engine        gnet.Engine
	started       bool
}

func NewAcceptor(cfg *ServerConfig, sessCfg *session.Config, handler session.SessionHandler) *Acceptor {
	if handler == nil {
		handler = &session.NopSessionHandler{}
	}
	return &Acceptor{
		config:        cfg,
		sessionConfig: sessCfg,
		handler:       handler,
	}
}

func (a *Acceptor) Start() error {
	opts := []gnet.Option{
		gnet.WithMulticore(a.config.Multicore),
		gnet.WithReusePort(a.config.ReusePort),
		gnet.WithReuseAddr(a.config.ReuseAddr),
		gnet.WithTCPKeepAlive(a.config.TCPKeepAlive),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
	}
	if a.config.NumEventLoop > 0 {
		opts = append(opts, gnet.WithNumEventLoop(a.config.NumEventLoop))
	}

	protoAddr := fmt.Sprintf("%s://%s", a.config.Network, a.config.Addr)
	
	errCh := make(chan error, 1)
	conc.Go(func() (struct{}, error) {
		errCh <- gnet.Run(a, protoAddr, opts...)
		return struct{}{}, nil
	})

	// 等待一小段时间看是否启动失败
	select {
	case err := <-errCh:
		return err
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (a *Acceptor) Stop() error {
	if a.started {
		return a.engine.Stop(context.Background())
	}
	return nil
}

// OnBoot 实现 gnet.EventHandler。
func (a *Acceptor) OnBoot(eng gnet.Engine) (action gnet.Action) {
	a.engine = eng
	a.started = true
	return gnet.None
}

// OnOpen 实现 gnet.EventHandler。
func (a *Acceptor) OnOpen(c gnet.Conn) (out []byte, action gnet.Action) {
	s := NewTCPSession(c, a.sessionConfig)
	c.SetContext(s)
	a.handler.OnOpened(s)
	return nil, gnet.None
}

// OnClose 实现 gnet.EventHandler。
func (a *Acceptor) OnClose(c gnet.Conn, err error) (action gnet.Action) {
	if s, ok := c.Context().(*TCPSession); ok {
		a.handler.OnClosed(s, err)
	}
	return
}

// OnTraffic 实现 gnet.EventHandler。
func (a *Acceptor) OnTraffic(c gnet.Conn) gnet.Action {
	data, _ := c.Next(-1)
	if s, ok := c.Context().(*TCPSession); ok {
		env, err := framer.Unmarshal(data)
		if err != nil {
			a.handler.OnError(s, err)
			return gnet.None
		}
		// 验证签名并解密/解压
		op, payload, err := s.Framer().Decode(env)
		if err != nil {
			a.handler.OnError(s, err)
			return gnet.None
		}
		// 构建解码后的 Envelope
		decodedEnv := &common.Envelope{
			Header:  &common.MessageHeader{Op: op},
			Payload: payload,
		}
		if err := s.PushRecv(decodedEnv); err != nil {
			a.handler.OnError(s, err)
			return gnet.None
		}
		a.handler.OnMessage(s, decodedEnv)
	}
	return gnet.None
}