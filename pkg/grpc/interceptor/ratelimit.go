package interceptor

import (
	"context"
	"sync"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RateLimitConfig 限流拦截器配置
type RateLimitConfig struct {
	// 是否启用（默认 true）
	Enabled bool

	// 每秒允许的请求数（默认 100）
	RequestsPerSecond int

	// 突发容量（默认 RequestsPerSecond * 2）
	Burst int

	// 是否按方法限流（默认 false，全局限流）
	PerMethod bool

	// 是否记录限流日志（默认 true）
	LogRateLimits bool
}

// DefaultRateLimitConfig 默认配置
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 100,
		Burst:             200,
		PerMethod:         false,
		LogRateLimits:     true,
	}
}

// RateLimiter 限流器
type RateLimiter struct {
	cfg      *RateLimitConfig
	logger   *logger.Logger
	global   *rate.Limiter
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
}

// NewRateLimiter 创建限流器
func NewRateLimiter(logger *logger.Logger, cfg *RateLimitConfig) *RateLimiter {
	if cfg == nil {
		cfg = DefaultRateLimitConfig()
	}

	// 设置默认 Burst
	if cfg.Burst == 0 {
		cfg.Burst = cfg.RequestsPerSecond * 2
	}

	rl := &RateLimiter{
		cfg:    cfg,
		logger: logger,
	}

	// 全局限流器
	if !cfg.PerMethod {
		rl.global = rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst)
	} else {
		rl.limiters = make(map[string]*rate.Limiter)
	}

	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(method string) bool {
	if !rl.cfg.Enabled {
		return true
	}

	if rl.global != nil {
		return rl.global.Allow()
	}

	// 按方法限流
	limiter := rl.getLimiter(method)
	return limiter.Allow()
}

// Wait 等待直到允许请求（带超时）
func (rl *RateLimiter) Wait(ctx context.Context, method string) error {
	if !rl.cfg.Enabled {
		return nil
	}

	if rl.global != nil {
		return rl.global.Wait(ctx)
	}

	// 按方法限流
	limiter := rl.getLimiter(method)
	return limiter.Wait(ctx)
}

// getLimiter 获取或创建方法级别的限流器
func (rl *RateLimiter) getLimiter(method string) *rate.Limiter {
	rl.mu.RLock()
	limiter, exists := rl.limiters[method]
	rl.mu.RUnlock()

	if exists {
		return limiter
	}

	// 创建新的限流器
	rl.mu.Lock()
	defer rl.mu.Unlock()

	// 双重检查
	if limiter, exists := rl.limiters[method]; exists {
		return limiter
	}

	limiter = rate.NewLimiter(rate.Limit(rl.cfg.RequestsPerSecond), rl.cfg.Burst)
	rl.limiters[method] = limiter
	return limiter
}

// ServerRateLimitInterceptor Server 端限流拦截器（Unary）
func ServerRateLimitInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if !limiter.Allow(info.FullMethod) {
			if limiter.cfg.LogRateLimits {
				limiter.logger.Warn("gRPC request rate limited",
					zap.String("grpc.method", info.FullMethod),
				)
			}
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}

// StreamServerRateLimitInterceptor Server 端限流拦截器（Stream）
func StreamServerRateLimitInterceptor(limiter *RateLimiter) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if !limiter.Allow(info.FullMethod) {
			if limiter.cfg.LogRateLimits {
				limiter.logger.Warn("gRPC stream rate limited",
					zap.String("grpc.method", info.FullMethod),
				)
			}
			return status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(srv, ss)
	}
}

// ServerRateLimitWithWaitInterceptor Server 端限流拦截器（等待模式）
// 不是直接拒绝，而是等待直到允许请求
func ServerRateLimitWithWaitInterceptor(limiter *RateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := limiter.Wait(ctx, info.FullMethod); err != nil {
			if limiter.cfg.LogRateLimits {
				limiter.logger.Warn("gRPC request rate limit wait failed",
					zap.String("grpc.method", info.FullMethod),
					zap.Error(err),
				)
			}
			return nil, status.Error(codes.ResourceExhausted, "rate limit wait failed")
		}

		return handler(ctx, req)
	}
}

// MethodRateLimitConfig 方法级别限流配置
type MethodRateLimitConfig struct {
	Methods map[string]*RateLimitConfig // 方法名 -> 限流配置
	Default *RateLimitConfig             // 默认配置
}

// MethodRateLimiter 方法级别限流器
type MethodRateLimiter struct {
	limiters map[string]*RateLimiter
	default_ *RateLimiter
	logger   *logger.Logger
}

// NewMethodRateLimiter 创建方法级别限流器
func NewMethodRateLimiter(logger *logger.Logger, cfg *MethodRateLimitConfig) *MethodRateLimiter {
	mrl := &MethodRateLimiter{
		limiters: make(map[string]*RateLimiter),
		logger:   logger,
	}

	// 创建默认限流器
	if cfg.Default != nil {
		mrl.default_ = NewRateLimiter(logger, cfg.Default)
	}

	// 创建方法级别限流器
	for method, methodCfg := range cfg.Methods {
		mrl.limiters[method] = NewRateLimiter(logger, methodCfg)
	}

	return mrl
}

// ServerMethodRateLimitInterceptor 方法级别限流拦截器
func ServerMethodRateLimitInterceptor(limiter *MethodRateLimiter) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// 获取方法对应的限流器
		methodLimiter, exists := limiter.limiters[info.FullMethod]
		if !exists {
			methodLimiter = limiter.default_
		}

		// 没有配置限流器，直接通过
		if methodLimiter == nil {
			return handler(ctx, req)
		}

		// 检查限流
		if !methodLimiter.Allow(info.FullMethod) {
			if methodLimiter.cfg.LogRateLimits {
				limiter.logger.Warn("gRPC method rate limited",
					zap.String("grpc.method", info.FullMethod),
				)
			}
			return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
		}

		return handler(ctx, req)
	}
}
