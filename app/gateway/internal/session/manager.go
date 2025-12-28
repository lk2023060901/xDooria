package session

import (
	"sync"

	"github.com/lk2023060901/xdooria/pkg/network/session"
)

// Manager Gateway Session 管理器
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*GatewaySession // sessionID -> GatewaySession

	// UID 索引，用于快速查找同一用户的所有会话
	uidIndex map[int64]map[string]*GatewaySession // uid -> (sessionID -> GatewaySession)

	// RoleID 索引，用于快速查找角色对应的会话
	roleIndex map[int64]*GatewaySession // roleID -> GatewaySession
}

// NewManager 创建 Session 管理器
func NewManager() *Manager {
	return &Manager{
		sessions:  make(map[string]*GatewaySession),
		uidIndex:  make(map[int64]map[string]*GatewaySession),
		roleIndex: make(map[int64]*GatewaySession),
	}
}

// Register 注册新的 Session
func (m *Manager) Register(base session.Session) *GatewaySession {
	m.mu.Lock()
	defer m.mu.Unlock()

	gwSess := NewGatewaySession(base)
	m.sessions[base.ID()] = gwSess
	return gwSess
}

// Unregister 注销 Session
func (m *Manager) Unregister(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gwSess, ok := m.sessions[sessionID]
	if !ok {
		return
	}

	// 从 UID 索引中移除
	if gwSess.IsAuthenticated() {
		uid := gwSess.GetUID()
		if uidSessions, exists := m.uidIndex[uid]; exists {
			delete(uidSessions, sessionID)
			if len(uidSessions) == 0 {
				delete(m.uidIndex, uid)
			}
		}
	}

	// 从 RoleID 索引中移除
	if gwSess.IsRoleSelected() {
		roleID := gwSess.GetRoleID()
		delete(m.roleIndex, roleID)
	}

	// 从主索引中移除
	delete(m.sessions, sessionID)
}

// Get 获取 Session
func (m *Manager) Get(sessionID string) (*GatewaySession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.sessions[sessionID]
	return sess, ok
}

// UpdateAuthState 更新认证状态（认证成功后调用）
func (m *Manager) UpdateAuthState(sessionID string, uid int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gwSess, ok := m.sessions[sessionID]
	if !ok {
		return
	}

	// 设置认证状态
	gwSess.SetUID(uid)
	gwSess.SetAuthenticated(true)

	// 更新 UID 索引
	if _, exists := m.uidIndex[uid]; !exists {
		m.uidIndex[uid] = make(map[string]*GatewaySession)
	}
	m.uidIndex[uid][sessionID] = gwSess
}

// UpdateRoleState 更新角色状态（选择角色后调用）
func (m *Manager) UpdateRoleState(sessionID string, roleID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	gwSess, ok := m.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}

	if !gwSess.IsAuthenticated() {
		return ErrNotAuthenticated
	}

	// 如果之前选择过角色，先清除旧的索引
	if gwSess.IsRoleSelected() {
		oldRoleID := gwSess.GetRoleID()
		delete(m.roleIndex, oldRoleID)
	}

	// 设置新的角色
	gwSess.SetRoleID(roleID)
	gwSess.SetRoleSelected(true)

	// 更新 RoleID 索引
	m.roleIndex[roleID] = gwSess

	return nil
}

// GetByUID 根据 UID 获取所有会话
func (m *Manager) GetByUID(uid int64) []*GatewaySession {
	m.mu.RLock()
	defer m.mu.RUnlock()

	uidSessions, ok := m.uidIndex[uid]
	if !ok {
		return nil
	}

	sessions := make([]*GatewaySession, 0, len(uidSessions))
	for _, sess := range uidSessions {
		sessions = append(sessions, sess)
	}
	return sessions
}

// GetByRoleID 根据角色 ID 获取会话
func (m *Manager) GetByRoleID(roleID int64) (*GatewaySession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sess, ok := m.roleIndex[roleID]
	return sess, ok
}

// Count 获取当前会话总数
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// OnlineUserCount 获取在线用户数（已认证的不同 UID 数量）
func (m *Manager) OnlineUserCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.uidIndex)
}

// OnlineRoleCount 获取在线角色数（已选择角色的会话数量）
func (m *Manager) OnlineRoleCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.roleIndex)
}
