package otel

import "errors"

var (
	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("otel: invalid config")

	// ErrInvalidServiceName 服务名称无效
	ErrInvalidServiceName = errors.New("otel: invalid service name")

	// ErrInvalidSamplerRatio 采样比率无效
	ErrInvalidSamplerRatio = errors.New("otel: sampler ratio must be between 0 and 1")

	// ErrProviderClosed 提供者已关闭
	ErrProviderClosed = errors.New("otel: provider is closed")

	// ErrExporterFailed 导出器创建失败
	ErrExporterFailed = errors.New("otel: failed to create exporter")

	// ErrUnsupportedExporter 不支持的导出器类型
	ErrUnsupportedExporter = errors.New("otel: unsupported exporter type")
)
