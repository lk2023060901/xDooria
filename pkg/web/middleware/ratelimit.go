package middleware

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/cache/lru"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/web/errors"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	// RequestsPerSecond 每秒请求数
	RequestsPerSecond int
	// Burst 突发容量
	Burst int
	// PerIP 是否按 IP 限流
	PerIP bool
	// PerPath 是否按路径限流
	PerPath bool
	// SkipPaths 跳过的路径
	SkipPaths []string
	// WaitMode 等待模式（true=等待，false=拒绝）
	WaitMode bool
	// WaitTimeout 等待超时
	WaitTimeout time.Duration
	// KeyFunc 自定义限流键生成函数
	KeyFunc func(*gin.Context) string

	// MaxLimiters 最大限流器数量
	MaxLimiters int
	// LimiterTTL 限流器过期时间
	LimiterTTL time.Duration
	// CleanupInterval 清理间隔
	CleanupInterval time.Duration
}

// RateLimiter 限流器
type RateLimiter struct {
	cfg      *RateLimitConfig
	global   *rate.Limiter
	limiters *lru.LRU[string, *rate.Limiter]
	logger   logger.Logger
}

// NewRateLimiter 创建限流器
func NewRateLimiter(l logger.Logger, cfg *RateLimitConfig) *RateLimiter {
	rl := &RateLimiter{
		cfg:    cfg,
		global: rate.NewLimiter(rate.Limit(cfg.RequestsPerSecond), cfg.Burst),
		logger: l,
	}

	rl.limiters = lru.New[string, *rate.Limiter](
		&lru.Config{
			MaxSize:         cfg.MaxLimiters,
			DefaultTTL:      cfg.LimiterTTL,
			CleanupInterval: cfg.CleanupInterval,
		},
		lru.WithOnEvict(func(key string, limiter *rate.Limiter) {
			l.Debug("rate limiter evicted", zap.String("key", key))
		}),
	)

	return rl
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow(key string) bool {
	if key == "" {
		return rl.global.Allow()
	}
	limiter := rl.getLimiter(key)
	return limiter.Allow()
}

// Wait 等待直到允许请求
func (rl *RateLimiter) Wait(ctx context.Context, key string) error {
	if key == "" {
		return rl.global.Wait(ctx)
	}
	limiter := rl.getLimiter(key)
	return limiter.Wait(ctx)
}

// getLimiter 获取或创建限流器
func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
	return rl.limiters.GetOrCreate(key, func() *rate.Limiter {
		return rate.NewLimiter(rate.Limit(rl.cfg.RequestsPerSecond), rl.cfg.Burst)
	})
}

// Close 关闭限流器
func (rl *RateLimiter) Close() error {
	return rl.limiters.Close()
}

// RateLimit 限流中间件
func RateLimit(limiter *RateLimiter) gin.HandlerFunc {
	skipPaths := make(map[string]struct{})
	for _, path := range limiter.cfg.SkipPaths {
		skipPaths[path] = struct{}{}
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 检查跳过路径
		if _, skip := skipPaths[path]; skip {
			c.Next()
			return
		}

		// 生成限流键
		key := generateKey(c, limiter.cfg)

		if limiter.cfg.WaitMode {
			// 等待模式
			ctx := c.Request.Context()
			if limiter.cfg.WaitTimeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, limiter.cfg.WaitTimeout)
				defer cancel()
			}

			if err := limiter.Wait(ctx, key); err != nil {
				limiter.logger.Warn("rate limit wait timeout",
					zap.String("key", key),
					zap.String("path", path),
					zap.Error(err),
				)
				abortWithRateLimitError(c)
				return
			}
		} else {
			// 拒绝模式
			if !limiter.Allow(key) {
				limiter.logger.Warn("rate limit exceeded",
					zap.String("key", key),
					zap.String("path", path),
				)
				abortWithRateLimitError(c)
				return
			}
		}

		c.Next()
	}
}

// generateKey 生成限流键
func generateKey(c *gin.Context, cfg *RateLimitConfig) string {
	if cfg.KeyFunc != nil {
		return cfg.KeyFunc(c)
	}

	var key string

	if cfg.PerIP {
		key = "ip:" + c.ClientIP()
	}

	if cfg.PerPath {
		if key != "" {
			key += ":path:" + c.Request.URL.Path
		} else {
			key = "path:" + c.Request.URL.Path
		}
	}

	return key
}

// abortWithRateLimitError 返回限流错误
func abortWithRateLimitError(c *gin.Context) {
	c.Header("Retry-After", strconv.Itoa(1))
	c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
		"code":    errors.CodeRateLimited,
		"message": "too many requests",
		"data":    nil,
	})
}
