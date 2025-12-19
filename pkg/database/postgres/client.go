package postgres

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client PostgreSQL 客户端
type Client struct {
	master *pgxpool.Pool   // 主库连接池（单机模式或主从模式的写库）
	slaves []*pgxpool.Pool // 从库连接池（仅主从模式）
	cfg    *Config

	// 从库负载均衡
	slaveIndex uint64 // round_robin 计数器
}

// New 创建 PostgreSQL 客户端
func New(cfg *Config) (*Client, error) {
	// 合并配置，确保有最小可用的配置
	defaultCfg := DefaultConfig()
	newCfg, err := MergeConfig(defaultCfg, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	// 验证配置
	if err := validateConfig(newCfg); err != nil {
		return nil, err
	}

	client := &Client{
		cfg:    newCfg,
		slaves: make([]*pgxpool.Pool, 0),
	}

	// 单机模式
	if newCfg.IsStandaloneMode() {
		masterPool, err := createPool(newCfg, newCfg.Standalone)
		if err != nil {
			return nil, fmt.Errorf("failed to create standalone pool: %w", err)
		}
		client.master = masterPool
		return client, nil
	}

	// 主从模式
	if !newCfg.IsMasterSlaveMode() {
		return nil, fmt.Errorf("%w: master config is required in master-slave mode", ErrInvalidConfig)
	}

	// 创建主库连接池
	masterPool, err := createPool(newCfg, newCfg.Master)
	if err != nil {
		return nil, fmt.Errorf("failed to create master pool: %w", err)
	}
	client.master = masterPool

	// 创建从库连接池（可选）
	// 如果没有配置从库，所有流量都走主库
	for i := range newCfg.Slaves {
		slavePool, err := createPool(newCfg, &newCfg.Slaves[i])
		if err != nil {
			// 从库连接失败不应该阻止服务启动，只记录错误
			fmt.Printf("warning: failed to create slave pool %d: %v\n", i, err)
			continue
		}
		client.slaves = append(client.slaves, slavePool)
	}

	return client, nil
}

// getMaster 获取主库连接池（内部使用）
func (c *Client) getMaster() *pgxpool.Pool {
	return c.master
}

// getSlave 获取从库连接池（内部使用）
func (c *Client) getSlave() *pgxpool.Pool {
	if len(c.slaves) == 0 {
		return c.master
	}

	switch c.cfg.SlaveLoadBalance {
	case "round_robin":
		idx := atomic.AddUint64(&c.slaveIndex, 1)
		return c.slaves[idx%uint64(len(c.slaves))]
	case "random":
		fallthrough
	default:
		return c.slaves[rand.Intn(len(c.slaves))]
	}
}

// Close 关闭客户端
func (c *Client) Close() {
	if c.master != nil {
		c.master.Close()
	}

	for _, slave := range c.slaves {
		if slave != nil {
			slave.Close()
		}
	}
}

// Ping 检查数据库连接
func (c *Client) Ping(ctx context.Context) error {
	// 检查主库
	if err := c.master.Ping(ctx); err != nil {
		return fmt.Errorf("master ping failed: %w", err)
	}

	// 检查从库（失败不影响整体）
	for i, slave := range c.slaves {
		if err := slave.Ping(ctx); err != nil {
			fmt.Printf("warning: slave %d ping failed: %v\n", i, err)
		}
	}

	return nil
}

// Stats 获取主库连接池状态
func (c *Client) Stats() *PoolStats {
	stat := c.master.Stat()
	return &PoolStats{
		AcquireCount:            stat.AcquireCount(),
		AcquireDuration:         stat.AcquireDuration(),
		AcquiredConns:           stat.AcquiredConns(),
		CanceledAcquireCount:    stat.CanceledAcquireCount(),
		ConstructingConns:       stat.ConstructingConns(),
		EmptyAcquireCount:       stat.EmptyAcquireCount(),
		IdleConns:               stat.IdleConns(),
		MaxConns:                stat.MaxConns(),
		TotalConns:              stat.TotalConns(),
		NewConnsCount:           stat.NewConnsCount(),
		MaxLifetimeDestroyCount: stat.MaxLifetimeDestroyCount(),
		MaxIdleDestroyCount:     stat.MaxIdleDestroyCount(),
	}
}

// SlaveStats 获取所有从库连接池状态
func (c *Client) SlaveStats() []*PoolStats {
	stats := make([]*PoolStats, len(c.slaves))
	for i, slave := range c.slaves {
		stat := slave.Stat()
		stats[i] = &PoolStats{
			AcquireCount:            stat.AcquireCount(),
			AcquireDuration:         stat.AcquireDuration(),
			AcquiredConns:           stat.AcquiredConns(),
			CanceledAcquireCount:    stat.CanceledAcquireCount(),
			ConstructingConns:       stat.ConstructingConns(),
			EmptyAcquireCount:       stat.EmptyAcquireCount(),
			IdleConns:               stat.IdleConns(),
			MaxConns:                stat.MaxConns(),
			TotalConns:              stat.TotalConns(),
			NewConnsCount:           stat.NewConnsCount(),
			MaxLifetimeDestroyCount: stat.MaxLifetimeDestroyCount(),
			MaxIdleDestroyCount:     stat.MaxIdleDestroyCount(),
		}
	}
	return stats
}

// validateConfig 验证配置
func validateConfig(cfg *Config) error {
	if cfg == nil {
		return ErrNilConfig
	}

	// 检查模式互斥性
	if cfg.IsStandaloneMode() && cfg.IsMasterSlaveMode() {
		return fmt.Errorf("%w: standalone and master-slave mode cannot be both configured", ErrInvalidConfig)
	}

	if !cfg.IsStandaloneMode() && !cfg.IsMasterSlaveMode() {
		return fmt.Errorf("%w: must configure either standalone or master-slave mode", ErrInvalidConfig)
	}

	// 验证单机模式配置
	if cfg.IsStandaloneMode() {
		if err := validateDBConfig(cfg.Standalone); err != nil {
			return fmt.Errorf("invalid standalone config: %w", err)
		}
	}

	// 验证主从模式配置
	if cfg.IsMasterSlaveMode() {
		if err := validateDBConfig(cfg.Master); err != nil {
			return fmt.Errorf("invalid master config: %w", err)
		}

		for i := range cfg.Slaves {
			if err := validateDBConfig(&cfg.Slaves[i]); err != nil {
				return fmt.Errorf("invalid slave %d config: %w", i, err)
			}
		}
	}

	// 验证连接池配置
	if cfg.Pool.MaxConns <= 0 {
		return fmt.Errorf("%w: max_conns must be positive", ErrInvalidConfig)
	}

	if cfg.Pool.MinConns < 0 {
		return fmt.Errorf("%w: min_conns must be non-negative", ErrInvalidConfig)
	}

	if cfg.Pool.MinConns > cfg.Pool.MaxConns {
		return fmt.Errorf("%w: min_conns cannot be greater than max_conns", ErrInvalidConfig)
	}

	return nil
}

// validateDBConfig 验证单个数据库配置
func validateDBConfig(cfg *DBConfig) error {
	if cfg == nil {
		return fmt.Errorf("%w: db config is nil", ErrInvalidConfig)
	}

	if cfg.Host == "" {
		return fmt.Errorf("%w: host is empty", ErrInvalidConfig)
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return fmt.Errorf("%w: invalid port %d", ErrInvalidConfig, cfg.Port)
	}

	if cfg.User == "" {
		return fmt.Errorf("%w: user is empty", ErrInvalidConfig)
	}

	if cfg.DBName == "" {
		return fmt.Errorf("%w: db_name is empty", ErrInvalidConfig)
	}

	return nil
}

// createPool 创建连接池
func createPool(cfg *Config, dbCfg *DBConfig) (*pgxpool.Pool, error) {
	// 构建连接字符串
	connString := buildConnString(cfg, dbCfg)

	// 创建连接池配置
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse pool config: %w", err)
	}

	// 设置连接池参数
	poolConfig.MaxConns = cfg.Pool.MaxConns
	poolConfig.MinConns = cfg.Pool.MinConns
	poolConfig.MaxConnLifetime = cfg.Pool.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.Pool.MaxConnIdleTime
	poolConfig.HealthCheckPeriod = cfg.Pool.HealthCheckPeriod

	// 创建连接池
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	// 测试连接
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return pool, nil
}

// buildConnString 构建连接字符串
func buildConnString(cfg *Config, dbCfg *DBConfig) string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s connect_timeout=%d",
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.User,
		dbCfg.Password,
		dbCfg.DBName,
		dbCfg.SSLMode,
		int(cfg.ConnectTimeout.Seconds()),
	)
}
