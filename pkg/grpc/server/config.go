package server

import (
	"fmt"
	"time"

	"google.golang.org/grpc/keepalive"
)

// Config Server 配置
type Config struct {
	// 基础配置
	Name    string `mapstructure:"name" json:"name"`       // 服务名称
	Network string `mapstructure:"network" json:"network"` // 网络协议：tcp, unix
	Address string `mapstructure:"address" json:"address"` // 监听地址，如 :50051 或 /tmp/app.sock

	// 消息大小限制
	MaxRecvMsgSize int `mapstructure:"max_recv_msg_size" json:"max_recv_msg_size"` // 最大接收消息大小（字节）
	MaxSendMsgSize int `mapstructure:"max_send_msg_size" json:"max_send_msg_size"` // 最大发送消息大小（字节）

	// KeepAlive 配置（直接使用 gRPC 原生类型）
	KeepAliveParams      keepalive.ServerParameters  `mapstructure:"keep_alive_params" json:"keep_alive_params"`
	KeepAliveEnforcement keepalive.EnforcementPolicy `mapstructure:"keep_alive_enforcement" json:"keep_alive_enforcement"`

	// 服务注册配置（可选）
	ServiceRegistry *ServiceRegistryConfig `mapstructure:"service_registry" json:"service_registry"`

	// 优雅关闭超时
	GracefulStopTimeout time.Duration `mapstructure:"graceful_stop_timeout" json:"graceful_stop_timeout"`

	// 健康检查和反射
	EnableHealthCheck bool `mapstructure:"enable_health_check" json:"enable_health_check"` // 启用健康检查
	EnableReflection  bool `mapstructure:"enable_reflection" json:"enable_reflection"`     // 启用反射（调试用）
}

// ServiceRegistryConfig 服务注册配置
type ServiceRegistryConfig struct {
	Enabled     bool              `mapstructure:"enabled" json:"enabled"`
	Endpoints   []string          `mapstructure:"endpoints" json:"endpoints"`       // etcd endpoints
	ServiceName string            `mapstructure:"service_name" json:"service_name"` // 服务名
	Metadata    map[string]string `mapstructure:"metadata" json:"metadata"`         // 元数据
	TTL         time.Duration     `mapstructure:"ttl" json:"ttl"`                   // 租约 TTL
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Name:            "grpc-server",
		Network:         "tcp",
		Address:         ":50051",
		MaxRecvMsgSize:  4 * 1024 * 1024, // 4MB
		MaxSendMsgSize:  4 * 1024 * 1024, // 4MB
		KeepAliveParams: keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Minute,
			MaxConnectionAge:      30 * time.Minute,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Minute,
			Timeout:               10 * time.Second,
		},
		KeepAliveEnforcement: keepalive.EnforcementPolicy{
			MinTime:             1 * time.Minute,
			PermitWithoutStream: true,
		},
		ServiceRegistry:     nil, // 默认不启用
		GracefulStopTimeout: 30 * time.Second,
		EnableHealthCheck:   true,
		EnableReflection:    false,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("%w: name is required", ErrInvalidConfig)
	}

	if c.Network != "tcp" && c.Network != "unix" {
		return fmt.Errorf("%w: network must be tcp or unix", ErrInvalidConfig)
	}

	if c.Address == "" {
		return fmt.Errorf("%w: address is required", ErrInvalidConfig)
	}

	if c.MaxRecvMsgSize <= 0 {
		return fmt.Errorf("%w: max_recv_msg_size must be positive", ErrInvalidConfig)
	}

	if c.MaxSendMsgSize <= 0 {
		return fmt.Errorf("%w: max_send_msg_size must be positive", ErrInvalidConfig)
	}

	if c.GracefulStopTimeout <= 0 {
		return fmt.Errorf("%w: graceful_stop_timeout must be positive", ErrInvalidConfig)
	}

	// 验证服务注册配置
	if c.ServiceRegistry != nil && c.ServiceRegistry.Enabled {
		if len(c.ServiceRegistry.Endpoints) == 0 {
			return fmt.Errorf("%w: service_registry.endpoints is required", ErrInvalidConfig)
		}
		if c.ServiceRegistry.ServiceName == "" {
			return fmt.Errorf("%w: service_registry.service_name is required", ErrInvalidConfig)
		}
		if c.ServiceRegistry.TTL <= 0 {
			c.ServiceRegistry.TTL = 10 * time.Second // 默认 10s
		}
	}

	return nil
}
