// pkg/websocket/options.go
package websocket

import (
	"net/http"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/serializer"
	"github.com/prometheus/client_golang/prometheus"
)

// ================================
// Server Options
// ================================

// ServerOption 服务端选项
type ServerOption func(*Server)

// WithServerLogger 设置服务端日志记录器
func WithServerLogger(l logger.Logger) ServerOption {
	return func(s *Server) {
		s.logger = l
	}
}

// WithServerHandler 设置服务端消息处理器
func WithServerHandler(h MessageHandler) ServerOption {
	return func(s *Server) {
		s.handler = h
	}
}

// WithServerSerializer 设置服务端序列化器
func WithServerSerializer(ser serializer.Serializer) ServerOption {
	return func(s *Server) {
		s.serializer = ser
	}
}

// WithServerMiddleware 设置服务端中间件
func WithServerMiddleware(middlewares ...Middleware) ServerOption {
	return func(s *Server) {
		s.middlewares = append(s.middlewares, middlewares...)
	}
}

// WithServerMetricsRegisterer 设置 Prometheus 注册器
func WithServerMetricsRegisterer(registerer prometheus.Registerer) ServerOption {
	return func(s *Server) {
		s.metricsRegisterer = registerer
	}
}

// WithCheckOrigin 设置跨域检查函数
func WithCheckOrigin(fn func(r *http.Request) bool) ServerOption {
	return func(s *Server) {
		s.config.CheckOrigin = fn
	}
}

// WithAllowAllOrigins 允许所有跨域请求
func WithAllowAllOrigins() ServerOption {
	return WithCheckOrigin(func(r *http.Request) bool {
		return true
	})
}

// ================================
// Client Options
// ================================

// ClientOption 客户端选项
type ClientOption func(*Client)

// WithClientLogger 设置客户端日志记录器
func WithClientLogger(l logger.Logger) ClientOption {
	return func(c *Client) {
		c.logger = l
	}
}

// WithClientHandler 设置客户端消息处理器
func WithClientHandler(h MessageHandler) ClientOption {
	return func(c *Client) {
		c.handler = h
	}
}

// WithClientSerializer 设置客户端序列化器
func WithClientSerializer(ser serializer.Serializer) ClientOption {
	return func(c *Client) {
		c.serializer = ser
	}
}

// WithClientMiddleware 设置客户端中间件
func WithClientMiddleware(middlewares ...Middleware) ClientOption {
	return func(c *Client) {
		c.middlewares = append(c.middlewares, middlewares...)
	}
}

// WithClientMetricsRegisterer 设置 Prometheus 注册器
func WithClientMetricsRegisterer(registerer prometheus.Registerer) ClientOption {
	return func(c *Client) {
		c.metricsRegisterer = registerer
	}
}

// WithHTTPHeaders 设置 HTTP 请求头
func WithHTTPHeaders(headers map[string]string) ClientOption {
	return func(c *Client) {
		if c.config.Headers == nil {
			c.config.Headers = make(map[string]string)
		}
		for k, v := range headers {
			c.config.Headers[k] = v
		}
	}
}

// WithHTTPHeader 设置单个 HTTP 请求头
func WithHTTPHeader(key, value string) ClientOption {
	return func(c *Client) {
		if c.config.Headers == nil {
			c.config.Headers = make(map[string]string)
		}
		c.config.Headers[key] = value
	}
}

// WithOnReconnecting 设置重连中回调
func WithOnReconnecting(fn func(attempt int)) ClientOption {
	return func(c *Client) {
		c.onReconnecting = fn
	}
}

// WithOnReconnected 设置重连成功回调
func WithOnReconnected(fn func()) ClientOption {
	return func(c *Client) {
		c.onReconnected = fn
	}
}

// WithOnReconnectFailed 设置重连失败回调
func WithOnReconnectFailed(fn func(err error)) ClientOption {
	return func(c *Client) {
		c.onReconnectFailed = fn
	}
}

// ================================
// Connection Options
// ================================

// ConnectionOption 连接选项
type ConnectionOption func(*Connection)

// WithConnectionLogger 设置连接日志记录器
func WithConnectionLogger(l logger.Logger) ConnectionOption {
	return func(c *Connection) {
		c.logger = l
	}
}

// WithConnectionSerializer 设置连接序列化器
func WithConnectionSerializer(ser serializer.Serializer) ConnectionOption {
	return func(c *Connection) {
		c.serializer = ser
	}
}

// WithConnectionMetadata 设置连接元数据
func WithConnectionMetadata(key string, value interface{}) ConnectionOption {
	return func(c *Connection) {
		c.SetMetadata(key, value)
	}
}
