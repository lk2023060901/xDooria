package jaeger

import "errors"

var (
	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("jaeger: invalid config")

	// ErrTracerClosed Tracer 已关闭
	ErrTracerClosed = errors.New("jaeger: tracer is closed")

	// ErrNoSpanInContext context 中没有 span
	ErrNoSpanInContext = errors.New("jaeger: no span in context")

	// ErrExporterFailed 导出器创建失败
	ErrExporterFailed = errors.New("jaeger: failed to create exporter")
)
