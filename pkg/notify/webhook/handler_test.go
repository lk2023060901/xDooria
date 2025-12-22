package webhook

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lk2023060901/xdooria/pkg/notify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// MockNotifier 模拟 Notifier
type MockNotifier struct {
	sendCalled      bool
	sendBatchCalled bool
	lastAlert       *notify.Alert
	lastAlerts      []*notify.Alert
	sendError       error
}

func (m *MockNotifier) Send(ctx context.Context, alert *notify.Alert) error {
	m.sendCalled = true
	m.lastAlert = alert
	return m.sendError
}

func (m *MockNotifier) SendBatch(ctx context.Context, alerts []*notify.Alert) error {
	m.sendBatchCalled = true
	m.lastAlerts = alerts
	return m.sendError
}

func (m *MockNotifier) Name() string {
	return "mock"
}

// TestHandler_HandleGrafana_Success 测试成功处理
func TestHandler_HandleGrafana_Success(t *testing.T) {
	mockNotifier := &MockNotifier{}
	handler := NewHandler(mockNotifier, zap.NewNop())

	// 构造 Grafana Webhook 请求
	body := `{
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {
					"alertname": "HighCPU",
					"service": "api-server"
				},
				"annotations": {
					"summary": "CPU 使用率过高"
				},
				"startsAt": "2023-10-01T10:00:00Z",
				"fingerprint": "abc123"
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 调用 Handler
	handler.HandleGrafana(w, req)

	// 断言
	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockNotifier.sendBatchCalled)
	require.Len(t, mockNotifier.lastAlerts, 1)
	assert.Equal(t, "api-server", mockNotifier.lastAlerts[0].Service)
	assert.Equal(t, "CPU 使用率过高", mockNotifier.lastAlerts[0].Summary)
}

// TestHandler_HandleGrafana_InvalidMethod 测试错误的 HTTP 方法
func TestHandler_HandleGrafana_InvalidMethod(t *testing.T) {
	mockNotifier := &MockNotifier{}
	handler := NewHandler(mockNotifier, zap.NewNop())

	req := httptest.NewRequest(http.MethodGet, "/webhook/grafana", nil)
	w := httptest.NewRecorder()

	handler.HandleGrafana(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	assert.False(t, mockNotifier.sendCalled)
	assert.False(t, mockNotifier.sendBatchCalled)
}

// TestHandler_HandleGrafana_InvalidJSON 测试无效 JSON
func TestHandler_HandleGrafana_InvalidJSON(t *testing.T) {
	mockNotifier := &MockNotifier{}
	handler := NewHandler(mockNotifier, zap.NewNop())

	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana", bytes.NewBufferString("invalid json"))
	w := httptest.NewRecorder()

	handler.HandleGrafana(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.False(t, mockNotifier.sendCalled)
}

// TestHandler_HandleGrafana_NoAlerts 测试空告警
func TestHandler_HandleGrafana_NoAlerts(t *testing.T) {
	mockNotifier := &MockNotifier{}
	handler := NewHandler(mockNotifier, zap.NewNop())

	body := `{"status": "firing", "alerts": []}`
	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handler.HandleGrafana(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.False(t, mockNotifier.sendCalled)
	assert.False(t, mockNotifier.sendBatchCalled)
}

// TestHandler_HandleGrafana_MultipleAlerts 测试多条告警
func TestHandler_HandleGrafana_MultipleAlerts(t *testing.T) {
	mockNotifier := &MockNotifier{}
	handler := NewHandler(mockNotifier, zap.NewNop())

	body := `{
		"status": "firing",
		"alerts": [
			{
				"status": "firing",
				"labels": {"alertname": "Alert1", "service": "service1"},
				"annotations": {"summary": "告警1"}
			},
			{
				"status": "firing",
				"labels": {"alertname": "Alert2", "service": "service2"},
				"annotations": {"summary": "告警2"}
			}
		]
	}`

	req := httptest.NewRequest(http.MethodPost, "/webhook/grafana", bytes.NewBufferString(body))
	w := httptest.NewRecorder()

	handler.HandleGrafana(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.True(t, mockNotifier.sendBatchCalled)
	require.Len(t, mockNotifier.lastAlerts, 2)
	assert.Equal(t, "告警1", mockNotifier.lastAlerts[0].Summary)
	assert.Equal(t, "告警2", mockNotifier.lastAlerts[1].Summary)
}
