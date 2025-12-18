package logger

import (
	"context"
	"os"
	"sync"

	"github.com/lk2023060901/xdooria/pkg/config"
	"go.uber.org/zap"
)

var (
	defaultLogger     *Logger
	defaultLoggerOnce sync.Once
	defaultLoggerMu   sync.RWMutex
)

// InitDefault 初始化默认 logger
func InitDefault(config *Config, opts ...Option) error {
	logger, err := New(config, opts...)
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
func SetDefault(logger *Logger) {
	defaultLoggerMu.Lock()
	defer defaultLoggerMu.Unlock()
	defaultLogger = logger
}

// Default 获取默认 logger
func Default() *Logger {
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

// SetGlobalFields 设置全局字段
func SetGlobalFields(fields ...interface{}) {
	logger := Default()
	newLogger := logger.WithFields(fields...)
	SetDefault(newLogger)
}

// --- 便捷函数 (使用默认 logger) ---

func Debug(msg string, fields ...zap.Field) {
	Default().Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	Default().Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	Default().Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	Default().Error(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	Default().Panic(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	Default().Fatal(msg, fields...)
}

func Named(name string) *Logger {
	return Default().Named(name)
}

func WithFields(fields ...interface{}) *Logger {
	return Default().WithFields(fields...)
}

func Sync() error {
	return Default().Sync()
}

// --- Context 版本的便捷函数 ---

func DebugContext(ctx context.Context, msg string, fields ...zap.Field) {
	Default().DebugContext(ctx, msg, fields...)
}

func InfoContext(ctx context.Context, msg string, fields ...zap.Field) {
	Default().InfoContext(ctx, msg, fields...)
}

func WarnContext(ctx context.Context, msg string, fields ...zap.Field) {
	Default().WarnContext(ctx, msg, fields...)
}

func ErrorContext(ctx context.Context, msg string, fields ...zap.Field) {
	Default().ErrorContext(ctx, msg, fields...)
}

func PanicContext(ctx context.Context, msg string, fields ...zap.Field) {
	Default().PanicContext(ctx, msg, fields...)
}

func FatalContext(ctx context.Context, msg string, fields ...zap.Field) {
	Default().FatalContext(ctx, msg, fields...)
}

