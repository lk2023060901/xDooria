package grpc

import (
	"context"
	"io"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/session"
)

// Connector 实现 session.Connector。
type Connector struct {
	client        common.CommonServiceClient
	sessionConfig *session.Config
	handler       session.SessionHandler
}

func NewConnector(client common.CommonServiceClient, sessCfg *session.Config, handler session.SessionHandler) *Connector {
	if handler == nil {
		handler = &session.NopSessionHandler{}
	}
	return &Connector{
		client:        client,
		sessionConfig: sessCfg,
		handler:       handler,
	}
}

func (c *Connector) Connect(ctx context.Context, addr string) (session.Session, error) {
	// gRPC 客户端通常通过 client 实例发起流请求
	stream, err := c.client.Stream(ctx)
	if err != nil {
		return nil, err
	}

	s := NewGRPCSession(stream, c.sessionConfig)
	c.handler.OnOpened(s)

	// 驱动读取循环
	go func() {
		for {
			envelope, err := s.Recv()
			if err != nil {
				if err == io.EOF {
					c.handler.OnClosed(s, nil)
				} else {
					c.handler.OnClosed(s, err)
				}
				return
			}
			_ = s.PushRecv(envelope)
			c.handler.OnMessage(s, envelope)
		}
	}()

	return s, nil
}
