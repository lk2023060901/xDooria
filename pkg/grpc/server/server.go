package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

// Server gRPC Server 封装
type Server struct {
	config *Config
	server *grpc.Server
	logger *logger.Logger

	// 选项
	grpcOpts           []grpc.ServerOption
	unaryInterceptors  []grpc.UnaryServerInterceptor
	streamInterceptors []grpc.StreamServerInterceptor

	// 服务注册
	registrar registry.Registrar

	// 健康检查
	healthServer *health.Server

	// 异步任务
	serveFuture *conc.Future[struct{}]

	// 状态管理
	mu       sync.RWMutex
	started  bool
	listener net.Listener
}

// New 创建 gRPC Server
func New(cfg *Config, opts ...Option) (*Server, error) {
	// 合并配置
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	s := &Server{
		config:             newCfg,
		logger:             logger.Default().Named("grpc.server"),
		grpcOpts:           make([]grpc.ServerOption, 0),
		unaryInterceptors:  make([]grpc.UnaryServerInterceptor, 0),
		streamInterceptors: make([]grpc.StreamServerInterceptor, 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(s)
	}

	// 构建 gRPC ServerOptions 并创建 Server
	serverOpts := s.buildServerOptions()
	s.server = grpc.NewServer(serverOpts...)

	// 注册健康检查服务
	if newCfg.EnableHealthCheck {
		s.healthServer = health.NewServer()
		grpc_health_v1.RegisterHealthServer(s.server, s.healthServer)
	}

	// 注册反射服务（用于调试，如 grpcurl）
	if newCfg.EnableReflection {
		reflection.Register(s.server)
	}

	return s, nil
}

// Start 启动 Server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return ErrServerAlreadyStarted
	}

	// 创建监听器
	listener, err := net.Listen(s.config.Network, s.config.Address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s://%s: %w",
			s.config.Network, s.config.Address, err)
	}
	s.listener = listener

	// 设置健康检查为 SERVING
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	}

	// 服务注册到 registry（如 etcd）
	if err := s.registerService(); err != nil {
		listener.Close()
		return fmt.Errorf("failed to register service: %w", err)
	}

	s.started = true

	s.logger.Info("gRPC server starting",
		zap.String("name", s.config.Name),
		zap.String("network", s.config.Network),
		zap.String("address", listener.Addr().String()),
	)

	// 使用 conc.Go 启动 Server（而非原始 goroutine）
	s.serveFuture = conc.Go(func() (struct{}, error) {
		err := s.server.Serve(listener)
		return struct{}{}, err
	})

	return nil
}

// Stop 立即停止 Server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return ErrServerNotStarted
	}

	s.logger.Info("stopping gRPC server")

	// 设置健康检查为 NOT_SERVING
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	// 取消服务注册
	if err := s.deregisterService(); err != nil {
		s.logger.Warn("failed to deregister service", zap.Error(err))
	}

	// 立即停止
	s.server.Stop()

	// 检查 Serve 是否有错误
	if s.serveFuture != nil {
		if err := s.serveFuture.Err(); err != nil {
			s.logger.Warn("serve ended with error", zap.Error(err))
		}
	}

	s.started = false
	return nil
}

// GracefulStop 优雅停止 Server
func (s *Server) GracefulStop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return ErrServerNotStarted
	}

	s.logger.Info("gracefully stopping gRPC server")

	// 设置健康检查为 NOT_SERVING
	if s.healthServer != nil {
		s.healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_NOT_SERVING)
	}

	// 取消服务注册
	if err := s.deregisterService(); err != nil {
		s.logger.Warn("failed to deregister service", zap.Error(err))
	}

	// 使用 conc.Go 优雅停止（带超时控制）
	stopFuture := conc.Go(func() (struct{}, error) {
		s.server.GracefulStop()
		return struct{}{}, nil
	})

	select {
	case <-stopFuture.Inner():
		s.logger.Info("gRPC server stopped gracefully")
	case <-time.After(s.config.GracefulStopTimeout):
		s.logger.Warn("graceful stop timeout, forcing stop")
		s.server.Stop()
	}

	// 检查 Serve 是否有错误
	if s.serveFuture != nil {
		if err := s.serveFuture.Err(); err != nil {
			s.logger.Warn("serve ended with error", zap.Error(err))
		}
	}

	s.started = false
	return nil
}

// RegisterService 注册 gRPC 服务
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
}

// GetGRPCServer 获取底层 gRPC Server（用于高级用法）
func (s *Server) GetGRPCServer() *grpc.Server {
	return s.server
}

// GetListener 获取监听器
func (s *Server) GetListener() net.Listener {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listener
}

// buildServerOptions 构建 gRPC ServerOptions
func (s *Server) buildServerOptions() []grpc.ServerOption {
	opts := make([]grpc.ServerOption, 0)

	// 消息大小限制
	opts = append(opts,
		grpc.MaxRecvMsgSize(s.config.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(s.config.MaxSendMsgSize),
	)

	// KeepAlive 配置
	opts = append(opts,
		grpc.KeepaliveParams(s.config.KeepAliveParams),
		grpc.KeepaliveEnforcementPolicy(s.config.KeepAliveEnforcement),
	)

	// 拦截器链
	if len(s.unaryInterceptors) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(s.unaryInterceptors...))
	}
	if len(s.streamInterceptors) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(s.streamInterceptors...))
	}

	// 用户自定义选项
	opts = append(opts, s.grpcOpts...)

	return opts
}

// registerService 注册服务到 registry
func (s *Server) registerService() error {
	if s.registrar == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serviceInfo := &registry.ServiceInfo{
		ServiceName: s.config.Name,
		Address:     s.listener.Addr().String(),
		Metadata:    make(map[string]string),
	}

	// 如果配置了 ServiceRegistry，使用其 Metadata
	if s.config.ServiceRegistry != nil {
		serviceInfo.Metadata = s.config.ServiceRegistry.Metadata
	}

	return s.registrar.Register(ctx, serviceInfo)
}

// deregisterService 从 registry 取消注册
func (s *Server) deregisterService() error {
	if s.registrar == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.registrar.Deregister(ctx)
}
