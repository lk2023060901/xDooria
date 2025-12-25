//go:build wireinject
// +build wireinject

package main

import (
	"github.com/google/wire"
	"github.com/lk2023060901/xdooria/app/login/internal/dao"
	"github.com/lk2023060901/xdooria/app/login/internal/manager"
	"github.com/lk2023060901/xdooria/app/login/internal/service"
	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/framer"
	pb "github.com/xDooria/xDooria-proto-common"
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
		
		// 10. 组装与应用配置
		provideAppOptions,
		provideAppComponents,
		app.InitApp,
	))
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
	opts []app.Option,
) app.AppComponents {
	// 注册认证器插件
	authMgr.Register(localAuth)
	
	// 初始化路由映射 (OpCode -> Handler)
	loginSvc.Init(r)
	
	// 将 Bridge (实现的 CommonService) 注册到 gRPC Server
	pb.RegisterCommonServiceServer(srv.GetGRPCServer(), bridge)
	
	return app.AppComponents{
		Servers: []app.Server{
			srv, // gRPC Server 实现了 app.Server
		},
	}
}
