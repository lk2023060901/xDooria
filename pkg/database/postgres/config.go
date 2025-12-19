package postgres

import (
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// DBConfig 单个数据库实例配置
type DBConfig struct {
	Host     string `json:"host" yaml:"host"`
	Port     int    `json:"port" yaml:"port"`
	User     string `json:"user" yaml:"user"`
	Password string `json:"password" yaml:"password"`
	DBName   string `json:"db_name" yaml:"db_name"`
	SSLMode  string `json:"ssl_mode" yaml:"ssl_mode"` // disable, require, verify-ca, verify-full
}

// PoolConfig 连接池配置
type PoolConfig struct {
	MaxConns          int32         `json:"max_conns" yaml:"max_conns"`                     // 最大连接数
	MinConns          int32         `json:"min_conns" yaml:"min_conns"`                     // 最小连接数
	MaxConnLifetime   time.Duration `json:"max_conn_lifetime" yaml:"max_conn_lifetime"`     // 连接最大生命周期
	MaxConnIdleTime   time.Duration `json:"max_conn_idle_time" yaml:"max_conn_idle_time"`   // 连接最大空闲时间
	HealthCheckPeriod time.Duration `json:"health_check_period" yaml:"health_check_period"` // 健康检查周期
}

// Config PostgreSQL 配置
type Config struct {
	// 单机模式配置（与主从模式互斥）
	Standalone *DBConfig `json:"standalone,omitempty" yaml:"standalone,omitempty"`

	// 主从模式配置（与单机模式互斥）
	Master *DBConfig  `json:"master,omitempty" yaml:"master,omitempty"`
	Slaves []DBConfig `json:"slaves,omitempty" yaml:"slaves,omitempty"`

	// 连接池配置
	Pool PoolConfig `json:"pool" yaml:"pool"`

	// 连接超时配置
	ConnectTimeout time.Duration `json:"connect_timeout" yaml:"connect_timeout"` // 连接超时
	QueryTimeout   time.Duration `json:"query_timeout" yaml:"query_timeout"`     // 查询超时

	// 从库负载均衡策略（仅主从模式有效）
	SlaveLoadBalance string `json:"slave_load_balance,omitempty" yaml:"slave_load_balance,omitempty"` // random, round_robin
}

// DefaultConfig 返回默认配置（单机模式）
func DefaultConfig() *Config {
	return &Config{
		Standalone: &DBConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "",
			DBName:   "xdooria",
			SSLMode:  "disable",
		},
		Pool: PoolConfig{
			MaxConns:          25,
			MinConns:          5,
			MaxConnLifetime:   time.Hour,
			MaxConnIdleTime:   30 * time.Minute,
			HealthCheckPeriod: time.Minute,
		},
		ConnectTimeout: 10 * time.Second,
		QueryTimeout:   30 * time.Second,
	}
}

// MergeConfig 合并配置（使用通用的 config.MergeConfig）
func MergeConfig(dst, src *Config) (*Config, error) {
	return config.MergeConfig(dst, src)
}

// IsStandaloneMode 判断是否为单机模式
func (c *Config) IsStandaloneMode() bool {
	return c.Standalone != nil
}

// IsMasterSlaveMode 判断是否为主从模式
func (c *Config) IsMasterSlaveMode() bool {
	return c.Master != nil
}
