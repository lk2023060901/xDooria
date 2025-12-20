package sentry

import (
	"sync"

	"github.com/getsentry/sentry-go"
)

// EventHook 事件钩子接口
type EventHook interface {
	// OnCapture 事件捕获时调用
	OnCapture(event *sentry.Event)
}

// EventHookFunc 函数式钩子实现
type EventHookFunc func(event *sentry.Event)

// OnCapture 实现 EventHook 接口
func (f EventHookFunc) OnCapture(event *sentry.Event) {
	f(event)
}

// hookManager 钩子管理器
type hookManager struct {
	hooks []EventHook
	mu    sync.RWMutex
}

// newHookManager 创建钩子管理器
func newHookManager() *hookManager {
	return &hookManager{
		hooks: make([]EventHook, 0),
	}
}

// register 注册钩子
func (m *hookManager) register(hook EventHook) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.hooks = append(m.hooks, hook)
}

// unregister 注销钩子
func (m *hookManager) unregister(hook EventHook) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, h := range m.hooks {
		if h == hook {
			m.hooks = append(m.hooks[:i], m.hooks[i+1:]...)
			return
		}
	}
}

// trigger 触发所有钩子
func (m *hookManager) trigger(event *sentry.Event) {
	m.mu.RLock()
	hooks := make([]EventHook, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.RUnlock()

	// 异步调用钩子，不阻塞事件上报
	for _, hook := range hooks {
		go func(h EventHook) {
			defer func() {
				if r := recover(); r != nil {
					// 钩子执行失败不应该影响主流程
				}
			}()
			h.OnCapture(event)
		}(hook)
	}
}
