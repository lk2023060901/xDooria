package tcp

import (
	"fmt"
	"time"
)

// ServerConfig 服务端配置
type ServerConfig struct {
	// 监听地址，如 "0.0.0.0:8080"
	Addr string `mapstructure:"addr" json:"addr" yaml:"addr"`

	// 网络类型，tcp/tcp4/tcp6
	Network string `mapstructure:"network" json:"network" yaml:"network"`

	// 是否启用多核
	Multicore bool `mapstructure:"multicore" json:"multicore" yaml:"multicore"`

	// 事件循环数量，0 表示使用 CPU 核心数
	NumEventLoop int `mapstructure:"num_event_loop" json:"num_event_loop" yaml:"num_event_loop"`

	// 是否启用端口复用
	ReusePort bool `mapstructure:"reuse_port" json:"reuse_port" yaml:"reuse_port"`

	// 是否启用地址复用
	ReuseAddr bool `mapstructure:"reuse_addr" json:"reuse_addr" yaml:"reuse_addr"`

	// 读缓冲区大小
	ReadBufferSize int `mapstructure:"read_buffer_size" json:"read_buffer_size" yaml:"read_buffer_size"`

	// 写缓冲区大小
	WriteBufferSize int `mapstructure:"write_buffer_size" json:"write_buffer_size" yaml:"write_buffer_size"`

	// 最大消息大小
	MaxMessageSize int `mapstructure:"max_message_size" json:"max_message_size" yaml:"max_message_size"`

	// 发送队列大小
	SendQueueSize int `mapstructure:"send_queue_size" json:"send_queue_size" yaml:"send_queue_size"`

	// 读超时
	ReadTimeout time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`

	// 写超时
	WriteTimeout time.Duration `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`

	// TCP KeepAlive 间隔
	TCPKeepAlive time.Duration `mapstructure:"tcp_keep_alive" json:"tcp_keep_alive" yaml:"tcp_keep_alive"`

	// 是否禁用 Nagle 算法（启用 TCP_NODELAY）
	TCPNoDelay bool `mapstructure:"tcp_no_delay" json:"tcp_no_delay" yaml:"tcp_no_delay"`
}

// DefaultServerConfig 返回默认服务端配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		Addr:            "0.0.0.0:9000",
		Network:         "tcp",
		Multicore:       true,
		NumEventLoop:    0,
		ReusePort:       true,
		ReuseAddr:       true,
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 64 * 1024,
		MaxMessageSize:  1024 * 1024, // 1MB
		SendQueueSize:   256,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    10 * time.Second,
		TCPKeepAlive:    30 * time.Second,
		TCPNoDelay:      true,
	}
}

// Validate 验证服务端配置
func (c *ServerConfig) Validate() error {
	if c == nil {
		return ErrInvalidConfig
	}
	if c.Addr == "" {
		return fmt.Errorf("%w: addr is required", ErrInvalidConfig)
	}
	if c.Network == "" {
		c.Network = "tcp"
	}
	if c.ReadBufferSize <= 0 {
		c.ReadBufferSize = 64 * 1024
	}
	if c.WriteBufferSize <= 0 {
		c.WriteBufferSize = 64 * 1024
	}
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = 1024 * 1024
	}
	if c.SendQueueSize <= 0 {
		c.SendQueueSize = 256
	}
	return nil
}

// ClientConfig 客户端配置
type ClientConfig struct {
	// 服务端地址，如 "127.0.0.1:8080"
	Addr string `mapstructure:"addr" json:"addr" yaml:"addr"`

	// 网络类型，tcp/tcp4/tcp6
	Network string `mapstructure:"network" json:"network" yaml:"network"`

	// 读缓冲区大小
	ReadBufferSize int `mapstructure:"read_buffer_size" json:"read_buffer_size" yaml:"read_buffer_size"`

	// 写缓冲区大小
	WriteBufferSize int `mapstructure:"write_buffer_size" json:"write_buffer_size" yaml:"write_buffer_size"`

	// 最大消息大小
	MaxMessageSize int `mapstructure:"max_message_size" json:"max_message_size" yaml:"max_message_size"`

	// 发送队列大小
	SendQueueSize int `mapstructure:"send_queue_size" json:"send_queue_size" yaml:"send_queue_size"`

	// 连接超时
	DialTimeout time.Duration `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`

	// 读超时
	ReadTimeout time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`

	// 写超时
	WriteTimeout time.Duration `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`

	// TCP KeepAlive 间隔
	TCPKeepAlive time.Duration `mapstructure:"tcp_keep_alive" json:"tcp_keep_alive" yaml:"tcp_keep_alive"`

	// 是否禁用 Nagle 算法（启用 TCP_NODELAY）
	TCPNoDelay bool `mapstructure:"tcp_no_delay" json:"tcp_no_delay" yaml:"tcp_no_delay"`

	// 重连配置
	Reconnect ReconnectConfig `mapstructure:"reconnect" json:"reconnect" yaml:"reconnect"`
}

// DefaultClientConfig 返回默认客户端配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		Network:         "tcp",
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 64 * 1024,
		MaxMessageSize:  1024 * 1024,
		SendQueueSize:   256,
		DialTimeout:     10 * time.Second,
		ReadTimeout:     60 * time.Second,
		WriteTimeout:    10 * time.Second,
		TCPKeepAlive:    30 * time.Second,
		TCPNoDelay:      true,
		Reconnect:       DefaultReconnectConfig(),
	}
}

// Validate 验证客户端配置
func (c *ClientConfig) Validate() error {
	if c == nil {
		return ErrInvalidConfig
	}
	if c.Addr == "" {
		return fmt.Errorf("%w: addr is required", ErrInvalidConfig)
	}
	if c.Network == "" {
		c.Network = "tcp"
	}
	if c.ReadBufferSize <= 0 {
		c.ReadBufferSize = 64 * 1024
	}
	if c.WriteBufferSize <= 0 {
		c.WriteBufferSize = 64 * 1024
	}
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = 1024 * 1024
	}
	if c.SendQueueSize <= 0 {
		c.SendQueueSize = 256
	}
	return nil
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	// 是否启用自动重连
	Enable bool `mapstructure:"enable" json:"enable" yaml:"enable"`

	// 最大重试次数（0 = 无限重试）
	MaxRetries int `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`

	// 初始延迟
	InitialDelay time.Duration `mapstructure:"initial_delay" json:"initial_delay" yaml:"initial_delay"`

	// 最大延迟
	MaxDelay time.Duration `mapstructure:"max_delay" json:"max_delay" yaml:"max_delay"`

	// 延迟倍数
	Multiplier float64 `mapstructure:"multiplier" json:"multiplier" yaml:"multiplier"`
}

// DefaultReconnectConfig 返回默认重连配置
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		Enable:       true,
		MaxRetries:   0,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}
