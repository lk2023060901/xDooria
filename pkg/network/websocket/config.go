// pkg/websocket/config.go
package websocket

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"time"
)

// ================================
// TLS 配置
// ================================

// TLSConfig TLS 配置
type TLSConfig struct {
	// CertFile 证书文件路径
	CertFile string `mapstructure:"cert_file" json:"cert_file" yaml:"cert_file"`
	// KeyFile 私钥文件路径
	KeyFile string `mapstructure:"key_file" json:"key_file" yaml:"key_file"`
	// CAFile CA 证书文件路径（用于客户端验证）
	CAFile string `mapstructure:"ca_file" json:"ca_file,omitempty" yaml:"ca_file,omitempty"`
	// InsecureSkipVerify 是否跳过证书验证（仅用于测试）
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify" yaml:"insecure_skip_verify"`
	// MinVersion TLS 最低版本 ("1.2" 或 "1.3")
	MinVersion string `mapstructure:"min_version" json:"min_version" yaml:"min_version"`
}

// Validate 验证 TLS 配置
func (c *TLSConfig) Validate() error {
	if c.CertFile == "" && c.KeyFile == "" {
		return nil // 不使用 TLS
	}
	if c.CertFile == "" {
		return fmt.Errorf("%w: cert_file is required when using TLS", ErrTLSConfigInvalid)
	}
	if c.KeyFile == "" {
		return fmt.Errorf("%w: key_file is required when using TLS", ErrTLSConfigInvalid)
	}
	return nil
}

// BuildTLSConfig 构建 tls.Config
func (c *TLSConfig) BuildTLSConfig() (*tls.Config, error) {
	if c == nil || (c.CertFile == "" && c.KeyFile == "") {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: c.InsecureSkipVerify,
	}

	// 设置最低版本
	switch c.MinVersion {
	case "1.3":
		tlsConfig.MinVersion = tls.VersionTLS13
	case "1.2", "":
		tlsConfig.MinVersion = tls.VersionTLS12
	default:
		return nil, fmt.Errorf("%w: invalid min_version %s", ErrTLSConfigInvalid, c.MinVersion)
	}

	// 加载证书
	if c.CertFile != "" && c.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to load certificate: %v", ErrTLSConfigInvalid, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// ================================
// 连接池配置
// ================================

// PoolConfig 连接池配置
type PoolConfig struct {
	// MaxConnections 最大连接数
	MaxConnections int `mapstructure:"max_connections" json:"max_connections" yaml:"max_connections"`
	// MaxConnectionsPerIP 每 IP 最大连接数
	MaxConnectionsPerIP int `mapstructure:"max_connections_per_ip" json:"max_connections_per_ip" yaml:"max_connections_per_ip"`
	// CleanupInterval 清理间隔
	CleanupInterval time.Duration `mapstructure:"cleanup_interval" json:"cleanup_interval" yaml:"cleanup_interval"`
}

// DefaultPoolConfig 返回默认连接池配置
func DefaultPoolConfig() PoolConfig {
	return PoolConfig{
		MaxConnections:      10000,
		MaxConnectionsPerIP: 100,
		CleanupInterval:     30 * time.Second,
	}
}

// ================================
// 心跳配置
// ================================

// HeartbeatConfig 心跳配置
type HeartbeatConfig struct {
	// Enable 是否启用心跳
	Enable bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// Interval 心跳间隔
	Interval time.Duration `mapstructure:"interval" json:"interval" yaml:"interval"`
	// Timeout 心跳超时时间
	Timeout time.Duration `mapstructure:"timeout" json:"timeout" yaml:"timeout"`
	// MaxMissCount 最大丢失次数
	MaxMissCount int `mapstructure:"max_miss_count" json:"max_miss_count" yaml:"max_miss_count"`
}

// DefaultHeartbeatConfig 返回默认心跳配置
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Enable:       true,
		Interval:     30 * time.Second,
		Timeout:      10 * time.Second,
		MaxMissCount: 3,
	}
}

// ================================
// 重连配置
// ================================

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	// Enable 是否启用自动重连
	Enable bool `mapstructure:"enable" json:"enable" yaml:"enable"`
	// MaxRetries 最大重试次数（0 = 无限重试）
	MaxRetries int `mapstructure:"max_retries" json:"max_retries" yaml:"max_retries"`
	// InitialDelay 初始延迟
	InitialDelay time.Duration `mapstructure:"initial_delay" json:"initial_delay" yaml:"initial_delay"`
	// MaxDelay 最大延迟
	MaxDelay time.Duration `mapstructure:"max_delay" json:"max_delay" yaml:"max_delay"`
	// Multiplier 延迟倍数
	Multiplier float64 `mapstructure:"multiplier" json:"multiplier" yaml:"multiplier"`
	// RandomFactor 随机因子（0-1）
	RandomFactor float64 `mapstructure:"random_factor" json:"random_factor" yaml:"random_factor"`
}

// DefaultReconnectConfig 返回默认重连配置
func DefaultReconnectConfig() ReconnectConfig {
	return ReconnectConfig{
		Enable:       true,
		MaxRetries:   0, // 无限重试
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		RandomFactor: 0.1,
	}
}

// ================================
// 服务端配置
// ================================

