package service

import (
	"context"
	"fmt"

	api "github.com/lk2023060901/xdooria-proto-api"
	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/app/game/internal/manager"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// SceneService 场景服务
type SceneService struct {
	logger   logger.Logger
	roleMgr  *manager.RoleManager
	sceneMgr *manager.SceneManager
	metrics  *metrics.GameMetrics
}

// NewSceneService 创建场景服务
func NewSceneService(
	l logger.Logger,
	roleMgr *manager.RoleManager,
	sceneMgr *manager.SceneManager,
	m *metrics.GameMetrics,
) *SceneService {
	return &SceneService{
		logger:   l.Named("service.scene"),
		roleMgr:  roleMgr,
		sceneMgr: sceneMgr,
		metrics:  m,
	}
}

// HandleEnterScene 处理进入场景请求
func (s *SceneService) HandleEnterScene(ctx context.Context, roleID int64) (*api.EnterSceneResponse, error) {
	s.logger.Info("handle enter scene", "role_id", roleID)

	// 1. 获取角色信息
	role, ok := s.roleMgr.GetRole(roleID)
	if !ok {
		s.logger.Error("role not found", "role_id", roleID)
		return &api.EnterSceneResponse{
			Code: common.ErrCode_ERR_CODE_NOT_FOUND,
		}, nil
	}

	// 2. 检查角色状态
	if role.IsBanned() {
		s.logger.Warn("banned role attempted to enter scene", "role_id", roleID)
		return &api.EnterSceneResponse{
			Code: common.ErrCode_ERR_CODE_PERMISSION_DENIED,
		}, nil
	}

	// 3. 获取或创建场景实例（默认进入新手场景）
	mapID := int32(1001) // 默认新手场景
	scene, err := s.sceneMgr.GetOrCreateScene(mapID)
	if err != nil {
		s.logger.Error("failed to get or create scene",
			"role_id", roleID,
			"map_id", mapID,
			"error", err,
		)
		return &api.EnterSceneResponse{
			Code: common.ErrCode_ERR_CODE_INTERNAL,
		}, nil
	}

	// 4. 将角色加入场景
	position := &api.Position{
		X:        100.0, // 默认出生点
		Y:        0.0,
		Z:        100.0,
		Rotation: 0.0,
	}

	if err := s.sceneMgr.EnterScene(roleID, scene.MapID, position); err != nil {
		s.logger.Error("failed to enter scene",
			"role_id", roleID,
			"map_id", mapID,
			"error", err,
		)
		return &api.EnterSceneResponse{
			Code: common.ErrCode_ERR_CODE_INTERNAL,
		}, nil
	}

	s.logger.Info("role entered scene",
		"role_id", roleID,
		"map_id", mapID,
		"position", fmt.Sprintf("(%.1f, %.1f, %.1f)", position.X, position.Y, position.Z),
	)

	// 5. 返回场景信息
	return &api.EnterSceneResponse{
		Code: common.ErrCode_ERR_CODE_OK,
		Role: roleToProto(role),
		Scene: &api.SceneInfo{
			MapId: mapID,
			Pos:   position,
		},
	}, nil
}

// roleToProto 将 model.Role 转换为 api.RoleInfo
func roleToProto(role *model.Role) *api.RoleInfo {
	return &api.RoleInfo{
		RoleId:    role.ID,
		Nickname:  role.Nickname,
		AvatarUrl: role.AvatarURL,
		Level:     role.Level,
		VipExp:    int32(role.VIPExp),
		Status:    int32(role.Status),
	}
}
