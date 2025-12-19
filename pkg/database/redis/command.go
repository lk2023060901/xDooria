package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// ==================== String 操作 ====================

// Get 获取字符串值（从从库读取）
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	val, err := c.getSlave().Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNil
		}
		return "", fmt.Errorf("get failed: %w", err)
	}
	return val, nil
}

// Set 设置字符串值（写入主库）
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.getMaster().Set(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("set failed: %w", err)
	}
	return nil
}

// SetNX 设置字符串值（仅当键不存在时）
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	ok, err := c.getMaster().SetNX(ctx, key, value, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("setnx failed: %w", err)
	}
	return ok, nil
}

// SetEX 设置字符串值并指定过期时间
func (c *Client) SetEX(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := c.getMaster().SetEx(ctx, key, value, expiration).Err(); err != nil {
		return fmt.Errorf("setex failed: %w", err)
	}
	return nil
}

// Del 删除键
func (c *Client) Del(ctx context.Context, keys ...string) (int64, error) {
	n, err := c.getMaster().Del(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("del failed: %w", err)
	}
	return n, nil
}

// Exists 检查键是否存在（从从库读取）
func (c *Client) Exists(ctx context.Context, keys ...string) (int64, error) {
	n, err := c.getSlave().Exists(ctx, keys...).Result()
	if err != nil {
		return 0, fmt.Errorf("exists failed: %w", err)
	}
	return n, nil
}

// Expire 设置键的过期时间
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	ok, err := c.getMaster().Expire(ctx, key, expiration).Result()
	if err != nil {
		return false, fmt.Errorf("expire failed: %w", err)
	}
	return ok, nil
}

// TTL 获取键的剩余过期时间（从从库读取）
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := c.getSlave().TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("ttl failed: %w", err)
	}
	return ttl, nil
}

// Incr 自增
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	val, err := c.getMaster().Incr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("incr failed: %w", err)
	}
	return val, nil
}

// IncrBy 按指定值自增
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.getMaster().IncrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("incrby failed: %w", err)
	}
	return val, nil
}

// Decr 自减
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	val, err := c.getMaster().Decr(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("decr failed: %w", err)
	}
	return val, nil
}

// DecrBy 按指定值自减
func (c *Client) DecrBy(ctx context.Context, key string, value int64) (int64, error) {
	val, err := c.getMaster().DecrBy(ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("decrby failed: %w", err)
	}
	return val, nil
}

// ==================== Hash 操作 ====================

// HGet 获取哈希字段值（从从库读取）
func (c *Client) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := c.getSlave().HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNil
		}
		return "", fmt.Errorf("hget failed: %w", err)
	}
	return val, nil
}

// HSet 设置哈希字段值（写入主库）
func (c *Client) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	n, err := c.getMaster().HSet(ctx, key, values...).Result()
	if err != nil {
		return 0, fmt.Errorf("hset failed: %w", err)
	}
	return n, nil
}

// HGetAll 获取哈希所有字段（从从库读取）
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	vals, err := c.getSlave().HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall failed: %w", err)
	}
	return vals, nil
}

// HDel 删除哈希字段
func (c *Client) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	n, err := c.getMaster().HDel(ctx, key, fields...).Result()
	if err != nil {
		return 0, fmt.Errorf("hdel failed: %w", err)
	}
	return n, nil
}

// HExists 检查哈希字段是否存在（从从库读取）
func (c *Client) HExists(ctx context.Context, key, field string) (bool, error) {
	ok, err := c.getSlave().HExists(ctx, key, field).Result()
	if err != nil {
		return false, fmt.Errorf("hexists failed: %w", err)
	}
	return ok, nil
}

// HLen 获取哈希字段数量（从从库读取）
func (c *Client) HLen(ctx context.Context, key string) (int64, error) {
	n, err := c.getSlave().HLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("hlen failed: %w", err)
	}
	return n, nil
}

// ==================== List 操作 ====================

// LPush 从左侧推入元素
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	n, err := c.getMaster().LPush(ctx, key, values...).Result()
	if err != nil {
		return 0, fmt.Errorf("lpush failed: %w", err)
	}
	return n, nil
}

// RPush 从右侧推入元素
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	n, err := c.getMaster().RPush(ctx, key, values...).Result()
	if err != nil {
		return 0, fmt.Errorf("rpush failed: %w", err)
	}
	return n, nil
}

// LPop 从左侧弹出元素
func (c *Client) LPop(ctx context.Context, key string) (string, error) {
	val, err := c.getMaster().LPop(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNil
		}
		return "", fmt.Errorf("lpop failed: %w", err)
	}
	return val, nil
}

// RPop 从右侧弹出元素
func (c *Client) RPop(ctx context.Context, key string) (string, error) {
	val, err := c.getMaster().RPop(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrNil
		}
		return "", fmt.Errorf("rpop failed: %w", err)
	}
	return val, nil
}

