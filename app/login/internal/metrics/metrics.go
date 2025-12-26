package metrics

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/metrics/sliding"
	"github.com/lk2023060901/xdooria/pkg/metrics/system"
	"github.com/prometheus/client_golang/prometheus"
)

// Config 指标配置
type Config struct {
	// Namespace 指标命名空间
	Namespace string `mapstructure:"namespace" json:"namespace" yaml:"namespace"`
	// Reporter 上报器配置
	Reporter ReporterConfig `mapstructure:"reporter" json:"reporter" yaml:"reporter"`
	// System 系统指标配置
	SystemCollectInterval time.Duration `mapstructure:"system_collect_interval" json:"system_collect_interval" yaml:"system_collect_interval"`
	// Bandwidth 带宽统计配置
	Bandwidth interceptor.BandwidthConfig `mapstructure:"bandwidth" json:"bandwidth" yaml:"bandwidth"`
	// SlidingWindow 滑动窗口配置
	SlidingWindow sliding.WindowConfig `mapstructure:"sliding_window" json:"sliding_window" yaml:"sliding_window"`
}

// DefaultConfig 默认配置（保障最小可用）
func DefaultConfig() *Config {
	return &Config{
		Namespace:             "login",
		Reporter:              *DefaultReporterConfig(),
		SystemCollectInterval: 5 * time.Second,
		Bandwidth:             *interceptor.DefaultBandwidthConfig(),
		SlidingWindow:         *sliding.DefaultWindowConfig(),
	}
}

// LoginMetrics Login 服务指标
type LoginMetrics struct {
	config *Config
	// 登录请求总数（按登录类型、结果）
	LoginTotal *prometheus.CounterVec
	// 登录请求延迟
	LoginDuration *prometheus.HistogramVec
	// 认证失败次数（按原因）
	AuthFailures *prometheus.CounterVec
	// 当前活跃连接数
	ActiveConnections prometheus.Gauge

	// 内部统计（用于上报 etcd）
	totalRequests   atomic.Int64
	successRequests atomic.Int64
	failedRequests  atomic.Int64
	activeConns     atomic.Int64

	// 系统指标收集器
	systemCollector *system.Collector
	// 带宽统计
	bandwidthStats *interceptor.BandwidthStats
	// 滑动窗口统计（QPS/延迟）
	slidingWindow *sliding.Window
}

// New 创建 Login 指标
func New(cfg *Config) (*LoginMetrics, error) {
	// 使用 MergeConfig 合并默认配置和用户配置
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge metrics config: %w", err)
	}

	// 创建系统指标收集器
	sysCollector, err := system.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create system collector: %w", err)
	}

	// 创建带宽统计
	bandwidthStats, err := interceptor.NewBandwidthStats(&newCfg.Bandwidth)
	if err != nil {
		return nil, fmt.Errorf("failed to create bandwidth stats: %w", err)
	}

	// 创建滑动窗口统计
	slidingWindow, err := sliding.NewWindow(&newCfg.SlidingWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to create sliding window: %w", err)
	}

	m := &LoginMetrics{
		config: newCfg,
		// 登录请求总数
		LoginTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "requests_total",
				Help:      "登录请求总数",
			},
			[]string{"type", "result"}, // type: local/wechat/apple, result: success/failed
		),

		// 登录延迟分布
		LoginDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: newCfg.Namespace,
				Name:      "request_duration_seconds",
				Help:      "登录请求延迟（秒）",
				Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"type"},
		),

		// 认证失败原因
		AuthFailures: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "auth_failures_total",
				Help:      "认证失败次数（按原因）",
			},
			[]string{"type", "reason"}, // reason: invalid_credentials/user_not_found/account_locked/token_expired
		),

		// 当前活跃连接数
		ActiveConnections: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: newCfg.Namespace,
				Name:      "active_connections",
				Help:      "当前活跃连接数",
			},
		),

		systemCollector: sysCollector,
		bandwidthStats:  bandwidthStats,
		slidingWindow:   slidingWindow,
	}

	// 启动系统指标收集
	sysCollector.Start(newCfg.SystemCollectInterval)

	return m, nil
}

// Register 注册指标到 Prometheus Registry
func (m *LoginMetrics) Register(registerer prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.LoginTotal,
		m.LoginDuration,
		m.AuthFailures,
		m.ActiveConnections,
	}

	for _, c := range collectors {
		if err := registerer.Register(c); err != nil {
			return err
		}
	}

	return nil
}

