package redis

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

// redisClient 内部 Redis 客户端接口（隐藏 go-redis 类型）
type redisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.BoolCmd
	SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
	Exists(ctx context.Context, keys ...string) *redis.IntCmd
	Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	TTL(ctx context.Context, key string) *redis.DurationCmd
	Incr(ctx context.Context, key string) *redis.IntCmd
	IncrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	Decr(ctx context.Context, key string) *redis.IntCmd
	DecrBy(ctx context.Context, key string, value int64) *redis.IntCmd
	HGet(ctx context.Context, key, field string) *redis.StringCmd
	HSet(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	HGetAll(ctx context.Context, key string) *redis.MapStringStringCmd
	HDel(ctx context.Context, key string, fields ...string) *redis.IntCmd
	HExists(ctx context.Context, key, field string) *redis.BoolCmd
	HLen(ctx context.Context, key string) *redis.IntCmd
	LPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	RPush(ctx context.Context, key string, values ...interface{}) *redis.IntCmd
	LPop(ctx context.Context, key string) *redis.StringCmd
	RPop(ctx context.Context, key string) *redis.StringCmd
	LLen(ctx context.Context, key string) *redis.IntCmd
	LRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	SAdd(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	SMembers(ctx context.Context, key string) *redis.StringSliceCmd
	SIsMember(ctx context.Context, key string, member interface{}) *redis.BoolCmd
	SCard(ctx context.Context, key string) *redis.IntCmd
	ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	ZRem(ctx context.Context, key string, members ...interface{}) *redis.IntCmd
	ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd
	ZRevRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd
	ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) *redis.StringSliceCmd
	ZCard(ctx context.Context, key string) *redis.IntCmd
	ZScore(ctx context.Context, key, member string) *redis.FloatCmd
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
	EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) *redis.Cmd
	ScriptLoad(ctx context.Context, script string) *redis.StringCmd
	Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd
	PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub
	Pipeline() redis.Pipeliner
	Pipelined(ctx context.Context, fn func(redis.Pipeliner) error) ([]redis.Cmder, error)
	PoolStats() *redis.PoolStats
	Ping(ctx context.Context) *redis.StatusCmd
	Close() error
}

// Client Redis 客户端（隐藏 go-redis 类型，支持主从读写分离）
type Client struct {
	master         redisClient   // 主节点（或单机/集群客户端）
	slaves         []redisClient // 从节点列表（主从模式）
	cfg            *Config       // 配置
	slaveIndex     uint64        // 轮询索引（round_robin 策略使用）
	loadBalanceRng *rand.Rand    // 随机数生成器（random 策略使用）
}

// NewClient 创建 Redis 客户端
func NewClient(cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client := &Client{
		cfg:            cfg,
		loadBalanceRng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// 根据配置模式创建客户端
	if cfg.IsStandalone() {
		return client.createStandaloneClient()
	} else if cfg.IsMasterSlave() {
		return client.createMasterSlaveClient()
	} else if cfg.IsCluster() {
		return client.createClusterClient()
	}

	return nil, ErrInvalidConfig
}

// createStandaloneClient 创建单机模式客户端
func (c *Client) createStandaloneClient() (*Client, error) {
	opts := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", c.cfg.Standalone.Host, c.cfg.Standalone.Port),
		Password:        c.cfg.Standalone.Password,
		DB:              c.cfg.Standalone.DB,
		MaxIdleConns:    c.cfg.Pool.MaxIdleConns,
		MaxActiveConns:  c.cfg.Pool.MaxOpenConns,
		ConnMaxLifetime: c.cfg.Pool.ConnMaxLifetime,
		ConnMaxIdleTime: c.cfg.Pool.ConnMaxIdleTime,
		DialTimeout:     c.cfg.Pool.DialTimeout,
		ReadTimeout:     c.cfg.Pool.ReadTimeout,
		WriteTimeout:    c.cfg.Pool.WriteTimeout,
		PoolTimeout:     c.cfg.Pool.PoolTimeout,
	}

	c.master = redis.NewClient(opts)
	return c, nil
}

