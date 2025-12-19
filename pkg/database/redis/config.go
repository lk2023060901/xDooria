package redis

import "time"

// Config Redis 配置（Standalone/Master-Slave/Cluster 三种模式，必须且只能配置一种）
type Config struct {
	// Standalone 单机模式配置
	Standalone *NodeConfig `json:"standalone,omitempty" yaml:"standalone,omitempty"`

	// Master 主节点配置（主从模式）
	Master *NodeConfig `json:"master,omitempty" yaml:"master,omitempty"`
	// Slaves 从节点配置列表（主从模式）
	Slaves []NodeConfig `json:"slaves,omitempty" yaml:"slaves,omitempty"`

	// Cluster 集群模式配置
	Cluster *ClusterConfig `json:"cluster,omitempty" yaml:"cluster,omitempty"`

	// Pool 连接池配置（所有模式共享）
	Pool PoolConfig `json:"pool" yaml:"pool"`

	// SlaveLoadBalance 从库负载均衡策略（主从模式使用）
	// 可选值: "random"（随机）, "round_robin"（轮询）
	// 默认: "random"
	SlaveLoadBalance string `json:"slave_load_balance,omitempty" yaml:"slave_load_balance,omitempty"`
}

// NodeConfig 单节点配置（用于 Standalone 和 Master-Slave 模式）
type NodeConfig struct {
	Host     string `json:"host" yaml:"host"`         // 主机地址
	Port     int    `json:"port" yaml:"port"`         // 端口
	Password string `json:"password" yaml:"password"` // 密码
	DB       int    `json:"db" yaml:"db"`             // 数据库索引（0-15）
}

// ClusterConfig 集群配置
type ClusterConfig struct {
	Addrs    []string `json:"addrs" yaml:"addrs"`       // 集群节点地址列表 (格式: "host:port")
	Password string   `json:"password" yaml:"password"` // 密码
}

// PoolConfig 连接池配置
type PoolConfig struct {
	// MaxIdleConns 最大空闲连接数
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`

	// MaxOpenConns 最大打开连接数
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`

	// ConnMaxLifetime 连接最大生命周期
	ConnMaxLifetime time.Duration `json:"conn_max_lifetime" yaml:"conn_max_lifetime"`

	// ConnMaxIdleTime 连接最大空闲时间
	ConnMaxIdleTime time.Duration `json:"conn_max_idle_time" yaml:"conn_max_idle_time"`

	// DialTimeout 连接超时时间
	DialTimeout time.Duration `json:"dial_timeout" yaml:"dial_timeout"`

	// ReadTimeout 读超时时间
	ReadTimeout time.Duration `json:"read_timeout" yaml:"read_timeout"`

	// WriteTimeout 写超时时间
	WriteTimeout time.Duration `json:"write_timeout" yaml:"write_timeout"`

	// PoolTimeout 从连接池获取连接的超时时间
	PoolTimeout time.Duration `json:"pool_timeout" yaml:"pool_timeout"`
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c == nil {
		return ErrNilConfig
	}

	// 计算配置了几种模式
	modeCount := 0
	if c.Standalone != nil {
		modeCount++
	}
	if c.Master != nil {
		modeCount++
	}
	if c.Cluster != nil {
		modeCount++
	}

	// 必须且只能配置一种模式
	if modeCount != 1 {
		return ErrInvalidConfig
	}

	// 验证主从模式的从库负载均衡策略
	if c.Master != nil && len(c.Slaves) > 0 {
		if c.SlaveLoadBalance != "" &&
			c.SlaveLoadBalance != "random" &&
			c.SlaveLoadBalance != "round_robin" {
			return ErrInvalidSlaveLoadBalance
		}
	}

	return nil
}

// IsStandalone 是否为单机模式
func (c *Config) IsStandalone() bool {
	return c.Standalone != nil
}

// IsMasterSlave 是否为主从模式
func (c *Config) IsMasterSlave() bool {
	return c.Master != nil
}

// IsCluster 是否为集群模式
func (c *Config) IsCluster() bool {
	return c.Cluster != nil
}

// GetSlaveLoadBalance 获取从库负载均衡策略（默认为 random）
func (c *Config) GetSlaveLoadBalance() string {
	if c.SlaveLoadBalance == "" {
		return "random"
	}
	return c.SlaveLoadBalance
}
