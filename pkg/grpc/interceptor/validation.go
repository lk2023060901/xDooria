package interceptor

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Validator 参数校验接口
// protobuf 消息可以实现这个接口来提供自定义校验
type Validator interface {
	Validate() error
}

// ValidationConfig 校验拦截器配置
type ValidationConfig struct {
	// 是否启用（默认 true）
	Enabled bool

	// 是否记录校验失败日志（默认 true）
	LogValidationErrors bool
}

// DefaultValidationConfig 默认配置
func DefaultValidationConfig() *ValidationConfig {
	return &ValidationConfig{
		Enabled:             true,
		LogValidationErrors: true,
	}
}

// ServerValidationInterceptor Server 端校验拦截器（Unary）
func ServerValidationInterceptor(logger *logger.Logger, cfg *ValidationConfig) grpc.UnaryServerInterceptor {
	if cfg == nil {
		cfg = DefaultValidationConfig()
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !cfg.Enabled {
			return handler(ctx, req)
		}

		// 校验请求参数
		if err := validate(req); err != nil {
			if cfg.LogValidationErrors {
				logger.Warn("gRPC request validation failed",
					zap.String("grpc.method", info.FullMethod),
					zap.Error(err),
				)
			}
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		return handler(ctx, req)
	}
}

// StreamServerValidationInterceptor Server 端校验拦截器（Stream）
func StreamServerValidationInterceptor(logger *logger.Logger, cfg *ValidationConfig) grpc.StreamServerInterceptor {
	if cfg == nil {
		cfg = DefaultValidationConfig()
	}

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !cfg.Enabled {
			return handler(srv, ss)
		}

		// 对于流式调用，在 RecvMsg 时进行校验
		wrapped := &validationServerStream{
			ServerStream: ss,
			logger:       logger,
			cfg:          cfg,
			method:       info.FullMethod,
		}

		return handler(srv, wrapped)
	}
}

// validationServerStream 包装 ServerStream 以支持校验
type validationServerStream struct {
	grpc.ServerStream
	logger *logger.Logger
	cfg    *ValidationConfig
	method string
}

// RecvMsg 接收消息时进行校验
func (w *validationServerStream) RecvMsg(m interface{}) error {
	if err := w.ServerStream.RecvMsg(m); err != nil {
		return err
	}

	// 校验接收到的消息
	if err := validate(m); err != nil {
		if w.cfg.LogValidationErrors {
			w.logger.Warn("gRPC stream message validation failed",
				zap.String("grpc.method", w.method),
				zap.Error(err),
			)
		}
		return status.Error(codes.InvalidArgument, err.Error())
	}

	return nil
}

// ClientValidationInterceptor Client 端校验拦截器（Unary）
func ClientValidationInterceptor(logger *logger.Logger, cfg *ValidationConfig) grpc.UnaryClientInterceptor {
	if cfg == nil {
		cfg = DefaultValidationConfig()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !cfg.Enabled {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		// 校验请求参数
		if err := validate(req); err != nil {
			if cfg.LogValidationErrors {
				logger.Warn("gRPC client request validation failed",
					zap.String("grpc.method", method),
					zap.Error(err),
				)
			}
			return status.Error(codes.InvalidArgument, err.Error())
		}

		// 执行调用
		err := invoker(ctx, method, req, reply, cc, opts...)

		// 校验响应
		if err == nil {
			if err := validate(reply); err != nil {
				if cfg.LogValidationErrors {
					logger.Warn("gRPC client response validation failed",
						zap.String("grpc.method", method),
						zap.Error(err),
					)
				}
				return status.Error(codes.Internal, fmt.Sprintf("invalid response: %v", err))
			}
		}

		return err
	}
}

// validate 执行校验
func validate(msg interface{}) error {
	if msg == nil {
		return nil
	}

	// 如果消息实现了 Validator 接口，调用 Validate 方法
	if validator, ok := msg.(Validator); ok {
		return validator.Validate()
	}

	// 没有实现 Validator 接口，不校验
	return nil
}

// ValidationError 校验错误包装
type ValidationError struct {
	Field   string
	Message string
}

// Error 实现 error 接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on field '%s': %s", e.Field, e.Message)
}

// NewValidationError 创建校验错误
func NewValidationError(field, message string) error {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// MultiValidationError 多个校验错误
type MultiValidationError struct {
	Errors []error
}

// Error 实现 error 接口
func (e *MultiValidationError) Error() string {
	if len(e.Errors) == 0 {
		return "validation failed"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("validation failed: %d errors", len(e.Errors))
}

// Add 添加错误
func (e *MultiValidationError) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors 是否有错误
func (e *MultiValidationError) HasErrors() bool {
	return len(e.Errors) > 0
}

// ToError 转换为 error（如果没有错误则返回 nil）
func (e *MultiValidationError) ToError() error {
	if !e.HasErrors() {
		return nil
	}
	return e
}

// NewMultiValidationError 创建多重校验错误
func NewMultiValidationError() *MultiValidationError {
	return &MultiValidationError{
		Errors: make([]error, 0),
	}
}
