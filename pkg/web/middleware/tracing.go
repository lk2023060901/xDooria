package middleware

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/otel"
	"go.opentelemetry.io/otel/propagation"
)

// Tracing 分布式追踪中间件
func Tracing(serviceName string) gin.HandlerFunc {
	tracer := otel.GetTracerProvider().Tracer("web")
	propagator := otel.GetTextMapPropagator()

	return func(c *gin.Context) {
		// 从 Header 中提取 trace context
		ctx := propagator.Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		// 创建 span
		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		if c.FullPath() == "" {
			spanName = fmt.Sprintf("%s %s", c.Request.Method, c.Request.URL.Path)
		}

		ctx, span := tracer.Start(
			ctx,
			spanName,
			otel.WithSpanKind(otel.SpanKindServer),
			otel.WithAttributes(
				otel.String("http.method", c.Request.Method),
				otel.String("http.path", c.Request.URL.Path),
				otel.String("http.route", c.FullPath()),
				otel.String("service.name", serviceName),
			),
		)
		defer span.End()

		// 更新 context
		c.Request = c.Request.WithContext(ctx)

		c.Next()

		// 记录状态码和错误
		status := c.Writer.Status()
		span.SetAttributes(otel.Int("http.status_code", status))
		if status >= 400 {
			span.SetStatus(otel.CodeError, fmt.Sprintf("HTTP status %d", status))
		} else {
			span.SetStatus(otel.CodeOk, "")
		}

		if len(c.Errors) > 0 {
			span.RecordError(c.Errors.Last().Err)
		}
	}
}
