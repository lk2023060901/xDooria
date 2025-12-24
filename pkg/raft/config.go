// pkg/raft/config.go
package raft

import (
	"fmt"
	"net"
	"time"

	"github.com/hashicorp/raft"
)

// Config Raft 配置
type Config struct {
	// 网络配置
	BindAddr string `mapstructure:"bind_addr"` // 绑定地址 (host:port)

	// 数据目录
	DataDir string `mapstructure:"data_dir"` // 数据存储目录

	// 超时配置
	HeartbeatTimeout   time.Duration `mapstructure:"heartbeat_timeout"`    // 心跳超时
	ElectionTimeout    time.Duration `mapstructure:"election_timeout"`     // 选举超时
	CommitTimeout      time.Duration `mapstructure:"commit_timeout"`       // 提交超时
	LeaderLeaseTimeout time.Duration `mapstructure:"leader_lease_timeout"` // Leader 租约超时

	// 快照配置
	SnapshotInterval  time.Duration `mapstructure:"snapshot_interval"`  // 快照间隔
	SnapshotThreshold uint64        `mapstructure:"snapshot_threshold"` // 触发快照的日志条目数
	SnapshotRetain    int           `mapstructure:"snapshot_retain"`    // 保留的快照数量

	// 性能配置
	MaxAppendEntries int `mapstructure:"max_append_entries"` // 单次 AppendEntries 最大条目数
	TrailingLogs     uint64 `mapstructure:"trailing_logs"`      // 快照后保留的日志数

	// 传输配置
	MaxPool int `mapstructure:"max_pool"` // 连接池大小

	// 日志配置
	LogLevel string `mapstructure:"log_level"` // 日志级别: debug, info, warn, error
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		BindAddr:           "127.0.0.1:7000",
		DataDir:            "./raft-data",
		HeartbeatTimeout:   1000 * time.Millisecond,
		ElectionTimeout:    1000 * time.Millisecond,
		CommitTimeout:      50 * time.Millisecond,
		LeaderLeaseTimeout: 500 * time.Millisecond,
		SnapshotInterval:   5 * time.Minute,
		SnapshotThreshold:  8192,
		SnapshotRetain:     3,
		MaxAppendEntries:   64,
		TrailingLogs:       10240,
		MaxPool:            3,
		LogLevel:           "info",
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.BindAddr == "" {
		return fmt.Errorf("%w: bind_addr is required", ErrInvalidConfig)
	}

	// 验证地址格式
	if _, _, err := net.SplitHostPort(c.BindAddr); err != nil {
		return fmt.Errorf("%w: invalid bind_addr: %v", ErrInvalidConfig, err)
	}

	if c.DataDir == "" {
		return fmt.Errorf("%w: data_dir is required", ErrInvalidConfig)
	}

	if c.HeartbeatTimeout <= 0 {
		return fmt.Errorf("%w: heartbeat_timeout must be positive", ErrInvalidConfig)
	}

	if c.ElectionTimeout <= 0 {
		return fmt.Errorf("%w: election_timeout must be positive", ErrInvalidConfig)
	}

	if c.ElectionTimeout < c.HeartbeatTimeout {
		return fmt.Errorf("%w: election_timeout must be >= heartbeat_timeout", ErrInvalidConfig)
	}

	if c.SnapshotThreshold == 0 {
		return fmt.Errorf("%w: snapshot_threshold must be positive", ErrInvalidConfig)
	}

	if c.SnapshotRetain <= 0 {
		return fmt.Errorf("%w: snapshot_retain must be positive", ErrInvalidConfig)
	}

	return nil
}

// ToRaftConfig 转换为 HashiCorp Raft 配置
func (c *Config) ToRaftConfig(nodeID string) *raft.Config {
	cfg := raft.DefaultConfig()

	cfg.LocalID = raft.ServerID(nodeID)

	cfg.HeartbeatTimeout = c.HeartbeatTimeout
	cfg.ElectionTimeout = c.ElectionTimeout
	cfg.CommitTimeout = c.CommitTimeout
	cfg.LeaderLeaseTimeout = c.LeaderLeaseTimeout

	cfg.SnapshotInterval = c.SnapshotInterval
	cfg.SnapshotThreshold = c.SnapshotThreshold
	cfg.TrailingLogs = c.TrailingLogs
	cfg.MaxAppendEntries = c.MaxAppendEntries

	// 设置日志级别
	switch c.LogLevel {
	case "debug":
		cfg.LogLevel = "DEBUG"
	case "info":
		cfg.LogLevel = "INFO"
	case "warn":
		cfg.LogLevel = "WARN"
	case "error":
		cfg.LogLevel = "ERROR"
	default:
		cfg.LogLevel = "INFO"
	}

	return cfg
}

