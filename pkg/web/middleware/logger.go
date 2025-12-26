package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Logger 适配 pkg/logger 的 Gin 日志中间件
func Logger(l logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		status := c.Writer.Status()
		
		fields := []interface{}{
			"status",     status,
			"method",     c.Request.Method,
			"path",       path,
			"query",      query,
			"ip",         c.ClientIP(),
			"latency",    latency.String(),
			"user_agent", c.Request.UserAgent(),
		}

		if len(c.Errors) > 0 {
			for _, e := range c.Errors.Errors() {
				l.ErrorContext(c.Request.Context(), e, fields...)
			}
		} else {
			msg := "http request"
			if status >= 400 {
				l.WarnContext(c.Request.Context(), msg, fields...)
			} else {
				l.InfoContext(c.Request.Context(), msg, fields...)
			}
		}
	}
}
