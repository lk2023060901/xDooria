package interceptor

import (
	"context"
	"time"

	"google.golang.org/grpc"
)

// ClientTimeoutInterceptor Client 端超时拦截器
// 如果调用时未设置 Context deadline，则使用默认超时
func ClientTimeoutInterceptor(defaultTimeout time.Duration) grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply interface{},
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		// 检查是否已设置 deadline
		if _, ok := ctx.Deadline(); !ok && defaultTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
			defer cancel()
		}

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// StreamClientTimeoutInterceptor 流式调用超时拦截器
func StreamClientTimeoutInterceptor(defaultTimeout time.Duration) grpc.StreamClientInterceptor {
	return func(
		ctx context.Context,
		desc *grpc.StreamDesc,
		cc *grpc.ClientConn,
		method string,
		streamer grpc.Streamer,
		opts ...grpc.CallOption,
	) (grpc.ClientStream, error) {
		if _, ok := ctx.Deadline(); !ok && defaultTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
			defer cancel()
		}

		return streamer(ctx, desc, cc, method, opts...)
	}
}

// ServerTimeoutInterceptor Server 端超时拦截器
// 用于设置方法级别的默认超时（如果客户端未设置）
func ServerTimeoutInterceptor(defaultTimeout time.Duration) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		// 检查客户端是否已设置 deadline
		if _, ok := ctx.Deadline(); !ok && defaultTimeout > 0 {
			// 客户端未设置，应用默认超时
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
			defer cancel()
		}

		// 执行实际处理
		return handler(ctx, req)
	}
}

// StreamServerTimeoutInterceptor 流式调用超时拦截器
func StreamServerTimeoutInterceptor(defaultTimeout time.Duration) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		ctx := ss.Context()

		if _, ok := ctx.Deadline(); !ok && defaultTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
			defer cancel()

			// 包装 ServerStream 以使用新的 context
			ss = &wrappedServerStream{
				ServerStream: ss,
				ctx:          ctx,
			}
		}

		return handler(srv, ss)
	}
}

type wrappedServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedServerStream) Context() context.Context {
	return w.ctx
}
