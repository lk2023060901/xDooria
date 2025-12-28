//go:build wireinject
// +build wireinject

package main

import (
	gamepb "github.com/lk2023060901/xdooria-proto-internal/game"
	"context"

	"github.com/google/wire"
	"github.com/lk2023060901/xdooria/app/game/internal/dao"
	"github.com/lk2023060901/xdooria/app/game/internal/handler"
	"github.com/lk2023060901/xdooria/app/game/internal/manager"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/service"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/database/postgres"
	"github.com/lk2023060901/xdooria/pkg/database/redis"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/prometheus"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/router"
)

func InitApp(cfg *Config, l logger.Logger) (app.Application, func(), error) {
	panic(wire.Build(
		// 1. 基础框架 (BaseApp)
		app.ProviderSet,

		// 2. PostgreSQL 配置和客户端
		providePostgresConfig,
		postgres.New,

		// 3. Redis 配置和客户端
		provideRedisConfig,
		redis.NewClient,

		// 4. 数据层 (DAO)
		dao.NewRoleDAO,
		dao.NewCacheDAO,

		// 5. 指标收集
		provideMetricsConfig,
		metrics.New,
		provideMetricsReporter,

		// 6. 管理层 (Manager)
		manager.NewRoleManager,
		manager.NewSessionManager,
		manager.NewSceneManager,

		// 7. 服务层 (Service)
		service.NewRoleService,
		service.NewSceneService,
		service.NewMessageService,

		// 8. 接口层 (Handler)
		handler.NewGameHandler,

		// 9. gRPC Server 配置和选项
		provideGRPCServerConfig,
		provideGRPCServerOptions,
		server.New,

		// Router (消息路由器)
		provideRouter,

		// 10. Prometheus 客户端
		providePrometheusConfig,
		prometheus.New,

		// 11. 服务注册与发现（etcd）
		provideRegistryConfig,
		etcd.NewRegistrar,
		wire.Bind(new(registry.Registrar), new(*etcd.Registrar)),
		etcd.NewResolver,

		// 12. 组装与应用配置
		provideAppOptions,
		provideAppComponents,
		app.InitApp,
	))
}

// providePostgresConfig 提供 PostgreSQL 配置
func providePostgresConfig(cfg *Config) *postgres.Config {
	return &cfg.Database
}

// provideRedisConfig 提供 Redis 配置
func provideRedisConfig(cfg *Config) *redis.Config {
	return &cfg.Redis
}

// provideGameConfigConfig 提供游戏配置表加载配置
func provideGameConfigConfig(cfg *Config) *dao.GameConfigConfig {
	return &dao.GameConfigConfig{
		RequiredTables: cfg.GameConfig.RequiredTables,
		OptionalTables: cfg.GameConfig.OptionalTables,
	}
}

// provideGRPCServerConfig 提供 gRPC Server 配置
func provideGRPCServerConfig(cfg *Config) *server.Config {
	return &cfg.GRPC
}

// provideGRPCServerOptions 提供 gRPC Server 选项
func provideGRPCServerOptions() []server.Option {
	return nil // 暂时不需要额外选项
}

// provideRouter 提供消息路由器
func provideRouter() router.Router {
	return router.New()
}

// providePrometheusConfig 提供 Prometheus 配置
func providePrometheusConfig(cfg *Config) *prometheus.Config {
	return &cfg.Prometheus
}

// provideRegistryConfig 提供服务注册配置
func provideRegistryConfig(cfg *Config) *etcd.Config {
	return &cfg.Registry
}

// provideMetricsConfig 提供指标配置
func provideMetricsConfig(cfg *Config) *metrics.Config {
	return &cfg.Metrics
}

// provideMetricsReporter 提供指标上报器
func provideMetricsReporter(
	m *metrics.GameMetrics,
	registrar registry.Registrar,
	l logger.Logger,
) (*metrics.Reporter, error) {
	cfg := m.GetConfig()
	return metrics.NewReporter(&cfg.Reporter, m, registrar, l)
}

// provideAppOptions 提供应用选项
func provideAppOptions(cfg *Config, l logger.Logger) []app.Option {
	return []app.Option{
		app.WithName(app.AppName),
		app.WithLogger(l),
		app.WithLogConfig(&cfg.Log),
		app.WithNamedLoggers(cfg.Loggers),
	}
}

// provideAppComponents 提供应用组件
func provideAppComponents(
	baseApp *app.BaseApp,
	grpcServer *server.Server,
	gameHandler *handler.GameHandler,
	promClient *prometheus.Client,
	gameMetrics *metrics.GameMetrics,
	reporter *metrics.Reporter,
	registrar *etcd.Registrar,
	resolver *etcd.Resolver,
	postgresClient *postgres.Client,
	redisClient *redis.Client,
	cfg *Config,
	opts []app.Option,
) app.AppComponents {
	// 注册 gRPC 服务
	gamepb.RegisterGameServiceServer(grpcServer.GetGRPCServer(), gameHandler)

	// 注册 Game 指标到 Prometheus
	_ = gameMetrics.Register(promClient.Registry())

	// 启动指标上报器
	reporter.Start()

	// 创建服务注册启动器
	serviceStarter := &serviceRegistrar{
		registrar:   registrar,
		serviceName: cfg.Registry.ServiceName,
		serviceAddr: cfg.Registry.ServiceAddr,
		logger:      baseApp.AppLogger(),
	}

	return app.AppComponents{
		Servers: []app.Server{
			grpcServer, // gRPC Server 实现了 app.Server
			serviceStarter,
		},
		Closers: []app.Closer{
			&metricsCloser{
				reporter:    reporter,
				gameMetrics: gameMetrics,
			},
			promClient,
			&registrarCloser{registrar: registrar},
			resolver,
			&postgresCloser{client: postgresClient},
			redisClient,
		},
	}
}

// metricsCloser 指标上报器关闭器
type metricsCloser struct {
	reporter    *metrics.Reporter
	gameMetrics *metrics.GameMetrics
}

func (c *metricsCloser) Close() error {
	c.reporter.Stop()
	c.gameMetrics.Stop()
	return nil
}

// registrarCloser 服务注册器关闭器
type registrarCloser struct {
	registrar *etcd.Registrar
}

func (c *registrarCloser) Close() error {
	return c.registrar.Deregister(context.Background())
}

// postgresCloser PostgreSQL 客户端关闭器
type postgresCloser struct {
	client *postgres.Client
}

func (c *postgresCloser) Close() error {
	c.client.Close()
	return nil
}

// serviceRegistrar 服务注册启动器，实现 app.Server 接口
type serviceRegistrar struct {
	registrar   *etcd.Registrar
	serviceName string
	serviceAddr string
	logger      logger.Logger
}

func (s *serviceRegistrar) Start() error {
	info := &registry.ServiceInfo{
		ServiceName: s.serviceName,
		Address:     s.serviceAddr,
		Metadata:    make(map[string]string),
	}
	if err := s.registrar.Register(context.Background(), info); err != nil {
		s.logger.Error("failed to register service", "error", err)
		return err
	}
	s.logger.Info("service registered to etcd", "name", s.serviceName, "addr", s.serviceAddr)
	return nil
}

func (s *serviceRegistrar) Stop() error {
	return nil // Deregister 由 registrarCloser 处理
}
