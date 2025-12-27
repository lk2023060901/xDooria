//go:build wireinject
// +build wireinject

package main

import (
	"context"

	"github.com/google/wire"
	"github.com/lk2023060901/xdooria/app/login/internal/dao"
	"github.com/lk2023060901/xdooria/app/login/internal/handler"
	"github.com/lk2023060901/xdooria/app/login/internal/manager"
	"github.com/lk2023060901/xdooria/app/login/internal/metrics"
	"github.com/lk2023060901/xdooria/app/login/internal/service"
	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/balancer"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/network/tcp"
	"github.com/lk2023060901/xdooria/pkg/prometheus"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
)

func InitApp(cfg *Config, l logger.Logger) (app.Application, func(), error) {
	panic(wire.Build(
		// 1. 基础框架 (BaseApp)
		app.ProviderSet,

		// 2. 路由
		router.ProviderSet,

		// 3. Framer (安全信封)
		wire.FieldsOf(new(*Config), "Framer"),
		framer.New,

		// 4. Session 配置
		provideSessionConfig,

		// 5. TCP 配置
		wire.FieldsOf(new(*Config), "TCP"),

		// 6. 业务 Handler
		handler.NewLoginHandler,
		wire.Bind(new(session.SessionHandler), new(*handler.LoginHandler)),

		// 7. Session Server
		provideSessionServer,

		// 8. 业务组件 (AuthManager)
		auth.ProviderSet,

		// 9. 数据层 (Luban Config)
		dao.NewConfigDAO,
		wire.FieldsOf(new(*dao.ConfigDAO), "Tables"),

		// 10. 逻辑层 (Authenticator)
		manager.NewLocalAuthenticator,

		// 11. 安全层 (JWT)
		wire.FieldsOf(new(*Config), "JWT"),
		security.NewJWTManager,

		// 12. 负载均衡器
		provideBalancer,

		// 13. 接口层 (Service)
		service.NewLoginService,

		// 13. Prometheus 客户端
		providePrometheusConfig,
		prometheus.New,

		// 14. 服务注册与发现（etcd）
		provideRegistryConfig,
		etcd.NewRegistrar,
		wire.Bind(new(registry.Registrar), new(*etcd.Registrar)),
		etcd.NewResolver,
		wire.Bind(new(registry.Resolver), new(*etcd.Resolver)),

		// 15. 指标收集
		provideMetricsConfig,
		metrics.New,
		provideMetricsReporter,

		// 16. 组装与应用配置
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

// provideSessionConfig 提供 Session 配置（注入 Framer）
func provideSessionConfig(cfg *Config, fr framer.Framer) *session.Config {
	sessCfg := cfg.Session
	sessCfg.Framer = fr
	return &sessCfg
}

// provideSessionServer 提供 Session Server
func provideSessionServer(
	tcpCfg *tcp.ServerConfig,
	sessCfg *session.Config,
	h session.SessionHandler,
) *session.Server {
	sessServer := session.NewServer(&session.ServerConfig{
		Session: sessCfg,
		Handler: h,
	})
	// 创建 TCP Acceptor 并包装托管逻辑
	sessServer.Config().Acceptor = sessServer.ManagedAcceptor(func(handler session.SessionHandler) session.Acceptor {
		return tcp.NewAcceptor(tcpCfg, sessCfg, handler)
	})
	return sessServer
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
	sessServer *session.Server,
	loginSvc *service.LoginService,
	authMgr *auth.Manager,
	localAuth *manager.LocalAuthenticator,
	r router.Router,
	promClient *prometheus.Client,
	loginMetrics *metrics.LoginMetrics,
	reporter *metrics.Reporter,
	registrar *etcd.Registrar,
	resolver *etcd.Resolver,
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

	// 创建服务注册启动器
	serviceStarter := &serviceRegistrar{
		registrar:   registrar,
		serviceName: cfg.Registry.ServiceName,
		serviceAddr: cfg.TCP.Addr,
		logger:      baseApp.AppLogger(),
	}

	return app.AppComponents{
		Servers: []app.Server{
			sessServer, // Session Server 实现了 app.Server
			serviceStarter,
		},
		Closers: []app.Closer{
			&metricsCloser{reporter: reporter},
			promClient,
			&registrarCloser{registrar: registrar},
			resolver,
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

// provideBalancer 提供负载均衡器
func provideBalancer() balancer.Balancer {
	return balancer.New(balancer.RoundRobinName)
}
