package logger

import (
	"context"

	"go.uber.org/zap"
)

// DebugContext 记录 debug 级别日志，并从 context 中提取字段
func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Debug(msg, fields...)
}

// InfoContext 记录 info 级别日志，并从 context 中提取字段
func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Info(msg, fields...)
}

// WarnContext 记录 warn 级别日志，并从 context 中提取字段
func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Warn(msg, fields...)
}

// ErrorContext 记录 error 级别日志，并从 context 中提取字段
func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Error(msg, fields...)
}

// PanicContext 记录 panic 级别日志，并从 context 中提取字段
func (l *Logger) PanicContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Panic(msg, fields...)
}

// FatalContext 记录 fatal 级别日志，并从 context 中提取字段
func (l *Logger) FatalContext(ctx context.Context, msg string, fields ...zap.Field) {
	fields = append(l.contextExtractor(ctx), fields...)
	l.Fatal(msg, fields...)
}
