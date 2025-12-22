package main

import (
	"context"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 示例 1: 不使用 context 提取器（默认行为）
	log1, _ := logger.New(nil)
	ctx := context.WithValue(context.Background(), "trace_id", "abc123")
	log1.InfoContext(ctx, "默认行为：不提取字段")

	// 示例 2: 通过 Config 设置提取器
	log2, _ := logger.New(&logger.Config{
		Level:       logger.InfoLevel,
		Development: true, // 彩色输出
		ContextExtractor: func(ctx context.Context) []zap.Field {
			fields := make([]zap.Field, 0)
			if traceID, ok := ctx.Value("trace_id").(string); ok {
				fields = append(fields, zap.String("trace_id", traceID))
			}
			if userID, ok := ctx.Value("user_id").(string); ok {
				fields = append(fields, zap.String("user_id", userID))
			}
			return fields
		},
	})

	ctx = context.WithValue(context.Background(), "trace_id", "xyz789")
	ctx = context.WithValue(ctx, "user_id", "user-456")
	log2.InfoContext(ctx, "通过 Config 提取字段")

	// 示例 3: 通过 Option 设置提取器
	log3, _ := logger.New(nil,
		logger.WithDevelopment(true),
		logger.WithContextExtractor(func(ctx context.Context) []zap.Field {
			fields := make([]zap.Field, 0, 3)
			if traceID, ok := ctx.Value("trace_id").(string); ok {
				fields = append(fields, zap.String("trace_id", traceID))
			}
			if requestID, ok := ctx.Value("request_id").(string); ok {
				fields = append(fields, zap.String("request_id", requestID))
			}
			if clientIP, ok := ctx.Value("client_ip").(string); ok {
				fields = append(fields, zap.String("client_ip", clientIP))
			}
			return fields
		}),
	)

	ctx = context.Background()
	ctx = context.WithValue(ctx, "trace_id", "trace-001")
	ctx = context.WithValue(ctx, "request_id", "req-002")
	ctx = context.WithValue(ctx, "client_ip", "192.168.1.100")
	log3.InfoContext(ctx, "通过 Option 提取字段", "action", "login")

	// 示例 4: 全局 logger 使用 Context
	// 先初始化全局 logger 带提取器
	logger.InitDefault(&logger.Config{
		Level:       logger.InfoLevel,
		Development: true,
		ContextExtractor: func(ctx context.Context) []zap.Field {
			fields := make([]zap.Field, 0)
			if traceID, ok := ctx.Value("trace_id").(string); ok {
				fields = append(fields, zap.String("trace_id", traceID))
			}
			return fields
		},
	})

	ctx = context.WithValue(context.Background(), "trace_id", "global-trace")
	logger.Default().InfoContext(ctx, "全局 logger 使用 Context")

	// 示例 5: 混合使用 Context 字段和手动字段
	ctx = context.WithValue(context.Background(), "trace_id", "mixed-trace")
	logger.Default().InfoContext(ctx, "混合字段",
		"method", "POST",
		"status", 200,
	)

	// 示例 6: 测试所有日志级别的 Context 方法
	ctx = context.WithValue(context.Background(), "trace_id", "level-test")
	logger.Default().DebugContext(ctx, "Debug 级别")
	logger.Default().InfoContext(ctx, "Info 级别")
	logger.Default().WarnContext(ctx, "Warn 级别")
	logger.Default().ErrorContext(ctx, "Error 级别")
}
