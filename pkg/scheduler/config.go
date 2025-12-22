// pkg/scheduler/config.go
package scheduler

import "time"

// Config 调度器配置
type Config struct {
	// Timezone 时区，默认 Asia/Shanghai
	Timezone string `mapstructure:"timezone"`

	// WithSeconds 是否启用秒级精度（6位表达式），默认 false
	WithSeconds bool `mapstructure:"with_seconds"`

	// JobTimeout 任务执行超时时间，0 表示不限制
	JobTimeout time.Duration `mapstructure:"job_timeout"`

	// SkipIfStillRunning 如果上次执行未完成则跳过，默认 true
	SkipIfStillRunning bool `mapstructure:"skip_if_still_running"`

	// Middleware 中间件配置
	Middleware MiddlewareConfig `mapstructure:"middleware"`

	// DefaultJobOptions 默认任务选项（可被单个任务覆盖）
	DefaultJobOptions JobOptions `mapstructure:"default_job_options"`
}

// MiddlewareConfig 中间件配置
type MiddlewareConfig struct {
	// Logging 启用日志记录
	Logging bool `mapstructure:"logging"`

	// Recovery 启用 panic 恢复
	Recovery bool `mapstructure:"recovery"`

	// Metrics 启用 Prometheus 指标
	Metrics bool `mapstructure:"metrics"`
}

// BackoffStrategy 退避策略
type BackoffStrategy string

const (
	// BackoffNone 不重试
	BackoffNone BackoffStrategy = "none"
	// BackoffFixed 固定间隔重试
	BackoffFixed BackoffStrategy = "fixed"
	// BackoffExponential 指数退避重试
	BackoffExponential BackoffStrategy = "exponential"
)

// JobOptions 任务选项
type JobOptions struct {
	// MaxRetries 失败重试次数，0 表示不重试
	MaxRetries int `mapstructure:"max_retries"`

	// BackoffStrategy 退避策略
	BackoffStrategy BackoffStrategy `mapstructure:"backoff_strategy"`

	// InitialBackoff 初始退避时间
	InitialBackoff time.Duration `mapstructure:"initial_backoff"`

	// MaxBackoff 最大退避时间
	MaxBackoff time.Duration `mapstructure:"max_backoff"`

	// BackoffMultiplier 退避乘数（仅 exponential 有效）
	BackoffMultiplier float64 `mapstructure:"backoff_multiplier"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timezone:           "Asia/Shanghai",
		WithSeconds:        false,
		JobTimeout:         0,
		SkipIfStillRunning: true,
		Middleware: MiddlewareConfig{
			Logging:  true,
			Recovery: true,
			Metrics:  false,
		},
		DefaultJobOptions: DefaultJobOptions(),
	}
}

// DefaultJobOptions 返回默认任务选项
func DefaultJobOptions() JobOptions {
	return JobOptions{
		MaxRetries:        3,
		BackoffStrategy:   BackoffExponential,
		InitialBackoff:    time.Second,
		MaxBackoff:        30 * time.Second,
		BackoffMultiplier: 2.0,
	}
}
