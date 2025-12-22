package jaeger

import (
	"context"
	"sync/atomic"

	"github.com/lk2023060901/xdooria/pkg/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

// Tracer Jaeger 追踪器
type Tracer struct {
	config   *Config
	provider *sdktrace.TracerProvider
	exporter *otlptrace.Exporter
	closed   atomic.Bool
}

// New 创建 Jaeger Tracer
func New(cfg *Config) (*Tracer, error) {
	// 合并配置
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, err
	}

	// 处理 Enabled 字段：如果用户显式传入 cfg 且 Enabled 为 false，则保留用户设置
	// （因为 MergeConfig 不会用零值覆盖）
	if cfg != nil && !cfg.Enabled {
		newCfg.Enabled = false
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	if !newCfg.Enabled {
		return &Tracer{
			config: newCfg,
		}, nil
	}

	// 创建导出器
	exporter, err := createExporter(context.Background(), newCfg)
	if err != nil {
		return nil, err
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

	return &Tracer{
		config:   newCfg,
		provider: provider,
		exporter: exporter,
	}, nil
}

// createExporter 创建导出器
func createExporter(ctx context.Context, cfg *Config) (*otlptrace.Exporter, error) {
	var opts []otlptracehttp.Option

	opts = append(opts, otlptracehttp.WithEndpoint(cfg.Endpoint))

	// 如果不是 HTTPS，允许不安全连接
	opts = append(opts, otlptracehttp.WithInsecure())

	client := otlptracehttp.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, ErrExporterFailed
	}

	return exporter, nil
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
func (t *Tracer) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	if t.provider == nil {
		return otel.GetTracerProvider().Tracer(name, opts...)
	}
	return t.provider.Tracer(name, opts...)
}

// Provider 获取 TracerProvider
func (t *Tracer) Provider() trace.TracerProvider {
	if t.provider == nil {
		return otel.GetTracerProvider()
	}
	return t.provider
}

// Start 开始一个新的 Span
func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.Tracer(t.config.ServiceName).Start(ctx, spanName, opts...)
}

// Close 关闭 Tracer
func (t *Tracer) Close() error {
	if t.closed.Swap(true) {
		return ErrTracerClosed
	}

	if t.provider == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), t.config.ShutdownTimeout)
	defer cancel()

	if err := t.provider.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

// IsClosed 是否已关闭
func (t *Tracer) IsClosed() bool {
	return t.closed.Load()
}

// IsEnabled 是否启用
func (t *Tracer) IsEnabled() bool {
	return t.config.Enabled
}
