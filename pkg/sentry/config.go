package sentry

import (
	"time"

	"github.com/getsentry/sentry-go"
)

// Config Sentry 配置
type Config struct {
	// 基础配置
	DSN         string `json:"dsn" yaml:"dsn" mapstructure:"dsn"`                               // Sentry DSN
	Environment string `json:"environment" yaml:"environment" mapstructure:"environment"`       // 环境 (dev/test/prod)
	Release     string `json:"release" yaml:"release" mapstructure:"release"`                   // 版本号
	ServerName  string `json:"server_name" yaml:"server_name" mapstructure:"server_name"`       // 服务器名称

	// 采样配置
	SampleRate float64 `json:"sample_rate" yaml:"sample_rate" mapstructure:"sample_rate"` // 错误采样率 (0.0-1.0)

	// 上下文配置
	AttachStacktrace bool `json:"attach_stacktrace" yaml:"attach_stacktrace" mapstructure:"attach_stacktrace"` // 附加堆栈
	MaxBreadcrumbs   int  `json:"max_breadcrumbs" yaml:"max_breadcrumbs" mapstructure:"max_breadcrumbs"`       // 最大面包屑数

	// 超时配置
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"` // 关闭超时

	// 调试配置
	Debug bool `json:"debug" yaml:"debug" mapstructure:"debug"` // 调试模式

	// 全局标签
	Tags map[string]string `json:"tags" yaml:"tags" mapstructure:"tags"` // 全局标签
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Environment:      "production",
		SampleRate:       1.0,
		AttachStacktrace: true,
		MaxBreadcrumbs:   100,
		ShutdownTimeout:  2 * time.Second,
		Debug:            false,
		Tags:             make(map[string]string),
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c == nil {
		return ErrNilConfig
	}

	if c.DSN == "" {
		return ErrInvalidDSN
	}

	if c.SampleRate < 0 || c.SampleRate > 1 {
		return ErrInvalidConfig
	}

	if c.MaxBreadcrumbs < 0 {
		return ErrInvalidConfig
	}

	return nil
}

// toClientOptions 转换为 Sentry SDK 的 ClientOptions
func (c *Config) toClientOptions() sentry.ClientOptions {
	opts := sentry.ClientOptions{
		Dsn:              c.DSN,
		Environment:      c.Environment,
		Release:          c.Release,
		ServerName:       c.ServerName,
		SampleRate:       c.SampleRate,
		AttachStacktrace: c.AttachStacktrace,
		MaxBreadcrumbs:   c.MaxBreadcrumbs,
		Debug:            c.Debug,
	}

	return opts
}
