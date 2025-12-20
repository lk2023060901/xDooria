package client

import (
	"fmt"
	"time"

	"google.golang.org/grpc/keepalive"
)

// Config Client 配置
type Config struct {
	// 目标地址
	Target string `mapstructure:"target" json:"target"` // "etcd:///service-name" 或 "ip:port"

	// 连接超时
	DialTimeout time.Duration `mapstructure:"dial_timeout" json:"dial_timeout"`

	// 请求超时（默认超时，可被调用时的 Context 覆盖）
	RequestTimeout time.Duration `mapstructure:"request_timeout" json:"request_timeout"`

	// KeepAlive 配置（直接使用 gRPC 原生类型）
	KeepAlive keepalive.ClientParameters `mapstructure:"keep_alive" json:"keep_alive"`

	// 负载均衡策略
	LoadBalancer string `mapstructure:"load_balancer" json:"load_balancer"` // "round_robin", "pick_first"

	// 重试配置
	MaxRetries   int           `mapstructure:"max_retries" json:"max_retries"`
	RetryBackoff time.Duration `mapstructure:"retry_backoff" json:"retry_backoff"`

	// 消息大小限制
	MaxRecvMsgSize int `mapstructure:"max_recv_msg_size" json:"max_recv_msg_size"`
	MaxSendMsgSize int `mapstructure:"max_send_msg_size" json:"max_send_msg_size"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Target:         "localhost:50051",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 10 * time.Second, // 默认请求超时 10s
		KeepAlive: keepalive.ClientParameters{
			Time:                5 * time.Minute,
			Timeout:             10 * time.Second,
			PermitWithoutStream: false,
		},
		LoadBalancer:   "round_robin",
		MaxRetries:     3,
		RetryBackoff:   100 * time.Millisecond,
		MaxRecvMsgSize: 4 * 1024 * 1024, // 4MB
		MaxSendMsgSize: 4 * 1024 * 1024, // 4MB
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Target == "" {
		return fmt.Errorf("%w: target is required", ErrInvalidConfig)
	}

	if c.DialTimeout <= 0 {
		return fmt.Errorf("%w: dial_timeout must be positive", ErrInvalidConfig)
	}

	// RequestTimeout 允许为 0（表示不设置默认超时）

	if c.MaxRetries < 0 {
		return fmt.Errorf("%w: max_retries must be non-negative", ErrInvalidConfig)
	}

	if c.MaxRecvMsgSize <= 0 {
		return fmt.Errorf("%w: max_recv_msg_size must be positive", ErrInvalidConfig)
	}

	if c.MaxSendMsgSize <= 0 {
		return fmt.Errorf("%w: max_send_msg_size must be positive", ErrInvalidConfig)
	}

	return nil
}
