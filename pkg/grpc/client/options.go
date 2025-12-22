package client

import (
	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc"
)

// Option Client 配置选项
type Option func(*Client)

// WithLogger 设置自定义 logger
func WithLogger(l logger.Logger) Option {
	return func(c *Client) {
		c.logger = l
	}
}

// WithDialOptions 添加 gRPC DialOption
func WithDialOptions(opts ...grpc.DialOption) Option {
	return func(c *Client) {
		c.dialOpts = append(c.dialOpts, opts...)
	}
}

// WithUnaryInterceptors 添加一元拦截器
func WithUnaryInterceptors(interceptors ...grpc.UnaryClientInterceptor) Option {
	return func(c *Client) {
		c.unaryInterceptors = append(c.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors 添加流式拦截器
func WithStreamInterceptors(interceptors ...grpc.StreamClientInterceptor) Option {
	return func(c *Client) {
		c.streamInterceptors = append(c.streamInterceptors, interceptors...)
	}
}
