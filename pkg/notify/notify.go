package notify

import "context"

// Notifier 通知器接口（核心抽象）
type Notifier interface {
	// Send 发送告警
	Send(ctx context.Context, alert *Alert) error

	// SendBatch 批量发送（可选优化）
	SendBatch(ctx context.Context, alerts []*Alert) error

	// Name 返回通知器名称（用于日志）
	Name() string
}
