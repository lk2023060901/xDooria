package main

import (
	"github.com/lk2023060901/xdooria/app/login/internal/metrics"
	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/network/tcp"
	"github.com/lk2023060901/xdooria/pkg/prometheus"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/security"
)

// Config 定义 Login 服务的完整配置结构
type Config struct {
	Log     logger.Config             `mapstructure:"log"`
	Loggers map[string]*logger.Config `mapstructure:"loggers"`

	// JWT 配置
	JWT security.JWTConfig `mapstructure:"jwt"`

	// TCP Server 配置
	TCP tcp.ServerConfig `mapstructure:"tcp"`

	// Session 配置
	Session session.Config `mapstructure:"session"`

	// Framer 配置
	Framer framer.Config `mapstructure:"framer"`

	// Prometheus 配置
	Prometheus prometheus.Config `mapstructure:"prometheus"`

	// 服务注册配置
	Registry etcd.Config `mapstructure:"registry"`

	// 指标配置
	Metrics metrics.Config `mapstructure:"metrics"`
}

func main() {
	var cfg Config

	// 1. 加载配置
	if err := app.LoadConfig(&cfg); err != nil {
		panic(err)
	}

	// 2. 初始化主日志
	l, err := logger.New(&cfg.Log)
	if err != nil {
		panic(err)
	}

	// 3. 通过 Wire 初始化应用
	application, cleanup, err := InitApp(&cfg, l)
	if err != nil {
		l.Error("failed to initialize application", "error", err)
		return
	}
	defer cleanup()

	// 4. 运行服务
	if err := application.Run(); err != nil {
		l.Error("application exited with error", "error", err)
	}
}