package metrics

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// ReporterConfig 上报器配置
type ReporterConfig struct {
	// ReportInterval 上报间隔
	ReportInterval time.Duration `mapstructure:"report_interval" json:"report_interval" yaml:"report_interval"`
	// Enabled 是否启用
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
}

// DefaultReporterConfig 默认配置
func DefaultReporterConfig() *ReporterConfig {
	return &ReporterConfig{
		ReportInterval: 10 * time.Second,
		Enabled:        true,
	}
}

// Reporter 指标上报器（上报到 etcd）
type Reporter struct {
	config    *ReporterConfig
	metrics   *GameMetrics
	registrar registry.Registrar
	logger    logger.Logger
	pool      *conc.Pool[struct{}]
	stopCh    chan struct{}
}

// NewReporter 创建上报器
func NewReporter(
	cfg *ReporterConfig,
	metrics *GameMetrics,
	registrar registry.Registrar,
	l logger.Logger,
) (*Reporter, error) {
	// 使用 MergeConfig 合并默认配置和用户配置
	newCfg, err := config.MergeConfig(DefaultReporterConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge reporter config: %w", err)
	}

	return &Reporter{
		config:    newCfg,
		metrics:   metrics,
		registrar: registrar,
		logger:    l.Named("metrics.reporter"),
		pool:      conc.NewDefaultPool[struct{}](),
		stopCh:    make(chan struct{}),
	}, nil
}

// Start 启动上报器
func (r *Reporter) Start() {
	if !r.config.Enabled {
		r.logger.Info("metrics reporter disabled")
		return
	}

	r.pool.Submit(func() (struct{}, error) {
		r.run()
		return struct{}{}, nil
	})

	r.logger.Info("metrics reporter started",
		"interval", r.config.ReportInterval,
	)
}

// Stop 停止上报器
func (r *Reporter) Stop() {
	close(r.stopCh)
	r.pool.Release()
	r.logger.Info("metrics reporter stopped")
}

// run 运行上报循环
func (r *Reporter) run() {
	ticker := time.NewTicker(r.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.report()
		case <-r.stopCh:
			return
		}
	}
}

// report 执行一次上报
func (r *Reporter) report() {
	stats := r.metrics.GetStats()

	// 构建元数据（用于负载均衡决策的关键指标）
	metadata := map[string]string{
		// 负载指标
		"qps":          fmt.Sprintf("%.2f", stats.QPS),
		"avg_latency":  fmt.Sprintf("%.4f", stats.AvgLatency),
		"success_rate": fmt.Sprintf("%.2f", stats.SuccessRate),
		"online_roles": fmt.Sprintf("%d", stats.OnlineRoles),
		// 系统资源
		"cpu_percent":    fmt.Sprintf("%.2f", stats.CPUPercent),
		"memory_percent": fmt.Sprintf("%.2f", stats.MemoryPercent),
		"memory_bytes":   fmt.Sprintf("%d", stats.MemoryBytes),
		"goroutines":     fmt.Sprintf("%d", stats.Goroutines),
		// 时间戳
		"updated_at": time.Now().Format(time.RFC3339),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := r.registrar.UpdateMetadata(ctx, metadata); err != nil {
		r.logger.Warn("failed to report metrics to etcd",
			"error", err,
		)
		return
	}

	r.logger.Debug("metrics reported to etcd",
		"qps", stats.QPS,
		"avg_latency", stats.AvgLatency,
		"online_roles", stats.OnlineRoles,
		"cpu_percent", stats.CPUPercent,
		"memory_percent", stats.MemoryPercent,
	)
}
