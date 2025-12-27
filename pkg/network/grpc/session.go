package grpc

import (
	"context"

	"github.com/google/uuid"
	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"google.golang.org/grpc/peer"
)

// GRPCStream 定义了 gRPC 双向流的通用接口（兼容 Server 和 Client 流）。
type GRPCStream interface {
	Send(*common.Envelope) error
	Recv() (*common.Envelope, error)
	Context() context.Context
}

// GRPCSession 基于 gRPC 双向流的会话实现。
type GRPCSession struct {
	*session.BaseSession
	stream GRPCStream
}

// NewGRPCSession 创建一个新的 gRPC 会话。
func NewGRPCSession(stream GRPCStream, cfg *session.Config) *GRPCSession {
	id := uuid.New().String()
	remoteAddr := "unknown"
	if p, ok := peer.FromContext(stream.Context()); ok {
		remoteAddr = p.Addr.String()
	}

	s := &GRPCSession{
		BaseSession: session.NewBaseSession(id, remoteAddr, cfg),
		stream:      stream,
	}
	conc.Go(func() (struct{}, error) {
		s.writeLoop()
		return struct{}{}, nil
	})
	return s
}

// Send 发送消息信封，压入发送队列。
func (s *GRPCSession) Send(ctx context.Context, env *common.Envelope) error {
	select {
	case s.SendChan() <- env:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-s.Context().Done():
		return session.ErrConnectionClosed
	}
}

func (s *GRPCSession) writeLoop() {
	ctx := s.Context()
	f := s.Framer()
	for {
		select {
		case env := <-s.SendChan():
			signedEnv, err := f.Encode(env.Header.Op, env.Payload)
			if err != nil {
				continue
			}
			_ = s.stream.Send(signedEnv)
		case <-ctx.Done():
			return
		}
	}
}

// Recv 接收 Envelope。
func (s *GRPCSession) Recv() (*common.Envelope, error) {
	return s.stream.Recv()
}

// Close 关闭会话。
func (s *GRPCSession) Close() error {
	_ = s.BaseSession.Close()
	return nil
}

// UnderlyingStream 返回底层 gRPC 流。
func (s *GRPCSession) UnderlyingStream() GRPCStream {
	return s.stream
}
