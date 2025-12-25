// pkg/websocket/server_metrics.go
package websocket

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ServerMetrics 服务端指标
type ServerMetrics struct {
	// 连接指标
	activeConnections prometheus.Gauge
	totalConnections  prometheus.Counter
	connectionErrors  *prometheus.CounterVec

	// 消息指标
	messagesSent     prometheus.Counter
	messagesReceived prometheus.Counter
	bytesSent        prometheus.Counter
	bytesReceived    prometheus.Counter

	// 升级指标
	upgradeTotal  prometheus.Counter
	upgradeErrors prometheus.Counter
}

// NewServerMetrics 创建服务端指标
func NewServerMetrics(registerer prometheus.Registerer) *ServerMetrics {
	m := &ServerMetrics{
		activeConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "active_connections",
			Help:      "Number of active WebSocket connections",
		}),
		totalConnections: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "connections_total",
			Help:      "Total number of WebSocket connections",
		}),
		connectionErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "connection_errors_total",
			Help:      "Total number of connection errors",
		}, []string{"type"}),
		messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "messages_sent_total",
			Help:      "Total number of messages sent",
		}),
		messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "messages_received_total",
			Help:      "Total number of messages received",
		}),
		bytesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "bytes_sent_total",
			Help:      "Total bytes sent",
		}),
		bytesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "bytes_received_total",
			Help:      "Total bytes received",
		}),
		upgradeTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "upgrades_total",
			Help:      "Total number of WebSocket upgrades",
		}),
		upgradeErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "server",
			Name:      "upgrade_errors_total",
			Help:      "Total number of upgrade errors",
		}),
	}

	// 注册指标
	if registerer != nil {
		registerer.MustRegister(
			m.activeConnections,
			m.totalConnections,
			m.connectionErrors,
			m.messagesSent,
			m.messagesReceived,
			m.bytesSent,
			m.bytesReceived,
			m.upgradeTotal,
			m.upgradeErrors,
		)
	}

	return m
}

// OnConnectionOpened 连接打开
func (m *ServerMetrics) OnConnectionOpened() {
	m.activeConnections.Inc()
	m.totalConnections.Inc()
	m.upgradeTotal.Inc()
}

// OnConnectionClosed 连接关闭
func (m *ServerMetrics) OnConnectionClosed() {
	m.activeConnections.Dec()
}

// OnConnectionError 连接错误
func (m *ServerMetrics) OnConnectionError(errType string) {
	m.connectionErrors.WithLabelValues(errType).Inc()
}

// OnUpgradeError 升级错误
func (m *ServerMetrics) OnUpgradeError() {
	m.upgradeErrors.Inc()
}

// OnMessageSent 消息发送
func (m *ServerMetrics) OnMessageSent(size int64) {
	m.messagesSent.Inc()
	m.bytesSent.Add(float64(size))
}

// OnMessageReceived 消息接收
func (m *ServerMetrics) OnMessageReceived(size int64) {
	m.messagesReceived.Inc()
	m.bytesReceived.Add(float64(size))
}
