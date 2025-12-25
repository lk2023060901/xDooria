package app

import (
	"time"

	"github.com/google/uuid"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Options 应用程序配置选项
type Options struct {
	ID          string
	Name        string
	Version     string
	Metadata    map[string]string
	StopTimeout time.Duration
	Logger      logger.Logger

	// 日志配置
	LogConfig    *logger.Config
	NamedLoggers map[string]*logger.Config
}

// Option 定义配置函数
type Option func(*Options)

// DefaultOptions 返回默认配置
func DefaultOptions() Options {
	return Options{
		ID:          uuid.New().String(),
		Name:        AppName,
		Version:     Version,
		Metadata:    make(map[string]string),
		StopTimeout: 30 * time.Second,
		// 初始仅提供基础控制台 Logger
		Logger: logger.Default(),
	}
}

// WithLogConfig 设置主日志配置
func WithLogConfig(cfg *logger.Config) Option {
	return func(o *Options) { o.LogConfig = cfg }
}

// WithNamedLoggers 设置具名日志配置
func WithNamedLoggers(loggers map[string]*logger.Config) Option {
	return func(o *Options) { o.NamedLoggers = loggers }
}

// WithLogger 设置应用日志器
func WithLogger(l logger.Logger) Option {
	return func(o *Options) { o.Logger = l }
}

// WithID 设置应用 ID
func WithID(id string) Option {
	return func(o *Options) { o.ID = id }
}

// WithName 设置应用名称
func WithName(name string) Option {
	return func(o *Options) { o.Name = name }
}

// WithVersion 设置应用版本
func WithVersion(v string) Option {
	return func(o *Options) { o.Version = v }
}

// WithMetadata 设置应用元数据
func WithMetadata(md map[string]string) Option {
	return func(o *Options) { o.Metadata = md }
}

// WithStopTimeout 设置优雅停止超时时间
func WithStopTimeout(t time.Duration) Option {
	return func(o *Options) { o.StopTimeout = t }
}
