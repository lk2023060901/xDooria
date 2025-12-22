package jaeger

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// SpanFromContext 从 context 中获取当前 Span
func SpanFromContext(ctx context.Context) trace.Span {
	return trace.SpanFromContext(ctx)
}

// ContextWithSpan 将 Span 放入 context
func ContextWithSpan(ctx context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(ctx, span)
}

// TraceIDFromContext 从 context 中获取 TraceID
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}

	spanCtx := span.SpanContext()
	if !spanCtx.HasTraceID() {
		return ""
	}

	return spanCtx.TraceID().String()
}

// SpanIDFromContext 从 context 中获取 SpanID
func SpanIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return ""
	}

	spanCtx := span.SpanContext()
	if !spanCtx.HasSpanID() {
		return ""
	}

	return spanCtx.SpanID().String()
}

// IsSampled 判断当前 Span 是否被采样
func IsSampled(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return false
	}

	return span.SpanContext().IsSampled()
}

// SpanContext 获取 SpanContext
func SpanContext(ctx context.Context) trace.SpanContext {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return trace.SpanContext{}
	}

	return span.SpanContext()
}

// IsRecording 判断当前 Span 是否正在记录
func IsRecording(ctx context.Context) bool {
	span := trace.SpanFromContext(ctx)
	if span == nil {
		return false
	}

	return span.IsRecording()
}
