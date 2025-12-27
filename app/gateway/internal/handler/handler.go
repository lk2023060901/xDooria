package handler

import (
	"context"

	"google.golang.org/protobuf/proto"

	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria-proto-api/gateway"
	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
)

// GatewayHandler 处理客户端连接和消息。
type GatewayHandler struct {
	session.NopSessionHandler
	logger    logger.Logger
	jwtMgr    *security.JWTManager
	processor router.Processor
}

func NewGatewayHandler(l logger.Logger, jwtMgr *security.JWTManager, p router.Processor) *GatewayHandler {
	return &GatewayHandler{
		logger:    l.Named("gateway.handler"),
		jwtMgr:    jwtMgr,
		processor: p,
	}
}

func (h *GatewayHandler) OnOpened(s session.Session) {
	h.logger.Info("client connected", "id", s.ID(), "addr", s.RemoteAddr())
}

func (h *GatewayHandler) OnClosed(s session.Session, err error) {
	h.logger.Info("client disconnected", "id", s.ID(), "error", err)
}

func (h *GatewayHandler) OnMessage(s session.Session, env *common.Envelope) {
	op := env.Header.Op
	payload := env.Payload

	switch op {
	case uint32(api.OpCode_OP_AUTH_REQ):
		h.handleAuth(s, payload)
		return
	case uint32(api.OpCode_OP_RECONNECT_REQ):
		h.handleReconnect(s, payload)
		return
	}

	h.logger.Debug("received message", "id", s.ID(), "op", op, "len", len(payload))

	// 使用 Processor 路由到对应的 Handler
	respOp, respPayload, err := h.processor.Process(context.Background(), op, payload)
	if err != nil {
		h.logger.Error("process message failed", "id", s.ID(), "op", op, "error", err)
		return
	}

	// 发送响应
	respEnv := &common.Envelope{
		Header:  &common.MessageHeader{Op: respOp},
		Payload: respPayload,
	}
	if err := s.Send(s.Context(), respEnv); err != nil {
		h.logger.Error("send response failed", "id", s.ID(), "error", err)
	}
}

// handleAuth 处理首次认证请求
func (h *GatewayHandler) handleAuth(s session.Session, payload []byte) {
	// 解析 AuthRequest
	req := &gateway.AuthRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Warn("failed to unmarshal AuthRequest", "id", s.ID(), "error", err)
		h.sendAuthResponse(s, uint32(api.ErrorCode_ERR_INTERNAL), "")
		return
	}

	// 验证 Login 签发的 token
	claims, err := h.jwtMgr.ValidateToken(req.LoginToken)
	if err != nil {
		h.logger.Warn("token validation failed", "id", s.ID(), "error", err)
		code := uint32(api.ErrorCode_ERR_TOKEN_INVALID)
		if err == security.ErrTokenExpired {
			code = uint32(api.ErrorCode_ERR_TOKEN_EXPIRED)
		}
		h.sendAuthResponse(s, code, "")
		return
	}

	// 生成 Gateway 的 SessionToken（使用 Gateway 配置的过期时间）
	sessionToken, err := h.jwtMgr.GenerateToken(&security.Claims{
		Payload: claims.Payload, // 继承 uid 等信息
	})
	if err != nil {
		h.logger.Error("failed to generate session token", "id", s.ID(), "error", err)
		h.sendAuthResponse(s, uint32(api.ErrorCode_ERR_INTERNAL), "")
		return
	}

	h.logger.Info("client authenticated", "id", s.ID(), "uid", claims.Get("uid"))
	h.sendAuthResponse(s, uint32(api.ErrorCode_ERR_SUCCESS), sessionToken)
}

// handleReconnect 处理重连请求
func (h *GatewayHandler) handleReconnect(s session.Session, payload []byte) {
	// 解析 ReconnectRequest
	req := &gateway.ReconnectRequest{}
	if err := proto.Unmarshal(payload, req); err != nil {
		h.logger.Warn("failed to unmarshal ReconnectRequest", "id", s.ID(), "error", err)
		h.sendReconnectResponse(s, uint32(api.ErrorCode_ERR_INTERNAL), "")
		return
	}

	// 验证 SessionToken
	claims, err := h.jwtMgr.ValidateToken(req.Token)
	if err != nil {
		h.logger.Warn("reconnect token validation failed", "id", s.ID(), "error", err)
		code := uint32(api.ErrorCode_ERR_TOKEN_INVALID)
		if err == security.ErrTokenExpired {
			code = uint32(api.ErrorCode_ERR_TOKEN_EXPIRED)
		}
		h.sendReconnectResponse(s, code, "")
		return
	}

	// 续期：生成新的 SessionToken
	newToken, err := h.jwtMgr.GenerateToken(&security.Claims{
		Payload: claims.Payload,
	})
	if err != nil {
		h.logger.Error("failed to generate new session token", "id", s.ID(), "error", err)
		h.sendReconnectResponse(s, uint32(api.ErrorCode_ERR_INTERNAL), "")
		return
	}

	h.logger.Info("client reconnected", "id", s.ID(), "uid", claims.Get("uid"))
	h.sendReconnectResponse(s, uint32(api.ErrorCode_ERR_SUCCESS), newToken)
}

// sendAuthResponse 发送认证响应
func (h *GatewayHandler) sendAuthResponse(s session.Session, code uint32, token string) {
	resp := &gateway.AuthResponse{
		Code:  code,
		Token: token,
	}
	data, _ := proto.Marshal(resp)

	env := &common.Envelope{
		Header:  &common.MessageHeader{Op: uint32(api.OpCode_OP_AUTH_RES)},
		Payload: data,
	}
	_ = s.Send(s.Context(), env)
}

// sendReconnectResponse 发送重连响应
func (h *GatewayHandler) sendReconnectResponse(s session.Session, code uint32, token string) {
	resp := &gateway.ReconnectResponse{
		Code:  code,
		Token: token,
	}
	data, _ := proto.Marshal(resp)

	env := &common.Envelope{
		Header:  &common.MessageHeader{Op: uint32(api.OpCode_OP_RECONNECT_RES)},
		Payload: data,
	}
	_ = s.Send(s.Context(), env)
}

func (h *GatewayHandler) OnError(s session.Session, err error) {
	h.logger.Warn("session error", "id", s.ID(), "error", err)
}
