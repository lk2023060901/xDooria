// pkg/websocket/middleware.go
package websocket

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// middlewarePool 中间件专用工作池
var middlewarePool = conc.NewPool[struct{}](100)

// Middleware 中间件函数类型
type Middleware func(HandlerFunc) HandlerFunc

// MiddlewareChain 中间件链
type MiddlewareChain struct {
	middlewares []Middleware
}

// NewMiddlewareChain 创建中间件链
func NewMiddlewareChain(middlewares ...Middleware) *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: middlewares,
	}
}

// Use 添加中间件
func (c *MiddlewareChain) Use(middlewares ...Middleware) *MiddlewareChain {
	c.middlewares = append(c.middlewares, middlewares...)
	return c
}

// Then 将中间件链应用到处理函数
func (c *MiddlewareChain) Then(handler HandlerFunc) HandlerFunc {
	// 从后向前包装，确保执行顺序正确
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		handler = c.middlewares[i](handler)
	}
	return handler
}

// Len 返回中间件数量
func (c *MiddlewareChain) Len() int {
	return len(c.middlewares)
}

// ================================
// 内置中间件
// ================================

// Recovery Panic 恢复中间件
func Recovery(log logger.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) (err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := debug.Stack()
					if log != nil {
						log.Error("websocket handler panic recovered",
							"panic", r,
							"conn_id", conn.ID(),
							"stack", string(stack),
						)
					}
					// 将 panic 转换为错误
					switch v := r.(type) {
					case error:
						err = v
					default:
						err = ErrConnectionFailed
					}
				}
			}()
			return next(conn, msg)
		}
	}
}

// Logger 日志中间件
func Logger(log logger.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			start := time.Now()

			// 执行下一个处理器
			err := next(conn, msg)

			// 记录日志
			duration := time.Since(start)
			fields := []interface{}{
				"conn_id", conn.ID(),
				"msg_size", len(msg.Data),
				"duration", duration.String(),
			}

			if err != nil {
				fields = append(fields, "error", err.Error())
				log.Warn("websocket message handled with error", fields...)
			} else {
				log.Debug("websocket message handled", fields...)
			}

			return err
		}
	}
}

// RateLimiter 限流器接口
type RateLimiter interface {
	// Allow 检查是否允许请求
	Allow(key string) bool
}

// simpleRateLimiter 简单的滑动窗口限流器
type simpleRateLimiter struct {
	limit    int
	window   time.Duration
	counters sync.Map // map[string]*rateLimitCounter
}

type rateLimitCounter struct {
	mu       sync.Mutex
	count    int
	windowAt time.Time
}

// NewSimpleRateLimiter 创建简单限流器
func NewSimpleRateLimiter(limit int, window time.Duration) RateLimiter {
	return &simpleRateLimiter{
		limit:  limit,
		window: window,
	}
}

func (r *simpleRateLimiter) Allow(key string) bool {
	now := time.Now()

	// 获取或创建计数器
	val, _ := r.counters.LoadOrStore(key, &rateLimitCounter{
		windowAt: now,
	})
	counter := val.(*rateLimitCounter)

	counter.mu.Lock()
	defer counter.mu.Unlock()

	// 检查是否需要重置窗口
	if now.Sub(counter.windowAt) >= r.window {
		counter.count = 0
		counter.windowAt = now
	}

	// 检查是否超过限制
	if counter.count >= r.limit {
		return false
	}

	counter.count++
	return true
}

// RateLimit 限流中间件
func RateLimit(limiter RateLimiter) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			if !limiter.Allow(conn.ID()) {
				return ErrSendQueueFull // 复用错误，表示请求被限流
			}
			return next(conn, msg)
		}
	}
}

// RateLimitPerConnection 每连接限流中间件
func RateLimitPerConnection(limit int, window time.Duration) Middleware {
	limiter := NewSimpleRateLimiter(limit, window)
	return RateLimit(limiter)
}

// RateLimitPerIP 每 IP 限流中间件
func RateLimitPerIP(limit int, window time.Duration) Middleware {
	limiter := NewSimpleRateLimiter(limit, window)
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			// 使用远程地址作为限流 key
			if !limiter.Allow(conn.RemoteAddr()) {
				return ErrSendQueueFull
			}
			return next(conn, msg)
		}
	}
}

// AuthValidator 认证验证器接口
type AuthValidator interface {
	// Validate 验证连接
	Validate(conn *Connection, msg *Message) error
}

// Auth 认证中间件
func Auth(validator AuthValidator) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			if err := validator.Validate(conn, msg); err != nil {
				return err
			}
			return next(conn, msg)
		}
	}
}

// TokenAuthValidator Token 认证验证器
type TokenAuthValidator struct {
	tokenKey   string // 从元数据中获取 token 的 key
	validateFn func(token string) error
}

// NewTokenAuthValidator 创建 Token 认证验证器
func NewTokenAuthValidator(tokenKey string, validateFn func(token string) error) *TokenAuthValidator {
	return &TokenAuthValidator{
		tokenKey:   tokenKey,
		validateFn: validateFn,
	}
}

// Validate 验证 Token
func (v *TokenAuthValidator) Validate(conn *Connection, msg *Message) error {
	token, ok := conn.GetMetadata(v.tokenKey)
	if !ok {
		return ErrConnectionFailed
	}
	tokenStr, ok := token.(string)
	if !ok {
		return ErrConnectionFailed
	}
	return v.validateFn(tokenStr)
}

// MessageValidator 消息验证器接口
type MessageValidator interface {
	// Validate 验证消息
	Validate(msg *Message) error
}

// Validator 消息验证中间件
func Validator(v MessageValidator) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			if err := v.Validate(msg); err != nil {
				return err
			}
			return next(conn, msg)
		}
	}
}

// MaxMessageSize 消息大小限制中间件
func MaxMessageSize(maxSize int64) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			if int64(len(msg.Data)) > maxSize {
				return ErrMessageTooBig
			}
			return next(conn, msg)
		}
	}
}

// Timeout 超时中间件
func Timeout(timeout time.Duration) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(conn *Connection, msg *Message) error {
			done := make(chan error, 1)
			middlewarePool.Submit(func() (struct{}, error) {
				done <- next(conn, msg)
				return struct{}{}, nil
			})

			select {
			case err := <-done:
				return err
			case <-time.After(timeout):
				return ErrConnectionTimeout
			}
		}
	}
}
