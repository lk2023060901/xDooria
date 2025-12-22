package jaeger

import "time"

// Config Jaeger 配置
type Config struct {
	// 是否启用（默认 true）
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// 服务名称（必填）
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`

	// Jaeger Collector 端点
	// 例如: http://localhost:14268/api/traces (HTTP)
	// 或者: localhost:6831 (UDP Agent)
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// 端点类型: "collector" 或 "agent"
	EndpointType EndpointType `json:"endpoint_type" yaml:"endpoint_type" mapstructure:"endpoint_type"`

	// 采样配置
	Sampler SamplerConfig `json:"sampler" yaml:"sampler" mapstructure:"sampler"`

	// 资源属性
	Attributes map[string]string `json:"attributes" yaml:"attributes" mapstructure:"attributes"`

	// 批量导出配置
	BatchExport BatchExportConfig `json:"batch_export" yaml:"batch_export" mapstructure:"batch_export"`

	// 关闭超时
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`
}

// EndpointType 端点类型
type EndpointType string

const (
	// EndpointTypeCollector HTTP Collector 端点
	EndpointTypeCollector EndpointType = "collector"

	// EndpointTypeAgent UDP Agent 端点
	EndpointTypeAgent EndpointType = "agent"
)

// SamplerConfig 采样配置
type SamplerConfig struct {
	// 采样类型: "always", "never", "ratio", "parent"
	Type SamplerType `json:"type" yaml:"type" mapstructure:"type"`

	// 采样比率（0.0-1.0），仅当 Type 为 "ratio" 时有效
	Ratio float64 `json:"ratio" yaml:"ratio" mapstructure:"ratio"`
}

// SamplerType 采样类型
type SamplerType string

const (
	// SamplerTypeAlways 始终采样
	SamplerTypeAlways SamplerType = "always"

	// SamplerTypeNever 从不采样
	SamplerTypeNever SamplerType = "never"

	// SamplerTypeRatio 按比率采样
	SamplerTypeRatio SamplerType = "ratio"

	// SamplerTypeParent 跟随父 Span 采样决策
	SamplerTypeParent SamplerType = "parent"
)

// BatchExportConfig 批量导出配置
type BatchExportConfig struct {
	// 批量大小（默认 512）
	BatchSize int `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`

	// 导出超时（默认 30s）
	ExportTimeout time.Duration `json:"export_timeout" yaml:"export_timeout" mapstructure:"export_timeout"`

	// 最大队列大小（默认 2048）
	MaxQueueSize int `json:"max_queue_size" yaml:"max_queue_size" mapstructure:"max_queue_size"`

	// 批量导出间隔（默认 5s）
	BatchTimeout time.Duration `json:"batch_timeout" yaml:"batch_timeout" mapstructure:"batch_timeout"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:      true,
		ServiceName:  "unknown-service",
		Endpoint:     "http://localhost:14268/api/traces",
		EndpointType: EndpointTypeCollector,
		Sampler: SamplerConfig{
			Type:  SamplerTypeParent,
			Ratio: 1.0,
		},
		Attributes: make(map[string]string),
		BatchExport: BatchExportConfig{
			BatchSize:     512,
			ExportTimeout: 30 * time.Second,
			MaxQueueSize:  2048,
			BatchTimeout:  5 * time.Second,
		},
		ShutdownTimeout: 5 * time.Second,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.ServiceName == "" {
		return ErrInvalidConfig
	}

	if c.Endpoint == "" {
		return ErrInvalidConfig
	}

	if c.EndpointType != EndpointTypeCollector && c.EndpointType != EndpointTypeAgent {
		return ErrInvalidConfig
	}

	if c.Sampler.Type == SamplerTypeRatio {
		if c.Sampler.Ratio < 0 || c.Sampler.Ratio > 1 {
			return ErrInvalidConfig
		}
	}

	return nil
}
