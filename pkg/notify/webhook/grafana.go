package webhook

import (
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/notify"
)

// GrafanaPayload Grafana Webhook 的 JSON 格式
// 参考: https://grafana.com/docs/grafana/latest/alerting/manage-notifications/webhook-notifier/
type GrafanaPayload struct {
	Receiver          string         `json:"receiver"`
	Status            string         `json:"status"`
	Alerts            []GrafanaAlert `json:"alerts"`
	GroupLabels       map[string]string `json:"groupLabels"`
	CommonLabels      map[string]string `json:"commonLabels"`
	CommonAnnotations map[string]string `json:"commonAnnotations"`
	ExternalURL       string         `json:"externalURL"`
	Version           string         `json:"version"`
	GroupKey          string         `json:"groupKey"`
	TruncatedAlerts   int            `json:"truncatedAlerts"`
	Title             string         `json:"title"`
	State             string         `json:"state"`
	Message           string         `json:"message"`
}

// GrafanaAlert Grafana 单条告警
type GrafanaAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     string            `json:"startsAt"`
	EndsAt       string            `json:"endsAt"`
	GeneratorURL string            `json:"generatorURL"`
	Fingerprint  string            `json:"fingerprint"`
	SilenceURL   string            `json:"silenceURL"`
	DashboardURL string            `json:"dashboardURL"`
	PanelURL     string            `json:"panelURL"`
	ValueString  string            `json:"valueString"`
}

// ToAlerts 转换为统一的 notify.Alert 列表
func (p *GrafanaPayload) ToAlerts() []*notify.Alert {
	alerts := make([]*notify.Alert, 0, len(p.Alerts))

	for _, grafanaAlert := range p.Alerts {
		alert := &notify.Alert{
			Level:        p.mapAlertLevel(grafanaAlert.Status),
			Service:      p.extractService(grafanaAlert.Labels),
			Summary:      p.extractSummary(grafanaAlert),
			Description:  p.extractDescription(grafanaAlert),
			Labels:       grafanaAlert.Labels,
			Fingerprint:  grafanaAlert.Fingerprint,
			StartsAt:     p.parseTime(grafanaAlert.StartsAt),
			EndsAt:       p.parseTime(grafanaAlert.EndsAt),
			DashboardURL: p.selectDashboardURL(grafanaAlert),
			RunbookURL:   p.extractRunbookURL(grafanaAlert.Annotations),
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// mapAlertLevel 映射 Grafana 状态到告警级别
func (p *GrafanaPayload) mapAlertLevel(status string) notify.AlertLevel {
	switch status {
	case "firing":
		// Grafana 没有明确的严重程度，默认为 Critical
		// 可以通过 Labels 中的 severity 字段来判断
		return notify.AlertLevelCritical
	case "resolved":
		return notify.AlertLevelInfo
	default:
		return notify.AlertLevelWarning
	}
}

// extractService 从 Labels 中提取服务名
func (p *GrafanaPayload) extractService(labels map[string]string) string {
	// 尝试多个可能的字段
	if service, ok := labels["service"]; ok {
		return service
	}
	if job, ok := labels["job"]; ok {
		return job
	}
	if instance, ok := labels["instance"]; ok {
		return instance
	}
	if alertname, ok := labels["alertname"]; ok {
		return alertname
	}
	return "未知服务"
}

// extractSummary 提取告警摘要
func (p *GrafanaPayload) extractSummary(alert GrafanaAlert) string {
	// 优先使用 annotations 中的 summary
	if summary, ok := alert.Annotations["summary"]; ok && summary != "" {
		return summary
	}
	if description, ok := alert.Annotations["description"]; ok && description != "" {
		return description
	}
	// 使用 alertname
	if alertname, ok := alert.Labels["alertname"]; ok {
		return alertname
	}
	// 使用 message
	if alert.ValueString != "" {
		return alert.ValueString
	}
	return "告警触发"
}

// extractDescription 提取详细描述
func (p *GrafanaPayload) extractDescription(alert GrafanaAlert) string {
	// 如果 summary 和 description 都存在，description 作为详情
	if description, ok := alert.Annotations["description"]; ok && description != "" {
		return description
	}
	// 否则使用 message
	if message, ok := alert.Annotations["message"]; ok && message != "" {
		return message
	}
	// 拼接关键 Labels
	if len(alert.Labels) > 0 {
		desc := ""
		for k, v := range alert.Labels {
			if k != "alertname" && k != "__name__" {
				desc += fmt.Sprintf("%s: %s\n", k, v)
			}
		}
		return desc
	}
	return ""
}

// selectDashboardURL 选择 Dashboard URL
func (p *GrafanaPayload) selectDashboardURL(alert GrafanaAlert) string {
	if alert.DashboardURL != "" {
		return alert.DashboardURL
	}
	if alert.PanelURL != "" {
		return alert.PanelURL
	}
	if alert.GeneratorURL != "" {
		return alert.GeneratorURL
	}
	return ""
}

// extractRunbookURL 从 annotations 中提取 Runbook URL
func (p *GrafanaPayload) extractRunbookURL(annotations map[string]string) string {
	if runbook, ok := annotations["runbook_url"]; ok {
		return runbook
	}
	if runbook, ok := annotations["runbook"]; ok {
		return runbook
	}
	return ""
}

// parseTime 解析时间字符串
func (p *GrafanaPayload) parseTime(timeStr string) time.Time {
	if timeStr == "" {
		return time.Time{}
	}
	// 尝试多种时间格式
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05.999Z",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	return time.Time{}
}
