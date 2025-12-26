//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/lk2023060901/xdooria/app/login/internal/dao"
	"github.com/lk2023060901/xdooria/app/login/internal/manager"
	"github.com/lk2023060901/xdooria/app/login/internal/metrics"
	"github.com/lk2023060901/xdooria/app/login/internal/service"
	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/framer"
	"github.com/lk2023060901/xdooria/pkg/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/prometheus"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
	pb "github.com/lk2023060901/xdooria-proto-common"
)

func InitApp(cfg *Config, l logger.Logger) (app.Application, func(), error) {
	panic(wire.Build(
		// 1. 基础框架 (BaseApp)
		app.ProviderSet,

		// 2. 通信与路由 (Router, Bridge)
		router.ProviderSet,
		wire.Value([]router.Option{}), // 默认选项

		// 3. gRPC Server 封装 (pkg/grpc/server)
		wire.FieldsOf(new(*Config), "Server"),
		server.New,
		wire.Value([]server.Option{}), // 默认选项

		// 4. Framer (安全信封)
		wire.FieldsOf(new(*Config), "Framer"),
		framer.New,

		// 5. 业务组件 (AuthManager)
		auth.ProviderSet,

		// 6. 数据层 (Luban Config)
		dao.NewConfigDAO,
		wire.FieldsOf(new(*dao.ConfigDAO), "Tables"),

		// 7. 逻辑层 (Authenticator)
		manager.NewLocalAuthenticator,

		// 8. 安全层 (JWT)
		wire.FieldsOf(new(*Config), "JWT"),
		security.NewJWTManager,

		// 9. 接口层 (Service)
		service.NewLoginService,

		// 10. Prometheus 客户端
		providePrometheusConfig,
		prometheus.New,

		// 11. 服务注册（etcd）
		provideRegistryConfig,
		etcd.NewRegistrar,
		wire.Bind(new(registry.Registrar), new(*etcd.Registrar)),

		// 12. 指标收集
		provideMetricsConfig,
		metrics.New,
		provideMetricsReporter,

		// 13. 组装与应用配置
		provideAppOptions,
		provideAppComponents,
		app.InitApp,
	))
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
	m *metrics.LoginMetrics,
	registrar registry.Registrar,
	l logger.Logger,
) (*metrics.Reporter, error) {
	cfg := m.GetConfig()
	return metrics.NewReporter(&cfg.Reporter, m, registrar, l)
}

func provideAppOptions(cfg *Config, l logger.Logger) []app.Option {
	return []app.Option{
		app.WithName(app.AppName),
		app.WithLogger(l),
		app.WithLogConfig(&cfg.Log),
		app.WithNamedLoggers(cfg.Loggers),
	}
}

func provideAppComponents(
	baseApp *app.BaseApp,
	srv *server.Server,
	bridge *router.Bridge,
	loginSvc *service.LoginService,
	authMgr *auth.Manager,
	localAuth *manager.LocalAuthenticator,
	r router.Router,
	promClient *prometheus.Client,
	loginMetrics *metrics.LoginMetrics,
	reporter *metrics.Reporter,
	registrar *etcd.Registrar,
	cfg *Config,
	opts []app.Option,
) app.AppComponents {
	// 注册认证器插件
	authMgr.Register(localAuth)

	// 初始化路由映射 (OpCode -> Handler)
	loginSvc.Init(r)

	// 注册 Login 指标到 Prometheus
	_ = loginMetrics.Register(promClient.Registry())

	// 启动指标上报器
	reporter.Start()

	// 将 Bridge (实现的 CommonService) 注册到 gRPC Server
	pb.RegisterCommonServiceServer(srv.GetGRPCServer(), bridge)

	// 创建服务注册启动器
	serviceStarter := &serviceRegistrar{
		registrar:   registrar,
		serviceName: cfg.Registry.ServiceName,
		serviceAddr: cfg.Registry.ServiceAddr,
		logger:      baseApp.AppLogger(),
	}

	return app.AppComponents{
		Servers: []app.Server{
			srv, // gRPC Server 实现了 app.Server
			serviceStarter,
		},
		Closers: []app.Closer{
			&metricsCloser{reporter: reporter},
			promClient,
			&registrarCloser{registrar: registrar},
		},
	}
}

// metricsCloser 指标上报器关闭器
type metricsCloser struct {
	reporter *metrics.Reporter
}

func (c *metricsCloser) Close() error {
	c.reporter.Stop()
	return nil
}

// registrarCloser 服务注册器关闭器
type registrarCloser struct {
	registrar *etcd.Registrar
}

func (c *registrarCloser) Close() error {
	return c.registrar.Deregister(context.Background())
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
