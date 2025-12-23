package otel

import "time"

// Config TracerProvider 配置
type Config struct {
	// Enabled 是否启用追踪（默认 true）
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// ServiceName 服务名称（必填）
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`

	// Endpoint 导出器端点
	// OTLP HTTP: localhost:4318
	// OTLP gRPC: localhost:4317
	Endpoint string `json:"endpoint" yaml:"endpoint" mapstructure:"endpoint"`

	// ExporterType 导出器类型: "otlp-http", "otlp-grpc", "stdout", "noop"
	ExporterType ExporterType `json:"exporter_type" yaml:"exporter_type" mapstructure:"exporter_type"`

	// Sampler 采样配置
	Sampler SamplerConfig `json:"sampler" yaml:"sampler" mapstructure:"sampler"`

	// BatchExport 批量导出配置
	BatchExport BatchExportConfig `json:"batch_export" yaml:"batch_export" mapstructure:"batch_export"`

	// Resource 资源属性
	Attributes map[string]string `json:"attributes" yaml:"attributes" mapstructure:"attributes"`

	// ShutdownTimeout 关闭超时
	ShutdownTimeout time.Duration `json:"shutdown_timeout" yaml:"shutdown_timeout" mapstructure:"shutdown_timeout"`

	// Insecure 是否使用不安全连接（不使用 TLS）
	Insecure bool `json:"insecure" yaml:"insecure" mapstructure:"insecure"`
}

// ExporterType 导出器类型
type ExporterType string

const (
	// ExporterTypeOTLPHTTP OTLP HTTP 导出器
	ExporterTypeOTLPHTTP ExporterType = "otlp-http"

	// ExporterTypeOTLPGRPC OTLP gRPC 导出器
	ExporterTypeOTLPGRPC ExporterType = "otlp-grpc"

	// ExporterTypeStdout 标准输出导出器（调试用）
	ExporterTypeStdout ExporterType = "stdout"

	// ExporterTypeNoop 空导出器（禁用追踪）
	ExporterTypeNoop ExporterType = "noop"
)

// SamplerConfig 采样配置
type SamplerConfig struct {
	// Type 采样类型: "always", "never", "ratio", "parent"
	Type SamplerType `json:"type" yaml:"type" mapstructure:"type"`

	// Ratio 采样比率（0.0-1.0），仅当 Type 为 "ratio" 时有效
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
	// BatchSize 批量大小（默认 512）
	BatchSize int `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`

	// ExportTimeout 导出超时（默认 30s）
	ExportTimeout time.Duration `json:"export_timeout" yaml:"export_timeout" mapstructure:"export_timeout"`

	// MaxQueueSize 最大队列大小（默认 2048）
	MaxQueueSize int `json:"max_queue_size" yaml:"max_queue_size" mapstructure:"max_queue_size"`

	// BatchTimeout 批量导出间隔（默认 5s）
	BatchTimeout time.Duration `json:"batch_timeout" yaml:"batch_timeout" mapstructure:"batch_timeout"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Enabled:      true,
		ServiceName:  "unknown-service",
		Endpoint:     "localhost:4318",
		ExporterType: ExporterTypeOTLPHTTP,
		Sampler: SamplerConfig{
			Type:  SamplerTypeParent,
			Ratio: 1.0,
		},
		BatchExport: BatchExportConfig{
			BatchSize:     512,
			ExportTimeout: 30 * time.Second,
			MaxQueueSize:  2048,
			BatchTimeout:  5 * time.Second,
		},
		Attributes:      make(map[string]string),
		ShutdownTimeout: 5 * time.Second,
		Insecure:        true,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.ServiceName == "" {
		return ErrInvalidServiceName
	}

	if c.Sampler.Type == SamplerTypeRatio {
		if c.Sampler.Ratio < 0 || c.Sampler.Ratio > 1 {
			return ErrInvalidSamplerRatio
		}
	}

	return nil
}
