package interceptor

import (
	"context"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// TracingConfig Tracing 拦截器配置
type TracingConfig struct {
	// 启用 Tracing（默认 true）
	Enabled bool `mapstructure:"enabled" json:"enabled"`

	// Tracer 名称（默认 "grpc"）
	TracerName string `mapstructure:"tracer_name" json:"tracer_name"`

	// 是否记录请求和响应（默认 false）
	RecordPayload bool `mapstructure:"record_payload" json:"record_payload"`
}

// DefaultTracingConfig 返回默认配置
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:       true,
		TracerName:    "grpc",
		RecordPayload: false,
	}
}

// ServerTracingInterceptor 服务端 Tracing 拦截器
func ServerTracingInterceptor(cfg *TracingConfig) grpc.UnaryServerInterceptor {
	// 合并配置
	newCfg, err := config.MergeConfig(DefaultTracingConfig(), cfg)
	if err != nil {
		panic("failed to merge tracing config: " + err.Error())
	}

	if !newCfg.Enabled {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	tracer := otel.GetTracerProvider().Tracer(newCfg.TracerName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 从 metadata 中提取 trace context
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = propagator.Extract(ctx, &metadataCarrier{md: md})

		// 创建 span
		ctx, span := tracer.Start(
			ctx,
			info.FullMethod,
			otel.WithSpanKind(otel.SpanKindServer),
			otel.WithAttributes(
				otel.String("rpc.system", "grpc"),
				otel.String("rpc.service", extractServiceName(info.FullMethod)),
				otel.String("rpc.method", extractMethodName(info.FullMethod)),
			),
		)
		defer span.End()

		// 执行处理
		resp, err := handler(ctx, req)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otel.CodeError, err.Error())

			if st, ok := status.FromError(err); ok {
				span.SetAttributes(
					otel.String("rpc.grpc.status_code", st.Code().String()),
				)
			}
		} else {
			span.SetStatus(otel.CodeOk, "")
			span.SetAttributes(
				otel.String("rpc.grpc.status_code", "OK"),
			)
		}

		return resp, err
	}
}

// StreamServerTracingInterceptor 流式服务端 Tracing 拦截器
func StreamServerTracingInterceptor(cfg *TracingConfig) grpc.StreamServerInterceptor {
	newCfg, err := config.MergeConfig(DefaultTracingConfig(), cfg)
	if err != nil {
		panic("failed to merge tracing config: " + err.Error())
	}

	if !newCfg.Enabled {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	tracer := otel.GetTracerProvider().Tracer(newCfg.TracerName)
	propagator := otel.GetTextMapPropagator()

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		ctx := ss.Context()

		// 从 metadata 中提取 trace context
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = propagator.Extract(ctx, &metadataCarrier{md: md})

		// 创建 span
		ctx, span := tracer.Start(
			ctx,
			info.FullMethod,
			otel.WithSpanKind(otel.SpanKindServer),
			otel.WithAttributes(
				otel.String("rpc.system", "grpc"),
				otel.String("rpc.service", extractServiceName(info.FullMethod)),
				otel.String("rpc.method", extractMethodName(info.FullMethod)),
				otel.Bool("rpc.stream", true),
			),
		)
		defer span.End()

		// 包装 stream
		wrappedStream := &wrappedStreamWithTracing{
			ServerStream: ss,
			ctx:          ctx,
		}

		// 执行处理
		err := handler(srv, wrappedStream)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otel.CodeError, err.Error())

			if st, ok := status.FromError(err); ok {
				span.SetAttributes(
					otel.String("rpc.grpc.status_code", st.Code().String()),
				)
			}
		} else {
			span.SetStatus(otel.CodeOk, "")
			span.SetAttributes(
				otel.String("rpc.grpc.status_code", "OK"),
			)
		}

		return err
	}
}

