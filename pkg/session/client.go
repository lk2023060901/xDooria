package session

import (
	"context"

	"github.com/lk2023060901/xdooria-proto-common"
)

// Client 通用的客户端实现。
type Client struct {
	connector Connector
	session   Session
	handler   SessionHandler
}

func NewClient(connector Connector, handler SessionHandler) *Client {
	return &Client{
		connector: connector,
		handler:   handler,
	}
}

// Connect 连接到服务端。
func (c *Client) Connect(ctx context.Context, addr string) (Session, error) {
	s, err := c.connector.Connect(ctx, addr)
	if err != nil {
		return nil, err
	}
	c.session = s
	
	// 这里通常需要启动一个读取循环来驱动 SessionHandler
	// 但具体的读取逻辑可能在 Session 内部 or Connector 中处理
	if c.handler != nil {
		c.handler.OnOpened(s)
	}
	
	return s, nil
}

// Session 返回当前会话。
func (c *Client) Session() Session {
	return c.session
}

// Close 关闭客户端。
func (c *Client) Close() error {
	if c.session != nil {
		return c.session.Close()
	}
	return nil
}

// Send 发送数据。
func (c *Client) Send(ctx context.Context, env *common.Envelope) error {
	if c.session == nil {
		return ErrSessionNotFound
	}
	return c.session.Send(ctx, env)
}