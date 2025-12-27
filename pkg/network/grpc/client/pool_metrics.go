package client

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// PoolMetrics 连接池 Prometheus 指标
type PoolMetrics struct {
	// 连接池容量（配置值）
	poolCapacity prometheus.Gauge

	// 当前活跃连接数（正在使用的连接）
	activeConnections prometheus.Gauge

	// 当前空闲连接数
	idleConnections prometheus.Gauge

	// 获取连接的等待时间分布
	getConnectionDuration prometheus.Histogram

	// 获取连接失败次数
	getConnectionErrors prometheus.Counter

	// 获取连接超时次数
	getConnectionTimeouts prometheus.Counter

	// 健康检查失败次数
	healthCheckFailures prometheus.Counter

	// 连接重建次数
	connectionRecreations prometheus.Counter

	// 连接总数
	totalConnections prometheus.Gauge
}

// NewPoolMetrics 创建连接池指标
func NewPoolMetrics(namespace, subsystem, target string, registerer prometheus.Registerer) *PoolMetrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	labels := prometheus.Labels{"target": target}

	m := &PoolMetrics{
		poolCapacity: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_capacity",
			Help:        "Configured capacity of the connection pool",
			ConstLabels: labels,
		}),

		activeConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_active_connections",
			Help:        "Current number of active (in-use) connections in the pool",
			ConstLabels: labels,
		}),

		idleConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_idle_connections",
			Help:        "Current number of idle connections in the pool",
			ConstLabels: labels,
		}),

		getConnectionDuration: prometheus.NewHistogram(prometheus.HistogramOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_get_connection_duration_seconds",
			Help:        "Time spent waiting to get a connection from the pool",
			ConstLabels: labels,
			Buckets:     []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10},
		}),

		getConnectionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_get_connection_errors_total",
			Help:        "Total number of errors when getting a connection from the pool",
			ConstLabels: labels,
		}),

		getConnectionTimeouts: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_get_connection_timeouts_total",
			Help:        "Total number of timeouts when getting a connection from the pool",
			ConstLabels: labels,
		}),

		healthCheckFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_health_check_failures_total",
			Help:        "Total number of health check failures",
			ConstLabels: labels,
		}),

		connectionRecreations: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_connection_recreations_total",
			Help:        "Total number of connection recreations due to health check failures",
			ConstLabels: labels,
		}),

		totalConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace:   namespace,
			Subsystem:   subsystem,
			Name:        "pool_total_connections",
			Help:        "Total number of connections in the pool",
			ConstLabels: labels,
		}),
	}

	// 注册指标
	registerer.MustRegister(
		m.poolCapacity,
		m.activeConnections,
		m.idleConnections,
		m.getConnectionDuration,
		m.getConnectionErrors,
		m.getConnectionTimeouts,
		m.healthCheckFailures,
		m.connectionRecreations,
		m.totalConnections,
	)

	return m
}

// RecordGetConnection 记录获取连接的操作
func (m *PoolMetrics) RecordGetConnection(duration time.Duration, err error, timeout bool) {
	m.getConnectionDuration.Observe(duration.Seconds())
	if err != nil {
		m.getConnectionErrors.Inc()
	}
	if timeout {
		m.getConnectionTimeouts.Inc()
	}
}

// RecordHealthCheckFailure 记录健康检查失败
func (m *PoolMetrics) RecordHealthCheckFailure() {
	m.healthCheckFailures.Inc()
}

// RecordConnectionRecreation 记录连接重建
func (m *PoolMetrics) RecordConnectionRecreation() {
	m.connectionRecreations.Inc()
}

// UpdatePoolStats 更新连接池统计信息
func (m *PoolMetrics) UpdatePoolStats(capacity, active, idle, total int) {
	m.poolCapacity.Set(float64(capacity))
	m.activeConnections.Set(float64(active))
	m.idleConnections.Set(float64(idle))
	m.totalConnections.Set(float64(total))
}

// Unregister 取消注册所有指标
func (m *PoolMetrics) Unregister(registerer prometheus.Registerer) {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	registerer.Unregister(m.poolCapacity)
	registerer.Unregister(m.activeConnections)
	registerer.Unregister(m.idleConnections)
	registerer.Unregister(m.getConnectionDuration)
	registerer.Unregister(m.getConnectionErrors)
	registerer.Unregister(m.getConnectionTimeouts)
	registerer.Unregister(m.healthCheckFailures)
	registerer.Unregister(m.connectionRecreations)
	registerer.Unregister(m.totalConnections)
}
