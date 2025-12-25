package auth

import (
	"context"
	"fmt"
	"sync"

	pb "github.com/xDooria/xDooria-proto-common"
)

// Manager 认证管理器
type Manager struct {
	mu             sync.RWMutex
	authenticators map[pb.LoginType]Authenticator
}

func NewManager() *Manager {
	return &Manager{
		authenticators: make(map[pb.LoginType]Authenticator),
	}
}

// Register 注册认证器
func (m *Manager) Register(a Authenticator) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.authenticators[a.Type()] = a
}

// Verify 执行认证分发
func (m *Manager) Verify(ctx context.Context, loginType pb.LoginType, cred []byte) (*Identity, error) {
	m.mu.RLock()
	a, ok := m.authenticators[loginType]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("authenticator not found for login type: %v", loginType)
	}

	return a.Authenticate(ctx, cred)
}
