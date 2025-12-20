package server

import (
	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc"
)

// Option Server 配置选项
type Option func(*Server)

// WithLogger 设置自定义 logger
func WithLogger(l *logger.Logger) Option {
	return func(s *Server) {
		s.logger = l
	}
}

// WithServerOptions 添加 gRPC ServerOption
func WithServerOptions(opts ...grpc.ServerOption) Option {
	return func(s *Server) {
		s.grpcOpts = append(s.grpcOpts, opts...)
	}
}

// WithUnaryInterceptors 添加一元拦截器
func WithUnaryInterceptors(interceptors ...grpc.UnaryServerInterceptor) Option {
	return func(s *Server) {
		s.unaryInterceptors = append(s.unaryInterceptors, interceptors...)
	}
}

// WithStreamInterceptors 添加流式拦截器
func WithStreamInterceptors(interceptors ...grpc.StreamServerInterceptor) Option {
	return func(s *Server) {
		s.streamInterceptors = append(s.streamInterceptors, interceptors...)
	}
}

// WithServiceRegistrar 设置服务注册器（用于自定义服务注册逻辑）
func WithServiceRegistrar(registrar ServiceRegistrar) Option {
	return func(s *Server) {
		s.serviceRegistrar = registrar
	}
}
