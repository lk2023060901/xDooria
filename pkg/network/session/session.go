// Package session 提供通用的会话管理框架。
package session

import (
	"context"
	"errors"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
)

// Session 定义会话的基础接口。
type Session interface {
	// ID 返回会话的唯一标识。
	ID() string
	// Send 发送消息信封。
	Send(ctx context.Context, env *common.Envelope) error
	// Close 关闭会话。
	Close() error
	// RemoteAddr 返回远程地址。
	RemoteAddr() string
	// Context 返回会话的 Context，用于生命周期管理。
	Context() context.Context
	// SendChan 返回发送通道。
	SendChan() chan *common.Envelope
	// RecvChan 返回接收通道。
	RecvChan() chan *common.Envelope
}

// BaseSession 提供 Session 接口的基础实现。
// 仅包含最基本的标识和生命周期管理。
type BaseSession struct {
	id         string
	remoteAddr string
	ctx        context.Context
	cancel     context.CancelFunc
	sendCh     chan *common.Envelope
	recvCh     chan *common.Envelope
	framer     framer.Framer
}

// NewBaseSession 创建一个新的基础会话。
func NewBaseSession(id string, remoteAddr string, cfg *Config) *BaseSession {
	// 使用 MergeConfig 确保配置完整
	newCfg, _ := config.MergeConfig(DefaultConfig(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	return &BaseSession{
		id:         id,
		remoteAddr: remoteAddr,
		ctx:        ctx,
		cancel:     cancel,
		sendCh:     make(chan *common.Envelope, newCfg.SendChannelSize),
		recvCh:     make(chan *common.Envelope, newCfg.RecvChannelSize),
		framer:     newCfg.Framer,
	}
}

// ID 返回会话 ID。
func (s *BaseSession) ID() string {
	return s.id
}

// RemoteAddr 返回远程地址。
func (s *BaseSession) RemoteAddr() string {
	return s.remoteAddr
}

// Context 返回会话的 Context。
func (s *BaseSession) Context() context.Context {
	return s.ctx
}

// Close 关闭会话并取消 Context，同时关闭 Channel。
func (s *BaseSession) Close() error {
	s.cancel()
	return nil
}

// SendChan 返回发送通道。
func (s *BaseSession) SendChan() chan *common.Envelope {
	return s.sendCh
}

// RecvChan 返回接收通道。
func (s *BaseSession) RecvChan() chan *common.Envelope {
	return s.recvCh
}

// PushRecv 将接收到的消息信封压入接收队列。
func (s *BaseSession) PushRecv(env *common.Envelope) error {
	select {
	case s.recvCh <- env:
		return nil
	case <-s.ctx.Done():
		return s.ctx.Err()
	default:
		return errors.New("recv channel full")
	}
}

// Framer 返回消息帧处理器。
func (s *BaseSession) Framer() framer.Framer {
	return s.framer
}