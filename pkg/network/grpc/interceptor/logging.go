package interceptor

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// LoggingConfig 日志拦截器配置
type LoggingConfig struct {
	// 是否启用（默认 true）
	Enabled bool

	// 是否记录请求（默认 true）
	LogRequest bool

	// 是否记录响应（默认 true）
	LogResponse bool

	// 是否记录耗时（默认 true）
	LogDuration bool

	// 敏感字段列表（需要脱敏）
	SensitiveFields []string

	// 跳过的方法列表
	SkipMethods []string

	// 最大参数大小（0 表示不限制，默认 0）
	MaxPayloadSize int

	// 是否记录 peer 信息（默认 true）
	LogPeer bool
}

// DefaultLoggingConfig 默认配置
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Enabled:         true,
		LogRequest:      true,
		LogResponse:     true,
		LogDuration:     true,
		SensitiveFields: []string{"password", "token", "secret", "credential", "authorization", "api_key"},
		SkipMethods:     []string{"/grpc.health.v1.Health/Check", "/grpc.health.v1.Health/Watch"},
		MaxPayloadSize:  0,
		LogPeer:         true,
	}
}

// ServerLoggingInterceptor Server 端日志拦截器（Unary）
func ServerLoggingInterceptor(log logger.Logger, cfg *LoggingConfig) grpc.UnaryServerInterceptor {
	if cfg == nil {
		cfg = DefaultLoggingConfig()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !cfg.Enabled || shouldSkipMethod(info.FullMethod, cfg.SkipMethods) {
			return handler(ctx, req)
		}

		start := time.Now()

		// 构建请求日志字段
		fields := []interface{}{
			"grpc.method", info.FullMethod,
			"grpc.type", "unary",
		}

		// 提取并记录 trace ID
		if traceID := extractTraceID(ctx); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		// 记录 peer 信息
		if cfg.LogPeer {
			if p, ok := peer.FromContext(ctx); ok {
				fields = append(fields, "grpc.peer", p.Addr.String())
			}
		}

		// 记录请求参数
		if cfg.LogRequest {
			if reqJSON := marshalPayload(req, cfg); reqJSON != "" {
				fields = append(fields, "grpc.request", reqJSON)
			}
		}

		log.Info("gRPC request started", fields...)

		// 执行处理
		resp, err := handler(ctx, req)

		// 构建响应日志字段
		duration := time.Since(start)
		code := status.Code(err)

		fields = []interface{}{
			"grpc.method", info.FullMethod,
			"grpc.code", code.String(),
		}

		// 提取并记录 trace ID
		if traceID := extractTraceID(ctx); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		if cfg.LogDuration {
			fields = append(fields, "grpc.duration", duration)
		}

		// 记录响应参数
		if cfg.LogResponse && resp != nil {
			if respJSON := marshalPayload(resp, cfg); respJSON != "" {
				fields = append(fields, "grpc.response", respJSON)
			}
		}

		// 根据结果选择日志级别
		if err != nil {
			fields = append(fields, "error", err)
			if code == codes.Internal || code == codes.Unknown {
				log.Error("gRPC request failed", fields...)
			} else {
				log.Warn("gRPC request failed", fields...)
			}
		} else {
			log.Info("gRPC request completed", fields...)
		}

		return resp, err
	}
}

// StreamServerLoggingInterceptor Server 端日志拦截器（Stream）
func StreamServerLoggingInterceptor(log logger.Logger, cfg *LoggingConfig) grpc.StreamServerInterceptor {
	if cfg == nil {
		cfg = DefaultLoggingConfig()
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !cfg.Enabled || shouldSkipMethod(info.FullMethod, cfg.SkipMethods) {
			return handler(srv, ss)
		}

		start := time.Now()
		ctx := ss.Context()

		// 记录流开始
		fields := []interface{}{
			"grpc.method", info.FullMethod,
			"grpc.type", "stream",
			"grpc.is_client_stream", info.IsClientStream,
			"grpc.is_server_stream", info.IsServerStream,
		}

		// 提取并记录 trace ID
		if traceID := extractTraceID(ctx); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		if cfg.LogPeer {
			if p, ok := peer.FromContext(ctx); ok {
				fields = append(fields, "grpc.peer", p.Addr.String())
			}
		}

		log.Info("gRPC stream started", fields...)

		// 执行处理
		err := handler(srv, ss)

		// 记录流结束
		duration := time.Since(start)
		code := status.Code(err)

		fields = []interface{}{
			"grpc.method", info.FullMethod,
			"grpc.code", code.String(),
		}

		// 提取并记录 trace ID
		if traceID := extractTraceID(ctx); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		if cfg.LogDuration {
			fields = append(fields, "grpc.duration", duration)
		}

		if err != nil {
			fields = append(fields, "error", err)
			log.Error("gRPC stream failed", fields...)
		} else {
			log.Info("gRPC stream completed", fields...)
		}

		return err
	}
}

