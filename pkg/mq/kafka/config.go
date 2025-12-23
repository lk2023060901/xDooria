package kafka

import "time"

// Config Kafka 配置
type Config struct {
	// Brokers Kafka broker 地址列表
	Brokers []string `json:"brokers" yaml:"brokers" mapstructure:"brokers"`

	// Producer 生产者配置
	Producer ProducerConfig `json:"producer" yaml:"producer" mapstructure:"producer"`

	// Consumer 消费者配置
	Consumer ConsumerConfig `json:"consumer" yaml:"consumer" mapstructure:"consumer"`

	// SASL 认证配置（可选）
	SASL *SASLConfig `json:"sasl,omitempty" yaml:"sasl,omitempty" mapstructure:"sasl"`

	// TLS 配置（可选）
	TLS *TLSConfig `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}

// ProducerConfig 生产者配置
type ProducerConfig struct {
	// Async 是否异步发送（默认 false，同步发送）
	Async bool `json:"async" yaml:"async" mapstructure:"async"`

	// BatchSize 批量大小（异步模式下，累积多少条消息后发送）
	BatchSize int `json:"batch_size" yaml:"batch_size" mapstructure:"batch_size"`

	// BatchTimeout 批量超时时间（异步模式下，最长等待时间）
	BatchTimeout time.Duration `json:"batch_timeout" yaml:"batch_timeout" mapstructure:"batch_timeout"`

	// MaxRetries 最大重试次数
	MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`

	// RetryBackoff 重试间隔
	RetryBackoff time.Duration `json:"retry_backoff" yaml:"retry_backoff" mapstructure:"retry_backoff"`

	// RequiredAcks 确认模式
	// 0: NoResponse - 不等待确认
	// 1: Leader - 等待 Leader 确认
	// -1: All - 等待所有副本确认
	RequiredAcks int `json:"required_acks" yaml:"required_acks" mapstructure:"required_acks"`

	// Compression 压缩算法: none, gzip, snappy, lz4, zstd
	Compression string `json:"compression" yaml:"compression" mapstructure:"compression"`

	// WriteTimeout 写超时
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout" mapstructure:"write_timeout"`

	// ReadTimeout 读超时（等待 broker 响应）
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout" mapstructure:"read_timeout"`
}

// ConsumerConfig 消费者配置
type ConsumerConfig struct {
	// GroupID 消费者组 ID
	GroupID string `json:"group_id" yaml:"group_id" mapstructure:"group_id"`

	// MinBytes 最小拉取字节数（达到此值才返回）
	MinBytes int `json:"min_bytes" yaml:"min_bytes" mapstructure:"min_bytes"`

	// MaxBytes 最大拉取字节数
	MaxBytes int `json:"max_bytes" yaml:"max_bytes" mapstructure:"max_bytes"`

	// MaxWait 最大等待时间（未达到 MinBytes 时最长等待时间）
	MaxWait time.Duration `json:"max_wait" yaml:"max_wait" mapstructure:"max_wait"`

	// CommitInterval 自动提交间隔（0 表示手动提交）
	CommitInterval time.Duration `json:"commit_interval" yaml:"commit_interval" mapstructure:"commit_interval"`

	// StartOffset 起始偏移量
	// -1: Latest - 从最新位置开始
	// -2: Earliest - 从最早位置开始
	StartOffset int64 `json:"start_offset" yaml:"start_offset" mapstructure:"start_offset"`

	// HeartbeatInterval 心跳间隔
	HeartbeatInterval time.Duration `json:"heartbeat_interval" yaml:"heartbeat_interval" mapstructure:"heartbeat_interval"`

	// SessionTimeout 会话超时（超过此时间未收到心跳，消费者被踢出组）
	SessionTimeout time.Duration `json:"session_timeout" yaml:"session_timeout" mapstructure:"session_timeout"`

	// RebalanceTimeout 重平衡超时
	RebalanceTimeout time.Duration `json:"rebalance_timeout" yaml:"rebalance_timeout" mapstructure:"rebalance_timeout"`

	// RetryBackoff 消费失败重试间隔
	RetryBackoff time.Duration `json:"retry_backoff" yaml:"retry_backoff" mapstructure:"retry_backoff"`

	// MaxRetries 消费失败最大重试次数
	MaxRetries int `json:"max_retries" yaml:"max_retries" mapstructure:"max_retries"`

	// Concurrency 并发消费协程数
	Concurrency int `json:"concurrency" yaml:"concurrency" mapstructure:"concurrency"`
}

// SASLConfig SASL 认证配置
type SASLConfig struct {
	// Mechanism 认证机制: PLAIN, SCRAM-SHA-256, SCRAM-SHA-512
	Mechanism string `json:"mechanism" yaml:"mechanism" mapstructure:"mechanism"`

	// Username 用户名
	Username string `json:"username" yaml:"username" mapstructure:"username"`

	// Password 密码
	Password string `json:"password" yaml:"password" mapstructure:"password"`
}

// TLSConfig TLS 配置
type TLSConfig struct {
	// Enable 是否启用 TLS
	Enable bool `json:"enable" yaml:"enable" mapstructure:"enable"`

	// CertFile 客户端证书文件
	CertFile string `json:"cert_file" yaml:"cert_file" mapstructure:"cert_file"`

	// KeyFile 客户端私钥文件
	KeyFile string `json:"key_file" yaml:"key_file" mapstructure:"key_file"`

	// CAFile CA 证书文件
	CAFile string `json:"ca_file" yaml:"ca_file" mapstructure:"ca_file"`

	// InsecureSkipVerify 是否跳过证书验证
	InsecureSkipVerify bool `json:"insecure_skip_verify" yaml:"insecure_skip_verify" mapstructure:"insecure_skip_verify"`
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Brokers: []string{"localhost:9092"},
		Producer: ProducerConfig{
			Async:        false,
			BatchSize:    100,
			BatchTimeout: 1 * time.Second,
			MaxRetries:   3,
			RetryBackoff: 100 * time.Millisecond,
			RequiredAcks: -1, // All
			Compression:  "snappy",
			WriteTimeout: 10 * time.Second,
			ReadTimeout:  10 * time.Second,
		},
		Consumer: ConsumerConfig{
			GroupID:           "default-group",
			MinBytes:          10 * 1024,        // 10KB
			MaxBytes:          10 * 1024 * 1024, // 10MB
			MaxWait:           500 * time.Millisecond,
			CommitInterval:    0, // 手动提交
			StartOffset:       -2, // Earliest
			HeartbeatInterval: 3 * time.Second,
			SessionTimeout:    30 * time.Second,
			RebalanceTimeout:  60 * time.Second,
			RetryBackoff:      100 * time.Millisecond,
			MaxRetries:        3,
			Concurrency:       1,
		},
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.Brokers) == 0 {
		return ErrNoBrokers
	}

	if c.Consumer.GroupID == "" {
		return ErrEmptyGroupID
	}

	return nil
}