// createMasterSlaveClient 创建主从模式客户端
func (c *Client) createMasterSlaveClient() (*Client, error) {
	// 创建主节点
	masterOpts := &redis.Options{
		Addr:            fmt.Sprintf("%s:%d", c.cfg.Master.Host, c.cfg.Master.Port),
		Password:        c.cfg.Master.Password,
		DB:              c.cfg.Master.DB,
		MaxIdleConns:    c.cfg.Pool.MaxIdleConns,
		MaxActiveConns:  c.cfg.Pool.MaxOpenConns,
		ConnMaxLifetime: c.cfg.Pool.ConnMaxLifetime,
		ConnMaxIdleTime: c.cfg.Pool.ConnMaxIdleTime,
		DialTimeout:     c.cfg.Pool.DialTimeout,
		ReadTimeout:     c.cfg.Pool.ReadTimeout,
		WriteTimeout:    c.cfg.Pool.WriteTimeout,
		PoolTimeout:     c.cfg.Pool.PoolTimeout,
	}
	c.master = redis.NewClient(masterOpts)

	// 创建从节点
	c.slaves = make([]redisClient, len(c.cfg.Slaves))
	for i, slaveCfg := range c.cfg.Slaves {
		slaveOpts := &redis.Options{
			Addr:            fmt.Sprintf("%s:%d", slaveCfg.Host, slaveCfg.Port),
			Password:        slaveCfg.Password,
			DB:              slaveCfg.DB,
			MaxIdleConns:    c.cfg.Pool.MaxIdleConns,
			MaxActiveConns:  c.cfg.Pool.MaxOpenConns,
			ConnMaxLifetime: c.cfg.Pool.ConnMaxLifetime,
			ConnMaxIdleTime: c.cfg.Pool.ConnMaxIdleTime,
			DialTimeout:     c.cfg.Pool.DialTimeout,
			ReadTimeout:     c.cfg.Pool.ReadTimeout,
			WriteTimeout:    c.cfg.Pool.WriteTimeout,
			PoolTimeout:     c.cfg.Pool.PoolTimeout,
		}
		c.slaves[i] = redis.NewClient(slaveOpts)
	}

	return c, nil
}

// createClusterClient 创建集群模式客户端
func (c *Client) createClusterClient() (*Client, error) {
	opts := &redis.ClusterOptions{
		Addrs:           c.cfg.Cluster.Addrs,
		Password:        c.cfg.Cluster.Password,
		MaxIdleConns:    c.cfg.Pool.MaxIdleConns,
		ConnMaxLifetime: c.cfg.Pool.ConnMaxLifetime,
		ConnMaxIdleTime: c.cfg.Pool.ConnMaxIdleTime,
		DialTimeout:     c.cfg.Pool.DialTimeout,
		ReadTimeout:     c.cfg.Pool.ReadTimeout,
		WriteTimeout:    c.cfg.Pool.WriteTimeout,
		PoolTimeout:     c.cfg.Pool.PoolTimeout,
	}

	c.master = redis.NewClusterClient(opts)
	return c, nil
}

// getMaster 获取主节点（用于写操作）
func (c *Client) getMaster() redisClient {
	return c.master
}

// getSlave 获取从节点（用于读操作，支持负载均衡）
func (c *Client) getSlave() redisClient {
	// 如果没有从节点，使用主节点
	if len(c.slaves) == 0 {
		return c.master
	}

	// 根据负载均衡策略选择从节点
	strategy := c.cfg.GetSlaveLoadBalance()
	switch strategy {
	case "round_robin":
		// 轮询策略
		index := atomic.AddUint64(&c.slaveIndex, 1) % uint64(len(c.slaves))
		return c.slaves[index]
	default:
		// 随机策略（默认）
		index := c.loadBalanceRng.Intn(len(c.slaves))
		return c.slaves[index]
	}
}

// Ping 测试连接
func (c *Client) Ping(ctx context.Context) error {
	if err := c.master.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("master ping failed: %w", err)
	}

	// 测试所有从节点
	for i, slave := range c.slaves {
		if err := slave.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("slave[%d] ping failed: %w", i, err)
		}
	}

	return nil
}

// PoolStats 获取连接池统计信息（隐藏 go-redis 类型）
func (c *Client) PoolStats() PoolStats {
	stats := c.master.PoolStats()
	return PoolStats{
		Hits:       stats.Hits,
		Misses:     stats.Misses,
		Timeouts:   stats.Timeouts,
		TotalConns: stats.TotalConns,
		IdleConns:  stats.IdleConns,
		StaleConns: stats.StaleConns,
	}
}

// Close 关闭客户端
func (c *Client) Close() error {
	// 关闭主节点
	if err := c.master.Close(); err != nil {
		return fmt.Errorf("failed to close master: %w", err)
	}

	// 关闭所有从节点
	for i, slave := range c.slaves {
		if err := slave.Close(); err != nil {
			return fmt.Errorf("failed to close slave[%d]: %w", i, err)
		}
	}

	return nil
}

// ===== Pub/Sub =====

// Publish 发布消息到频道
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) *redis.IntCmd {
	return c.getMaster().Publish(ctx, channel, message)
}

// PSubscribe 订阅模式匹配的频道
func (c *Client) PSubscribe(ctx context.Context, patterns ...string) *redis.PubSub {
	return c.getMaster().PSubscribe(ctx, patterns...)
}

// ===== Lua Scripts =====

// Eval 执行 Lua 脚本
func (c *Client) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	return c.getMaster().Eval(ctx, script, keys, args...)
}
