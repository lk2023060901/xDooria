package etcd

import (
	"context"
	"fmt"
	"sync"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"go.uber.org/zap"
	"google.golang.org/grpc/resolver"
)

const (
	// Scheme etcd resolver scheme
	Scheme = "etcd"
)

// ResolverBuilder 实现 gRPC resolver.Builder 接口
type ResolverBuilder struct {
	config *Config
	logger *logger.Logger
}

// NewResolverBuilder 创建 gRPC Resolver Builder
func NewResolverBuilder(cfg *Config) (*ResolverBuilder, error) {
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := newCfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &ResolverBuilder{
		config: newCfg,
		logger: logger.Default().Named("grpc.resolver.etcd"),
	}, nil
}

// Build 创建 gRPC Resolver 实例
// target 格式: etcd:///service-name
func (b *ResolverBuilder) Build(target resolver.Target, cc resolver.ClientConn, opts resolver.BuildOptions) (resolver.Resolver, error) {
	r := &grpcResolver{
		target:  target,
		cc:      cc,
		config:  b.config,
		logger:  b.logger,
		closeCh: make(chan struct{}),
		readyCh: make(chan struct{}),
	}

	// 异步创建 Resolver 并连接 etcd（避免阻塞 Build 方法）
	conc.Go(func() (struct{}, error) {
		res, err := NewResolver(b.config)
		if err != nil {
			r.logger.Error("failed to create etcd resolver", zap.Error(err))
			r.cc.ReportError(err)
			close(r.readyCh) // 即使失败也关闭 ready channel
			return struct{}{}, err
		}

		r.mu.Lock()
		r.resolver = res
		r.mu.Unlock()

		// 连接成功后，立即执行一次解析并等待完成
		r.resolveAndNotify()

		// 启动监听服务变化
		r.watch()
		return struct{}{}, nil
	})

	return r, nil
}

// Scheme 返回 scheme 名称
func (b *ResolverBuilder) Scheme() string {
	return Scheme
}

// grpcResolver 实现 gRPC resolver.Resolver 接口
type grpcResolver struct {
	target    resolver.Target
	cc        resolver.ClientConn
	config    *Config
	resolver  *Resolver
	logger    *logger.Logger
	closeCh   chan struct{}
	readyCh   chan struct{} // 用于通知 resolver 已就绪
	closeOnce sync.Once
	mu        sync.RWMutex
}

// resolveAndNotify 执行解析并在完成后通知就绪
func (r *grpcResolver) resolveAndNotify() {
	r.mu.RLock()
	res := r.resolver
	r.mu.RUnlock()

	if res == nil {
		close(r.readyCh)
		return
	}

	ctx := context.Background()
	services, err := res.Resolve(ctx, r.target.Endpoint())
	if err != nil {
		r.logger.Error("failed to resolve services",
			zap.String("service", r.target.Endpoint()),
			zap.Error(err),
		)
		r.cc.ReportError(err)
		close(r.readyCh)
		return
	}

	addrs := make([]resolver.Address, 0, len(services))
	for _, svc := range services {
		addrs = append(addrs, resolver.Address{
			Addr:       svc.Address,
			ServerName: svc.ServiceName,
			Metadata:   svc.Metadata,
		})
	}

	if err := r.cc.UpdateState(resolver.State{Addresses: addrs}); err != nil {
		r.logger.Error("failed to update state",
			zap.String("service", r.target.Endpoint()),
			zap.Error(err),
		)
		close(r.readyCh)
		return
	}

	r.logger.Debug("resolved services",
		zap.String("service", r.target.Endpoint()),
		zap.Int("count", len(addrs)),
	)

	// UpdateState 完成后才标记为就绪
	close(r.readyCh)
}

// ResolveNow 触发立即解析
func (r *grpcResolver) ResolveNow(opts resolver.ResolveNowOptions) {
	r.mu.RLock()
	res := r.resolver
	r.mu.RUnlock()

	// 如果 resolver 还未初始化，跳过（异步初始化中）
	if res == nil {
		return
	}

	ctx := context.Background()
	services, err := res.Resolve(ctx, r.target.Endpoint())
	if err != nil {
		r.logger.Error("failed to resolve services",
			zap.String("service", r.target.Endpoint()),
			zap.Error(err),
		)
		r.cc.ReportError(err)
		return
	}

	addrs := make([]resolver.Address, 0, len(services))
	for _, svc := range services {
		addrs = append(addrs, resolver.Address{
			Addr:       svc.Address,
			ServerName: svc.ServiceName,
			Metadata:   svc.Metadata,
		})
	}

	if err := r.cc.UpdateState(resolver.State{Addresses: addrs}); err != nil {
		r.logger.Error("failed to update state",
			zap.String("service", r.target.Endpoint()),
			zap.Error(err),
		)
	}

	r.logger.Debug("resolved services",
		zap.String("service", r.target.Endpoint()),
		zap.Int("count", len(addrs)),
	)
}

// Close 关闭 resolver
func (r *grpcResolver) Close() {
	r.closeOnce.Do(func() {
		close(r.closeCh)
	})
}

// watch 监听服务变化
func (r *grpcResolver) watch() {
	r.mu.RLock()
	res := r.resolver
	r.mu.RUnlock()

	// resolver 应该已经初始化（由调用者保证）
	if res == nil {
		r.logger.Error("resolver not initialized in watch")
		return
	}

	ctx := context.Background()
	serviceName := r.target.Endpoint()

	watchCh, err := res.Watch(ctx, serviceName)
	if err != nil {
		r.logger.Error("failed to watch services",
			zap.String("service", serviceName),
			zap.Error(err),
		)
		return
	}

	for {
		select {
		case <-r.closeCh:
			return
		case services, ok := <-watchCh:
			if !ok {
				r.logger.Warn("watch channel closed",
					zap.String("service", serviceName),
				)
				return
			}

			addrs := make([]resolver.Address, 0, len(services))
			for _, svc := range services {
				addrs = append(addrs, resolver.Address{
					Addr:       svc.Address,
					ServerName: svc.ServiceName,
					Metadata:   svc.Metadata,
				})
			}

			if err := r.cc.UpdateState(resolver.State{Addresses: addrs}); err != nil {
				r.logger.Error("failed to update state on watch",
					zap.String("service", serviceName),
					zap.Error(err),
				)
			}

			r.logger.Debug("service list updated from watch",
				zap.String("service", serviceName),
				zap.Int("count", len(addrs)),
			)
		}
	}
}

// WaitForReady 等待 resolver 就绪（已连接 etcd 并完成首次解析）
// 超时返回 false，成功返回 true
func (r *grpcResolver) WaitForReady(ctx context.Context) bool {
	select {
	case <-r.readyCh:
		return true
	case <-ctx.Done():
		return false
	}
}

// RegisterBuilder 注册 gRPC Resolver Builder 到全局
func RegisterBuilder(cfg *Config) error {
	builder, err := NewResolverBuilder(cfg)
	if err != nil {
		return fmt.Errorf("failed to create resolver builder: %w", err)
	}

	resolver.Register(builder)
	logger.Default().Info("etcd resolver builder registered", zap.String("scheme", Scheme))
	return nil
}
