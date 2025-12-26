package session

import (
	"context"

	"github.com/lk2023060901/xdooria-proto-common"
)

// Acceptor 监听并接受连接的接口。
type Acceptor interface {
	// Start 启动监听。
	Start() error
	// Stop 停止监听。
	Stop() error
}

// Connector 发起连接的接口。
type Connector interface {
	// Connect 发起连接并返回会话。
	Connect(ctx context.Context, addr string) (Session, error)
}

// SessionHandler 会话事件处理器。

type SessionHandler interface {

	// OnOpened 会话建立回调。

	OnOpened(s Session)

	// OnClosed 会话关闭回调。

	OnClosed(s Session, err error)

	// OnMessage 收到消息回调。

	OnMessage(s Session, env *common.Envelope)

}



// NopSessionHandler 提供 SessionHandler 的空实现。

type NopSessionHandler struct{}



func (n *NopSessionHandler) OnOpened(s Session)                  {}

func (n *NopSessionHandler) OnClosed(s Session, err error)       {}

func (n *NopSessionHandler) OnMessage(s Session, env *common.Envelope) {}



var _ SessionHandler = (*NopSessionHandler)(nil)
