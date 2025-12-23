package otel

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// createExporter 根据配置创建导出器
func createExporter(ctx context.Context, cfg *Config) (sdktrace.SpanExporter, error) {
	switch cfg.ExporterType {
	case ExporterTypeOTLPHTTP:
		return createOTLPHTTPExporter(ctx, cfg)
	case ExporterTypeOTLPGRPC:
		return createOTLPGRPCExporter(ctx, cfg)
	case ExporterTypeStdout:
		return createStdoutExporter()
	case ExporterTypeNoop:
		return nil, nil
	default:
		// 默认使用 OTLP HTTP
		return createOTLPHTTPExporter(ctx, cfg)
	}
}

// createOTLPHTTPExporter 创建 OTLP HTTP 导出器
func createOTLPHTTPExporter(ctx context.Context, cfg *Config) (*otlptrace.Exporter, error) {
	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	client := otlptracehttp.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, ErrExporterFailed
	}

	return exporter, nil
}

// createOTLPGRPCExporter 创建 OTLP gRPC 导出器
func createOTLPGRPCExporter(ctx context.Context, cfg *Config) (*otlptrace.Exporter, error) {
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(cfg.Endpoint),
	}

	if cfg.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	client := otlptracegrpc.NewClient(opts...)
	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, ErrExporterFailed
	}

	return exporter, nil
}

// createStdoutExporter 创建标准输出导出器（用于调试）
func createStdoutExporter() (*stdouttrace.Exporter, error) {
	return stdouttrace.New(
		stdouttrace.WithWriter(os.Stdout),
		stdouttrace.WithPrettyPrint(),
	)
}
