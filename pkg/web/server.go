package web

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/web/middleware"
)

// Server Web 服务核心结构
type Server struct {
	engine *gin.Engine
	config *Config
	logger logger.Logger
	server *http.Server
}

// NewServer 创建 Web 服务
func NewServer(cfg *Config, l logger.Logger) *Server {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if l == nil {
		l = logger.Default()
	}

	gin.SetMode(cfg.Mode)
	
	engine := gin.New()

	// 挂载基础中间件
	engine.Use(middleware.Logger(l))
	engine.Use(middleware.Recovery(l, true))
	engine.Use(middleware.Tracing("web-service")) // 默认服务名，之后可配置

	return &Server{
		engine: engine,
		config: cfg,
		logger: l.Named("web.server"),
	}
}

// Router 返回 Gin 引擎，用于注册路由
func (s *Server) Router() *gin.Engine {
	return s.engine
}

// Handler 返回 http.Handler 接口
func (s *Server) Handler() http.Handler {
	return s.engine
}

// Run 启动服务（支持优雅关机）
func (s *Server) Run(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.config.Port)
	s.server = &http.Server{
		Addr:           addr,
		Handler:        s.engine,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	// 启动监听
	go func() {
		var err error
		if s.config.EnableTLS {
			s.logger.Info("starting https server", "addr", addr)
			err = s.server.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
		} else {
			s.logger.Info("starting http server", "addr", addr)
			err = s.server.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			s.logger.Error("server startup failed", "error", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	select {
	case <-quit:
		s.logger.Info("shutting down server...")
	case <-ctx.Done():
		s.logger.Info("context cancelled, shutting down...")
	}

	// 优雅关机，设置 5 秒超时
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	s.logger.Info("server exited")
	return nil
}