// ServerConfig 服务端配置
type ServerConfig struct {
	// 基础配置
	ReadBufferSize  int   `mapstructure:"read_buffer_size" json:"read_buffer_size" yaml:"read_buffer_size"`
	WriteBufferSize int   `mapstructure:"write_buffer_size" json:"write_buffer_size" yaml:"write_buffer_size"`
	MaxMessageSize  int64 `mapstructure:"max_message_size" json:"max_message_size" yaml:"max_message_size"`

	// 超时配置
	HandshakeTimeout time.Duration `mapstructure:"handshake_timeout" json:"handshake_timeout" yaml:"handshake_timeout"`
	ReadTimeout      time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout     time.Duration `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`
	PongTimeout      time.Duration `mapstructure:"pong_timeout" json:"pong_timeout" yaml:"pong_timeout"`
	PingInterval     time.Duration `mapstructure:"ping_interval" json:"ping_interval" yaml:"ping_interval"`

	// 压缩配置
	EnableCompression bool `mapstructure:"enable_compression" json:"enable_compression" yaml:"enable_compression"`
	CompressionLevel  int  `mapstructure:"compression_level" json:"compression_level" yaml:"compression_level"` // 1-9

	// TLS 配置
	TLS *TLSConfig `mapstructure:"tls" json:"tls,omitempty" yaml:"tls,omitempty"`

	// 连接池配置
	Pool PoolConfig `mapstructure:"pool" json:"pool" yaml:"pool"`

	// 消息队列配置
	SendQueueSize int `mapstructure:"send_queue_size" json:"send_queue_size" yaml:"send_queue_size"`
	RecvQueueSize int `mapstructure:"recv_queue_size" json:"recv_queue_size" yaml:"recv_queue_size"`

	// 跨域配置（运行时设置，不序列化）
	CheckOrigin func(r *http.Request) bool `mapstructure:"-" json:"-" yaml:"-"`
}

// DefaultServerConfig 返回默认服务端配置
func DefaultServerConfig() *ServerConfig {
	return &ServerConfig{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		MaxMessageSize:    512 * 1024, // 512KB
		HandshakeTimeout:  10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		PongTimeout:       60 * time.Second,
		PingInterval:      30 * time.Second,
		EnableCompression: true,
		CompressionLevel:  1, // BestSpeed
		Pool:              DefaultPoolConfig(),
		SendQueueSize:     256,
		RecvQueueSize:     256,
	}
}

// Validate 验证服务端配置
func (c *ServerConfig) Validate() error {
	if c == nil {
		return ErrInvalidConfig
	}
	if c.ReadBufferSize <= 0 {
		c.ReadBufferSize = 4096
	}
	if c.WriteBufferSize <= 0 {
		c.WriteBufferSize = 4096
	}
	if c.MaxMessageSize <= 0 {
		c.MaxMessageSize = 512 * 1024
	}
	if c.SendQueueSize <= 0 {
		c.SendQueueSize = 256
	}
	if c.RecvQueueSize <= 0 {
		c.RecvQueueSize = 256
	}
	if c.TLS != nil {
		if err := c.TLS.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// ================================
// 客户端配置
// ================================

// ClientConfig 客户端配置
type ClientConfig struct {
	// 连接地址
	URL string `mapstructure:"url" json:"url" yaml:"url"` // "ws://host:port/path" 或 "wss://..."

	// 缓冲区配置
	ReadBufferSize  int `mapstructure:"read_buffer_size" json:"read_buffer_size" yaml:"read_buffer_size"`
	WriteBufferSize int `mapstructure:"write_buffer_size" json:"write_buffer_size" yaml:"write_buffer_size"`

	// 超时配置
	DialTimeout  time.Duration `mapstructure:"dial_timeout" json:"dial_timeout" yaml:"dial_timeout"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" json:"read_timeout" yaml:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" json:"write_timeout" yaml:"write_timeout"`

	// 心跳配置
	Heartbeat HeartbeatConfig `mapstructure:"heartbeat" json:"heartbeat" yaml:"heartbeat"`

	// 重连配置
	Reconnect ReconnectConfig `mapstructure:"reconnect" json:"reconnect" yaml:"reconnect"`

	// 压缩配置
	EnableCompression bool `mapstructure:"enable_compression" json:"enable_compression" yaml:"enable_compression"`

	// TLS 配置
	TLS *TLSConfig `mapstructure:"tls" json:"tls,omitempty" yaml:"tls,omitempty"`

	// HTTP Headers（用于握手）
	Headers map[string]string `mapstructure:"headers" json:"headers,omitempty" yaml:"headers,omitempty"`

	// 消息队列配置
	SendQueueSize int `mapstructure:"send_queue_size" json:"send_queue_size" yaml:"send_queue_size"`
	RecvQueueSize int `mapstructure:"recv_queue_size" json:"recv_queue_size" yaml:"recv_queue_size"`
}

// DefaultClientConfig 返回默认客户端配置
func DefaultClientConfig() *ClientConfig {
	return &ClientConfig{
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		DialTimeout:       10 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      10 * time.Second,
		Heartbeat:         DefaultHeartbeatConfig(),
		Reconnect:         DefaultReconnectConfig(),
		EnableCompression: true,
		SendQueueSize:     256,
		RecvQueueSize:     256,
	}
}

// Validate 验证客户端配置
func (c *ClientConfig) Validate() error {
	if c == nil {
		return ErrInvalidConfig
	}
	if c.URL == "" {
		return ErrInvalidURL
	}
	if c.ReadBufferSize <= 0 {
		c.ReadBufferSize = 4096
	}
	if c.WriteBufferSize <= 0 {
		c.WriteBufferSize = 4096
	}
	if c.SendQueueSize <= 0 {
		c.SendQueueSize = 256
	}
	if c.RecvQueueSize <= 0 {
		c.RecvQueueSize = 256
	}
	if c.TLS != nil {
		if err := c.TLS.Validate(); err != nil {
			return err
		}
	}
	return nil
}
