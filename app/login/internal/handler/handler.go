package handler

import (
	"context"

	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/router"
)

// LoginHandler 处理 TCP 连接和消息。
type LoginHandler struct {
	session.NopSessionHandler
	logger    logger.Logger
	processor router.Processor
}

func NewLoginHandler(l logger.Logger, p router.Processor) *LoginHandler {
	return &LoginHandler{
		logger:    l.Named("login.handler"),
		processor: p,
	}
}

func (h *LoginHandler) OnOpened(s session.Session) {
	h.logger.Debug("client connected", "id", s.ID(), "addr", s.RemoteAddr())
}

func (h *LoginHandler) OnClosed(s session.Session, err error) {
	h.logger.Debug("client disconnected", "id", s.ID(), "error", err)
}

func (h *LoginHandler) OnMessage(s session.Session, env *common.Envelope) {
	// env 已经被 Acceptor 解码，直接使用 Op 和 Payload
	op := env.Header.Op
	payload := env.Payload

	h.logger.Debug("received message", "id", s.ID(), "op", op)

	// 使用 Processor 路由到对应的 Handler
	respOp, respPayload, err := h.processor.Process(context.Background(), op, payload)
	if err != nil {
		h.logger.Error("process message failed", "id", s.ID(), "op", op, "error", err)
		return
	}

	// 发送响应，writeLoop 会自动 Encode
	respEnv := &common.Envelope{
		Header:  &common.MessageHeader{Op: respOp},
		Payload: respPayload,
	}
	if err := s.Send(s.Context(), respEnv); err != nil {
		h.logger.Error("send response failed", "id", s.ID(), "error", err)
	}
}

func (h *LoginHandler) OnError(s session.Session, err error) {
	h.logger.Warn("session error", "id", s.ID(), "error", err)
}
