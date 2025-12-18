package logger

import (
	"fmt"
	"os"

	"github.com/lk2023060901/xdooria/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger 日志记录器
type Logger struct {
	*zap.Logger
	config       *Config
	name         string
	globalFields map[string]interface{}
	hooks        []Hook
}

// New 创建新的 Logger
func New(cfg *Config, opts ...Option) (*Logger, error) {
	// 使用 MergeConfig 合并默认配置和用户配置
	// 确保即使用户只传递部分配置，也能正常工作
	mergedConfig, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := mergedConfig.Validate(); err != nil {
		return nil, err
	}

	logger := &Logger{
		config:       mergedConfig,
		globalFields: make(map[string]interface{}),
		hooks:        make([]Hook, 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(logger)
	}

	// 合并配置中的全局字段
	for k, v := range mergedConfig.GlobalFields {
		logger.globalFields[k] = v
	}

	// 构建 zap logger
	zapLogger, err := logger.build()
	if err != nil {
		return nil, err
	}

	logger.Logger = zapLogger

	return logger, nil
}

// build 构建 zap logger
func (l *Logger) build() (*zap.Logger, error) {
	// 创建 encoder config
	encoderConfig := l.buildEncoderConfig()

	// 创建 encoder
	var encoder zapcore.Encoder
	switch l.config.Format {
	case JSONFormat:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	case ConsoleFormat:
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	default:
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// 创建 writer
	writers := make([]zapcore.WriteSyncer, 0, 2)

	// 控制台输出
	if l.config.EnableConsole {
		writers = append(writers, zapcore.AddSync(os.Stdout))
	}

	// 文件输出 (仅在 EnableFile=true 时创建 rotation writer)
	if l.config.EnableFile {
		fileWriter, err := NewRotationWriter(&l.config.Rotation, l.config.OutputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create rotation writer: %w", err)
		}
		writers = append(writers, zapcore.AddSync(fileWriter))
	}

	writeSyncer := zapcore.NewMultiWriteSyncer(writers...)

	// 创建 level
	level := l.parseLevel(l.config.Level)

	// 创建 core
	core := zapcore.NewCore(encoder, writeSyncer, level)

	// 添加钩子
	if len(l.hooks) > 0 {
		core = NewHookedCore(core, l.hooks...)
	}

	// 采样
	if l.config.EnableSampling {
		core = zapcore.NewSamplerWithOptions(
			core,
			1, // 1 秒
			l.config.SamplingInitial,
			l.config.SamplingThereafter,
		)
	}

	// 构建选项
	options := []zap.Option{
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	}

	// 堆栈跟踪
	if l.config.EnableStacktrace {
		stacktraceLevel := l.parseLevel(l.config.StacktraceLevel)
		options = append(options, zap.AddStacktrace(stacktraceLevel))
	}

	// 开发模式
	if l.config.Development {
		options = append(options, zap.Development())
	}

	// 创建 logger
	zapLogger := zap.New(core, options...)

	// 添加全局字段
	if len(l.globalFields) > 0 {
		fields := make([]zap.Field, 0, len(l.globalFields))
		for k, v := range l.globalFields {
			fields = append(fields, zap.Any(k, v))
		}
		zapLogger = zapLogger.With(fields...)
	}

	// 添加名称
	if l.name != "" {
		zapLogger = zapLogger.Named(l.name)
	}

	return zapLogger, nil
}

// buildEncoderConfig 构建 encoder 配置
func (l *Logger) buildEncoderConfig() zapcore.EncoderConfig {
	config := zapcore.EncoderConfig{
		MessageKey:     "msg",
		LevelKey:       "level",
		TimeKey:        "time",
		NameKey:        "logger",
		CallerKey:      "caller",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 时间格式
	if l.config.TimeFormat != "" {
		config.EncodeTime = zapcore.TimeEncoderOfLayout(l.config.TimeFormat)
	} else {
		config.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// 开发模式：彩色输出
	if l.config.Development {
		config.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	return config
}

// parseLevel 解析日志等级
func (l *Logger) parseLevel(level Level) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case PanicLevel:
		return zapcore.PanicLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// Named 创建具名 logger
func (l *Logger) Named(name string) *Logger {
	newLogger := &Logger{
		Logger:       l.Logger.Named(name),
		config:       l.config,
		name:         name,
		globalFields: l.globalFields,
		hooks:        l.hooks,
	}
	return newLogger
}

// WithFields 添加字段
func (l *Logger) WithFields(fields ...interface{}) *Logger {
	if len(fields)%2 != 0 {
		return l
	}

	zapFields := make([]zap.Field, 0, len(fields)/2)
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		zapFields = append(zapFields, zap.Any(key, fields[i+1]))
	}

	newLogger := &Logger{
		Logger:       l.Logger.With(zapFields...),
		config:       l.config,
		name:         l.name,
		globalFields: l.globalFields,
		hooks:        l.hooks,
	}
	return newLogger
}

// Sync 同步日志
func (l *Logger) Sync() error {
	return l.Logger.Sync()
}
