// pkg/logger/logger.go
package logger

import (
	"context"
	"fmt"
	"os"

	"github.com/lk2023060901/xdooria/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 确保 BaseLogger 实现了 Logger 接口
var _ Logger = (*BaseLogger)(nil)

// BaseLogger 基于 zap 的日志记录器实现
type BaseLogger struct {
	*zap.Logger
	config           *Config
	name             string
	globalFields     map[string]interface{}
	hooks            []Hook
	contextExtractor ContextFieldExtractor
}

// New 创建新的 BaseLogger
func New(cfg *Config, opts ...Option) (*BaseLogger, error) {
	// 使用 MergeConfig 合并默认配置和用户配置
	// 确保即使用户只传递部分配置，也能正常工作
	mergedConfig, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := mergedConfig.Validate(); err != nil {
		return nil, err
	}

	logger := &BaseLogger{
		config:           mergedConfig,
		globalFields:     make(map[string]interface{}),
		hooks:            make([]Hook, 0),
		contextExtractor: mergedConfig.ContextExtractor,
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
func (l *BaseLogger) build() (*zap.Logger, error) {
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
func (l *BaseLogger) buildEncoderConfig() zapcore.EncoderConfig {
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
func (l *BaseLogger) parseLevel(level Level) zapcore.Level {
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

// Debug 记录 debug 级别日志
func (l *BaseLogger) Debug(msg string, keysAndValues ...interface{}) {
	l.Logger.Debug(msg, l.toZapFields(keysAndValues...)...)
}

// Info 记录 info 级别日志
func (l *BaseLogger) Info(msg string, keysAndValues ...interface{}) {
	l.Logger.Info(msg, l.toZapFields(keysAndValues...)...)
}

// Warn 记录 warn 级别日志
func (l *BaseLogger) Warn(msg string, keysAndValues ...interface{}) {
	l.Logger.Warn(msg, l.toZapFields(keysAndValues...)...)
}

// Error 记录 error 级别日志
func (l *BaseLogger) Error(msg string, keysAndValues ...interface{}) {
	l.Logger.Error(msg, l.toZapFields(keysAndValues...)...)
}

// DebugContext 记录 debug 级别日志，并从 context 中提取字段
func (l *BaseLogger) DebugContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := append(l.contextExtractor(ctx), l.toZapFields(keysAndValues...)...)
	l.Logger.Debug(msg, fields...)
}

// InfoContext 记录 info 级别日志，并从 context 中提取字段
func (l *BaseLogger) InfoContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := append(l.contextExtractor(ctx), l.toZapFields(keysAndValues...)...)
	l.Logger.Info(msg, fields...)
}

// WarnContext 记录 warn 级别日志，并从 context 中提取字段
func (l *BaseLogger) WarnContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := append(l.contextExtractor(ctx), l.toZapFields(keysAndValues...)...)
	l.Logger.Warn(msg, fields...)
}

// ErrorContext 记录 error 级别日志，并从 context 中提取字段
func (l *BaseLogger) ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{}) {
	fields := append(l.contextExtractor(ctx), l.toZapFields(keysAndValues...)...)
	l.Logger.Error(msg, fields...)
}

// Named 创建具名 logger
func (l *BaseLogger) Named(name string) Logger {
	newLogger := &BaseLogger{
		Logger:           l.Logger.Named(name),
		config:           l.config,
		name:             name,
		globalFields:     l.globalFields,
		hooks:            l.hooks,
		contextExtractor: l.contextExtractor,
	}
	return newLogger
}

// WithFields 添加字段
func (l *BaseLogger) WithFields(keysAndValues ...interface{}) Logger {
	zapFields := l.toZapFields(keysAndValues...)
	if len(zapFields) == 0 {
		return l
	}

	newLogger := &BaseLogger{
		Logger:           l.Logger.With(zapFields...),
		config:           l.config,
		name:             l.name,
		globalFields:     l.globalFields,
		hooks:            l.hooks,
		contextExtractor: l.contextExtractor,
	}
	return newLogger
}

// Sync 同步日志
func (l *BaseLogger) Sync() error {
	return l.Logger.Sync()
}

// toZapFields 将 key-value 对转换为 zap.Field
func (l *BaseLogger) toZapFields(keysAndValues ...interface{}) []zap.Field {
	if len(keysAndValues) == 0 {
		return nil
	}

	// 如果第一个参数就是 zap.Field，直接返回
	if len(keysAndValues) > 0 {
		if _, ok := keysAndValues[0].(zap.Field); ok {
			fields := make([]zap.Field, 0, len(keysAndValues))
			for _, v := range keysAndValues {
				if f, ok := v.(zap.Field); ok {
					fields = append(fields, f)
				}
			}
			return fields
		}
	}

	// key-value 对形式
	if len(keysAndValues)%2 != 0 {
		return nil
	}

	fields := make([]zap.Field, 0, len(keysAndValues)/2)
	for i := 0; i < len(keysAndValues); i += 2 {
		key, ok := keysAndValues[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, zap.Any(key, keysAndValues[i+1]))
	}
	return fields
}

// Panic 记录 panic 级别日志
func (l *BaseLogger) Panic(msg string, fields ...zap.Field) {
	l.Logger.Panic(msg, fields...)
}

// Fatal 记录 fatal 级别日志
func (l *BaseLogger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// PanicContext 记录 panic 级别日志，并从 context 中提取字段
func (l *BaseLogger) PanicContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Logger.Panic(msg, fields...)
}

// FatalContext 记录 fatal 级别日志，并从 context 中提取字段
func (l *BaseLogger) FatalContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Logger.Fatal(msg, fields...)
}
