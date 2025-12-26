package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/web/metrics"
)

// Metrics 接口监控中间件
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath() // 获取路由定义的路径而非实际请求路径
		if path == "" {
			path = "unknown"
		}

		c.Next()

		latency := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())

		// 使用 pkg/web/metrics 中定义的指标
		metrics.HttpRequestsTotal.WithLabelValues(path, c.Request.Method, status).Inc()
		metrics.HttpRequestDuration.WithLabelValues(path, c.Request.Method).Observe(latency)
	}
}
