package session

import (
	"context"
	"sync"

	"github.com/lk2023060901/xdooria-proto-common"
)

// SessionManager 定义会话管理器的接口。
type SessionManager interface {
	// Add 添加一个会话。
	Add(session Session)
	// Remove 移除指定 ID 的会话。
	Remove(id string)
	// Get 获取指定 ID 的会话。
	Get(id string) (Session, bool)
	// Count 返回当前会话数量。
	Count() int
	// Range 遍历所有会话，若 f 返回 false 则停止遍历。
	Range(f func(session Session) bool)
	// Broadcast 向所有会话广播数据。
	Broadcast(ctx context.Context, env *common.Envelope)
	// Close 关闭并清空所有会话。
	Close() error
}

// BaseSessionManager 提供 SessionManager 接口的基础实现。
type BaseSessionManager struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

// NewBaseSessionManager 创建一个新的基础会话管理器。
func NewBaseSessionManager() *BaseSessionManager {
	return &BaseSessionManager{
		sessions: make(map[string]Session),
	}
}

// Add 添加一个会话。
func (m *BaseSessionManager) Add(session Session) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.ID()] = session
}

// Remove 移除指定 ID 的会话。
func (m *BaseSessionManager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.sessions, id)
}

// Get 获取指定 ID 的会话。
func (m *BaseSessionManager) Get(id string) (Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// Count 返回当前会话数量。
func (m *BaseSessionManager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.sessions)
}

// Range 遍历所有会话。
func (m *BaseSessionManager) Range(f func(session Session) bool) {
	m.mu.RLock()
	// 为了避免在遍历时长时间持有锁，可以先复制一份切片或者在回调中处理
	// 这里简单实现，遍历时持有读锁
	defer m.mu.RUnlock()
	for _, session := range m.sessions {
		if !f(session) {
			break
		}
	}
}

// Broadcast 向所有会话广播数据。
func (m *BaseSessionManager) Broadcast(ctx context.Context, env *common.Envelope) {
	m.mu.RLock()
	// 复制一份 session 列表以减少持锁时间
	sessions := make([]Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	m.mu.RUnlock()

	for _, s := range sessions {
		_ = s.Send(ctx, env)
	}
}

// Close 关闭并清空所有会话。
func (m *BaseSessionManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, s := range m.sessions {
		_ = s.Close()
		delete(m.sessions, id)
	}
	return nil
}