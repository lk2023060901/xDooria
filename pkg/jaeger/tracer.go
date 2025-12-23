package jaeger

import (
	"context"

	"github.com/lk2023060901/xdooria/pkg/otel"
	"go.opentelemetry.io/otel/trace"
)

// Tracer Jaeger 追踪器（基于 pkg/otel 的封装）
type Tracer struct {
	provider *otel.TracerProvider
	config   *Config
}

// New 创建 Jaeger Tracer
func New(cfg *Config) (*Tracer, error) {
	// 转换为 otel.Config
	otelCfg := convertConfig(cfg)

	// 创建 otel TracerProvider
	provider, err := otel.New(otelCfg)
	if err != nil {
		return nil, err
	}

	// 设置为全局 provider
	otel.SetGlobalTracerProvider(provider)

	return &Tracer{
		provider: provider,
		config:   cfg,
	}, nil
}

// convertConfig 将 jaeger.Config 转换为 otel.Config
func convertConfig(cfg *Config) *otel.Config {
	if cfg == nil {
		return nil
	}

	// 确定导出器类型
	exporterType := otel.ExporterTypeOTLPHTTP
	if cfg.EndpointType == EndpointTypeAgent {
		// Agent 模式暂时映射到 OTLP gRPC
		exporterType = otel.ExporterTypeOTLPGRPC
	}

	// 转换采样器类型
	samplerType := otel.SamplerTypeParent
	switch cfg.Sampler.Type {
	case SamplerTypeAlways:
		samplerType = otel.SamplerTypeAlways
	case SamplerTypeNever:
		samplerType = otel.SamplerTypeNever
	case SamplerTypeRatio:
		samplerType = otel.SamplerTypeRatio
	case SamplerTypeParent:
		samplerType = otel.SamplerTypeParent
	}

	return &otel.Config{
		Enabled:      cfg.Enabled,
		ServiceName:  cfg.ServiceName,
		Endpoint:     cfg.Endpoint,
		ExporterType: exporterType,
		Sampler: otel.SamplerConfig{
			Type:  samplerType,
			Ratio: cfg.Sampler.Ratio,
		},
		BatchExport: otel.BatchExportConfig{
			BatchSize:     cfg.BatchExport.BatchSize,
			ExportTimeout: cfg.BatchExport.ExportTimeout,
			MaxQueueSize:  cfg.BatchExport.MaxQueueSize,
			BatchTimeout:  cfg.BatchExport.BatchTimeout,
		},
		Attributes:      cfg.Attributes,
		ShutdownTimeout: cfg.ShutdownTimeout,
		Insecure:        true, // Jaeger 默认使用不安全连接
	}
}

// Tracer 获取指定名称的 Tracer
func (t *Tracer) Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return t.provider.Tracer(name, opts...)
}

// Provider 获取 TracerProvider
func (t *Tracer) Provider() trace.TracerProvider {
	return t.provider.Provider()
}

// Start 开始一个新的 Span
func (t *Tracer) Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	return t.provider.Start(ctx, spanName, opts...)
}

// Close 关闭 Tracer
func (t *Tracer) Close() error {
	return t.provider.Close()
}

// IsClosed 是否已关闭
func (t *Tracer) IsClosed() bool {
	return t.provider.IsClosed()
}

// IsEnabled 是否启用
func (t *Tracer) IsEnabled() bool {
	return t.provider.IsEnabled()
}
