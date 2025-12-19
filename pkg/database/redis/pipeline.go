package redis

import (
	"context"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Pipeline Redis Pipeline（批量操作，减少网络往返）
type Pipeline struct {
	client   *Client
	pipeliner goredis.Pipeliner
}

// Pipeline 创建 Pipeline 实例
func (c *Client) Pipeline() *Pipeline {
	return &Pipeline{
		client:    c,
		pipeliner: c.getMaster().Pipeline(),
	}
}

// ==================== String 操作 ====================

// Set 添加 Set 命令到 Pipeline
func (p *Pipeline) Set(key string, value interface{}, expiration time.Duration) *Pipeline {
	p.pipeliner.Set(context.Background(), key, value, expiration)
	return p
}

// Get 添加 Get 命令到 Pipeline
func (p *Pipeline) Get(key string) *Pipeline {
	p.pipeliner.Get(context.Background(), key)
	return p
}

// Del 添加 Del 命令到 Pipeline
func (p *Pipeline) Del(keys ...string) *Pipeline {
	p.pipeliner.Del(context.Background(), keys...)
	return p
}

// Incr 添加 Incr 命令到 Pipeline
func (p *Pipeline) Incr(key string) *Pipeline {
	p.pipeliner.Incr(context.Background(), key)
	return p
}

// IncrBy 添加 IncrBy 命令到 Pipeline
func (p *Pipeline) IncrBy(key string, value int64) *Pipeline {
	p.pipeliner.IncrBy(context.Background(), key, value)
	return p
}

// ==================== Hash 操作 ====================

// HSet 添加 HSet 命令到 Pipeline
func (p *Pipeline) HSet(key string, values ...interface{}) *Pipeline {
	p.pipeliner.HSet(context.Background(), key, values...)
	return p
}

// HGet 添加 HGet 命令到 Pipeline
func (p *Pipeline) HGet(key, field string) *Pipeline {
	p.pipeliner.HGet(context.Background(), key, field)
	return p
}

// HGetAll 添加 HGetAll 命令到 Pipeline
func (p *Pipeline) HGetAll(key string) *Pipeline {
	p.pipeliner.HGetAll(context.Background(), key)
	return p
}

// HDel 添加 HDel 命令到 Pipeline
func (p *Pipeline) HDel(key string, fields ...string) *Pipeline {
	p.pipeliner.HDel(context.Background(), key, fields...)
	return p
}

// ==================== List 操作 ====================

// LPush 添加 LPush 命令到 Pipeline
func (p *Pipeline) LPush(key string, values ...interface{}) *Pipeline {
	p.pipeliner.LPush(context.Background(), key, values...)
	return p
}

// RPush 添加 RPush 命令到 Pipeline
func (p *Pipeline) RPush(key string, values ...interface{}) *Pipeline {
	p.pipeliner.RPush(context.Background(), key, values...)
	return p
}

// LPop 添加 LPop 命令到 Pipeline
func (p *Pipeline) LPop(key string) *Pipeline {
	p.pipeliner.LPop(context.Background(), key)
	return p
}

// RPop 添加 RPop 命令到 Pipeline
func (p *Pipeline) RPop(key string) *Pipeline {
	p.pipeliner.RPop(context.Background(), key)
	return p
}

// LRange 添加 LRange 命令到 Pipeline
func (p *Pipeline) LRange(key string, start, stop int64) *Pipeline {
	p.pipeliner.LRange(context.Background(), key, start, stop)
	return p
}

// ==================== Set 操作 ====================

// SAdd 添加 SAdd 命令到 Pipeline
func (p *Pipeline) SAdd(key string, members ...interface{}) *Pipeline {
	p.pipeliner.SAdd(context.Background(), key, members...)
	return p
}

// SRem 添加 SRem 命令到 Pipeline
func (p *Pipeline) SRem(key string, members ...interface{}) *Pipeline {
	p.pipeliner.SRem(context.Background(), key, members...)
	return p
}

// SMembers 添加 SMembers 命令到 Pipeline
func (p *Pipeline) SMembers(key string) *Pipeline {
	p.pipeliner.SMembers(context.Background(), key)
	return p
}

// ==================== Sorted Set 操作 ====================

// ZAdd 添加 ZAdd 命令到 Pipeline
func (p *Pipeline) ZAdd(key string, members ...ZItem) *Pipeline {
	// 转换为 go-redis 类型
	zs := make([]goredis.Z, len(members))
	for i, m := range members {
		zs[i] = goredis.Z{
			Score:  m.Score,
			Member: m.Member,
		}
	}
	p.pipeliner.ZAdd(context.Background(), key, zs...)
	return p
}

// ZRem 添加 ZRem 命令到 Pipeline
func (p *Pipeline) ZRem(key string, members ...interface{}) *Pipeline {
	p.pipeliner.ZRem(context.Background(), key, members...)
	return p
}

// ZRange 添加 ZRange 命令到 Pipeline
func (p *Pipeline) ZRange(key string, start, stop int64) *Pipeline {
	p.pipeliner.ZRange(context.Background(), key, start, stop)
	return p
}

// ZRevRange 添加 ZRevRange 命令到 Pipeline
func (p *Pipeline) ZRevRange(key string, start, stop int64) *Pipeline {
	p.pipeliner.ZRevRange(context.Background(), key, start, stop)
	return p
}

// ==================== 执行 Pipeline ====================

// Exec 执行 Pipeline（提交所有命令）
func (p *Pipeline) Exec(ctx context.Context) ([]interface{}, error) {
	cmders, err := p.pipeliner.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("pipeline exec failed: %w", err)
	}

	// 提取结果
	results := make([]interface{}, len(cmders))
	for i, cmd := range cmders {
		// 获取命令的结果（可能是不同类型）
		switch c := cmd.(type) {
		case *goredis.StringCmd:
			val, err := c.Result()
			if err != nil {
				results[i] = err
			} else {
				results[i] = val
			}
		case *goredis.IntCmd:
			val, err := c.Result()
			if err != nil {
				results[i] = err
			} else {
				results[i] = val
			}
		case *goredis.StatusCmd:
			val, err := c.Result()
			if err != nil {
				results[i] = err
			} else {
				results[i] = val
			}
		case *goredis.StringSliceCmd:
			val, err := c.Result()
			if err != nil {
				results[i] = err
			} else {
				results[i] = val
			}
		case *goredis.MapStringStringCmd:
			val, err := c.Result()
			if err != nil {
				results[i] = err
			} else {
				results[i] = val
			}
		default:
			results[i] = cmd
		}
	}

	return results, nil
}

// Discard 丢弃 Pipeline（不执行任何命令）
func (p *Pipeline) Discard() {
	p.pipeliner.Discard()
}

// Pipelined 在 Pipeline 中执行函数（自动提交）
func (c *Client) Pipelined(ctx context.Context, fn func(*Pipeline) error) ([]interface{}, error) {
	pipe := c.Pipeline()

	// 执行函数
	if err := fn(pipe); err != nil {
		pipe.Discard()
		return nil, fmt.Errorf("pipelined function failed: %w", err)
	}

	// 提交 Pipeline
	results, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	return results, nil
}
