package prometheus

import "time"

// Config Prometheus 配置
type Config struct {
	// 命名空间（应用名称）
	Namespace string `json:"namespace" yaml:"namespace"`

	// 子系统（可选）
	Subsystem string `json:"subsystem" yaml:"subsystem"`

	// HTTP 服务器配置
	HTTPServer HTTPServerConfig `json:"http_server" yaml:"http_server"`

	// 是否注册默认 Go 采集器
	EnableGoCollector bool `json:"enable_go_collector" yaml:"enable_go_collector"`

	// 是否注册默认进程采集器
	EnableProcessCollector bool `json:"enable_process_collector" yaml:"enable_process_collector"`
}

// HTTPServerConfig HTTP 服务器配置
type HTTPServerConfig struct {
	// 是否启用独立的 HTTP 服务器暴露指标
	Enabled bool `json:"enabled" yaml:"enabled"`

	// 监听地址
	Addr string `json:"addr" yaml:"addr"`

	// 指标路径
	Path string `json:"path" yaml:"path"`

	// 读写超时
	Timeout time.Duration `json:"timeout" yaml:"timeout"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Namespace: "app",
		Subsystem: "",

		HTTPServer: HTTPServerConfig{
			Enabled: true,
			Addr:    ":9090",
			Path:    "/metrics",
			Timeout: 10 * time.Second,
		},

		EnableGoCollector:      true,
		EnableProcessCollector: true,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Namespace == "" {
		return ErrInvalidConfig
	}

	if c.HTTPServer.Enabled {
		if c.HTTPServer.Addr == "" {
			return ErrInvalidConfig
		}
		if c.HTTPServer.Path == "" {
			c.HTTPServer.Path = "/metrics"
		}
		if c.HTTPServer.Timeout == 0 {
			c.HTTPServer.Timeout = 10 * time.Second
		}
	}

	return nil
}
