package handler

import (
	"context"

	gamepb "github.com/lk2023060901/xdooria-proto-internal/game"
	"github.com/lk2023060901/xdooria/app/game/internal/service"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// GameHandler gRPC 处理器
type GameHandler struct {
	gamepb.UnimplementedGameServiceServer
	logger     logger.Logger
	roleSvc    *service.RoleService
	messageSvc *service.MessageService
}

// NewGameHandler 创建 gRPC 处理器
func NewGameHandler(
	l logger.Logger,
	roleSvc *service.RoleService,
	messageSvc *service.MessageService,
) *GameHandler {
	return &GameHandler{
		logger:     l.Named("handler.game"),
		roleSvc:    roleSvc,
		messageSvc: messageSvc,
	}
}

// ForwardMessage 处理从 Gateway 转发的消息
func (h *GameHandler) ForwardMessage(ctx context.Context, req *gamepb.ForwardMessageRequest) (*gamepb.ForwardMessageResponse, error) {
	h.logger.Debug("received forward message request",
		"role_id", req.RoleId,
		"op_code", req.OpCode,
	)

	resp, err := h.messageSvc.HandleMessage(ctx, req.RoleId, req.OpCode, req.Payload)
	if err != nil {
		h.logger.Error("failed to handle message",
			"role_id", req.RoleId,
			"op_code", req.OpCode,
			"error", err,
		)
		return &gamepb.ForwardMessageResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &gamepb.ForwardMessageResponse{
		Success: true,
		Payload: resp,
	}, nil
}
// RoleOnline 处理角色上线
// func (h *GameHandler) RoleOnline(ctx context.Context, req *gamepb.RoleOnlineRequest) (*gamepb.RoleOnlineResponse, error) {
// 	h.logger.Info("received role online request",
// 		"role_id", req.RoleId,
// 		"session_id", req.SessionId,
// 		"gateway_addr", req.GatewayAddr,
// 	)
//
// 	if err := h.roleSvc.HandleRoleOnline(ctx, req.RoleId, req.SessionId, req.GatewayAddr); err != nil {
// 		h.logger.Error("failed to handle role online",
// 			"role_id", req.RoleId,
// 			"error", err,
// 		)
// 		return &gamepb.RoleOnlineResponse{
// 			Success: false,
// 			Error:   err.Error(),
// 		}, nil
// 	}
//
// 	return &gamepb.RoleOnlineResponse{
// 		Success: true,
// 	}, nil
// }
//
// RoleOffline 处理角色下线
// func (h *GameHandler) RoleOffline(ctx context.Context, req *gamepb.RoleOfflineRequest) (*gamepb.RoleOfflineResponse, error) {
// 	h.logger.Info("received role offline request",
// 		"role_id", req.RoleId,
// 	)
//
// 	if err := h.roleSvc.HandleRoleOffline(ctx, req.RoleId); err != nil {
// 		h.logger.Error("failed to handle role offline",
// 			"role_id", req.RoleId,
// 			"error", err,
// 		)
// 		return &gamepb.RoleOfflineResponse{
// 			Success: false,
// 			Error:   err.Error(),
// 		}, nil
// 	}
//
// 	return &gamepb.RoleOfflineResponse{
// 		Success: true,
// 	}, nil
// }
