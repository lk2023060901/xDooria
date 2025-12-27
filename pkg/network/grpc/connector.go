package grpc

import (
	"context"
	"io"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
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
	f := s.Framer()
	conc.Go(func() (struct{}, error) {
		for {
			envelope, err := s.Recv()
			if err != nil {
				if err == io.EOF {
					c.handler.OnClosed(s, nil)
				} else {
					c.handler.OnClosed(s, err)
				}
				return struct{}{}, nil
			}
			// 验证签名并解密/解压
			op, payload, err := f.Decode(envelope)
			if err != nil {
				c.handler.OnError(s, err)
				continue
			}
			// 构建解码后的 Envelope
			decodedEnv := &common.Envelope{
				Header:  &common.MessageHeader{Op: op},
				Payload: payload,
			}
			if err := s.PushRecv(decodedEnv); err != nil {
				c.handler.OnError(s, err)
				continue
			}
			c.handler.OnMessage(s, decodedEnv)
		}
	})

	return s, nil
}
