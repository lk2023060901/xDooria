package interceptor

import (
	"context"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lk2023060901/xdooria/pkg/config"
	xdooriasentry "github.com/lk2023060901/xdooria/pkg/sentry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// SentryConfig Sentry 拦截器配置
type SentryConfig struct {
	// 是否启用（默认 true）
	Enabled bool `mapstructure:"enabled" json:"enabled"`

	// 是否上报 Panic（默认 true）
	ReportPanics bool `mapstructure:"report_panics" json:"report_panics"`

	// 是否上报错误（默认 true，上报非 OK 状态）
	ReportErrors bool `mapstructure:"report_errors" json:"report_errors"`

	// 需要上报的错误级别（默认只上报 Internal、Unknown、DataLoss）
	ReportCodes []codes.Code `mapstructure:"report_codes" json:"report_codes"`

	// 采样率（0.0 - 1.0，默认 1.0）
	SampleRate float64 `mapstructure:"sample_rate" json:"sample_rate"`

	// 是否包含请求详情（默认 false）
	IncludeRequestDetails bool `mapstructure:"include_request_details" json:"include_request_details"`

	// 敏感字段列表（需要过滤）
	SensitiveFields []string `mapstructure:"sensitive_fields" json:"sensitive_fields"`

	// 超时时间（默认 2s）
	FlushTimeout time.Duration `mapstructure:"flush_timeout" json:"flush_timeout"`
}

// DefaultSentryConfig 返回默认配置
func DefaultSentryConfig() *SentryConfig {
	return &SentryConfig{
		Enabled:               true,
		ReportPanics:          true,
		ReportErrors:          true,
		ReportCodes:           []codes.Code{codes.Internal, codes.Unknown, codes.DataLoss},
		SampleRate:            1.0,
		IncludeRequestDetails: false,
		SensitiveFields:       []string{"password", "token", "secret", "auth"},
		FlushTimeout:          2 * time.Second,
	}
}

// ServerSentryInterceptor Server 端 Sentry 拦截器（Unary）
func ServerSentryInterceptor(sentryClient *xdooriasentry.Client, cfg *SentryConfig) grpc.UnaryServerInterceptor {
	// 合并配置
	newCfg, err := config.MergeConfig(DefaultSentryConfig(), cfg)
	if err != nil {
		panic("failed to merge sentry config: " + err.Error())
	}

	if !newCfg.Enabled || sentryClient == nil {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 设置 Sentry Hub（每个请求独立的 scope）
		hub := sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)

		// 配置 scope
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("grpc", map[string]interface{}{
				"method": info.FullMethod,
				"type":   "unary",
			})

			// 添加 peer 信息
			if p, ok := peer.FromContext(ctx); ok {
				scope.SetContext("peer", map[string]interface{}{
					"address": p.Addr.String(),
				})
			}

			// 添加 trace ID（如果有）
			if md, ok := metadata.FromIncomingContext(ctx); ok {
				if traceID := md.Get("x-trace-id"); len(traceID) > 0 {
					scope.SetTag("trace_id", traceID[0])
				}
				if userID := md.Get("x-user-id"); len(userID) > 0 {
					scope.SetUser(sentry.User{ID: userID[0]})
				}
			}
		})

		// 执行处理
		resp, err := handler(ctx, req)

		// 检查是否需要上报错误
		if err != nil && newCfg.ReportErrors {
			st, _ := status.FromError(err)
			if shouldReportError(st.Code(), newCfg) {
				hub.CaptureException(err)
				hub.Flush(newCfg.FlushTimeout)
			}
		}

		return resp, err
	}
}

// StreamServerSentryInterceptor Server 端 Sentry 拦截器（Stream）
func StreamServerSentryInterceptor(sentryClient *xdooriasentry.Client, cfg *SentryConfig) grpc.StreamServerInterceptor {
	newCfg, err := config.MergeConfig(DefaultSentryConfig(), cfg)
	if err != nil {
		panic("failed to merge sentry config: " + err.Error())
	}

	if !newCfg.Enabled || sentryClient == nil {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// 设置 Sentry Hub
		hub := sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)

		// 配置 scope
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("grpc", map[string]interface{}{
				"method":        info.FullMethod,
				"type":          "stream",
				"is_client_stream": info.IsClientStream,
				"is_server_stream": info.IsServerStream,
			})

			if p, ok := peer.FromContext(ctx); ok {
				scope.SetContext("peer", map[string]interface{}{
					"address": p.Addr.String(),
				})
			}

			if md, ok := metadata.FromIncomingContext(ctx); ok {
				if traceID := md.Get("x-trace-id"); len(traceID) > 0 {
					scope.SetTag("trace_id", traceID[0])
				}
				if userID := md.Get("x-user-id"); len(userID) > 0 {
					scope.SetUser(sentry.User{ID: userID[0]})
				}
			}
		})

		// 包装 stream
		wrappedStream := &wrappedStreamWithSentry{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 执行处理
		err := handler(srv, wrappedStream)

		// 检查是否需要上报错误
		if err != nil && newCfg.ReportErrors {
			st, _ := status.FromError(err)
			if shouldReportError(st.Code(), newCfg) {
				hub.CaptureException(err)
				hub.Flush(newCfg.FlushTimeout)
			}
		}

		return err
	}
}

