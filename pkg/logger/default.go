// pkg/logger/default.go
package logger

import (
	"os"
	"sync"

	"github.com/lk2023060901/xdooria/pkg/config"
)

var (
	defaultLogger     Logger
	defaultLoggerOnce sync.Once
	defaultLoggerMu   sync.RWMutex
)

// InitDefault 初始化默认 logger
func InitDefault(cfg *Config, opts ...Option) error {
	logger, err := New(cfg, opts...)
	if err != nil {
		return err
	}

	SetDefault(logger)
	return nil
}

// InitDefaultFromEnv 从环境变量初始化默认 logger
// 环境变量前缀: XDOORIA_LOG_
func InitDefaultFromEnv() error {
	// 从默认配置开始
	defaultCfg := DefaultConfig()

	// 创建环境变量覆盖配置
	envConfig := &Config{}

	// 从环境变量读取配置
	if level := os.Getenv("XDOORIA_LOG_LEVEL"); level != "" {
		envConfig.Level = Level(level)
	}
	if format := os.Getenv("XDOORIA_LOG_FORMAT"); format != "" {
		envConfig.Format = Format(format)
	}
	if path := os.Getenv("XDOORIA_LOG_PATH"); path != "" {
		envConfig.EnableFile = true
		envConfig.OutputPath = path
	}
	if os.Getenv("XDOORIA_LOG_CONSOLE") == "false" {
		envConfig.EnableConsole = false
	}
	if os.Getenv("XDOORIA_LOG_DEVELOPMENT") == "true" {
		envConfig.Development = true
	}

	// 合并配置：默认配置 + 环境变量覆盖
	mergedConfig, err := config.MergeConfig(defaultCfg, envConfig)
	if err != nil {
		return err
	}

	return InitDefault(mergedConfig)
}

// SetDefault 设置默认 logger
func SetDefault(logger Logger) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	defaultLogger = logger
}

// Default 获取默认 logger
func Default() Logger {
	defaultLoggerOnce.Do(func() {
		// 懒加载：使用默认配置 (仅控制台输出)
		if defaultLogger == nil {
			logger, err := New(DefaultConfig())
			if err != nil {
				panic(err)
			}
			defaultLogger = logger
		}
	})

	defaultLoggerMu.RLock()
	defer defaultLoggerMu.RUnlock()
	return defaultLogger
}

// Noop 返回空日志记录器
func Noop() Logger {
	return NewNoop()
}

// SetGlobalFields 设置全局字段
func SetGlobalFields(fields ...interface{}) {
	logger := Default()
	newLogger := logger.WithFields(fields...)
	SetDefault(newLogger)
}

// --- 便捷函数 (使用默认 logger) ---

func Debug(msg string, keysAndValues ...interface{}) {
	Default().Debug(msg, keysAndValues...)
}

func Info(msg string, keysAndValues ...interface{}) {
	Default().Info(msg, keysAndValues...)
}

func Warn(msg string, keysAndValues ...interface{}) {
	Default().Warn(msg, keysAndValues...)
}

func Error(msg string, keysAndValues ...interface{}) {
	Default().Error(msg, keysAndValues...)
}

func Named(name string) Logger {
	return Default().Named(name)
}

func WithFields(fields ...interface{}) Logger {
	return Default().WithFields(fields...)
}

func Sync() error {
	return Default().Sync()
}
