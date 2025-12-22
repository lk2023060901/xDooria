// pkg/logger/noop.go
package logger

import "context"

// 确保 NoopLogger 实现了 Logger 接口
var _ Logger = (*NoopLogger)(nil)

// NoopLogger 空日志记录器，不做任何操作
// 用于不需要日志输出的场景，或作为其他模块的默认 Logger
type NoopLogger struct{}

// NewNoop 创建空日志记录器
func NewNoop() *NoopLogger {
	return &NoopLogger{}
}

// Debug 空实现
func (l *NoopLogger) Debug(msg string, keysAndValues ...interface{}) {}

// Info 空实现
func (l *NoopLogger) Info(msg string, keysAndValues ...interface{}) {}

// Warn 空实现
func (l *NoopLogger) Warn(msg string, keysAndValues ...interface{}) {}

// Error 空实现
func (l *NoopLogger) Error(msg string, keysAndValues ...interface{}) {}

// DebugContext 空实现
func (l *NoopLogger) DebugContext(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// InfoContext 空实现
func (l *NoopLogger) InfoContext(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// WarnContext 空实现
func (l *NoopLogger) WarnContext(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// ErrorContext 空实现
func (l *NoopLogger) ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{}) {}

// Named 返回自身
func (l *NoopLogger) Named(name string) Logger {
	return l
}

// WithFields 返回自身
func (l *NoopLogger) WithFields(keysAndValues ...interface{}) Logger {
	return l
}

// Sync 空实现
func (l *NoopLogger) Sync() error {
	return nil
}
