package sentry

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/getsentry/sentry-go"
)

// Client Sentry 客户端
type Client struct {
	hub    *sentry.Hub    // Sentry Hub（隔离的上下文）
	config *Config        // 配置
	hooks  *hookManager   // 钩子管理器
	closed atomic.Bool    // 关闭状态

	// 统计信息
	stats struct {
		eventsTotal    atomic.Uint64 // 总事件数
		eventsCaptured atomic.Uint64 // 成功捕获数
		eventsDropped  atomic.Uint64 // 丢弃数
	}
}

// New 创建 Sentry 客户端
func New(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	// 初始化 Sentry SDK
	client, err := sentry.NewClient(cfg.toClientOptions())
	if err != nil {
		return nil, fmt.Errorf("failed to create sentry client: %w", err)
	}

	// 创建独立的 Hub
	hub := sentry.NewHub(client, sentry.NewScope())

	// 设置全局标签
	hub.ConfigureScope(func(scope *sentry.Scope) {
		for key, value := range cfg.Tags {
			scope.SetTag(key, value)
		}
	})

	c := &Client{
		hub:    hub,
		config: cfg,
		hooks:  newHookManager(),
	}

	return c, nil
}

// CaptureException 捕获异常
func (c *Client) CaptureException(err error) *sentry.EventID {
	if c.closed.Load() {
		return nil
	}

	c.stats.eventsTotal.Add(1)

	eventID := c.hub.CaptureException(err)
	if eventID != nil && *eventID != "" {
		c.stats.eventsCaptured.Add(1)

		// 触发钩子
		c.hooks.trigger(&sentry.Event{
			EventID: *eventID,
			Level:   sentry.LevelError,
		})
	} else {
		c.stats.eventsDropped.Add(1)
	}

	return eventID
}

// CaptureMessage 捕获消息
func (c *Client) CaptureMessage(message string, level Level) *sentry.EventID {
	if c.closed.Load() {
		return nil
	}

	c.stats.eventsTotal.Add(1)

	eventID := c.hub.CaptureMessage(message)
	if eventID != nil && *eventID != "" {
		c.stats.eventsCaptured.Add(1)

		// 触发钩子
		c.hooks.trigger(&sentry.Event{
			EventID: *eventID,
			Level:   level.toSentryLevel(),
			Message: message,
		})
	} else {
		c.stats.eventsDropped.Add(1)
	}

	return eventID
}

// Recover 恢复 panic（用于 defer）
func (c *Client) Recover() {
	if r := recover(); r != nil {
		c.RecoverWithContext(r)
		panic(r) // 重新抛出 panic
	}
}

// RecoverWithContext 恢复 panic 并上报（不重新抛出）
func (c *Client) RecoverWithContext(recovered interface{}) *sentry.EventID {
	if c.closed.Load() {
		return nil
	}

	c.stats.eventsTotal.Add(1)

	eventID := c.hub.RecoverWithContext(nil, recovered)
	if eventID != nil && *eventID != "" {
		c.stats.eventsCaptured.Add(1)

		// 触发钩子
		c.hooks.trigger(&sentry.Event{
			EventID: *eventID,
			Level:   sentry.LevelFatal,
		})
	} else {
		c.stats.eventsDropped.Add(1)
	}

	return eventID
}

// Hub 获取底层 Hub（高级用户使用）
func (c *Client) Hub() *sentry.Hub {
	return c.hub
}

// RegisterHook 注册事件钩子
func (c *Client) RegisterHook(hook EventHook) {
	c.hooks.register(hook)
}

// UnregisterHook 注销事件钩子
func (c *Client) UnregisterHook(hook EventHook) {
	c.hooks.unregister(hook)
}

// Flush 等待所有事件上报完成
func (c *Client) Flush(timeout time.Duration) bool {
	return c.hub.Flush(timeout)
}

// Close 关闭客户端
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return ErrClientClosed
	}

	c.hub.Flush(c.config.ShutdownTimeout)
	return nil
}

// Stats 获取统计信息
func (c *Client) Stats() Stats {
	return Stats{
		EventsTotal:    c.stats.eventsTotal.Load(),
		EventsCaptured: c.stats.eventsCaptured.Load(),
		EventsDropped:  c.stats.eventsDropped.Load(),
	}
}

// IsClosed 是否已关闭
func (c *Client) IsClosed() bool {
	return c.closed.Load()
}