// LLen 获取列表长度（从从库读取）
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	n, err := c.getSlave().LLen(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("llen failed: %w", err)
	}
	return n, nil
}

// LRange 获取列表范围元素（从从库读取）
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vals, err := c.getSlave().LRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("lrange failed: %w", err)
	}
	return vals, nil
}

// ==================== Set 操作 ====================

// SAdd 添加集合成员
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	n, err := c.getMaster().SAdd(ctx, key, members...).Result()
	if err != nil {
		return 0, fmt.Errorf("sadd failed: %w", err)
	}
	return n, nil
}

// SRem 删除集合成员
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	n, err := c.getMaster().SRem(ctx, key, members...).Result()
	if err != nil {
		return 0, fmt.Errorf("srem failed: %w", err)
	}
	return n, nil
}

// SMembers 获取集合所有成员（从从库读取）
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	vals, err := c.getSlave().SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("smembers failed: %w", err)
	}
	return vals, nil
}

// SIsMember 检查成员是否在集合中（从从库读取）
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	ok, err := c.getSlave().SIsMember(ctx, key, member).Result()
	if err != nil {
		return false, fmt.Errorf("sismember failed: %w", err)
	}
	return ok, nil
}

// SCard 获取集合成员数量（从从库读取）
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	n, err := c.getSlave().SCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("scard failed: %w", err)
	}
	return n, nil
}

// ==================== Sorted Set 操作 ====================

// ZAdd 添加有序集合成员
func (c *Client) ZAdd(ctx context.Context, key string, members ...ZItem) (int64, error) {
	// 转换为 go-redis 类型
	zs := make([]redis.Z, len(members))
	for i, m := range members {
		zs[i] = redis.Z{
			Score:  m.Score,
			Member: m.Member,
		}
	}

	n, err := c.getMaster().ZAdd(ctx, key, zs...).Result()
	if err != nil {
		return 0, fmt.Errorf("zadd failed: %w", err)
	}
	return n, nil
}

// ZRem 删除有序集合成员
func (c *Client) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	n, err := c.getMaster().ZRem(ctx, key, members...).Result()
	if err != nil {
		return 0, fmt.Errorf("zrem failed: %w", err)
	}
	return n, nil
}

// ZRange 获取有序集合范围成员（从从库读取，按分数从小到大）
func (c *Client) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vals, err := c.getSlave().ZRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange failed: %w", err)
	}
	return vals, nil
}

// ZRangeWithScores 获取有序集合范围成员（带分数，从从库读取，按分数从小到大）
func (c *Client) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]ZItem, error) {
	zs, err := c.getSlave().ZRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("zrange with scores failed: %w", err)
	}

	// 转换为自定义类型
	items := make([]ZItem, len(zs))
	for i, z := range zs {
		items[i] = ZItem{
			Member: z.Member.(string),
			Score:  z.Score,
		}
	}
	return items, nil
}

// ZRevRange 获取有序集合范围成员（从从库读取，按分数从大到小）
func (c *Client) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	vals, err := c.getSlave().ZRevRange(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange failed: %w", err)
	}
	return vals, nil
}

// ZRevRangeWithScores 获取有序集合范围成员（带分数，从从库读取，按分数从大到小）
func (c *Client) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]ZItem, error) {
	zs, err := c.getSlave().ZRevRangeWithScores(ctx, key, start, stop).Result()
	if err != nil {
		return nil, fmt.Errorf("zrevrange with scores failed: %w", err)
	}

	// 转换为自定义类型
	items := make([]ZItem, len(zs))
	for i, z := range zs {
		items[i] = ZItem{
			Member: z.Member.(string),
			Score:  z.Score,
		}
	}
	return items, nil
}

// ZRangeByScore 按分数范围获取有序集合成员（从从库读取）
func (c *Client) ZRangeByScore(ctx context.Context, key string, min, max string, offset, count int64) ([]string, error) {
	opt := &redis.ZRangeBy{
		Min:    min,
		Max:    max,
		Offset: offset,
		Count:  count,
	}

	vals, err := c.getSlave().ZRangeByScore(ctx, key, opt).Result()
	if err != nil {
		return nil, fmt.Errorf("zrangebyscore failed: %w", err)
	}
	return vals, nil
}

// ZCard 获取有序集合成员数量（从从库读取）
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	n, err := c.getSlave().ZCard(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("zcard failed: %w", err)
	}
	return n, nil
}

// ZScore 获取有序集合成员分数（从从库读取）
func (c *Client) ZScore(ctx context.Context, key, member string) (float64, error) {
	score, err := c.getSlave().ZScore(ctx, key, member).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, ErrNil
		}
		return 0, fmt.Errorf("zscore failed: %w", err)
	}
	return score, nil
}
