package interceptor

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
)

func TestExtractTraceID_FromOTelSpan(t *testing.T) {
	// 初始化 OpenTelemetry tracer
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	// 创建带有 span 的 context
	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	// 提取 trace ID
	traceID := extractTraceID(ctx)

	if traceID == "" {
		t.Error("Expected non-empty trace ID from OpenTelemetry span")
	}

	// 验证格式
	if len(traceID) != 32 { // trace ID should be 32 hex characters
		t.Errorf("Expected trace ID length 32, got %d: %s", len(traceID), traceID)
	}

	// 验证与 span context 匹配
	expectedTraceID := span.SpanContext().TraceID().String()
	if traceID != expectedTraceID {
		t.Errorf("Expected trace ID %s, got %s", expectedTraceID, traceID)
	}
}

func TestExtractTraceID_FromMetadata(t *testing.T) {
	// 创建带有 x-trace-id 的 metadata context
	md := metadata.New(map[string]string{
		"x-trace-id": "custom-trace-id-123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// 提取 trace ID
	traceID := extractTraceID(ctx)

	if traceID != "custom-trace-id-123" {
		t.Errorf("Expected trace ID 'custom-trace-id-123', got '%s'", traceID)
	}
}

func TestExtractTraceID_OTelPriority(t *testing.T) {
	// 创建带有 OpenTelemetry span 的 context
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	otelTraceID := span.SpanContext().TraceID().String()

	// 同时添加 metadata
	md := metadata.New(map[string]string{
		"x-trace-id": "metadata-trace-id",
	})
	ctx = metadata.NewIncomingContext(ctx, md)

	// 提取 trace ID - 应该优先使用 OpenTelemetry
	traceID := extractTraceID(ctx)

	if traceID != otelTraceID {
		t.Errorf("Expected OpenTelemetry trace ID %s (priority), got %s", otelTraceID, traceID)
	}
}

func TestExtractTraceID_NoTraceID(t *testing.T) {
	// 空 context
	ctx := context.Background()

	traceID := extractTraceID(ctx)

	if traceID != "" {
		t.Errorf("Expected empty trace ID, got '%s'", traceID)
	}
}

func TestExtractTraceID_InvalidSpan(t *testing.T) {
	// 创建无效的 span context
	ctx := context.Background()
	ctx = oteltrace.ContextWithSpan(ctx, oteltrace.SpanFromContext(ctx))

	// 不应该从无效 span 提取 trace ID
	traceID := extractTraceID(ctx)

	if traceID != "" {
		t.Errorf("Expected empty trace ID from invalid span, got '%s'", traceID)
	}
}

func TestExtractTraceID_MetadataWithoutTraceID(t *testing.T) {
	// 创建不包含 x-trace-id 的 metadata
	md := metadata.New(map[string]string{
		"other-header": "value",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	traceID := extractTraceID(ctx)

	if traceID != "" {
		t.Errorf("Expected empty trace ID, got '%s'", traceID)
	}
}

func TestExtractTraceID_MetadataMultipleValues(t *testing.T) {
	// 创建包含多个 x-trace-id 值的 metadata
	md := metadata.MD{
		"x-trace-id": []string{"first-trace-id", "second-trace-id"},
	}
	ctx := metadata.NewIncomingContext(context.Background(), md)

	// 应该返回第一个值
	traceID := extractTraceID(ctx)

	if traceID != "first-trace-id" {
		t.Errorf("Expected 'first-trace-id', got '%s'", traceID)
	}
}

func BenchmarkExtractTraceID_WithOTelSpan(b *testing.B) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)
	tracer := tp.Tracer("test")

	ctx, span := tracer.Start(context.Background(), "test-operation")
	defer span.End()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractTraceID(ctx)
	}
}

func BenchmarkExtractTraceID_WithMetadata(b *testing.B) {
	md := metadata.New(map[string]string{
		"x-trace-id": "trace-123",
	})
	ctx := metadata.NewIncomingContext(context.Background(), md)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractTraceID(ctx)
	}
}

func BenchmarkExtractTraceID_Empty(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractTraceID(ctx)
	}
}
