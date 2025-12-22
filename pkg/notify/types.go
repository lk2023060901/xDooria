package notify

import "time"

// Alert 统一告警结构（平台无关）
type Alert struct {
	// 基础信息
	Level       AlertLevel // critical/warning/info
	Service     string     // 服务名
	Summary     string     // 一句话摘要
	Description string     // 详细描述

	// 元数据
	Labels      map[string]string // 自定义标签
	Fingerprint string            // 告警指纹（用于去重）

	// 时间
	StartsAt time.Time // 告警开始时间
	EndsAt   time.Time // 恢复时间（零值表示未恢复）

	// 交互
	DashboardURL string // Grafana/Kibana 链接
	RunbookURL   string // 处理手册链接
	AtUsers      []User // @ 的用户列表
	AtAll        bool   // 是否 @ 所有人
}

// AlertLevel 告警级别
type AlertLevel string

const (
	AlertLevelCritical AlertLevel = "critical" // 严重
	AlertLevelWarning  AlertLevel = "warning"  // 警告
	AlertLevelInfo     AlertLevel = "info"     // 信息
)

// User 用户标识（多平台兼容）
type User struct {
	ID     string // 飞书 user_id/open_id，Slack user_id
	Name   string // 显示名称（可选）
	Mobile string // 钉钉手机号
	Email  string // 邮箱（可选）
}
