package manager

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// BroadcastManager 广播管理器，负责消息的广播和转发
// TODO: 需要等待 game.proto 定义完成后实现 gRPC 客户端
type BroadcastManager struct {
	logger     logger.Logger
	sessionMgr *SessionManager
	// TODO: 添加 Gateway gRPC 客户端池
	// gatewayClients map[string]gamepb.GameServiceClient
}

// NewBroadcastManager 创建广播管理器
func NewBroadcastManager(
	l logger.Logger,
	sessionMgr *SessionManager,
) *BroadcastManager {
	return &BroadcastManager{
		logger:     l.Named("manager.broadcast"),
		sessionMgr: sessionMgr,
	}
}

// SendToRole 发送消息给指定角色
func (m *BroadcastManager) SendToRole(ctx context.Context, roleID int64, opCode uint32, payload []byte) error {
	// 获取角色所在的网关地址
	gatewayAddr, ok := m.sessionMgr.GetGatewayAddr(roleID)
	if !ok {
		return fmt.Errorf("role %d is not online", roleID)
	}

	m.logger.Debug("sending message to role",
		"role_id", roleID,
		"gateway_addr", gatewayAddr,
		"op_code", opCode,
	)

	// TODO: 实现 gRPC 调用网关的 SendToPlayer 方法
	// client := m.getGatewayClient(gatewayAddr)
	// _, err := client.SendToPlayer(ctx, &gamepb.SendToPlayerRequest{
	// 	RoleId: roleID,
	// 	OpCode: opCode,
	// 	Payload: payload,
	// })
	// return err

	return fmt.Errorf("not implemented: need game.proto definition")
}

// BroadcastToRoles 广播消息给指定角色列表
func (m *BroadcastManager) BroadcastToRoles(ctx context.Context, roleIDs []int64, opCode uint32, payload []byte) error {
	if len(roleIDs) == 0 {
		return nil
	}

	m.logger.Debug("broadcasting message to roles",
		"role_count", len(roleIDs),
		"op_code", opCode,
	)

	// TODO: 按网关分组，批量发送
	// 1. 将角色按网关地址分组
	// gatewayRoles := make(map[string][]int64)
	// for _, roleID := range roleIDs {
	// 	if gatewayAddr, ok := m.sessionMgr.GetGatewayAddr(roleID); ok {
	// 		gatewayRoles[gatewayAddr] = append(gatewayRoles[gatewayAddr], roleID)
	// 	}
	// }
	//
	// 2. 并发调用各网关的 SendToPlayers 方法
	// var wg sync.WaitGroup
	// for gatewayAddr, roles := range gatewayRoles {
	// 	wg.Add(1)
	// 	go func(addr string, roleIDs []int64) {
	// 		defer wg.Done()
	// 		client := m.getGatewayClient(addr)
	// 		_, err := client.SendToPlayers(ctx, &gamepb.SendToPlayersRequest{
	// 			RoleIds: roleIDs,
	// 			OpCode: opCode,
	// 			Payload: payload,
	// 		})
	// 		if err != nil {
	// 			m.logger.Error("failed to broadcast to gateway",
	// 				"gateway_addr", addr,
	// 				"error", err,
	// 			)
	// 		}
	// 	}(gatewayAddr, roles)
	// }
	// wg.Wait()

	return fmt.Errorf("not implemented: need game.proto definition")
}

// BroadcastToAll 广播消息给所有在线角色
func (m *BroadcastManager) BroadcastToAll(ctx context.Context, opCode uint32, payload []byte) error {
	sessions := m.sessionMgr.GetAllSessions()
	if len(sessions) == 0 {
		return nil
	}

	m.logger.Info("broadcasting message to all online roles",
		"role_count", len(sessions),
		"op_code", opCode,
	)

	roleIDs := make([]int64, len(sessions))
	for i, session := range sessions {
		roleIDs[i] = session.RoleID
	}

	return m.BroadcastToRoles(ctx, roleIDs, opCode, payload)
}