// ClientSentryInterceptor Client 端 Sentry 拦截器（Unary）
func ClientSentryInterceptor(sentryClient *xdooriasentry.Client, cfg *SentryConfig) grpc.UnaryClientInterceptor {
	newCfg, err := config.MergeConfig(DefaultSentryConfig(), cfg)
	if err != nil {
		panic("failed to merge sentry config: " + err.Error())
	}

	if !newCfg.Enabled || sentryClient == nil {
		return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 设置 Sentry Hub
		hub := sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)

		// 配置 scope
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("grpc", map[string]interface{}{
				"method": method,
				"type":   "client_unary",
				"target": cc.Target(),
			})
		})

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 检查是否需要上报错误
		if err != nil && newCfg.ReportErrors {
			st, _ := status.FromError(err)
			if shouldReportError(st.Code(), newCfg) {
				hub.CaptureException(err)
				hub.Flush(newCfg.FlushTimeout)
			}
		}

		return err
	}
}

// StreamClientSentryInterceptor Client 端 Sentry 拦截器（Stream）
func StreamClientSentryInterceptor(sentryClient *xdooriasentry.Client, cfg *SentryConfig) grpc.StreamClientInterceptor {
	newCfg, err := config.MergeConfig(DefaultSentryConfig(), cfg)
	if err != nil {
		panic("failed to merge sentry config: " + err.Error())
	}

	if !newCfg.Enabled || sentryClient == nil {
		return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return streamer(ctx, desc, cc, method, opts...)
		}
	}

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// 设置 Sentry Hub
		hub := sentry.CurrentHub().Clone()
		ctx = sentry.SetHubOnContext(ctx, hub)

		// 配置 scope
		hub.ConfigureScope(func(scope *sentry.Scope) {
			scope.SetContext("grpc", map[string]interface{}{
				"method":            method,
				"type":              "client_stream",
				"target":            cc.Target(),
				"client_streams":    desc.ClientStreams,
				"server_streams":    desc.ServerStreams,
			})
		})

		// 执行流式调用
		stream, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil && newCfg.ReportErrors {
			st, _ := status.FromError(err)
			if shouldReportError(st.Code(), newCfg) {
				hub.CaptureException(err)
				hub.Flush(newCfg.FlushTimeout)
			}
		}

		return stream, err
	}
}

// SentryRecoveryHandler 创建用于 Recovery 拦截器的 Sentry 处理器
func SentryRecoveryHandler(sentryClient *xdooriasentry.Client, flushTimeout time.Duration) func(context.Context, interface{}) error {
	if sentryClient == nil {
		return nil
	}

	return func(ctx context.Context, p interface{}) error {
		hub := sentry.GetHubFromContext(ctx)
		if hub == nil {
			hub = sentry.CurrentHub()
		}

		// 捕获 panic
		hub.RecoverWithContext(ctx, p)
		hub.Flush(flushTimeout)

		return status.Errorf(codes.Internal, "panic recovered: %v", p)
	}
}

// shouldReportError 判断是否应该上报错误
func shouldReportError(code codes.Code, cfg *SentryConfig) bool {
	// 不上报正常状态
	if code == codes.OK {
		return false
	}

	// 如果没有配置 ReportCodes，默认上报所有错误
	if len(cfg.ReportCodes) == 0 {
		return true
	}

	// 检查是否在配置的错误列表中
	for _, c := range cfg.ReportCodes {
		if c == code {
			return true
		}
	}

	return false
}

// wrappedStreamWithSentry 包装 ServerStream 以传递 context
type wrappedStreamWithSentry struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStreamWithSentry) Context() context.Context {
	return w.ctx
}

// CapturePanic 手动捕获 panic 并上报到 Sentry
func CapturePanic(ctx context.Context, p interface{}) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.RecoverWithContext(ctx, p)
}

// CaptureError 手动捕获错误并上报到 Sentry
func CaptureError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	hub.CaptureException(err)
}

// CaptureMessage 捕获消息并上报到 Sentry
func CaptureMessage(ctx context.Context, message string, level sentry.Level) {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
	}

	event := sentry.NewEvent()
	event.Message = message
	event.Level = level

	hub.CaptureEvent(event)
}

// WithSentryContext 添加额外的 Sentry 上下文信息
func WithSentryContext(ctx context.Context, key string, value map[string]interface{}) context.Context {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetContext(key, value)
	})

	return ctx
}

// WithSentryTag 添加 Sentry 标签
func WithSentryTag(ctx context.Context, key, value string) context.Context {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetTag(key, value)
	})

	return ctx
}

// WithSentryUser 设置 Sentry 用户信息
func WithSentryUser(ctx context.Context, userID, username, email string) context.Context {
	hub := sentry.GetHubFromContext(ctx)
	if hub == nil {
		hub = sentry.CurrentHub()
		ctx = sentry.SetHubOnContext(ctx, hub)
	}

	hub.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:       userID,
			Username: username,
			Email:    email,
		})
	})

	return ctx
}
