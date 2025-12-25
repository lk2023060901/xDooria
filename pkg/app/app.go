package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

var (
	ErrAppAlreadyRunning = errors.New("application is already running")
)

// Application 定义了框架级应用的接口
type Application interface {
	Run() error
	Shutdown() error
	Logger(name string) logger.Logger
	AppLogger() logger.Logger
	SetAppLogger(l logger.Logger)
}

// Server 定义了服务接口（如 gRPC, HTTP）
type Server interface {
	Start() error
	Stop() error
}

// GracefulServer 定义了支持优雅停止的服务器
type GracefulServer interface {
	Server
	GracefulStop() error
}

// Closer 定义了资源清理接口（如 Redis, DB, Tracer）
type Closer interface {
	Close() error
}

// BaseApp 提供了 Application 接口的基础实现
type BaseApp struct {
	opts     Options
	logger   logger.Logger
	registry *LoggerRegistry
	servers  []Server
	closers  []Closer

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex

	// 状态管理
	started atomic.Bool
	closed  atomic.Bool
}

// NewBaseApp 创建一个新的 BaseApp 实例
func NewBaseApp(opts ...Option) *BaseApp {
	o := DefaultOptions()
	for _, opt := range opts {
		opt(&o)
	}

	ctx, cancel := context.WithCancel(context.Background())

	a := &BaseApp{
		opts:     o,
		logger:   o.Logger.Named(o.Name),
		registry: NewLoggerRegistry(),
		ctx:      ctx,
		cancel:   cancel,
	}

	// 如果配置中已经带了日志定义，立即执行初始化
	if o.LogConfig != nil {
		if l, err := logger.New(o.LogConfig); err == nil {
			a.logger = l.Named(o.Name)
		}
	}

	return a
}

// SetAppLogger 替换应用主日志对象
func (a *BaseApp) SetAppLogger(l logger.Logger) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.logger = l
}

// AppLogger 获取应用主日志对象
func (a *BaseApp) AppLogger() logger.Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.logger
}

// Logger 获取具名 Logger
func (a *BaseApp) Logger(name string) logger.Logger {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.registry.Get(name)
}

// RegisterLogger 注册具名 Logger
func (a *BaseApp) RegisterLogger(name string, l logger.Logger) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.registry.Register(name, l)
}

// Run 启动应用程序并阻塞
func (a *BaseApp) Run() error {
	if !a.started.CompareAndSwap(false, true) {
		return ErrAppAlreadyRunning
	}

	// 初始化具名日志对象
	if len(a.opts.NamedLoggers) > 0 {
		if err := a.registry.InitLoggers(a.opts.NamedLoggers); err != nil {
			a.logger.Error("failed to initialize named loggers from config", "error", err)
			return err
		}
	}

	info := GetInfo()
	// 1. 打印醒目的版本字符串（符合产品级规范）
	fmt.Println(info.String())

	// 2. 结构化日志记录启动参数
	a.logger.Info("application starting",
		"name", info.AppName,
		"version", info.Version,
		"commit", info.GitCommit,
		"build_date", info.BuildDate,
		"go_version", info.GoVersion,
		"id", a.opts.ID,
	)

	// 启动所有注册的服务
	for _, srv := range a.servers {
		if err := srv.Start(); err != nil {
			a.logger.Error("failed to start server", "error", err)
			return err
		}
	}

	// 监听系统信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		a.logger.Info("received signal, shutting down", "signal", sig.String())
	case <-a.ctx.Done():
		a.logger.Info("context cancelled, shutting down")
	}

	return a.Shutdown()
}

// Shutdown 停止应用程序并清理资源
func (a *BaseApp) Shutdown() error {
	if !a.closed.CompareAndSwap(false, true) {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.cancel()
	a.logger.Info("application shutting down")

	var wg sync.WaitGroup
	// 停止所有服务器
	for _, srv := range a.servers {
		wg.Add(1)
		s := srv
		conc.Go(func() (struct{}, error) {
			defer wg.Done()
			var err error
			if gs, ok := s.(GracefulServer); ok {
				err = gs.GracefulStop()
			} else {
				err = s.Stop()
			}
			if err != nil {
				a.logger.Error("failed to stop server", "error", err)
			}
			return struct{}{}, err
		})
	}

	// 使用 conc.Go 等待服务器停止或超时
	waitFuture := conc.Go(func() (struct{}, error) {
		wg.Wait()
		return struct{}{}, nil
	})

	select {
	case <-waitFuture.Inner():
		a.logger.Info("all servers stopped")
	case <-time.After(a.opts.StopTimeout):
		a.logger.Warn("shutdown timeout, forcing exit")
	}

	// 逆序关闭所有 Closer 组件（LIFO）
	for i := len(a.closers) - 1; i >= 0; i-- {
		if err := a.closers[i].Close(); err != nil {
			a.logger.Error("failed to close component", "error", err)
		}
	}

	// 同步所有日志
	a.registry.SyncAll()
	_ = a.logger.Sync()

	a.logger.Info("application exited")
	return nil
}

// AppendServer 添加服务器
func (a *BaseApp) AppendServer(srv ...Server) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.servers = append(a.servers, srv...)
}

// AppendCloser 添加资源清理组件
func (a *BaseApp) AppendCloser(closer ...Closer) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.closers = append(a.closers, closer...)
}