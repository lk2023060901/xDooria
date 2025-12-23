package otel

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/lk2023060901/xdooria/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

var (
	globalProvider   *TracerProvider
	globalProviderMu sync.RWMutex
)

// TracerProvider 追踪提供者
type TracerProvider struct {
	config   *Config
	provider *sdktrace.TracerProvider
	closed   atomic.Bool
}

// New 创建追踪提供者
func New(cfg *Config) (*TracerProvider, error) {
	// 合并配置
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, err
	}

	// 处理 Enabled 字段
	if cfg != nil && !cfg.Enabled {
		newCfg.Enabled = false
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	// 如果未启用，返回 noop provider
	if !newCfg.Enabled {
		return &TracerProvider{
			config: newCfg,
		}, nil
	}

	// 创建导出器
	exporter, err := createExporter(context.Background(), newCfg)
	if err != nil {
		return nil, err
	}

	// 如果是 noop 导出器，返回 noop provider
	if exporter == nil {
		return &TracerProvider{
			config: newCfg,
		}, nil
	}

	// 创建资源
	res, err := createResource(newCfg)
	if err != nil {
		return nil, err
	}

	// 创建采样器
	sampler := createSampler(newCfg.Sampler)

	// 创建 TracerProvider
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter,
			sdktrace.WithBatchTimeout(newCfg.BatchExport.BatchTimeout),
			sdktrace.WithExportTimeout(newCfg.BatchExport.ExportTimeout),
			sdktrace.WithMaxExportBatchSize(newCfg.BatchExport.BatchSize),
			sdktrace.WithMaxQueueSize(newCfg.BatchExport.MaxQueueSize),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sampler),
	)

	// 设置全局 TracerProvider
	otel.SetTracerProvider(provider)

	// 设置全局传播器
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracerProvider{
		config:   newCfg,
		provider: provider,
	}, nil
}

// createResource 创建资源
func createResource(cfg *Config) (*resource.Resource, error) {
	attrs := []attribute.KeyValue{
		semconv.ServiceName(cfg.ServiceName),
	}

	// 添加自定义属性
	for k, v := range cfg.Attributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	return resource.NewWithAttributes(
		semconv.SchemaURL,
		attrs...,
	), nil
}

// createSampler 创建采样器
func createSampler(cfg SamplerConfig) sdktrace.Sampler {
	switch cfg.Type {
	case SamplerTypeAlways:
		return sdktrace.AlwaysSample()
	case SamplerTypeNever:
		return sdktrace.NeverSample()
	case SamplerTypeRatio:
		return sdktrace.TraceIDRatioBased(cfg.Ratio)
	case SamplerTypeParent:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	default:
		return sdktrace.ParentBased(sdktrace.AlwaysSample())
	}
}

// Tracer 获取指定名称的 Tracer
func (p *TracerProvider) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if p.provider == nil {
		return otel.GetTracerProvider().Tracer(name, opts...)
	}
	return p.provider.Tracer(name, opts...)
}

// Provider 获取底层 TracerProvider
func (p *TracerProvider) Provider() trace.TracerProvider {
	if p.provider == nil {
		return otel.GetTracerProvider()
	}
	return p.provider
}

// Start 开始一个新的 Span
func (p *TracerProvider) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return p.Tracer(p.config.ServiceName).Start(ctx, spanName, opts...)
}

// Shutdown 关闭提供者
func (p *TracerProvider) Shutdown(ctx context.Context) error {
	if p.closed.Swap(true) {
		return ErrProviderClosed
	}

	if p.provider == nil {
		return nil
	}

	return p.provider.Shutdown(ctx)
}

// Close 关闭提供者（使用默认超时）
func (p *TracerProvider) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ShutdownTimeout)
	defer cancel()

	return p.Shutdown(ctx)
}

// ForceFlush 强制刷新
func (p *TracerProvider) ForceFlush(ctx context.Context) error {
	if p.provider == nil {
		return nil
	}
	return p.provider.ForceFlush(ctx)
}

// IsClosed 是否已关闭
func (p *TracerProvider) IsClosed() bool {
	return p.closed.Load()
}

// IsEnabled 是否启用
func (p *TracerProvider) IsEnabled() bool {
	return p.config.Enabled && p.provider != nil
}

// Config 获取配置
func (p *TracerProvider) Config() *Config {
	return p.config
}

// SetGlobalTracerProvider 设置全局追踪提供者
func SetGlobalTracerProvider(p *TracerProvider) {
	globalProviderMu.Lock()
	defer globalProviderMu.Unlock()

	globalProvider = p

	if p != nil && p.provider != nil {
		otel.SetTracerProvider(p.provider)
	}
}

// GetGlobalTracerProvider 获取全局追踪提供者
func GetGlobalTracerProvider() *TracerProvider {
	globalProviderMu.RLock()
	defer globalProviderMu.RUnlock()

	return globalProvider
}