// ClientLoggingInterceptor Client 端日志拦截器（Unary）
func ClientLoggingInterceptor(log logger.Logger, cfg *LoggingConfig) grpc.UnaryClientInterceptor {
	if cfg == nil {
		cfg = DefaultLoggingConfig()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !cfg.Enabled || shouldSkipMethod(method, cfg.SkipMethods) {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		start := time.Now()

		// 记录请求
		fields := []interface{}{
			"grpc.method", method,
			"grpc.target", cc.Target(),
		}

		// 提取并记录 trace ID
		if traceID := extractTraceID(ctx); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		if cfg.LogRequest {
			if reqJSON := marshalPayload(req, cfg); reqJSON != "" {
				fields = append(fields, "grpc.request", reqJSON)
			}
		}

		log.Info("gRPC client request started", fields...)

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 记录响应
		duration := time.Since(start)
		code := status.Code(err)

		fields = []interface{}{
			"grpc.method", method,
			"grpc.code", code.String(),
		}

		// 提取并记录 trace ID
		if traceID := extractTraceID(ctx); traceID != "" {
			fields = append(fields, "trace_id", traceID)
		}

		if cfg.LogDuration {
			fields = append(fields, "grpc.duration", duration)
		}

		if cfg.LogResponse && reply != nil {
			if respJSON := marshalPayload(reply, cfg); respJSON != "" {
				fields = append(fields, "grpc.response", respJSON)
			}
		}

		if err != nil {
			fields = append(fields, "error", err)
			log.Error("gRPC client request failed", fields...)
		} else {
			log.Info("gRPC client request completed", fields...)
		}

		return err
	}
}

// marshalPayload 序列化 payload 为 JSON 字符串
func marshalPayload(payload interface{}, cfg *LoggingConfig) string {
	if payload == nil {
		return ""
	}

	msg, ok := payload.(proto.Message)
	if !ok {
		return ""
	}

	// 使用 protojson 序列化
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}

	data, err := marshaler.Marshal(msg)
	if err != nil {
		return ""
	}

	jsonStr := string(data)

	// 脱敏处理
	jsonStr = sanitizeJSON(jsonStr, cfg.SensitiveFields)

	// 检查大小限制
	if cfg.MaxPayloadSize > 0 && len(jsonStr) > cfg.MaxPayloadSize {
		return jsonStr[:cfg.MaxPayloadSize] + "...[truncated]"
	}

	return jsonStr
}

// sanitizeJSON 脱敏处理 JSON 字符串
func sanitizeJSON(jsonStr string, sensitiveFields []string) string {
	if len(sensitiveFields) == 0 {
		return jsonStr
	}

	// 解析 JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}

	// 递归脱敏
	sanitizeMap(data, sensitiveFields)

	// 重新序列化
	sanitized, err := json.Marshal(data)
	if err != nil {
		return jsonStr
	}

	return string(sanitized)
}

// sanitizeMap 递归脱敏 map
func sanitizeMap(data map[string]interface{}, sensitiveFields []string) {
	for key, value := range data {
		// 检查是否是敏感字段
		if isSensitiveField(key, sensitiveFields) {
			data[key] = "***"
			continue
		}

		// 递归处理嵌套对象
		switch v := value.(type) {
		case map[string]interface{}:
			sanitizeMap(v, sensitiveFields)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					sanitizeMap(m, sensitiveFields)
				}
			}
		}
	}
}

// isSensitiveField 检查是否是敏感字段
func isSensitiveField(field string, sensitiveFields []string) bool {
	lowerField := strings.ToLower(field)
	for _, sensitive := range sensitiveFields {
		if strings.Contains(lowerField, strings.ToLower(sensitive)) {
			return true
		}
	}
	return false
}

// shouldSkipMethod 检查是否跳过方法
func shouldSkipMethod(method string, skipMethods []string) bool {
	for _, skip := range skipMethods {
		if method == skip {
			return true
		}
	}
	return false
}

// extractTraceID 从 context 中提取 trace ID
// 优先从 OpenTelemetry span 提取，其次从 metadata 提取
func extractTraceID(ctx context.Context) string {
	// 1. 尝试从 OpenTelemetry span 提取
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}

	// 2. 尝试从 metadata 提取 x-trace-id
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if traceIDs := md.Get("x-trace-id"); len(traceIDs) > 0 {
			return traceIDs[0]
		}
	}

	return ""
}
