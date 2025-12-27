package grpc

import (
	"io"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/session"
)

// Acceptor 实现 session.Acceptor 和 common.CommonServiceServer 接口。
type Acceptor struct {
	common.UnimplementedCommonServiceServer
	sessionConfig *session.Config
	handler       session.SessionHandler
}

func NewAcceptor(sessCfg *session.Config, handler session.SessionHandler) *Acceptor {
	if handler == nil {
		handler = &session.NopSessionHandler{}
	}
	return &Acceptor{
		sessionConfig: sessCfg,
		handler:       handler,
	}
}

func (a *Acceptor) Start() error {
	// gRPC Acceptor 通常作为服务注册到 grpc.Server 中
	return nil
}

func (a *Acceptor) Stop() error {
	return nil
}

// Stream 实现 common.CommonServiceServer。
func (a *Acceptor) Stream(stream common.CommonService_StreamServer) error {
	s := NewGRPCSession(stream, a.sessionConfig)
	a.handler.OnOpened(s)

	defer func() {
		a.handler.OnClosed(s, nil)
	}()

	f := s.Framer()
	for {
		envelope, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		// 验证签名并解密/解压
		op, payload, err := f.Decode(envelope)
		if err != nil {
			a.handler.OnError(s, err)
			continue
		}
		// 构建解码后的 Envelope
		decodedEnv := &common.Envelope{
			Header:  &common.MessageHeader{Op: op},
			Payload: payload,
		}
		if err := s.PushRecv(decodedEnv); err != nil {
			a.handler.OnError(s, err)
			continue
		}
		a.handler.OnMessage(s, decodedEnv)
	}
}