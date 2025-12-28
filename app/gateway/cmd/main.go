package main

import (
	"context"
	"fmt"

	gamepb "github.com/lk2023060901/xdooria-proto-internal/game"
	"github.com/lk2023060901/xdooria/app/gateway/internal/handler"
	"github.com/lk2023060901/xdooria/app/gateway/internal/role"
	gwsession "github.com/lk2023060901/xdooria/app/gateway/internal/session"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/database/postgres"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	grpcclient "github.com/lk2023060901/xdooria/pkg/network/grpc/client"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/network/tcp"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	// Database 配置
	Database postgres.Config `mapstructure:"database"`
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

	// 5. 初始化 PostgreSQL 客户端
	pgClient, err := postgres.New(&cfg.Database)
	if err != nil {
		l.Error("failed to create postgres client", "error", err)
		return
	}
	defer pgClient.Close()

	// 6. 初始化 Role Provider
	roleProvider := role.NewProvider(l, pgClient)

	// 7. 创建 etcd resolver 用于服务发现
	resolver, err := etcd.NewResolver(&cfg.Registry)
	if err != nil {
		l.Error("failed to create resolver", "error", err)
		return
	}
	defer resolver.Close()

	// 8. 解析 Game 服务地址
	gameServices, err := resolver.Resolve(context.Background(), "game")
	if err != nil || len(gameServices) == 0 {
		l.Error("failed to resolve game service", "error", err)
		return
	}
	gameAddr := gameServices[0].Address
	l.Info("resolved game service", "address", gameAddr)

	// 9. 创建 Game gRPC 客户端
	gameClientCfg := &grpcclient.Config{
		Target:      gameAddr,
		DialTimeout: 5,
	}
	gameConn, err := grpcclient.New(gameClientCfg, grpcclient.WithDialOptions(grpc.WithTransportCredentials(insecure.NewCredentials())))
	if err != nil {
		l.Error("failed to create game grpc client", "error", err)
		return
	}
	if err := gameConn.Dial(); err != nil {
		l.Error("failed to dial game service", "error", err)
		return
	}
	conn, _ := gameConn.GetConn()
	gameClient := gamepb.NewGameServiceClient(conn)

	// 10. 初始化 Router 和 Processor
	r := router.New()
	processor := router.NewProcessor(r)

	// 11. 初始化 Session Manager
	sessMgr := gwsession.NewManager()

	// 12. 初始化业务 Handler（传入 gameClient）
	gwHandler := handler.NewGatewayHandlerWithGame(l, jwtMgr, processor, sessMgr, roleProvider, gameClient)

	// 13. 初始化 Session 配置（注入 Framer）
	sessCfg := cfg.Session
	sessCfg.Framer = fr

	// 14. 初始化 Session Server
	sessServer := session.NewServer(&session.ServerConfig{
		Session: &sessCfg,
		Handler: gwHandler,
	})

	// 15. 初始化 TCP Acceptor (并包装托管逻辑)
	sessServer.Config().Acceptor = sessServer.ManagedAcceptor(func(h session.SessionHandler) session.Acceptor {
		return tcp.NewAcceptor(&cfg.TCP, &sessCfg, h)
	})

	// 16. 创建服务注册器
	registrar, err := etcd.NewRegistrar(&cfg.Registry)
	if err != nil {
		l.Error("failed to create registrar", "error", err)
		return
	}

	// 17. 创建应用并注册服务
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

	// 18. 运行
	if err := application.Run(); err != nil {
		l.Error("gateway exited with error", "error", err)
	}
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

