package grpc

import (
	"io"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/session"
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

	for {
		envelope, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		_ = s.PushRecv(envelope)
		a.handler.OnMessage(s, envelope)
	}
}