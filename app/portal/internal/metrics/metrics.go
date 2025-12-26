package metrics

import (
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/metrics/sliding"
	"github.com/lk2023060901/xdooria/pkg/metrics/system"
	"github.com/prometheus/client_golang/prometheus"
)

// Config 指标配置
type Config struct {
	// Namespace 指标命名空间
	Namespace string `mapstructure:"namespace" json:"namespace" yaml:"namespace"`
	// System 系统指标采集间隔
	SystemCollectInterval time.Duration `mapstructure:"system_collect_interval" json:"system_collect_interval" yaml:"system_collect_interval"`
	// SlidingWindow 滑动窗口配置
	SlidingWindow sliding.WindowConfig `mapstructure:"sliding_window" json:"sliding_window" yaml:"sliding_window"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Namespace:             "portal",
		SystemCollectInterval: 5 * time.Second,
		SlidingWindow:         *sliding.DefaultWindowConfig(),
	}
}

// PortalMetrics Portal 服务指标
type PortalMetrics struct {
	config *Config

	// HTTP 请求总数
	RequestTotal *prometheus.CounterVec
	// HTTP 请求延迟
	RequestDuration *prometheus.HistogramVec
	// 当前活跃请求数
	ActiveRequests prometheus.Gauge

	// 系统指标收集器
	systemCollector *system.Collector
	// 滑动窗口统计
	slidingWindow *sliding.Window
}

// New 创建 Portal 指标
func New(cfg *Config) (*PortalMetrics, error) {
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge metrics config: %w", err)
	}

	// 创建系统指标收集器
	sysCollector, err := system.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create system collector: %w", err)
	}

	// 创建滑动窗口统计
	slidingWindow, err := sliding.NewWindow(&newCfg.SlidingWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to create sliding window: %w", err)
	}

	m := &PortalMetrics{
		config: newCfg,

		RequestTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "requests_total",
				Help:      "HTTP 请求总数",
			},
			[]string{"endpoint", "result"},
		),

		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: newCfg.Namespace,
				Name:      "request_duration_seconds",
				Help:      "HTTP 请求延迟（秒）",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"endpoint"},
		),

		ActiveRequests: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: newCfg.Namespace,
				Name:      "active_requests",
				Help:      "当前活跃请求数",
			},
		),

		systemCollector: sysCollector,
		slidingWindow:   slidingWindow,
	}

	// 启动系统指标收集
	sysCollector.Start(newCfg.SystemCollectInterval)

	return m, nil
}

// Register 注册指标到 Prometheus Registry
func (m *PortalMetrics) Register(registerer prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.RequestTotal,
		m.RequestDuration,
		m.ActiveRequests,
	}

	for _, c := range collectors {
		if err := registerer.Register(c); err != nil {
			return err
		}
	}

	return nil
}

// RecordRequest 记录请求
func (m *PortalMetrics) RecordRequest(endpoint string, success bool, duration float64) {
	result := "success"
	if !success {
		result = "failed"
	}

	m.RequestTotal.WithLabelValues(endpoint, result).Inc()
	m.RequestDuration.WithLabelValues(endpoint).Observe(duration)
	m.slidingWindow.Record(duration, success)
}

// IncrActiveRequest 增加活跃请求
func (m *PortalMetrics) IncrActiveRequest() {
	m.ActiveRequests.Inc()
}

// DecrActiveRequest 减少活跃请求
func (m *PortalMetrics) DecrActiveRequest() {
	m.ActiveRequests.Dec()
}

// GetStats 获取统计数据
func (m *PortalMetrics) GetStats() Stats {
	windowStats := m.slidingWindow.GetStats()
	sysStats := m.systemCollector.GetStats()

	return Stats{
		QPS:           windowStats.QPS,
		AvgLatency:    windowStats.AvgLatency,
		SuccessRate:   windowStats.SuccessRate,
		CPUPercent:    sysStats.CPUPercent,
		MemoryPercent: sysStats.MemoryPercent,
		MemoryBytes:   sysStats.MemoryBytes,
		Goroutines:    sysStats.Goroutines,
	}
}

// Stats 统计数据
type Stats struct {
	QPS           float64 `json:"qps"`
	AvgLatency    float64 `json:"avg_latency"`
	SuccessRate   float64 `json:"success_rate"`
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryBytes   uint64  `json:"memory_bytes"`
	Goroutines    int     `json:"goroutines"`
}

// Stop 停止指标收集
func (m *PortalMetrics) Stop() {
	m.systemCollector.Stop()
	m.slidingWindow.Stop()
}
