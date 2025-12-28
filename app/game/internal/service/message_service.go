package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/app/game/internal/manager"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	gamerouter "github.com/lk2023060901/xdooria/app/game/internal/router"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/router"
)

// MessageService 消息服务，处理消息路由和转发
type MessageService struct {
	logger       logger.Logger
	roleRouter   *gamerouter.RoleRouter
	roleMgr      *manager.RoleManager
	broadcastMgr *manager.BroadcastManager
	metrics      *metrics.GameMetrics
}

// NewMessageService 创建消息服务
func NewMessageService(
	l logger.Logger,
	r router.Router,
	roleMgr *manager.RoleManager,
	broadcastMgr *manager.BroadcastManager,
	m *metrics.GameMetrics,
) *MessageService {
	s := &MessageService{
		logger:       l.Named("service.message"),
		roleRouter:   gamerouter.NewRoleRouter(r),
		roleMgr:      roleMgr,
		broadcastMgr: broadcastMgr,
		metrics:      m,
	}

	// 注册所有消息处理器
	s.registerHandlers()

	return s
}

// registerHandlers 注册所有游戏消息处理器
func (s *MessageService) registerHandlers() {
	// TODO: 注册具体的游戏消息处理器
	// 示例：
	// gamerouter.RegisterHandler(s.roleRouter,
	//     uint32(api.OpCode_OP_ENTER_SCENE_REQ),
	//     uint32(api.OpCode_OP_ENTER_SCENE_RES),
	//     s.handleEnterScene)
}

// HandleMessage 处理从 Gateway 转发来的消息
// TODO: 需要等待 game.proto 定义完成
// func (s *MessageService) HandleMessage(ctx context.Context, req *gamepb.ForwardMessageRequest) (*gamepb.ForwardMessageResponse, error)
func (s *MessageService) HandleMessage(ctx context.Context, roleID int64, opCode uint32, payload []byte) ([]byte, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		s.metrics.RecordMessage(fmt.Sprintf("%d", opCode), true, duration)
	}()

	s.logger.Debug("handling message",
		"role_id", roleID,
		"op_code", opCode,
		"payload_size", len(payload),
	)

	// 1. 验证角色是否在线
	role, ok := s.roleMgr.GetRole(roleID)
	if !ok {
		s.logger.Warn("role not found in memory",
			"role_id", roleID,
		)
		return nil, fmt.Errorf("role not online")
	}

	// 2. 检查角色状态
	if role.IsBanned() {
		s.logger.Warn("banned role attempted to send message",
			"role_id", roleID,
		)
		return nil, fmt.Errorf("role is banned")
	}

	// 3. 将 roleID 放入 Context，供 handler 使用
	ctx = router.WithRoleID(ctx, roleID)

	// 4. 使用 RoleRouter 路由到具体的处理器
	_, respPayload, err := s.roleRouter.Dispatch(ctx, opCode, payload)
	if err != nil {
		s.logger.Error("failed to route message",
			"role_id", roleID,
			"op_code", opCode,
			"error", err,
		)
		return nil, fmt.Errorf("failed to route message: %w", err)
	}

	s.logger.Debug("message handled",
		"role_id", roleID,
		"op_code", opCode,
	)

	return respPayload, nil
}

// BroadcastToAll 广播消息给所有在线角色
func (s *MessageService) BroadcastToAll(ctx context.Context, opCode uint32, payload []byte) error {
	s.logger.Info("broadcasting to all online roles",
		"op_code", opCode,
		"payload_size", len(payload),
	)

	if err := s.broadcastMgr.BroadcastToAll(ctx, opCode, payload); err != nil {
		s.logger.Error("failed to broadcast to all",
			"op_code", opCode,
			"error", err,
		)
		return fmt.Errorf("failed to broadcast: %w", err)
	}

	return nil
}

// SendToRole 发送消息给指定角色
func (s *MessageService) SendToRole(ctx context.Context, roleID int64, opCode uint32, payload []byte) error {
	s.logger.Debug("sending message to role",
		"role_id", roleID,
		"op_code", opCode,
		"payload_size", len(payload),
	)

	if err := s.broadcastMgr.SendToRole(ctx, roleID, opCode, payload); err != nil {
		s.logger.Error("failed to send message to role",
			"role_id", roleID,
			"op_code", opCode,
			"error", err,
		)
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}