// RecordLogin 记录登录请求
func (m *LoginMetrics) RecordLogin(loginType string, success bool, duration float64) {
	result := "success"
	if !success {
		result = "failed"
		m.failedRequests.Add(1)
	} else {
		m.successRequests.Add(1)
	}

	m.totalRequests.Add(1)
	m.LoginTotal.WithLabelValues(loginType, result).Inc()
	m.LoginDuration.WithLabelValues(loginType).Observe(duration)

	// 记录到滑动窗口
	m.slidingWindow.Record(duration, success)
}

// RecordAuthFailure 记录认证失败
func (m *LoginMetrics) RecordAuthFailure(loginType, reason string) {
	m.AuthFailures.WithLabelValues(loginType, reason).Inc()
}

// IncrConnection 增加连接数
func (m *LoginMetrics) IncrConnection() {
	m.activeConns.Add(1)
	m.ActiveConnections.Inc()
}

// DecrConnection 减少连接数
func (m *LoginMetrics) DecrConnection() {
	m.activeConns.Add(-1)
	m.ActiveConnections.Dec()
}

// GetStats 获取统计数据（用于 etcd 元数据上报）
func (m *LoginMetrics) GetStats() Stats {
	// 获取滑动窗口统计
	windowStats := m.slidingWindow.GetStats()
	// 获取系统统计
	sysStats := m.systemCollector.GetStats()
	// 获取带宽统计
	bandwidthSummary := m.bandwidthStats.GetSummary()

	return Stats{
		// 基础统计
		TotalRequests:     m.totalRequests.Load(),
		SuccessRequests:   m.successRequests.Load(),
		FailedRequests:    m.failedRequests.Load(),
		ActiveConnections: m.activeConns.Load(),
		// QPS 和延迟（滑动窗口）
		QPS:         windowStats.QPS,
		AvgLatency:  windowStats.AvgLatency,
		MinLatency:  windowStats.MinLatency,
		MaxLatency:  windowStats.MaxLatency,
		SuccessRate: windowStats.SuccessRate,
		// 系统指标
		CPUPercent:    sysStats.CPUPercent,
		MemoryPercent: sysStats.MemoryPercent,
		MemoryBytes:   sysStats.MemoryBytes,
		Goroutines:    sysStats.Goroutines,
		// 带宽统计
		BytesInPerSec:  bandwidthSummary.BytesInPerSec,
		BytesOutPerSec: bandwidthSummary.BytesOutPerSec,
		TotalBytesIn:   bandwidthSummary.TotalBytesIn,
		TotalBytesOut:  bandwidthSummary.TotalBytesOut,
	}
}

// Stats 统计数据结构（用于负载均衡）
type Stats struct {
	// 基础统计
	TotalRequests     int64 `json:"total_requests"`
	SuccessRequests   int64 `json:"success_requests"`
	FailedRequests    int64 `json:"failed_requests"`
	ActiveConnections int64 `json:"active_connections"`
	// QPS 和延迟（滑动窗口）
	QPS         float64 `json:"qps"`
	AvgLatency  float64 `json:"avg_latency"`
	MinLatency  float64 `json:"min_latency"`
	MaxLatency  float64 `json:"max_latency"`
	SuccessRate float64 `json:"success_rate"`
	// 系统指标
	CPUPercent    float64 `json:"cpu_percent"`
	MemoryPercent float64 `json:"memory_percent"`
	MemoryBytes   uint64  `json:"memory_bytes"`
	Goroutines    int     `json:"goroutines"`
	// 带宽统计
	BytesInPerSec  float64 `json:"bytes_in_per_sec"`
	BytesOutPerSec float64 `json:"bytes_out_per_sec"`
	TotalBytesIn   int64   `json:"total_bytes_in"`
	TotalBytesOut  int64   `json:"total_bytes_out"`
}

// GetConfig 获取配置
func (m *LoginMetrics) GetConfig() *Config {
	return m.config
}

// GetBandwidthStats 获取带宽统计（用于 gRPC 拦截器）
func (m *LoginMetrics) GetBandwidthStats() *interceptor.BandwidthStats {
	return m.bandwidthStats
}

// Stop 停止所有后台任务
func (m *LoginMetrics) Stop() {
	m.systemCollector.Stop()
	m.bandwidthStats.Stop()
	m.slidingWindow.Stop()
}
