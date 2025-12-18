package logger

import (
	"go.uber.org/zap/zapcore"
)

// Hook 日志钩子接口
type Hook interface {
	// OnWrite 日志写入前回调
	// 返回 false 则跳过该日志
	OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool
}

// HookFunc 函数式 Hook
type HookFunc func(entry zapcore.Entry, fields []zapcore.Field) bool

func (f HookFunc) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
	return f(entry, fields)
}

// HookedCore 带钩子的 Core
type HookedCore struct {
	zapcore.Core
	hooks []Hook
}

// NewHookedCore 创建带钩子的 Core
func NewHookedCore(core zapcore.Core, hooks ...Hook) zapcore.Core {
	return &HookedCore{
		Core:  core,
		hooks: hooks,
	}
}

// Check 检查日志等级
func (h *HookedCore) Check(entry zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if h.Enabled(entry.Level) {
		return ce.AddCore(entry, h)
	}
	return ce
}

// Write 写入日志
func (h *HookedCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// 执行钩子
	for _, hook := range h.hooks {
		if !hook.OnWrite(entry, fields) {
			return nil // 跳过该日志
		}
	}

	return h.Core.Write(entry, fields)
}

// With 添加字段
func (h *HookedCore) With(fields []zapcore.Field) zapcore.Core {
	return &HookedCore{
		Core:  h.Core.With(fields),
		hooks: h.hooks,
	}
}

// --- 内置 Hooks ---

// SensitiveDataHook 敏感数据脱敏 Hook
func SensitiveDataHook(sensitiveKeys []string) Hook {
	keyMap := make(map[string]bool)
	for _, key := range sensitiveKeys {
		keyMap[key] = true
	}

	return HookFunc(func(entry zapcore.Entry, fields []zapcore.Field) bool {
		for i := range fields {
			if keyMap[fields[i].Key] {
				fields[i].String = "***REDACTED***"
			}
		}
		return true
	})
}
