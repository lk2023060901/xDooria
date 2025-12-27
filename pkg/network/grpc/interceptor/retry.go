package interceptor

import (
	"context"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RetryConfig 重试拦截器配置
type RetryConfig struct {
	// 是否启用（默认 true）
	Enabled bool

	// 最大重试次数（默认 3）
	MaxAttempts int

	// 初始退避时间（默认 100ms）
	InitialBackoff time.Duration

	// 最大退避时间（默认 10s）
	MaxBackoff time.Duration

	// 退避倍数（默认 2.0）
	BackoffMultiplier float64

	// 可重试的状态码（默认 Unavailable, DeadlineExceeded, ResourceExhausted）
	RetryableCodes []codes.Code

	// 是否记录重试日志（默认 true）
	LogRetries bool
}

// DefaultRetryConfig 默认配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Enabled:           true,
		MaxAttempts:       3,
		InitialBackoff:    100 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableCodes: []codes.Code{
			codes.Unavailable,      // 服务不可用
			codes.DeadlineExceeded, // 超时
			codes.ResourceExhausted, // 资源耗尽（限流）
		},
		LogRetries: true,
	}
}

// ClientRetryInterceptor 客户端重试拦截器
func ClientRetryInterceptor(l logger.Logger, cfg *RetryConfig) grpc.UnaryClientInterceptor {
	if cfg == nil {
		cfg = DefaultRetryConfig()
	}

	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if !cfg.Enabled {
			return invoker(ctx, method, req, reply, cc, opts...)
		}

		var lastErr error
		backoff := cfg.InitialBackoff

		for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
			// 执行调用
			err := invoker(ctx, method, req, reply, cc, opts...)

			// 成功则返回
			if err == nil {
				if attempt > 1 && cfg.LogRetries {
					l.Info("gRPC retry succeeded",
						"grpc.method", method,
						"attempt", attempt,
					)
				}
				return nil
			}

			lastErr = err

			// 检查是否可重试
			if !isRetryable(err, cfg.RetryableCodes) {
				if cfg.LogRetries {
					l.Debug("gRPC error not retryable",
						"grpc.method", method,
						"grpc.code", status.Code(err).String(),
						"error", err,
					)
				}
				return err
			}

			// 最后一次尝试失败
			if attempt == cfg.MaxAttempts {
				if cfg.LogRetries {
					l.Warn("gRPC retry exhausted",
						"grpc.method", method,
						"attempts", attempt,
						"error", err,
					)
				}
				return err
			}

			// 检查 context 是否已取消
			if ctx.Err() != nil {
				return ctx.Err()
			}

			// 记录重试日志
			if cfg.LogRetries {
				l.Info("gRPC retrying request",
					"grpc.method", method,
					"attempt", attempt,
					"backoff", backoff,
					"grpc.code", status.Code(err).String(),
					"error", err,
				)
			}

			// 退避等待
			select {
			case <-time.After(backoff):
				// 计算下次退避时间
				backoff = time.Duration(float64(backoff) * cfg.BackoffMultiplier)
				if backoff > cfg.MaxBackoff {
					backoff = cfg.MaxBackoff
				}
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return lastErr
	}
}

// isRetryable 检查错误是否可重试
func isRetryable(err error, retryableCodes []codes.Code) bool {
	code := status.Code(err)
	for _, retryableCode := range retryableCodes {
		if code == retryableCode {
			return true
		}
	}
	return false
}

// WithRetryableCodes 设置可重试的状态码
func WithRetryableCodes(codes ...codes.Code) func(*RetryConfig) {
	return func(cfg *RetryConfig) {
		cfg.RetryableCodes = codes
	}
}

// WithMaxAttempts 设置最大重试次数
func WithMaxAttempts(maxAttempts int) func(*RetryConfig) {
	return func(cfg *RetryConfig) {
		if maxAttempts > 0 {
			cfg.MaxAttempts = maxAttempts
		}
	}
}

// WithBackoff 设置退避策略
func WithBackoff(initial, max time.Duration, multiplier float64) func(*RetryConfig) {
	return func(cfg *RetryConfig) {
		if initial > 0 {
			cfg.InitialBackoff = initial
		}
		if max > 0 {
			cfg.MaxBackoff = max
		}
		if multiplier > 0 {
			cfg.BackoffMultiplier = multiplier
		}
	}
}

// ApplyRetryOptions 应用重试选项到配置
func ApplyRetryOptions(cfg *RetryConfig, opts ...func(*RetryConfig)) *RetryConfig {
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg
}
