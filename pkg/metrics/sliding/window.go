package sliding

import (
	"fmt"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// WindowConfig 滑动窗口配置
type WindowConfig struct {
	// 是否启用
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 窗口大小
	WindowSize time.Duration `mapstructure:"window_size" json:"window_size" yaml:"window_size"`
	// 桶数量
	BucketCount int `mapstructure:"bucket_count" json:"bucket_count" yaml:"bucket_count"`
}

// DefaultWindowConfig 默认配置（保障最小可用）
func DefaultWindowConfig() *WindowConfig {
	return &WindowConfig{
		Enabled:     true,
		WindowSize:  60 * time.Second,
		BucketCount: 60,
	}
}

// bucket 时间桶
type bucket struct {
	count       int64   // 请求数量
	totalTime   float64 // 总耗时（秒）
	minLatency  float64 // 最小延迟
	maxLatency  float64 // 最大延迟
	successCnt  int64   // 成功数量
	failureCnt  int64   // 失败数量
	timestamp   time.Time
	initialized bool
}

// Window 滑动窗口统计器
type Window struct {
	config *WindowConfig
	mu     sync.RWMutex

	buckets       []bucket
	currentBucket int

	pool   *conc.Pool[struct{}]
	stopCh chan struct{}
}

// NewWindow 创建滑动窗口统计器
func NewWindow(cfg *WindowConfig) (*Window, error) {
	newCfg, err := config.MergeConfig(DefaultWindowConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge window config: %w", err)
	}

	w := &Window{
		config:  newCfg,
		buckets: make([]bucket, newCfg.BucketCount),
		pool:    conc.NewPool[struct{}](1),
		stopCh:  make(chan struct{}),
	}

	// 初始化桶
	now := time.Now()
	for i := range w.buckets {
		w.buckets[i].timestamp = now
		w.buckets[i].minLatency = -1 // 标记未初始化
	}

	// 启动桶轮转
	interval := newCfg.WindowSize / time.Duration(newCfg.BucketCount)
	w.startRotation(interval)

	return w, nil
}

// startRotation 启动桶轮转
func (w *Window) startRotation(interval time.Duration) {
	w.pool.Submit(func() (struct{}, error) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				w.rotate()
			case <-w.stopCh:
				return struct{}{}, nil
			}
		}
	})
}

// rotate 轮转到下一个桶
func (w *Window) rotate() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.currentBucket = (w.currentBucket + 1) % len(w.buckets)
	w.buckets[w.currentBucket] = bucket{
		timestamp:  time.Now(),
		minLatency: -1,
	}
}

// Record 记录一次请求
func (w *Window) Record(latency float64, success bool) {
	if !w.config.Enabled {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	b := &w.buckets[w.currentBucket]
	b.count++
	b.totalTime += latency
	b.initialized = true

	if success {
		b.successCnt++
	} else {
		b.failureCnt++
	}

	// 更新最小/最大延迟
	if b.minLatency < 0 || latency < b.minLatency {
		b.minLatency = latency
	}
	if latency > b.maxLatency {
		b.maxLatency = latency
	}
}

// Stats 统计结果
type Stats struct {
	// 每秒请求数（QPS）
	QPS float64 `json:"qps"`
	// 平均延迟（秒）
	AvgLatency float64 `json:"avg_latency"`
	// 最小延迟（秒）
	MinLatency float64 `json:"min_latency"`
	// 最大延迟（秒）
	MaxLatency float64 `json:"max_latency"`
	// 成功率 (0-100)
	SuccessRate float64 `json:"success_rate"`
	// 窗口内总请求数
	TotalCount int64 `json:"total_count"`
	// 窗口内成功数
	SuccessCount int64 `json:"success_count"`
	// 窗口内失败数
	FailureCount int64 `json:"failure_count"`
}

// GetStats 获取统计数据
func (w *Window) GetStats() Stats {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var stats Stats
	var totalCount int64
	var totalTime float64
	var successCnt int64
	var failureCnt int64
	minLatency := float64(-1)
	maxLatency := float64(0)

	now := time.Now()
	windowStart := now.Add(-w.config.WindowSize)

	for _, b := range w.buckets {
		if b.timestamp.After(windowStart) && b.initialized {
			totalCount += b.count
			totalTime += b.totalTime
			successCnt += b.successCnt
			failureCnt += b.failureCnt

			if b.minLatency >= 0 && (minLatency < 0 || b.minLatency < minLatency) {
				minLatency = b.minLatency
			}
			if b.maxLatency > maxLatency {
				maxLatency = b.maxLatency
			}
		}
	}

	// 计算 QPS
	seconds := w.config.WindowSize.Seconds()
	stats.QPS = float64(totalCount) / seconds
	stats.TotalCount = totalCount
	stats.SuccessCount = successCnt
	stats.FailureCount = failureCnt

	// 计算平均延迟
	if totalCount > 0 {
		stats.AvgLatency = totalTime / float64(totalCount)
	}

	// 设置最小/最大延迟
	if minLatency >= 0 {
		stats.MinLatency = minLatency
	}
	stats.MaxLatency = maxLatency

	// 计算成功率
	if totalCount > 0 {
		stats.SuccessRate = float64(successCnt) / float64(totalCount) * 100
	}

	return stats
}

// GetQPS 获取 QPS
func (w *Window) GetQPS() float64 {
	return w.GetStats().QPS
}

// GetAvgLatency 获取平均延迟
func (w *Window) GetAvgLatency() float64 {
	return w.GetStats().AvgLatency
}

// Stop 停止统计
func (w *Window) Stop() {
	close(w.stopCh)
	w.pool.Release()
}
