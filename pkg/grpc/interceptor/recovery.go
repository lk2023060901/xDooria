package interceptor

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RecoveryConfig Recovery 拦截器配置
type RecoveryConfig struct {
	// 是否启用（默认 true）
	Enabled bool

	// 自定义恢复处理函数
	RecoveryHandler func(ctx context.Context, p interface{}) error
}

// DefaultRecoveryConfig 默认配置
func DefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Enabled:         true,
		RecoveryHandler: nil, // 使用默认处理
	}
}

// ServerRecoveryInterceptor Server 端 Recovery 拦截器（Unary）
func ServerRecoveryInterceptor(l logger.Logger, cfg *RecoveryConfig) grpc.UnaryServerInterceptor {
	if cfg == nil {
		cfg = DefaultRecoveryConfig()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		if !cfg.Enabled {
			return handler(ctx, req)
		}

		defer func() {
			if p := recover(); p != nil {
				// 记录 panic 日志
				l.Error("gRPC panic recovered",
					"grpc.method", info.FullMethod,
					"panic", p,
					"stack", string(debug.Stack()),
				)

				// 使用自定义处理或默认处理
				if cfg.RecoveryHandler != nil {
					err = cfg.RecoveryHandler(ctx, p)
				} else {
					err = status.Errorf(codes.Internal, "Internal server error: %v", p)
				}
			}
		}()

		return handler(ctx, req)
	}
}

// StreamServerRecoveryInterceptor Server 端 Recovery 拦截器（Stream）
func StreamServerRecoveryInterceptor(l logger.Logger, cfg *RecoveryConfig) grpc.StreamServerInterceptor {
	if cfg == nil {
		cfg = DefaultRecoveryConfig()
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		if !cfg.Enabled {
			return handler(srv, ss)
		}

		defer func() {
			if p := recover(); p != nil {
				// 记录 panic 日志
				l.Error("gRPC stream panic recovered",
					"grpc.method", info.FullMethod,
					"panic", p,
					"stack", string(debug.Stack()),
				)

				// 使用自定义处理或默认处理
				if cfg.RecoveryHandler != nil {
					err = cfg.RecoveryHandler(ss.Context(), p)
				} else {
					err = status.Errorf(codes.Internal, "Internal server error: %v", p)
				}
			}
		}()

		return handler(srv, ss)
	}
}

// ClientRecoveryInterceptor Client 端 Recovery 拦截器（可选）
func ClientRecoveryInterceptor(l logger.Logger, cfg *RecoveryConfig) grpc.UnaryClientInterceptor {
	if cfg == nil {
		cfg = DefaultRecoveryConfig()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
		if !cfg.Enabled {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		defer func() {
			if p := recover(); p != nil {
				// 记录客户端 panic
				l.Error("gRPC client panic recovered",
					"grpc.method", method,
					"grpc.target", cc.Target(),
					"panic", p,
					"stack", string(debug.Stack()),
				)

				err = fmt.Errorf("client panic: %v", p)
			}
		}()

		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
