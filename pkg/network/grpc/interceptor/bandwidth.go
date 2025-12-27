package interceptor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

// BandwidthStats 带宽统计数据
type BandwidthStats struct {
	config *BandwidthConfig
	mu     sync.RWMutex

	// 滑动窗口统计
	buckets       []bandwidthBucket
	currentBucket int

	// 累计统计
	totalBytesIn  atomic.Int64
	totalBytesOut atomic.Int64

	pool   *conc.Pool[struct{}]
	stopCh chan struct{}
}

type bandwidthBucket struct {
	bytesIn   int64
	bytesOut  int64
	timestamp time.Time
}

// BandwidthConfig 带宽统计配置
type BandwidthConfig struct {
	// 是否启用
	Enabled bool `mapstructure:"enabled" json:"enabled" yaml:"enabled"`
	// 滑动窗口大小
	WindowSize time.Duration `mapstructure:"window_size" json:"window_size" yaml:"window_size"`
	// 窗口桶数量
	WindowBuckets int `mapstructure:"window_buckets" json:"window_buckets" yaml:"window_buckets"`
}

// DefaultBandwidthConfig 默认配置（保障最小可用）
func DefaultBandwidthConfig() *BandwidthConfig {
	return &BandwidthConfig{
		Enabled:       true,
		WindowSize:    60 * time.Second,
		WindowBuckets: 60,
	}
}

// NewBandwidthStats 创建带宽统计
func NewBandwidthStats(cfg *BandwidthConfig) (*BandwidthStats, error) {
	// 使用 MergeConfig 合并默认配置和用户配置
	newCfg, err := config.MergeConfig(DefaultBandwidthConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge bandwidth config: %w", err)
	}

	bs := &BandwidthStats{
		config:  newCfg,
		buckets: make([]bandwidthBucket, newCfg.WindowBuckets),
		pool:    conc.NewPool[struct{}](1),
		stopCh:  make(chan struct{}),
	}

	// 初始化桶
	now := time.Now()
	for i := range bs.buckets {
		bs.buckets[i].timestamp = now
	}

	// 启动桶轮转
	bs.startRotation(newCfg.WindowSize / time.Duration(newCfg.WindowBuckets))

	return bs, nil
}

// startRotation 启动桶轮转
func (bs *BandwidthStats) startRotation(interval time.Duration) {
	bs.pool.Submit(func() (struct{}, error) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				bs.rotate()
			case <-bs.stopCh:
				return struct{}{}, nil
			}
		}
	})
}

// rotate 轮转到下一个桶
func (bs *BandwidthStats) rotate() {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.currentBucket = (bs.currentBucket + 1) % len(bs.buckets)
	bs.buckets[bs.currentBucket] = bandwidthBucket{
		timestamp: time.Now(),
	}
}

// RecordIn 记录入站字节
func (bs *BandwidthStats) RecordIn(bytes int64) {
	bs.totalBytesIn.Add(bytes)

	bs.mu.Lock()
	bs.buckets[bs.currentBucket].bytesIn += bytes
	bs.mu.Unlock()
}

// RecordOut 记录出站字节
func (bs *BandwidthStats) RecordOut(bytes int64) {
	bs.totalBytesOut.Add(bytes)

	bs.mu.Lock()
	bs.buckets[bs.currentBucket].bytesOut += bytes
	bs.mu.Unlock()
}

// GetBytesPerSecond 获取每秒字节数（入站、出站）
func (bs *BandwidthStats) GetBytesPerSecond() (in, out float64) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	var totalIn, totalOut int64
	now := time.Now()
	windowStart := now.Add(-bs.config.WindowSize)

	for _, bucket := range bs.buckets {
		if bucket.timestamp.After(windowStart) {
			totalIn += bucket.bytesIn
			totalOut += bucket.bytesOut
		}
	}

	seconds := bs.config.WindowSize.Seconds()
	return float64(totalIn) / seconds, float64(totalOut) / seconds
}

// GetTotalBytes 获取累计字节数
func (bs *BandwidthStats) GetTotalBytes() (in, out int64) {
	return bs.totalBytesIn.Load(), bs.totalBytesOut.Load()
}

// Stop 停止统计
func (bs *BandwidthStats) Stop() {
	close(bs.stopCh)
	bs.pool.Release()
}

// BandwidthSummary 统计摘要
type BandwidthSummary struct {
	BytesInPerSec  float64 `json:"bytes_in_per_sec"`
	BytesOutPerSec float64 `json:"bytes_out_per_sec"`
	TotalBytesIn   int64   `json:"total_bytes_in"`
	TotalBytesOut  int64   `json:"total_bytes_out"`
}

// GetSummary 获取统计摘要
func (bs *BandwidthStats) GetSummary() BandwidthSummary {
	inPerSec, outPerSec := bs.GetBytesPerSecond()
	totalIn, totalOut := bs.GetTotalBytes()

	return BandwidthSummary{
		BytesInPerSec:  inPerSec,
		BytesOutPerSec: outPerSec,
		TotalBytesIn:   totalIn,
		TotalBytesOut:  totalOut,
	}
}

// ServerBandwidthInterceptor Server 端带宽统计拦截器（Unary）
func ServerBandwidthInterceptor(stats *BandwidthStats) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if stats == nil || !stats.config.Enabled {
			return handler(ctx, req)
		}

		// 统计入站字节
		if msg, ok := req.(proto.Message); ok {
			stats.RecordIn(int64(proto.Size(msg)))
		}

		// 执行处理
		resp, err := handler(ctx, req)

		// 统计出站字节
		if resp != nil {
			if msg, ok := resp.(proto.Message); ok {
				stats.RecordOut(int64(proto.Size(msg)))
			}
		}

		return resp, err
	}
}

// StreamServerBandwidthInterceptor Server 端带宽统计拦截器（Stream）
func StreamServerBandwidthInterceptor(stats *BandwidthStats) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if stats == nil || !stats.config.Enabled {
			return handler(srv, ss)
		}

		// 包装 Stream 以统计带宽
		wrapped := &bandwidthServerStream{
			ServerStream: ss,
			stats:        stats,
		}

		return handler(srv, wrapped)
	}
}

// bandwidthServerStream 带宽统计 ServerStream 包装器
type bandwidthServerStream struct {
	grpc.ServerStream
	stats *BandwidthStats
}

func (s *bandwidthServerStream) RecvMsg(m interface{}) error {
	err := s.ServerStream.RecvMsg(m)
	if err == nil {
		if msg, ok := m.(proto.Message); ok {
			s.stats.RecordIn(int64(proto.Size(msg)))
		}
	}
	return err
}

func (s *bandwidthServerStream) SendMsg(m interface{}) error {
	if msg, ok := m.(proto.Message); ok {
		s.stats.RecordOut(int64(proto.Size(msg)))
	}
	return s.ServerStream.SendMsg(m)
}

// ClientBandwidthInterceptor Client 端带宽统计拦截器（Unary）
func ClientBandwidthInterceptor(stats *BandwidthStats) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if stats == nil || !stats.config.Enabled {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// 统计出站字节（客户端发送）
		if msg, ok := req.(proto.Message); ok {
			stats.RecordOut(int64(proto.Size(msg)))
		}

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 统计入站字节（客户端接收）
		if reply != nil {
			if msg, ok := reply.(proto.Message); ok {
				stats.RecordIn(int64(proto.Size(msg)))
			}
		}

		return err
	}
}
