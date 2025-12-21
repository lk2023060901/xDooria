package etcd

import (
	"context"
	"fmt"
	"net"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// ServerOption gRPC Server 配置选项
type ServerOption func(*serverOptions)

type serverOptions struct {
	serviceName string
	address     string
	metadata    map[string]string
	registrar   *Registrar
}

// WithServiceName 设置服务名称
func WithServiceName(name string) ServerOption {
	return func(o *serverOptions) {
		o.serviceName = name
	}
}

// WithAddress 设置服务地址
func WithAddress(addr string) ServerOption {
	return func(o *serverOptions) {
		o.address = addr
	}
}

// WithMetadata 设置服务元数据
func WithMetadata(metadata map[string]string) ServerOption {
	return func(o *serverOptions) {
		o.metadata = metadata
	}
}

// WithRegistrar 设置注册器
func WithRegistrar(registrar *Registrar) ServerOption {
	return func(o *serverOptions) {
		o.registrar = registrar
	}
}

// RegisterServer 注册 gRPC Server 到 etcd
func RegisterServer(ctx context.Context, server *grpc.Server, opts ...ServerOption) error {
	options := &serverOptions{
		metadata: make(map[string]string),
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.serviceName == "" {
		return fmt.Errorf("service name is required")
	}

	if options.address == "" {
		return fmt.Errorf("address is required")
	}

	if options.registrar == nil {
		return fmt.Errorf("registrar is required")
	}

	info := &registry.ServiceInfo{
		ServiceName: options.serviceName,
		Address:     options.address,
		Metadata:    options.metadata,
	}

	if err := options.registrar.Register(ctx, info); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	logger.Default().Info("gRPC server registered",
		zap.String("service", options.serviceName),
		zap.String("address", options.address),
	)

	return nil
}

// ServeWithRegistry 启动 gRPC Server 并自动注册到 etcd
// 当 server 停止时，自动从 etcd 注销
func ServeWithRegistry(
	ctx context.Context,
	server *grpc.Server,
	listener net.Listener,
	opts ...ServerOption,
) error {
	options := &serverOptions{
		metadata: make(map[string]string),
	}

	for _, opt := range opts {
		opt(options)
	}

	// 注册服务
	if err := RegisterServer(ctx, server, opts...); err != nil {
		return err
	}

	// 启动 gRPC Server
	serveFuture := conc.Go(func() (error, error) {
		logger.Default().Info("gRPC server starting",
			zap.String("address", listener.Addr().String()),
		)
		err := server.Serve(listener)
		return err, err
	})

	// 等待 context 取消或 server 错误
	select {
	case <-ctx.Done():
		logger.Default().Info("shutting down gRPC server")
		server.GracefulStop()

		// 注销服务
		if options.registrar != nil {
			if err := options.registrar.Deregister(context.Background()); err != nil {
				logger.Default().Error("failed to deregister service", zap.Error(err))
			}
		}

		return ctx.Err()
	case <-serveFuture.Inner():
		err := serveFuture.Err()
		// 注销服务
		if options.registrar != nil {
			if deregErr := options.registrar.Deregister(context.Background()); deregErr != nil {
				logger.Default().Error("failed to deregister service", zap.Error(deregErr))
			}
		}
		return err
	}
}
