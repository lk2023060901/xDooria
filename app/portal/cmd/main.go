package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/app/portal/internal/client"
	"github.com/lk2023060901/xdooria/app/portal/internal/handler"
	"github.com/lk2023060901/xdooria/app/portal/internal/metrics"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/prometheus"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/web"
	"github.com/lk2023060901/xdooria/pkg/web/middleware"
)

// Config Portal 服务配置
type Config struct {
	Log     logger.Config             `mapstructure:"log"`
	Loggers map[string]*logger.Config `mapstructure:"loggers"`

	// Web Server 配置
	Web web.Config `mapstructure:"web"`

	// etcd 配置（服务发现）
	Etcd etcd.Config `mapstructure:"etcd"`

	// Login Client 配置
	LoginClient client.LoginClientConfig `mapstructure:"login_client"`

	// Prometheus 配置
	Prometheus prometheus.Config `mapstructure:"prometheus"`

	// 指标配置
	Metrics metrics.Config `mapstructure:"metrics"`
}

func main() {
	var cfg Config

	// 1. 加载配置
	if err := app.LoadConfig(&cfg); err != nil {
		panic(err)
	}

	// 2. 初始化 Logger
	l, err := logger.New(&cfg.Log)
	if err != nil {
		panic(err)
	}

	// 3. 创建 Prometheus 客户端
	promClient, err := prometheus.New(&cfg.Prometheus)
	if err != nil {
		l.Error("failed to create prometheus client", "error", err)
		return
	}
	defer promClient.Close()

	// 4. 创建指标收集器
	portalMetrics, err := metrics.New(&cfg.Metrics)
	if err != nil {
		l.Error("failed to create metrics", "error", err)
		return
	}
	defer portalMetrics.Stop()

	// 注册指标到 Prometheus
	if err := portalMetrics.Register(promClient.Registry()); err != nil {
		l.Error("failed to register metrics", "error", err)
		return
	}

	// 5. 创建 Login Client
	loginClient, err := client.NewLoginClient(&cfg.LoginClient, &cfg.Etcd, l)
	if err != nil {
		l.Error("failed to create login client", "error", err)
		return
	}
	defer loginClient.Close()

	// 6. 创建 Web Server
	webServer := web.NewServer(&cfg.Web, l)

	// 添加中间件
	webServer.Router().Use(middleware.CORS())

	// 创建限流器
	rateLimiter := middleware.NewRateLimiter(l, &middleware.RateLimitConfig{
		RequestsPerSecond: 1000,
		Burst:             2000,
		PerIP:             true,
		MaxLimiters:       10000,
		LimiterTTL:        10 * time.Minute,
		CleanupInterval:   time.Minute,
	})
	defer rateLimiter.Close()
	webServer.Router().Use(middleware.RateLimit(rateLimiter))

	// 7. 注册 Handler
	loginHandler := handler.NewLoginHandler(loginClient, portalMetrics, l)
	loginHandler.Register(webServer.Router())

	// 8. 健康检查
	webServer.Router().GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// 9. 运行服务
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听退出信号
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		l.Info("received shutdown signal")
		cancel()
	}()

	l.Info("starting portal server", "port", cfg.Web.Port)
	if err := webServer.Run(ctx); err != nil {
		l.Error("server exited with error", "error", err)
	}

	l.Info("portal server stopped")
}
