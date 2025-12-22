package feishu

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/notify"
)

// Adapter 飞书适配器（实现 notify.Notifier 接口）
type Adapter struct {
	client *Client
}

// NewAdapter 创建飞书适配器
func NewAdapter(cfg *Config) (*Adapter, error) {
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}

	return &Adapter{
		client: client,
	}, nil
}

// Send 实现 notify.Notifier 接口
func (a *Adapter) Send(ctx context.Context, alert *notify.Alert) error {
	msg := a.convertToPost(alert)
	return a.client.Send(msg)
}

// SendBatch 批量发送
func (a *Adapter) SendBatch(ctx context.Context, alerts []*notify.Alert) error {
	for _, alert := range alerts {
		if err := a.Send(ctx, alert); err != nil {
			return err
		}
	}
	return nil
}

// Name 实现 notify.Notifier 接口
func (a *Adapter) Name() string {
	return "feishu"
}

// convertToPost 转换为飞书富文本格式
func (a *Adapter) convertToPost(alert *notify.Alert) *PostMessage {
	emoji := a.getLevelEmoji(alert.Level)
	title := fmt.Sprintf("%s %s告警", emoji, alert.Service)

	msg := NewPostMessage(title)

	// 第一行：级别和服务
	msg.AddLine(
		Text(fmt.Sprintf("%s ", emoji)),
		Text(fmt.Sprintf("%s告警 - ", a.getLevelText(alert.Level))),
		Text(alert.Service),
	)

	// 第二行：摘要
	msg.AddLine(Text(fmt.Sprintf("摘要: %s", alert.Summary)))

	// 详情
	if alert.Description != "" {
		msg.AddLine(Text(fmt.Sprintf("详情: %s", alert.Description)))
	}

	// 标签
	if len(alert.Labels) > 0 {
		msg.AddLine(Text("标签:"))
		for k, v := range alert.Labels {
			msg.AddLine(Text(fmt.Sprintf("  • %s: %s", k, v)))
		}
	}

	// 时间
	if !alert.StartsAt.IsZero() {
		msg.AddLine(Text(fmt.Sprintf("开始时间: %s",
			alert.StartsAt.Format("2006-01-02 15:04:05"))))
	}

	if !alert.EndsAt.IsZero() {
		msg.AddLine(Text(fmt.Sprintf("恢复时间: %s",
			alert.EndsAt.Format("2006-01-02 15:04:05"))))
	}

	// Dashboard 链接
	if alert.DashboardURL != "" {
		msg.AddLine(Link("📊 查看监控大盘", alert.DashboardURL))
	}

	// Runbook 链接
	if alert.RunbookURL != "" {
		msg.AddLine(Link("📖 处理手册", alert.RunbookURL))
	}

	// @ 用户（飞书使用 user_id）
	if alert.AtAll {
		msg.AddLine(Text("相关人员: "), AtAll())
	} else if len(alert.AtUsers) > 0 {
		elements := []MessageElement{Text("相关人员: ")}
		for _, user := range alert.AtUsers {
			if user.ID != "" {
				elements = append(elements, At(user.ID))
				elements = append(elements, Text(" "))
			}
		}
		msg.AddLine(elements...)
	}

	return msg
}

func (a *Adapter) getLevelEmoji(level notify.AlertLevel) string {
	switch level {
	case notify.AlertLevelCritical:
		return "🔴"
	case notify.AlertLevelWarning:
		return "🟡"
	case notify.AlertLevelInfo:
		return "🟢"
	default:
		return "ℹ️"
	}
}

func (a *Adapter) getLevelText(level notify.AlertLevel) string {
	switch level {
	case notify.AlertLevelCritical:
		return "严重"
	case notify.AlertLevelWarning:
		return "警告"
	case notify.AlertLevelInfo:
		return "信息"
	default:
		return "未知"
	}
}
