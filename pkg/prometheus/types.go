package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Labels 标签类型
type Labels = prometheus.Labels

// Counter 计数器接口
type Counter interface {
	Inc()
	Add(float64)
}

// Gauge 仪表盘接口
type Gauge interface {
	Set(float64)
	Inc()
	Dec()
	Add(float64)
	Sub(float64)
}

// Histogram 直方图接口
type Histogram interface {
	Observe(float64)
}

// Summary 摘要接口
type Summary interface {
	Observe(float64)
}

// CounterVec Counter 向量
type CounterVec = prometheus.CounterVec

// GaugeVec Gauge 向量
type GaugeVec = prometheus.GaugeVec

// HistogramVec Histogram 向量
type HistogramVec = prometheus.HistogramVec

// SummaryVec Summary 向量
type SummaryVec = prometheus.SummaryVec

// Registry Prometheus 注册器
type Registry = prometheus.Registry

// Collector 采集器接口
type Collector = prometheus.Collector
