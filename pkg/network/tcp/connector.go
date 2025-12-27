package tcp

import (
	"context"
	"sync"
	"time"

	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/panjf2000/gnet/v2"
)

// Connector 实现 session.Connector 接口，基于 gnet 客户端。
type Connector struct {
	config        *ClientConfig
	sessionConfig *session.Config
	handler       session.SessionHandler

	client  *gnet.Client
	session *TCPSession

	mu      sync.Mutex
	started bool
}

// NewConnector 创建一个新的 TCP 连接器。
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

// Start 启动 gnet 客户端事件循环。
func (c *Connector) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.started {
		return nil
	}

	client, err := gnet.NewClient(
		c,
		gnet.WithReadBufferCap(c.config.ReadBufferSize),
		gnet.WithWriteBufferCap(c.config.WriteBufferSize),
		gnet.WithTCPKeepAlive(c.config.TCPKeepAlive),
		gnet.WithTCPNoDelay(gnet.TCPNoDelay),
	)
	if err != nil {
		return err
	}

	c.client = client
	c.started = true

	return client.Start()
}

// Stop 停止 gnet 客户端事件循环。
func (c *Connector) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.started || c.client == nil {
		return nil
	}

	c.started = false
	return c.client.Stop()
}

// Connect 发起连接并返回会话。
func (c *Connector) Connect(ctx context.Context, addr string) (session.Session, error) {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		if err := c.Start(); err != nil {
			return nil, err
		}
	} else {
		c.mu.Unlock()
	}

	conn, err := c.client.Dial(c.config.Network, addr)
	if err != nil {
		return nil, err
	}

	c.session = NewTCPSession(conn, c.sessionConfig)
	c.handler.OnOpened(c.session)

	return c.session, nil
}

// Session 返回当前会话。
func (c *Connector) Session() *TCPSession {
	return c.session
}

// OnBoot 实现 gnet.EventHandler。
func (c *Connector) OnBoot(eng gnet.Engine) gnet.Action {
	return gnet.None
}

// OnShutdown 实现 gnet.EventHandler。
func (c *Connector) OnShutdown(eng gnet.Engine) {
}

// OnOpen 实现 gnet.EventHandler。
func (c *Connector) OnOpen(conn gnet.Conn) (out []byte, action gnet.Action) {
	return nil, gnet.None
}

// OnClose 实现 gnet.EventHandler。
func (c *Connector) OnClose(conn gnet.Conn, err error) gnet.Action {
	if c.session != nil {
		c.handler.OnClosed(c.session, err)
	}
	return gnet.None
}

// OnTraffic 实现 gnet.EventHandler。
func (c *Connector) OnTraffic(conn gnet.Conn) gnet.Action {
	if c.session == nil {
		return gnet.None
	}

	data, _ := conn.Next(-1)
	if len(data) == 0 {
		return gnet.None
	}

	env, err := framer.Unmarshal(data)
	if err != nil {
		c.handler.OnError(c.session, err)
		return gnet.None
	}
	// 验证签名并解密/解压
	op, payload, err := c.session.Framer().Decode(env)
	if err != nil {
		c.handler.OnError(c.session, err)
		return gnet.None
	}
	// 构建解码后的 Envelope
	decodedEnv := &common.Envelope{
		Header:  &common.MessageHeader{Op: op},
		Payload: payload,
	}
	if err := c.session.PushRecv(decodedEnv); err != nil {
		c.handler.OnError(c.session, err)
		return gnet.None
	}
	c.handler.OnMessage(c.session, decodedEnv)

	return gnet.None
}

// OnTick 实现 gnet.EventHandler。
func (c *Connector) OnTick() (delay time.Duration, action gnet.Action) {
	return 0, gnet.None
}
