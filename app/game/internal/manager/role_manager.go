package manager

import (
	"context"
	"fmt"
	"sync"

	"github.com/lk2023060901/xdooria/app/game/internal/dao"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// RoleManager 角色管理器，负责角色数据的三级缓存管理
type RoleManager struct {
	logger   logger.Logger
	roleDAO  *dao.RoleDAO
	cacheDAO *dao.CacheDAO
	metrics  *metrics.GameMetrics

	// 内存缓存（第一级）
	mu    sync.RWMutex
	roles map[int64]*model.Role
}

// NewRoleManager 创建角色管理器
func NewRoleManager(
	l logger.Logger,
	roleDAO *dao.RoleDAO,
	cacheDAO *dao.CacheDAO,
	m *metrics.GameMetrics,
) *RoleManager {
	return &RoleManager{
		logger:   l.Named("manager.role"),
		roleDAO:  roleDAO,
		cacheDAO: cacheDAO,
		metrics:  m,
		roles:    make(map[int64]*model.Role),
	}
}

// LoadRole 加载角色（三级缓存：内存 -> Redis -> PostgreSQL）
func (m *RoleManager) LoadRole(ctx context.Context, roleID int64) (*model.Role, error) {
	// 1. 检查内存缓存
	m.mu.RLock()
	if role, ok := m.roles[roleID]; ok {
		m.mu.RUnlock()
		m.metrics.RecordCacheHit("memory")
		return role, nil
	}
	m.mu.RUnlock()

	m.metrics.RecordCacheMiss("memory")

	// 2. 检查 Redis 缓存
	role, err := m.cacheDAO.GetRole(ctx, roleID)
	if err != nil {
		m.logger.Warn("failed to get role from redis",
			"role_id", roleID,
			"error", err,
		)
		// 继续尝试从数据库加载
	}

	// 3. 如果 Redis 有数据，写入内存缓存
	if role != nil {
		m.mu.Lock()
		m.roles[roleID] = role
		m.mu.Unlock()
		return role, nil
	}

	// 4. 从 PostgreSQL 加载
	role, err = m.roleDAO.GetByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load role from db: %w", err)
	}

	// 5. 回写到 Redis 和内存缓存
	m.mu.Lock()
	m.roles[roleID] = role
	m.mu.Unlock()

	// 异步写入 Redis
	go func() {
		if err := m.cacheDAO.SetRole(context.Background(), role, 0); err != nil {
			m.logger.Warn("failed to cache role to redis",
				"role_id", roleID,
				"error", err,
			)
		}
	}()

	return role, nil
}

// LoadRoleByUID 根据 UID 加载角色
func (m *RoleManager) LoadRoleByUID(ctx context.Context, uid int64) (*model.Role, error) {
	// 直接从数据库查询（UID 不适合做内存缓存的 key）
	role, err := m.roleDAO.GetByUID(ctx, uid)
	if err != nil {
		return nil, fmt.Errorf("failed to load role by uid: %w", err)
	}

	// 加载后更新内存缓存
	m.mu.Lock()
	m.roles[role.ID] = role
	m.mu.Unlock()

	// 异步写入 Redis
	go func() {
		if err := m.cacheDAO.SetRole(context.Background(), role, 0); err != nil {
			m.logger.Warn("failed to cache role to redis",
				"role_id", role.ID,
				"error", err,
			)
		}
	}()

	return role, nil
}

// GetRole 获取内存中的角色（不触发加载）
func (m *RoleManager) GetRole(roleID int64) (*model.Role, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	role, ok := m.roles[roleID]
	return role, ok
}

// SaveRole 保存角色到数据库和缓存
func (m *RoleManager) SaveRole(ctx context.Context, roleID int64) error {
	m.mu.RLock()
	role, ok := m.roles[roleID]
	m.mu.RUnlock()

	if !ok {
		return fmt.Errorf("role %d not found in memory", roleID)
	}

	// 1. 保存到数据库
	if err := m.roleDAO.Update(ctx, role); err != nil {
		return fmt.Errorf("failed to update role in db: %w", err)
	}

	// 2. 更新 Redis 缓存
	if err := m.cacheDAO.SetRole(ctx, role, 0); err != nil {
		m.logger.Warn("failed to update role cache",
			"role_id", roleID,
			"error", err,
		)
		// 不返回错误，因为数据库已更新
	}

	m.logger.Debug("role saved",
		"role_id", roleID,
	)

	return nil
}

// UpdateRoleState 更新角色状态（通过更新函数）
func (m *RoleManager) UpdateRoleState(roleID int64, updateFunc func(*model.Role)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	role, ok := m.roles[roleID]
	if !ok {
		return fmt.Errorf("role %d not found in memory", roleID)
	}

	updateFunc(role)
	return nil
}

// MarkInactive 标记角色为不活跃（从内存中移除，但不影响数据库）
func (m *RoleManager) MarkInactive(roleID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.roles, roleID)

	m.logger.Debug("role marked as inactive",
		"role_id", roleID,
	)
}

// GetOnlineCount 获取在线角色数量
func (m *RoleManager) GetOnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.roles)
}

// GetAllOnlineRoleIDs 获取所有在线角色 ID
func (m *RoleManager) GetAllOnlineRoleIDs() []int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	roleIDs := make([]int64, 0, len(m.roles))
	for roleID := range m.roles {
		roleIDs = append(roleIDs, roleID)
	}

	return roleIDs
}