// ClientTracingInterceptor 客户端 Tracing 拦截器
func ClientTracingInterceptor(cfg *TracingConfig) grpc.UnaryClientInterceptor {
	newCfg, err := config.MergeConfig(DefaultTracingConfig(), cfg)
	if err != nil {
		panic("failed to merge tracing config: " + err.Error())
	}

	if !newCfg.Enabled {
		return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
			return invoker(ctx, method, req, reply, cc, opts...)
		}
	}

	tracer := otel.GetTracerProvider().Tracer(newCfg.TracerName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		// 创建 span
		ctx, span := tracer.Start(
			ctx,
			method,
			otel.WithSpanKind(otel.SpanKindClient),
			otel.WithAttributes(
				otel.String("rpc.system", "grpc"),
				otel.String("rpc.service", extractServiceName(method)),
				otel.String("rpc.method", extractMethodName(method)),
			),
		)
		defer span.End()

		// 注入 trace context 到 metadata
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		carrier := &metadataCarrier{md: md}
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, carrier.md)

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 记录错误
		if err != nil {
			span.RecordError(err)
			span.SetStatus(otel.CodeError, err.Error())

			if st, ok := status.FromError(err); ok {
				span.SetAttributes(
					otel.String("rpc.grpc.status_code", st.Code().String()),
				)
			}
		} else {
			span.SetStatus(otel.CodeOk, "")
			span.SetAttributes(
				otel.String("rpc.grpc.status_code", "OK"),
			)
		}

		return err
	}
}

// StreamClientTracingInterceptor 流式客户端 Tracing 拦截器
func StreamClientTracingInterceptor(cfg *TracingConfig) grpc.StreamClientInterceptor {
	newCfg, err := config.MergeConfig(DefaultTracingConfig(), cfg)
	if err != nil {
		panic("failed to merge tracing config: " + err.Error())
	}

	if !newCfg.Enabled {
		return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
			return streamer(ctx, desc, cc, method, opts...)
		}
	}

	tracer := otel.GetTracerProvider().Tracer(newCfg.TracerName)
	propagator := otel.GetTextMapPropagator()

	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		// 创建 span
		ctx, span := tracer.Start(
			ctx,
			method,
			otel.WithSpanKind(otel.SpanKindClient),
			otel.WithAttributes(
				otel.String("rpc.system", "grpc"),
				otel.String("rpc.service", extractServiceName(method)),
				otel.String("rpc.method", extractMethodName(method)),
				otel.Bool("rpc.stream", true),
			),
		)

		// 注入 trace context
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.New(nil)
		}
		carrier := &metadataCarrier{md: md}
		propagator.Inject(ctx, carrier)
		ctx = metadata.NewOutgoingContext(ctx, carrier.md)

		// 执行流式调用
		stream, err := streamer(ctx, desc, cc, method, opts...)

		if err != nil {
			span.RecordError(err)
			span.SetStatus(otel.CodeError, err.Error())
			span.End()
			return nil, err
		}

		// 包装 stream，在关闭时结束 span
		return &wrappedClientStreamWithTracing{
			ClientStream: stream,
			span:         span,
		}, nil
	}
}

// metadataCarrier 实现 TextMapCarrier 接口
type metadataCarrier struct {
	md metadata.MD
}

func (c *metadataCarrier) Get(key string) string {
	values := c.md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (c *metadataCarrier) Set(key, value string) {
	c.md.Set(key, value)
}

func (c *metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(c.md))
	for k := range c.md {
		keys = append(keys, k)
	}
	return keys
}

// wrappedStreamWithTracing 包装 ServerStream 以传递 context
type wrappedStreamWithTracing struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStreamWithTracing) Context() context.Context {
	return w.ctx
}

// wrappedClientStreamWithTracing 包装 ClientStream 以在结束时关闭 span
type wrappedClientStreamWithTracing struct {
	grpc.ClientStream
	span otel.Span
}

func (w *wrappedClientStreamWithTracing) RecvMsg(m interface{}) error {
	err := w.ClientStream.RecvMsg(m)
	if err != nil {
		w.span.RecordError(err)
		w.span.SetStatus(otel.CodeError, err.Error())
		w.span.End()
	}
	return err
}

func (w *wrappedClientStreamWithTracing) CloseSend() error {
	err := w.ClientStream.CloseSend()
	if err == nil {
		w.span.SetStatus(otel.CodeOk, "")
	}
	w.span.End()
	return err
}

// extractServiceName 从方法名提取服务名
func extractServiceName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}
	if fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}

// extractMethodName 从方法名提取方法名称
func extractMethodName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[i+1:]
		}
	}
	return fullMethod
}
