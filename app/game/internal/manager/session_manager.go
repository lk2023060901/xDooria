package manager

import (
	"context"
	"sync"

	"github.com/lk2023060901/xdooria/app/game/internal/dao"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// SessionManager 会话管理器，负责管理角色与网关的连接关系
type SessionManager struct {
	logger   logger.Logger
	cacheDAO *dao.CacheDAO

	// 内存会话映射
	mu       sync.RWMutex
	sessions map[int64]*model.Session // roleID -> Session
}

// NewSessionManager 创建会话管理器
func NewSessionManager(l logger.Logger, cacheDAO *dao.CacheDAO) *SessionManager {
	return &SessionManager{
		logger:   l.Named("manager.session"),
		cacheDAO: cacheDAO,
		sessions: make(map[int64]*model.Session),
	}
}

// RegisterSession 注册会话
func (m *SessionManager) RegisterSession(roleID int64, sessionID, gatewayAddr string) error {
	session := model.NewSession(roleID, sessionID, gatewayAddr)

	// 1. 更新内存
	m.mu.Lock()
	m.sessions[roleID] = session
	m.mu.Unlock()

	// 2. 持久化到 Redis
	ctx := context.Background()
	if err := m.cacheDAO.SetSession(ctx, session, 0); err != nil {
		m.logger.Warn("failed to cache session to redis",
			"role_id", roleID,
			"error", err,
		)
		// 不返回错误，因为内存已更新
	}

	// 3. 设置在线状态
	if err := m.cacheDAO.SetOnlineStatus(ctx, roleID); err != nil {
		m.logger.Warn("failed to set online status",
			"role_id", roleID,
			"error", err,
		)
	}

	m.logger.Info("session registered",
		"role_id", roleID,
		"session_id", sessionID,
		"gateway_addr", gatewayAddr,
	)

	return nil
}

// UnregisterSession 注销会话
func (m *SessionManager) UnregisterSession(roleID int64) error {
	// 1. 从内存移除
	m.mu.Lock()
	delete(m.sessions, roleID)
	m.mu.Unlock()

	// 2. 从 Redis 移除
	ctx := context.Background()
	if err := m.cacheDAO.DeleteSession(ctx, roleID); err != nil {
		m.logger.Warn("failed to delete session from redis",
			"role_id", roleID,
			"error", err,
		)
	}

	// 3. 删除在线状态
	if err := m.cacheDAO.DeleteOnlineStatus(ctx, roleID); err != nil {
		m.logger.Warn("failed to delete online status",
			"role_id", roleID,
			"error", err,
		)
	}

	m.logger.Info("session unregistered",
		"role_id", roleID,
	)

	return nil
}

// GetSession 获取会话信息
func (m *SessionManager) GetSession(roleID int64) (*model.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[roleID]
	return session, ok
}

// GetGatewayAddr 获取角色对应的网关地址
func (m *SessionManager) GetGatewayAddr(roleID int64) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, ok := m.sessions[roleID]
	if !ok {
		return "", false
	}

	return session.GatewayAddr, true
}

// GetSessionCount 获取会话数量
func (m *SessionManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// GetAllSessions 获取所有会话
func (m *SessionManager) GetAllSessions() []*model.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*model.Session, 0, len(m.sessions))
	for _, session := range m.sessions {
		sessions = append(sessions, session)
	}

	return sessions
}

// IsOnline 检查角色是否在线
func (m *SessionManager) IsOnline(roleID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	_, ok := m.sessions[roleID]
	return ok
}
