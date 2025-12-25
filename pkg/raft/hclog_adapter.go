// pkg/raft/hclog_adapter.go
package raft

import (
	"io"
	"log"

	"github.com/hashicorp/go-hclog"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// hclogAdapter 将 logger.Logger 适配为 hclog.Logger
type hclogAdapter struct {
	l     logger.Logger
	name  string
	level hclog.Level
}

// NewHclogAdapter 创建 hclog 适配器
func NewHclogAdapter(l logger.Logger, name string, level string) hclog.Logger {
	return &hclogAdapter{
		l:     l.Named(name),
		name:  name,
		level: hclog.LevelFromString(level),
	}
}

func (a *hclogAdapter) Log(level hclog.Level, msg string, args ...interface{}) {
	switch level {
	case hclog.Trace, hclog.Debug:
		a.l.Debug(msg, args...)
	case hclog.Info:
		a.l.Info(msg, args...)
	case hclog.Warn:
		a.l.Warn(msg, args...)
	case hclog.Error:
		a.l.Error(msg, args...)
	}
}

func (a *hclogAdapter) Trace(msg string, args ...interface{}) {
	a.l.Debug(msg, args...)
}

func (a *hclogAdapter) Debug(msg string, args ...interface{}) {
	a.l.Debug(msg, args...)
}

func (a *hclogAdapter) Info(msg string, args ...interface{}) {
	a.l.Info(msg, args...)
}

func (a *hclogAdapter) Warn(msg string, args ...interface{}) {
	a.l.Warn(msg, args...)
}

func (a *hclogAdapter) Error(msg string, args ...interface{}) {
	a.l.Error(msg, args...)
}

func (a *hclogAdapter) IsTrace() bool { return a.level <= hclog.Trace }
func (a *hclogAdapter) IsDebug() bool { return a.level <= hclog.Debug }
func (a *hclogAdapter) IsInfo() bool  { return a.level <= hclog.Info }
func (a *hclogAdapter) IsWarn() bool  { return a.level <= hclog.Warn }
func (a *hclogAdapter) IsError() bool { return a.level <= hclog.Error }

func (a *hclogAdapter) ImpliedArgs() []interface{} { return nil }

func (a *hclogAdapter) With(args ...interface{}) hclog.Logger {
	return &hclogAdapter{
		l:     a.l.WithFields(args...),
		name:  a.name,
		level: a.level,
	}
}

func (a *hclogAdapter) Name() string { return a.name }

func (a *hclogAdapter) Named(name string) hclog.Logger {
	newName := name
	if a.name != "" {
		newName = a.name + "." + name
	}
	return &hclogAdapter{
		l:     a.l.Named(name),
		name:  newName,
		level: a.level,
	}
}

func (a *hclogAdapter) ResetNamed(name string) hclog.Logger {
	return &hclogAdapter{
		l:     a.l.Named(name),
		name:  name,
		level: a.level,
	}
}

func (a *hclogAdapter) SetLevel(level hclog.Level) { a.level = level }
func (a *hclogAdapter) GetLevel() hclog.Level      { return a.level }

func (a *hclogAdapter) StandardLogger(opts *hclog.StandardLoggerOptions) *log.Logger {
	return log.New(a.StandardWriter(opts), "", 0)
}

func (a *hclogAdapter) StandardWriter(opts *hclog.StandardLoggerOptions) io.Writer {
	return &hclogWriter{a: a}
}

type hclogWriter struct{ a *hclogAdapter }

func (w *hclogWriter) Write(p []byte) (int, error) {
	msg := string(p)
	if len(msg) > 0 && msg[len(msg)-1] == '\n' {
		msg = msg[:len(msg)-1]
	}
	w.a.Info(msg)
	return len(p), nil
}

var _ hclog.Logger = (*hclogAdapter)(nil)
