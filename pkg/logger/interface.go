// pkg/logger/interface.go
package logger

import "context"

// Logger 日志接口
// 其他 pkg 模块可以引用此接口，避免重复定义
type Logger interface {
	// 基础日志方法
	Debug(msg string, keysAndValues ...interface{})
	Info(msg string, keysAndValues ...interface{})
	Warn(msg string, keysAndValues ...interface{})
	Error(msg string, keysAndValues ...interface{})

	// Context 版本
	DebugContext(ctx context.Context, msg string, keysAndValues ...interface{})
	InfoContext(ctx context.Context, msg string, keysAndValues ...interface{})
	WarnContext(ctx context.Context, msg string, keysAndValues ...interface{})
	ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{})

	// 派生方法
	Named(name string) Logger
	WithFields(keysAndValues ...interface{}) Logger

	// 同步
	Sync() error
}
