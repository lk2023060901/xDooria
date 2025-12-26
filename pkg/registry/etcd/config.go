package etcd

import (
	"fmt"
	"time"
)

// Config etcd 服务注册配置
type Config struct {
	// Endpoints etcd 集群地址
	Endpoints []string `mapstructure:"endpoints" json:"endpoints"`
	// DialTimeout 连接超时
	DialTimeout time.Duration `mapstructure:"dial_timeout" json:"dial_timeout"`
	// TTL 租约过期时间
	TTL time.Duration `mapstructure:"ttl" json:"ttl"`
	// Namespace 命名空间前缀（如 /services）
	Namespace string `mapstructure:"namespace" json:"namespace"`
	// VirtualNodes 一致性哈希虚拟节点数量
	VirtualNodes int `mapstructure:"virtual_nodes" json:"virtual_nodes"`
	// ServiceName 服务名称（用于服务注册）
	ServiceName string `mapstructure:"service_name" json:"service_name"`
	// ServiceAddr 服务地址（用于服务注册，如 localhost:50051）
	ServiceAddr string `mapstructure:"service_addr" json:"service_addr"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoints:    []string{"localhost:2379"},
		DialTimeout:  5 * time.Second,
		TTL:          10 * time.Second,
		Namespace:    "/services",
		VirtualNodes: 150,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("endpoints is required")
	}
	if c.DialTimeout <= 0 {
		return fmt.Errorf("dial_timeout must be positive")
	}
	if c.TTL <= 0 {
		return fmt.Errorf("ttl must be positive")
	}
	if c.Namespace == "" {
		c.Namespace = "/services"
	}
	return nil
}
