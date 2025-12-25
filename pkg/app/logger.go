package app

import (
	"sync"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// LoggerRegistry 管理应用中的具名日志对象
type LoggerRegistry struct {
	mu      sync.RWMutex
	loggers map[string]logger.Logger
}

func NewLoggerRegistry() *LoggerRegistry {
	return &LoggerRegistry{
		loggers: make(map[string]logger.Logger),
	}
}

// Register 注册一个具名 Logger
func (r *LoggerRegistry) Register(name string, l logger.Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.loggers[name] = l
}

// Get 获取一个具名 Logger，如果不存在则返回 nil
func (r *LoggerRegistry) Get(name string) logger.Logger {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.loggers[name]
}

// SyncAll 同步所有已注册的 Logger
func (r *LoggerRegistry) SyncAll() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, l := range r.loggers {
		_ = l.Sync()
	}
}

// InitLoggers 根据配置初始化多个具名 Logger
func (r *LoggerRegistry) InitLoggers(configs map[string]*logger.Config) error {
	for name, cfg := range configs {
		l, err := logger.New(cfg)
		if err != nil {
			return err
		}
		r.Register(name, l.Named(name))
	}
	return nil
}

// InitNamedLoggers 是一个方便的入口，用于从配置初始化并注册到 App
func InitNamedLoggers(a Application, configs map[string]*logger.Config) error {
	if base, ok := a.(*BaseApp); ok {
		return base.registry.InitLoggers(configs)
	}
	return nil
}
