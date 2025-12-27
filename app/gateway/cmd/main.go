package main

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/app/gateway/internal/handler"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/network/tcp"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
)

// Config Gateway 服务配置
type Config struct {
	Log     logger.Config             `mapstructure:"log"`
	Loggers map[string]*logger.Config `mapstructure:"loggers"`

	// TCP 配置
	TCP tcp.ServerConfig `mapstructure:"tcp"`

	// Session 配置
	Session session.Config `mapstructure:"session"`

	// Framer 配置
	Framer framer.Config `mapstructure:"framer"`

	// Registry 配置
	Registry etcd.Config `mapstructure:"registry"`

	// JWT 配置
	JWT security.JWTConfig `mapstructure:"jwt"`
}

func main() {
	var cfg Config

	// 1. 加载配置
	if err := app.LoadConfig(&cfg); err != nil {
		fmt.Printf("failed to load config: %v\n", err)
		return
	}

	// 2. 初始化日志
	l, err := logger.New(&cfg.Log)
	if err != nil {
		panic(err)
	}

	// 3. 初始化 Framer
	fr, err := framer.New(&cfg.Framer)
	if err != nil {
		l.Error("failed to create framer", "error", err)
		return
	}

	// 4. 初始化 JWT 管理器
	jwtMgr, err := security.NewJWTManager(&cfg.JWT)
	if err != nil {
		l.Error("failed to create jwt manager", "error", err)
		return
	}

	// 5. 初始化 Router 和 Processor
	r := router.New()
	processor := router.NewProcessor(r)

	// 6. 初始化业务 Handler
	gwHandler := handler.NewGatewayHandler(l, jwtMgr, processor)

	// 7. 初始化 Session 配置（注入 Framer）
	sessCfg := cfg.Session
	sessCfg.Framer = fr

	// 8. 初始化 Session Server
	sessServer := session.NewServer(&session.ServerConfig{
		Session: &sessCfg,
		Handler: gwHandler,
	})

	// 9. 初始化 TCP Acceptor (并包装托管逻辑)
	sessServer.Config().Acceptor = sessServer.ManagedAcceptor(func(h session.SessionHandler) session.Acceptor {
		return tcp.NewAcceptor(&cfg.TCP, &sessCfg, h)
	})

	// 10. 创建服务注册器
	registrar, err := etcd.NewRegistrar(&cfg.Registry)
	if err != nil {
		l.Error("failed to create registrar", "error", err)
		return
	}

	// 11. 创建应用并注册服务
	application := app.NewBaseApp(
		app.WithName("gateway"),
		app.WithLogger(l),
	)

	// 将 sessServer 注册到应用中
	application.AppendServer(sessServer)

	// 注册服务到 etcd 的启动器
	application.AppendServer(&serviceRegistrar{
		registrar: registrar,
		info: &registry.ServiceInfo{
			ServiceName: "gateway",
			Address:     cfg.TCP.Addr,
			Metadata:    make(map[string]string),
		},
	})

	application.AppendCloser(&registrarCloser{registrar: registrar})

	// 12. 运行
	if err := application.Run(); err != nil {
		l.Error("gateway exited with error", "error", err)
	}
}

type registrarCloser struct {
	registrar *etcd.Registrar
}

func (c *registrarCloser) Close() error {
	return c.registrar.Deregister(context.Background())
}

type serviceRegistrar struct {
	registrar registry.Registrar
	info      *registry.ServiceInfo
}

func (s *serviceRegistrar) Start() error {
	return s.registrar.Register(context.Background(), s.info)
}

func (s *serviceRegistrar) Stop() error {
	return s.registrar.Deregister(context.Background())
}

