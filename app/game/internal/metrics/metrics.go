package metrics

import (
	"fmt"
	"sync/atomic"
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
	// Reporter 上报器配置
	Reporter ReporterConfig `mapstructure:"reporter" json:"reporter" yaml:"reporter"`
	// System 系统指标配置
	SystemCollectInterval time.Duration `mapstructure:"system_collect_interval" json:"system_collect_interval" yaml:"system_collect_interval"`
	// SlidingWindow 滑动窗口配置
	SlidingWindow sliding.WindowConfig `mapstructure:"sliding_window" json:"sliding_window" yaml:"sliding_window"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Namespace:             "game",
		Reporter:              *DefaultReporterConfig(),
		SystemCollectInterval: 5 * time.Second,
		SlidingWindow:         *sliding.DefaultWindowConfig(),
	}
}

// GameMetrics Game 服务指标
type GameMetrics struct {
	config *Config

	// 玩家指标
	OnlineRoles   prometheus.Gauge        // 当前在线角色数
	TotalLogins   *prometheus.CounterVec  // 登录总数（按结果）

	// 消息指标
	MessageTotal    *prometheus.CounterVec    // 消息总数（按操作码、结果）
	MessageDuration *prometheus.HistogramVec  // 消息处理延迟

	// 数据库指标
	DBQueryTotal    *prometheus.CounterVec    // 数据库查询总数（按操作、结果）
	DBQueryDuration *prometheus.HistogramVec  // 数据库查询延迟

	// 缓存指标
	CacheHitTotal  *prometheus.CounterVec  // 缓存命中（按缓存类型）
	CacheMissTotal *prometheus.CounterVec  // 缓存未命中（按缓存类型）

	// 内部统计（用于上报 etcd）
	totalMessages   atomic.Int64
	successMessages atomic.Int64
	failedMessages  atomic.Int64
	onlineCount     atomic.Int64

	// 系统指标收集器
	systemCollector *system.Collector
	// 滑动窗口统计（QPS/延迟）
	slidingWindow *sliding.Window
}

// New 创建 Game 指标
func New(cfg *Config) (*GameMetrics, error) {
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

	// 创建滑动窗口统计
	slidingWindow, err := sliding.NewWindow(&newCfg.SlidingWindow)
	if err != nil {
		return nil, fmt.Errorf("failed to create sliding window: %w", err)
	}

	m := &GameMetrics{
		config: newCfg,

		// 玩家指标
		OnlineRoles: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: newCfg.Namespace,
				Name:      "online_roles",
				Help:      "当前在线角色数",
			},
		),
		TotalLogins: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "logins_total",
				Help:      "角色登录总数",
			},
			[]string{"result"}, // result: success/failed
		),

		// 消息指标
		MessageTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "messages_total",
				Help:      "消息处理总数",
			},
			[]string{"op_code", "result"}, // op_code: 操作码, result: success/failed
		),
		MessageDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: newCfg.Namespace,
				Name:      "message_duration_seconds",
				Help:      "消息处理延迟（秒）",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 2.5, 5},
			},
			[]string{"op_code"},
		),

		// 数据库指标
		DBQueryTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "db_queries_total",
				Help:      "数据库查询总数",
			},
			[]string{"operation", "result"}, // operation: select/insert/update/delete
		),
		DBQueryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: newCfg.Namespace,
				Name:      "db_query_duration_seconds",
				Help:      "数据库查询延迟（秒）",
				Buckets:   []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1},
			},
			[]string{"operation"},
		),

		// 缓存指标
		CacheHitTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "cache_hits_total",
				Help:      "缓存命中总数",
			},
			[]string{"cache_type"}, // cache_type: memory/redis
		),
		CacheMissTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: newCfg.Namespace,
				Name:      "cache_misses_total",
				Help:      "缓存未命中总数",
			},
			[]string{"cache_type"},
		),

		systemCollector: sysCollector,
		slidingWindow:   slidingWindow,
	}

	// 启动系统指标收集
	sysCollector.Start(newCfg.SystemCollectInterval)

	return m, nil
}

// Register 注册指标到 Prometheus Registry
func (m *GameMetrics) Register(registerer prometheus.Registerer) error {
	collectors := []prometheus.Collector{
		m.OnlineRoles,
		m.TotalLogins,
		m.MessageTotal,
		m.MessageDuration,
		m.DBQueryTotal,
		m.DBQueryDuration,
		m.CacheHitTotal,
		m.CacheMissTotal,
	}

	for _, c := range collectors {
		if err := registerer.Register(c); err != nil {
			return err
		}
	}

	return nil
}

// RecordRoleOnline 记录角色上线
func (m *GameMetrics) RecordRoleOnline(success bool) {
	result := "success"
	if !success {
		result = "failed"
	} else {
		m.onlineCount.Add(1)
		m.OnlineRoles.Inc()
	}
	m.TotalLogins.WithLabelValues(result).Inc()
}

// RecordRoleOffline 记录角色下线
func (m *GameMetrics) RecordRoleOffline() {
	m.onlineCount.Add(-1)
	m.OnlineRoles.Dec()
}

// RecordMessage 记录消息处理
func (m *GameMetrics) RecordMessage(opCode string, success bool, duration float64) {
	result := "success"
	if !success {
		result = "failed"
		m.failedMessages.Add(1)
	} else {
		m.successMessages.Add(1)
	}

	m.totalMessages.Add(1)
	m.MessageTotal.WithLabelValues(opCode, result).Inc()
	m.MessageDuration.WithLabelValues(opCode).Observe(duration)

	// 记录到滑动窗口
	m.slidingWindow.Record(duration, success)
}

// RecordDBQuery 记录数据库查询
func (m *GameMetrics) RecordDBQuery(operation string, success bool, duration float64) {
	result := "success"
	if !success {
		result = "failed"
	}
	m.DBQueryTotal.WithLabelValues(operation, result).Inc()
	m.DBQueryDuration.WithLabelValues(operation).Observe(duration)
}

// RecordCacheHit 记录缓存命中
func (m *GameMetrics) RecordCacheHit(cacheType string) {
	m.CacheHitTotal.WithLabelValues(cacheType).Inc()
}

// RecordCacheMiss 记录缓存未命中
func (m *GameMetrics) RecordCacheMiss(cacheType string) {
	m.CacheMissTotal.WithLabelValues(cacheType).Inc()
}

// GetStats 获取统计数据（用于 etcd 元数据上报）
func (m *GameMetrics) GetStats() Stats {
	// 获取滑动窗口统计
	windowStats := m.slidingWindow.GetStats()
	// 获取系统统计
	sysStats := m.systemCollector.GetStats()

	return Stats{
		// 基础统计
		TotalMessages:   m.totalMessages.Load(),
		SuccessMessages: m.successMessages.Load(),
		FailedMessages:  m.failedMessages.Load(),
		OnlineRoles:     m.onlineCount.Load(),
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
	}
}

// Stats 统计数据结构（用于负载均衡）
type Stats struct {
	// 基础统计
	TotalMessages   int64 `json:"total_messages"`
	SuccessMessages int64 `json:"success_messages"`
	FailedMessages  int64 `json:"failed_messages"`
	OnlineRoles     int64 `json:"online_roles"`
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
}

// GetConfig 获取配置
func (m *GameMetrics) GetConfig() *Config {
	return m.config
}

// Stop 停止所有后台任务
func (m *GameMetrics) Stop() {
	m.systemCollector.Stop()
	m.slidingWindow.Stop()
}
