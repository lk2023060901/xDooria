package handler

import (
	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/framer"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/security"
	"github.com/lk2023060901/xdooria/pkg/session"
)

// GatewayHandler 处理客户端连接和消息。
type GatewayHandler struct {
	session.NopSessionHandler
	logger      logger.Logger
	framer      framer.Framer
	jwtMgr      *security.JWTManager
	loginClient common.CommonServiceClient
}

func NewGatewayHandler(l logger.Logger, fr framer.Framer, jwtMgr *security.JWTManager, client common.CommonServiceClient) *GatewayHandler {
	return &GatewayHandler{
		logger:      l.Named("gateway.handler"),
		framer:      fr,
		jwtMgr:      jwtMgr,
		loginClient: client,
	}
}

func (h *GatewayHandler) OnOpened(s session.Session) {
	h.logger.Info("client connected", "id", s.ID(), "addr", s.RemoteAddr())
}

func (h *GatewayHandler) OnClosed(s session.Session, err error) {
	h.logger.Info("client disconnected", "id", s.ID(), "error", err)
}

func (h *GatewayHandler) OnMessage(s session.Session, env *common.Envelope) {
	// 1. 使用 Framer 解码
	op, payload, err := h.framer.Decode(env)
	if err != nil {
		h.logger.Warn("failed to decode message", "id", s.ID(), "error", err)
		return
	}

	// 2. 处理授权请求
	if op == uint32(api.OpCode_OP_AUTH_REQ) {
		token := string(payload)
		claims, err := h.jwtMgr.ValidateToken(token)
		if err != nil {
			h.logger.Warn("token validation failed", "id", s.ID(), "error", err)
			_ = s.Close() // 验证失败，关闭连接
			return
		}

		h.logger.Info("client authenticated", "id", s.ID(), "uid", claims.Get("uid"))
		
		// 记录验证结果到 Session (可以通过 Metadata，如果 Session 支持)
		// 目前 Session 接口没有 Metadata，我们简单回显一个响应
		resp, _ := h.framer.Encode(uint32(api.OpCode_OP_AUTH_RES), []byte("auth_ok"))
		_ = s.Send(s.Context(), resp)
		return
	}

	h.logger.Debug("received message", "id", s.ID(), "op", op, "len", len(payload))
	
	// 业务逻辑转发...
	
	// 使用 Framer 编码响应
	resp, err := h.framer.Encode(op, []byte("gateway_ack"))
	if err != nil {
		h.logger.Error("failed to encode response", "error", err)
		return
	}
	
	_ = s.Send(s.Context(), resp)
}
