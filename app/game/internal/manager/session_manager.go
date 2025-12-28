package manager

import (
	"context"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/app/game/internal/dao"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// RoleState 角色会话状态
type RoleState int

const (
	StateOnline      RoleState = iota // Gateway 连接正常
	StateDisconnected                  // Gateway 断线，保护期
	StateOffline                       // 完全下线
)

func (s RoleState) String() string {
	switch s {
	case StateOnline:
		return "Online"
	case StateDisconnected:
		return "Disconnected"
	case StateOffline:
		return "Offline"
	default:
		return "Unknown"
	}
}

// RoleSessionState 角色会话状态信息
type RoleSessionState struct {
	Session          *model.Session
	State            RoleState
	DisconnectTime   time.Time
	DisconnectTimer  *time.Timer
}

// SessionManager 会话管理器，负责管理角色与网关的连接关系
type SessionManager struct {
	logger   logger.Logger
	cacheDAO *dao.CacheDAO

	// 内存会话映射
	mu           sync.RWMutex
	sessions     map[int64]*RoleSessionState // roleID -> RoleSessionState

	// 断线超时配置
	disconnectTimeout time.Duration // 默认 60 秒
}

// NewSessionManager 创建会话管理器
func NewSessionManager(l logger.Logger, cacheDAO *dao.CacheDAO) *SessionManager {
	return &SessionManager{
		logger:            l.Named("manager.session"),
		cacheDAO:          cacheDAO,
		sessions:          make(map[int64]*RoleSessionState),
		disconnectTimeout: 60 * time.Second,
	}
}

// RegisterSession 注册会话
func (m *SessionManager) RegisterSession(roleID int64, sessionID, gatewayAddr string) error {
	session := model.NewSession(roleID, sessionID, gatewayAddr)

	m.mu.Lock()
	// 检查是否存在旧会话
	if oldState, exists := m.sessions[roleID]; exists {
		// 如果处于断线状态，取消断线定时器
		if oldState.State == StateDisconnected && oldState.DisconnectTimer != nil {
			oldState.DisconnectTimer.Stop()
			m.logger.Info("role reconnected, cancelled disconnect timer",
				"role_id", roleID,
			)
		}
	}

	// 创建新的会话状态
	m.sessions[roleID] = &RoleSessionState{
		Session: session,
		State:   StateOnline,
	}
	m.mu.Unlock()

	// 持久化到 Redis
	ctx := context.Background()
	if err := m.cacheDAO.SetSession(ctx, session, 0); err != nil {
		m.logger.Warn("failed to cache session to redis",
			"role_id", roleID,
			"error", err,
		)
	}

	// 设置在线状态
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
	m.mu.Lock()
	state, exists := m.sessions[roleID]
	if exists {
		// 取消断线定时器（如果存在）
		if state.DisconnectTimer != nil {
			state.DisconnectTimer.Stop()
		}
		delete(m.sessions, roleID)
	}
	m.mu.Unlock()

	// 从 Redis 移除
	ctx := context.Background()
	if err := m.cacheDAO.DeleteSession(ctx, roleID); err != nil {
		m.logger.Warn("failed to delete session from redis",
			"role_id", roleID,
			"error", err,
		)
	}

	// 删除在线状态
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

// SetDisconnected 设置角色为断线状态并启动断线定时器
func (m *SessionManager) SetDisconnected(roleID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.sessions[roleID]
	if !exists {
		return
	}

	// 如果已经是断线状态，不重复处理
	if state.State == StateDisconnected {
		return
	}

	// 设置为断线状态
	state.State = StateDisconnected
	state.DisconnectTime = time.Now()

	// 启动断线定时器
	state.DisconnectTimer = time.AfterFunc(m.disconnectTimeout, func() {
		m.handleDisconnectTimeout(roleID)
	})

	m.logger.Info("role disconnected, started disconnect timer",
		"role_id", roleID,
		"timeout", m.disconnectTimeout,
	)
}

// handleDisconnectTimeout 处理断线超时
func (m *SessionManager) handleDisconnectTimeout(roleID int64) {
	m.mu.Lock()
	state, exists := m.sessions[roleID]
	if !exists {
		m.mu.Unlock()
		return
	}

	// 确认仍然是断线状态
	if state.State != StateDisconnected {
		m.mu.Unlock()
		return
	}

	// 设置为离线状态
	state.State = StateOffline
	m.mu.Unlock()

	m.logger.Info("role disconnect timeout, saving data",
		"role_id", roleID,
	)

	// TODO: 调用 RoleService 保存角色数据

	// 注销会话
	if err := m.UnregisterSession(roleID); err != nil {
		m.logger.Error("unregister session failed after disconnect timeout",
			"role_id", roleID,
			"error", err,
		)
	}
}

// GetSession 获取会话信息
func (m *SessionManager) GetSession(roleID int64) (*model.Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sessions[roleID]
	if !ok {
		return nil, false
	}

	return state.Session, true
}

// GetState 获取角色状态
func (m *SessionManager) GetState(roleID int64) (RoleState, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sessions[roleID]
	if !ok {
		return StateOffline, false
	}

	return state.State, true
}

// GetGatewayAddr 获取角色对应的网关地址
func (m *SessionManager) GetGatewayAddr(roleID int64) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sessions[roleID]
	if !ok {
		return "", false
	}

	return state.Session.GatewayAddr, true
}

// GetSessionCount 获取会话数量
func (m *SessionManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// GetOnlineCount 获取在线（非断线）角色数量
func (m *SessionManager) GetOnlineCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	count := 0
	for _, state := range m.sessions {
		if state.State == StateOnline {
			count++
		}
	}
	return count
}

// GetAllSessions 获取所有会话
func (m *SessionManager) GetAllSessions() []*model.Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*model.Session, 0, len(m.sessions))
	for _, state := range m.sessions {
		sessions = append(sessions, state.Session)
	}

	return sessions
}

// IsOnline 检查角色是否在线
func (m *SessionManager) IsOnline(roleID int64) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.sessions[roleID]
	if !ok {
		return false
	}

	return state.State == StateOnline
}

// SetDisconnectTimeout 设置断线超时时间
func (m *SessionManager) SetDisconnectTimeout(timeout time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.disconnectTimeout = timeout
	m.logger.Info("disconnect timeout updated", "timeout", timeout)
}
