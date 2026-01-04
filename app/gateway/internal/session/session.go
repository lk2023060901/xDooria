package session

import (
	"context"
	"sync"

	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// GatewaySession Gateway 特定的 Session，扩展 BaseSession 添加认证状态和消息串行处理器
type GatewaySession struct {
	session.Session // 嵌入基础 Session

	// 认证状态
	mu            sync.RWMutex
	uid           int64 // 用户账号 ID
	roleID        int64 // 当前选择的角色 ID
	authenticated bool  // 是否已认证
	roleSelected  bool  // 是否已选择角色

	// 消息串行处理
	taskCh chan *common.Envelope // 玩家私有的待转发消息队列
	cancel context.CancelFunc    // 停止处理循环
}

// NewGatewaySession 创建 Gateway Session
func NewGatewaySession(base session.Session) *GatewaySession {
	return &GatewaySession{
		Session: base,
		taskCh:  make(chan *common.Envelope, 1024),
	}
}

// StartProcessor 启动该玩家的消息串行处理器
func (s *GatewaySession) StartProcessor(ctx context.Context, handler func(*common.Envelope)) {
	s.mu.Lock()
	if s.cancel != nil {
		s.mu.Unlock()
		return
	}
	pCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel
	s.mu.Unlock()

	// 使用项目封装的 conc.Go，确保安全性和风格统一
	conc.Go(func() (struct{}, error) {
		for {
			select {
			case env := <-s.taskCh:
				handler(env)
			case <-pCtx.Done():
				return struct{}{}, nil
			case <-s.Context().Done():
				return struct{}{}, nil
			}
		}
	})
}

// PushTask 投递一个待处理任务
func (s *GatewaySession) PushTask(env *common.Envelope) bool {
	select {
	case s.taskCh <- env:
		return true
	default:
		return false // 队列满，可能被恶意攻击或后端极慢
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
