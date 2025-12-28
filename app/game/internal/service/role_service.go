package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lk2023060901/xdooria/app/game/internal/dao"
	"github.com/lk2023060901/xdooria/app/game/internal/manager"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// RoleService 角色服务，处理角色相关的业务逻辑
type RoleService struct {
	logger     logger.Logger
	roleMgr    *manager.RoleManager
	sessionMgr *manager.SessionManager
	roleDAO    *dao.RoleDAO
	metrics    *metrics.GameMetrics
}

// NewRoleService 创建角色服务
func NewRoleService(
	l logger.Logger,
	roleMgr *manager.RoleManager,
	sessionMgr *manager.SessionManager,
	roleDAO *dao.RoleDAO,
	m *metrics.GameMetrics,
) *RoleService {
	return &RoleService{
		logger:     l.Named("service.role"),
		roleMgr:    roleMgr,
		sessionMgr: sessionMgr,
		roleDAO:    roleDAO,
		metrics:    m,
	}
}

// HandleRoleOnline 处理角色上线
// TODO: 需要等待 game.proto 定义完成
// func (s *RoleService) HandleRoleOnline(ctx context.Context, req *gamepb.RoleOnlineRequest) (*gamepb.RoleOnlineResponse, error)
func (s *RoleService) HandleRoleOnline(ctx context.Context, roleID int64, sessionID, gatewayAddr string) error {
	s.logger.Info("handling role online",
		"role_id", roleID,
		"session_id", sessionID,
		"gateway_addr", gatewayAddr,
	)

	// 1. 加载角色数据
	role, err := s.roleMgr.LoadRole(ctx, roleID)
	if err != nil {
		s.logger.Error("failed to load role",
			"role_id", roleID,
			"error", err,
		)
		s.metrics.RecordRoleOnline(false)
		return fmt.Errorf("failed to load role: %w", err)
	}

	// 2. 检查角色是否被封禁
	if role.IsBanned() {
		s.logger.Warn("role is banned",
			"role_id", roleID,
			"ban_expire_at", role.BanExpireAt,
		)
		s.metrics.RecordRoleOnline(false)
		return fmt.Errorf("role is banned")
	}

	// 3. 注册会话
	if err := s.sessionMgr.RegisterSession(roleID, sessionID, gatewayAddr); err != nil {
		s.logger.Error("failed to register session",
			"role_id", roleID,
			"error", err,
		)
		s.metrics.RecordRoleOnline(false)
		return fmt.Errorf("failed to register session: %w", err)
	}

	// 4. 更新最后登录时间
	role.UpdateLastLogin()
	if err := s.roleDAO.UpdateLastLogin(ctx, roleID); err != nil {
		s.logger.Warn("failed to update last login time",
			"role_id", roleID,
			"error", err,
		)
		// 不影响主流程
	}

	s.metrics.RecordRoleOnline(true)
	s.logger.Info("role online success",
		"role_id", roleID,
		"nickname", role.Nickname,
	)

	return nil
}

// HandleRoleOffline 处理角色下线
// TODO: 需要等待 game.proto 定义完成
// func (s *RoleService) HandleRoleOffline(ctx context.Context, req *gamepb.RoleOfflineRequest) (*gamepb.RoleOfflineResponse, error)
func (s *RoleService) HandleRoleOffline(ctx context.Context, roleID int64) error {
	s.logger.Info("handling role offline",
		"role_id", roleID,
	)

	// 1. 保存角色数据
	if err := s.roleMgr.SaveRole(ctx, roleID); err != nil {
		s.logger.Error("failed to save role",
			"role_id", roleID,
			"error", err,
		)
		// 继续下线流程
	}

	// 2. 注销会话
	if err := s.sessionMgr.UnregisterSession(roleID); err != nil {
		s.logger.Error("failed to unregister session",
			"role_id", roleID,
			"error", err,
		)
	}

	// 3. 标记角色为不活跃（从内存移除）
	s.roleMgr.MarkInactive(roleID)

	s.metrics.RecordRoleOffline()
	s.logger.Info("role offline success",
		"role_id", roleID,
	)

	return nil
}

// GetRoleInfo 获取角色信息
func (s *RoleService) GetRoleInfo(ctx context.Context, roleID int64) (*model.Role, error) {
	// 先从内存获取
	role, ok := s.roleMgr.GetRole(roleID)
	if ok {
		return role, nil
	}

	// 内存没有则加载
	role, err := s.roleMgr.LoadRole(ctx, roleID)
	if err != nil {
		s.logger.Error("failed to get role info",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get role info: %w", err)
	}

	return role, nil
}

// UpdateRoleData 更新角色数据
func (s *RoleService) UpdateRoleData(roleID int64, updateFunc func(*model.Role)) error {
	if err := s.roleMgr.UpdateRoleState(roleID, updateFunc); err != nil {
		s.logger.Error("failed to update role data",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("failed to update role data: %w", err)
	}

	return nil
}

// ListRolesByUID 根据 UID 查询所有角色
func (s *RoleService) ListRolesByUID(ctx context.Context, uid int64) ([]*model.Role, error) {
	roles, err := s.roleDAO.ListByUID(ctx, uid)
	if err != nil {
		s.logger.Error("failed to list roles by uid",
			"uid", uid,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list roles by uid: %w", err)
	}

	s.logger.Debug("listed roles by uid",
		"uid", uid,
		"count", len(roles),
	)

	return roles, nil
}

// CreateRole 创建角色
func (s *RoleService) CreateRole(ctx context.Context, uid int64, nickname string, gender int32, appearance string) (*model.Role, error) {
	// 检查昵称是否已存在
	exists, err := s.roleDAO.CheckNicknameExists(ctx, nickname)
	if err != nil {
		s.logger.Error("failed to check nickname exists",
			"nickname", nickname,
			"error", err,
		)
		return nil, fmt.Errorf("failed to check nickname: %w", err)
	}

	if exists {
		s.logger.Warn("nickname already exists", "nickname", nickname)
		return nil, fmt.Errorf("nickname already exists")
	}

	// 检查该用户的角色数量
	roles, err := s.roleDAO.ListByUID(ctx, uid)
	if err != nil {
		s.logger.Error("failed to list roles for count check",
			"uid", uid,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	if len(roles) >= 3 {
		s.logger.Warn("role limit exceeded",
			"uid", uid,
			"count", len(roles),
		)
		return nil, fmt.Errorf("role limit exceeded")
	}

	// 创建角色
	role := model.NewRole(uid, nickname, int16(gender), json.RawMessage(appearance))
	if err := s.roleDAO.Create(ctx, role); err != nil {
		s.logger.Error("failed to create role",
			"uid", uid,
			"nickname", nickname,
			"error", err,
		)
		return nil, fmt.Errorf("failed to create role: %w", err)
	}

	s.logger.Info("role created",
		"uid", uid,
		"role_id", role.ID,
		"nickname", nickname,
	)

	return role, nil
}

// CheckNicknameExists 检查昵称是否已存在
func (s *RoleService) CheckNicknameExists(ctx context.Context, nickname string) (bool, error) {
	exists, err := s.roleDAO.CheckNicknameExists(ctx, nickname)
	if err != nil {
		s.logger.Error("failed to check nickname exists",
			"nickname", nickname,
			"error", err,
		)
		return false, fmt.Errorf("failed to check nickname: %w", err)
	}

	return exists, nil
}
