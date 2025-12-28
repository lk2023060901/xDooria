package session

import (
	"sync"

	"github.com/lk2023060901/xdooria/pkg/network/session"
)

// GatewaySession Gateway 特定的 Session，扩展 BaseSession 添加认证状态
type GatewaySession struct {
	session.Session // 嵌入基础 Session

	// 认证状态
	mu            sync.RWMutex
	uid           int64 // 用户账号 ID
	roleID        int64 // 当前选择的角色 ID
	authenticated bool  // 是否已认证
	roleSelected  bool  // 是否已选择角色
}

// NewGatewaySession 创建 Gateway Session
func NewGatewaySession(base session.Session) *GatewaySession {
	return &GatewaySession{
		Session: base,
	}
}

// SetUID 设置用户 UID
func (s *GatewaySession) SetUID(uid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uid = uid
}

// GetUID 获取用户 UID
func (s *GatewaySession) GetUID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.uid
}

// SetRoleID 设置角色 ID
func (s *GatewaySession) SetRoleID(roleID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roleID = roleID
}

// GetRoleID 获取角色 ID
func (s *GatewaySession) GetRoleID() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.roleID
}

// SetAuthenticated 设置认证状态
func (s *GatewaySession) SetAuthenticated(authenticated bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.authenticated = authenticated
}

// IsAuthenticated 检查是否已认证
func (s *GatewaySession) IsAuthenticated() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.authenticated
}

// SetRoleSelected 设置角色选择状态
func (s *GatewaySession) SetRoleSelected(selected bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.roleSelected = selected
}

// IsRoleSelected 检查是否已选择角色
func (s *GatewaySession) IsRoleSelected() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.roleSelected
}

// Reset 重置认证状态（用于断线重连等场景）
func (s *GatewaySession) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uid = 0
	s.roleID = 0
	s.authenticated = false
	s.roleSelected = false
}
