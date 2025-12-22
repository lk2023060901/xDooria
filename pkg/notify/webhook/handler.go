package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lk2023060901/xdooria/pkg/notify"
	"go.uber.org/zap"
)

// Handler Webhook HTTP Handler
type Handler struct {
	notifier notify.Notifier
	logger   *zap.Logger
}

// NewHandler 创建 Webhook Handler
func NewHandler(notifier notify.Notifier, logger *zap.Logger) *Handler {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Handler{
		notifier: notifier,
		logger:   logger,
	}
}

// HandleGrafana 处理 Grafana Webhook 请求
func (h *Handler) HandleGrafana(w http.ResponseWriter, r *http.Request) {
	// 1. 只接受 POST 请求
	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "只支持 POST 方法")
		return
	}

	// 2. 读取请求体
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("读取请求体失败", zap.Error(err))
		h.respondError(w, http.StatusBadRequest, "读取请求体失败")
		return
	}
	defer r.Body.Close()

	// 3. 解析 JSON
	var payload GrafanaPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		h.logger.Error("解析 Grafana Webhook JSON 失败",
			zap.Error(err),
			zap.String("body", string(body)))
		h.respondError(w, http.StatusBadRequest, "JSON 格式错误")
		return
	}

	// 4. 转换为 notify.Alert
	alerts := payload.ToAlerts()
	if len(alerts) == 0 {
		h.logger.Warn("Grafana Webhook 中没有告警")
		h.respondSuccess(w, "没有告警需要发送")
		return
	}

	// 5. 发送告警
	ctx := r.Context()
	if err := h.sendAlerts(ctx, alerts); err != nil {
		h.logger.Error("发送告警失败",
			zap.Error(err),
			zap.Int("alert_count", len(alerts)))
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("发送告警失败: %v", err))
		return
	}

	// 6. 返回成功
	h.logger.Info("成功处理 Grafana Webhook",
		zap.Int("alert_count", len(alerts)),
		zap.String("status", payload.Status))
	h.respondSuccess(w, fmt.Sprintf("成功发送 %d 条告警", len(alerts)))
}

// sendAlerts 发送告警（支持批量）
func (h *Handler) sendAlerts(ctx context.Context, alerts []*notify.Alert) error {
	// 如果 notifier 支持批量发送，使用批量接口
	if batchNotifier, ok := h.notifier.(interface {
		SendBatch(context.Context, []*notify.Alert) error
	}); ok {
		return batchNotifier.SendBatch(ctx, alerts)
	}

	// 否则逐条发送
	for _, alert := range alerts {
		if err := h.notifier.Send(ctx, alert); err != nil {
			return fmt.Errorf("发送告警失败 (fingerprint=%s): %w", alert.Fingerprint, err)
		}
	}
	return nil
}

// respondError 返回错误响应
func (h *Handler) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

// respondSuccess 返回成功响应
func (h *Handler) respondSuccess(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": message,
	})
}
