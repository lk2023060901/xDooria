package sentry

import (
	"github.com/getsentry/sentry-go"
)

// Level 事件级别
type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
	LevelFatal   Level = "fatal"
)

// toSentryLevel 转换为 Sentry SDK 的 Level
func (l Level) toSentryLevel() sentry.Level {
	switch l {
	case LevelDebug:
		return sentry.LevelDebug
	case LevelInfo:
		return sentry.LevelInfo
	case LevelWarning:
		return sentry.LevelWarning
	case LevelError:
		return sentry.LevelError
	case LevelFatal:
		return sentry.LevelFatal
	default:
		return sentry.LevelError
	}
}

// Stats 统计信息
type Stats struct {
	EventsTotal    uint64 // 总事件数
	EventsCaptured uint64 // 成功捕获数
	EventsDropped  uint64 // 丢弃数
}
