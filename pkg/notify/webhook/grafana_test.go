package webhook

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/lk2023060901/xdooria/pkg/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGrafanaPayload_ToAlerts 测试 Grafana Payload 转换
func TestGrafanaPayload_ToAlerts(t *testing.T) {
	// 模拟 Grafana Webhook JSON
	jsonData := `{
		"receiver": "feishu-webhook",
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {
					"alertname": "HighCPU",
					"service": "api-server",
					"instance": "server-01",
					"severity": "critical"
				},
				"annotations": {
					"summary": "CPU 使用率超过 80%",
					"description": "服务器 server-01 的 CPU 使用率持续超过 80%，需要立即检查",
					"runbook_url": "https://wiki.example.com/runbook/high-cpu"
				},
				"startsAt": "2023-10-01T10:00:00Z",
				"endsAt": "0001-01-01T00:00:00Z",
				"generatorURL": "http://grafana.example.com/alerting/grafana/abc123/view",
				"fingerprint": "abc123def456",
				"dashboardURL": "http://grafana.example.com/d/cpu-dashboard",
				"panelURL": "http://grafana.example.com/d/cpu-dashboard?viewPanel=1"
			}
		],
		"groupLabels": {
			"alertname": "HighCPU"
		},
		"commonLabels": {
			"alertname": "HighCPU",
			"service": "api-server"
		},
		"commonAnnotations": {},
		"externalURL": "http://grafana.example.com",
		"version": "1",
		"groupKey": "{}:{alertname=\"HighCPU\"}",
		"truncatedAlerts": 0
	}`

	var payload GrafanaPayload
	err := json.Unmarshal([]byte(jsonData), &payload)
	require.NoError(t, err)

	// 转换为 notify.Alert
	alerts := payload.ToAlerts()

	// 断言
	require.Len(t, alerts, 1)

	alert := alerts[0]
	assert.Equal(t, notify.AlertLevelCritical, alert.Level)
	assert.Equal(t, "api-server", alert.Service)
	assert.Equal(t, "CPU 使用率超过 80%", alert.Summary)
	assert.Equal(t, "服务器 server-01 的 CPU 使用率持续超过 80%，需要立即检查", alert.Description)
	assert.Equal(t, "abc123def456", alert.Fingerprint)
	assert.Equal(t, "http://grafana.example.com/d/cpu-dashboard", alert.DashboardURL)
	assert.Equal(t, "https://wiki.example.com/runbook/high-cpu", alert.RunbookURL)
	assert.Equal(t, "HighCPU", alert.Labels["alertname"])
	assert.Equal(t, "critical", alert.Labels["severity"])
	assert.False(t, alert.StartsAt.IsZero())
}

// TestGrafanaPayload_ToAlerts_Resolved 测试 resolved 状态
func TestGrafanaPayload_ToAlerts_Resolved(t *testing.T) {
	jsonData := `{
		"status": "resolved",
		"alerts": [
			{
				"status": "resolved",
				"labels": {
					"alertname": "HighCPU",
					"service": "api-server"
				},
				"annotations": {
					"summary": "CPU 使用率已恢复正常"
				},
				"startsAt": "2023-10-01T10:00:00Z",
				"endsAt": "2023-10-01T10:15:00Z",
				"fingerprint": "abc123"
			}
		]
	}`

	var payload GrafanaPayload
	err := json.Unmarshal([]byte(jsonData), &payload)
	require.NoError(t, err)

	alerts := payload.ToAlerts()
	require.Len(t, alerts, 1)

	alert := alerts[0]
	assert.Equal(t, notify.AlertLevelInfo, alert.Level)
	assert.Equal(t, "CPU 使用率已恢复正常", alert.Summary)
	assert.False(t, alert.EndsAt.IsZero())
}

// TestGrafanaPayload_ToAlerts_MultipleAlerts 测试多条告警
func TestGrafanaPayload_ToAlerts_MultipleAlerts(t *testing.T) {
	jsonData := `{
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {"alertname": "HighCPU", "service": "api-server"},
				"annotations": {"summary": "CPU 高"}
			},
			{
				"status": "firing",
				"labels": {"alertname": "HighMemory", "service": "db-server"},
				"annotations": {"summary": "内存高"}
			},
			{
				"status": "firing",
				"labels": {"alertname": "DiskFull", "service": "storage-server"},
				"annotations": {"summary": "磁盘满"}
			}
		]
	}`

	var payload GrafanaPayload
	err := json.Unmarshal([]byte(jsonData), &payload)
	require.NoError(t, err)

	alerts := payload.ToAlerts()
	require.Len(t, alerts, 3)

	assert.Equal(t, "CPU 高", alerts[0].Summary)
	assert.Equal(t, "api-server", alerts[0].Service)

	assert.Equal(t, "内存高", alerts[1].Summary)
	assert.Equal(t, "db-server", alerts[1].Service)

	assert.Equal(t, "磁盘满", alerts[2].Summary)
	assert.Equal(t, "storage-server", alerts[2].Service)
}

// TestExtractService 测试服务名提取逻辑
func TestExtractService(t *testing.T) {
	p := &GrafanaPayload{}

	tests := []struct {
		name     string
		labels   map[string]string
		expected string
	}{
		{
			name:     "使用 service 字段",
			labels:   map[string]string{"service": "api-server"},
			expected: "api-server",
		},
		{
			name:     "使用 job 字段",
			labels:   map[string]string{"job": "prometheus"},
			expected: "prometheus",
		},
		{
			name:     "使用 instance 字段",
			labels:   map[string]string{"instance": "server-01"},
			expected: "server-01",
		},
		{
			name:     "使用 alertname 字段",
			labels:   map[string]string{"alertname": "HighCPU"},
			expected: "HighCPU",
		},
		{
			name:     "无可用字段",
			labels:   map[string]string{},
			expected: "未知服务",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.extractService(tt.labels)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestParseTime 测试时间解析
func TestParseTime(t *testing.T) {
	p := &GrafanaPayload{}

	tests := []struct {
		name     string
		timeStr  string
		isZero   bool
	}{
		{
			name:    "RFC3339 格式",
			timeStr: "2023-10-01T10:00:00Z",
			isZero:  false,
		},
		{
			name:    "RFC3339Nano 格式",
			timeStr: "2023-10-01T10:00:00.123456789Z",
			isZero:  false,
		},
		{
			name:    "空字符串",
			timeStr: "",
			isZero:  true,
		},
		{
			name:    "无效格式",
			timeStr: "invalid-time",
			isZero:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := p.parseTime(tt.timeStr)
			assert.Equal(t, tt.isZero, result.IsZero())
			if !tt.isZero {
				assert.True(t, result.After(time.Time{}))
			}
		})
	}
}
