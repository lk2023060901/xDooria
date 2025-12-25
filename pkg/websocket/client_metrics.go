// pkg/websocket/client_metrics.go
package websocket

import (
	"github.com/prometheus/client_golang/prometheus"
)

// ClientMetrics 客户端指标
type ClientMetrics struct {
	// 连接指标
	connectionState  prometheus.Gauge
	connectionsTotal prometheus.Counter
	disconnectsTotal prometheus.Counter

	// 重连指标
	reconnectAttempts prometheus.Counter
	reconnectSuccess  prometheus.Counter

	// 心跳指标
	heartbeatSent     prometheus.Counter
	heartbeatReceived prometheus.Counter
	heartbeatTimeouts prometheus.Counter

	// 消息指标
	messagesSent     prometheus.Counter
	messagesReceived prometheus.Counter
	bytesSent        prometheus.Counter
	bytesReceived    prometheus.Counter

	// 错误指标
	errors *prometheus.CounterVec
}

// NewClientMetrics 创建客户端指标
func NewClientMetrics(registerer prometheus.Registerer) *ClientMetrics {
	m := &ClientMetrics{
		connectionState: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "connection_state",
			Help:      "Current connection state (0=disconnected, 1=connecting, 2=connected, 3=reconnecting, 4=closed)",
		}),
		connectionsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "connections_total",
			Help:      "Total number of connections established",
		}),
		disconnectsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "disconnects_total",
			Help:      "Total number of disconnections",
		}),
		reconnectAttempts: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "reconnect_attempts_total",
			Help:      "Total number of reconnection attempts",
		}),
		reconnectSuccess: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "reconnect_success_total",
			Help:      "Total number of successful reconnections",
		}),
		heartbeatSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "heartbeat_sent_total",
			Help:      "Total number of heartbeat pings sent",
		}),
		heartbeatReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "heartbeat_received_total",
			Help:      "Total number of heartbeat pongs received",
		}),
		heartbeatTimeouts: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "heartbeat_timeouts_total",
			Help:      "Total number of heartbeat timeouts",
		}),
		messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "messages_sent_total",
			Help:      "Total number of messages sent",
		}),
		messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "messages_received_total",
			Help:      "Total number of messages received",
		}),
		bytesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "bytes_sent_total",
			Help:      "Total bytes sent",
		}),
		bytesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "bytes_received_total",
			Help:      "Total bytes received",
		}),
		errors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "websocket",
			Subsystem: "client",
			Name:      "errors_total",
			Help:      "Total number of errors",
		}, []string{"type"}),
	}

	// 注册指标
	if registerer != nil {
		registerer.MustRegister(
			m.connectionState,
			m.connectionsTotal,
			m.disconnectsTotal,
			m.reconnectAttempts,
			m.reconnectSuccess,
			m.heartbeatSent,
			m.heartbeatReceived,
			m.heartbeatTimeouts,
			m.messagesSent,
			m.messagesReceived,
			m.bytesSent,
			m.bytesReceived,
			m.errors,
		)
	}

	return m
}

// OnConnected 连接成功
func (m *ClientMetrics) OnConnected() {
	m.connectionState.Set(float64(StateConnected))
	m.connectionsTotal.Inc()
}

// OnDisconnected 断开连接
func (m *ClientMetrics) OnDisconnected() {
	m.connectionState.Set(float64(StateDisconnected))
	m.disconnectsTotal.Inc()
}

// OnReconnecting 重连中
func (m *ClientMetrics) OnReconnecting() {
	m.connectionState.Set(float64(StateReconnecting))
}

// OnReconnectAttempt 重连尝试
func (m *ClientMetrics) OnReconnectAttempt() {
	m.reconnectAttempts.Inc()
}

// OnReconnected 重连成功
func (m *ClientMetrics) OnReconnected() {
	m.connectionState.Set(float64(StateConnected))
	m.reconnectSuccess.Inc()
}

// OnHeartbeatSent 发送心跳
func (m *ClientMetrics) OnHeartbeatSent() {
	m.heartbeatSent.Inc()
}

// OnHeartbeatReceived 收到心跳响应
func (m *ClientMetrics) OnHeartbeatReceived() {
	m.heartbeatReceived.Inc()
}

// OnHeartbeatTimeout 心跳超时
func (m *ClientMetrics) OnHeartbeatTimeout() {
	m.heartbeatTimeouts.Inc()
}

// OnMessageSent 消息发送
func (m *ClientMetrics) OnMessageSent(size int64) {
	m.messagesSent.Inc()
	m.bytesSent.Add(float64(size))
}

// OnMessageReceived 消息接收
func (m *ClientMetrics) OnMessageReceived(size int64) {
	m.messagesReceived.Inc()
	m.bytesReceived.Add(float64(size))
}

// OnError 错误
func (m *ClientMetrics) OnError(errType string) {
	m.errors.WithLabelValues(errType).Inc()
}
