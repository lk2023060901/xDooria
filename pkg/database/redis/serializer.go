package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// GetObject 获取对象（从从库读取，自动反序列化 JSON）
func GetObject[T any](c *Client, ctx context.Context, key string) (*T, error) {
	val, err := c.getSlave().Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, ErrNil
		}
		return nil, fmt.Errorf("get object failed: %w", err)
	}

	var obj T
	if err := json.Unmarshal([]byte(val), &obj); err != nil {
		return nil, fmt.Errorf("unmarshal object failed: %w", err)
	}

	return &obj, nil
}

// SetObject 设置对象（写入主库，自动序列化为 JSON）
func SetObject(c *Client, ctx context.Context, key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal object failed: %w", err)
	}

	if err := c.getMaster().Set(ctx, key, data, expiration).Err(); err != nil {
		return fmt.Errorf("set object failed: %w", err)
	}

	return nil
}

// HGetObject 获取哈希字段对象（从从库读取，自动反序列化 JSON）
func HGetObject[T any](c *Client, ctx context.Context, key, field string) (*T, error) {
	val, err := c.getSlave().HGet(ctx, key, field).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return nil, ErrNil
		}
		return nil, fmt.Errorf("hget object failed: %w", err)
	}

	var obj T
	if err := json.Unmarshal([]byte(val), &obj); err != nil {
		return nil, fmt.Errorf("unmarshal object failed: %w", err)
	}

	return &obj, nil
}

// HSetObject 设置哈希字段对象（写入主库，自动序列化为 JSON）
func HSetObject(c *Client, ctx context.Context, key, field string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("marshal object failed: %w", err)
	}

	if err := c.getMaster().HSet(ctx, key, field, data).Err(); err != nil {
		return fmt.Errorf("hset object failed: %w", err)
	}

	return nil
}

// HGetAllObjects 获取哈希所有字段对象（从从库读取，自动反序列化 JSON）
func HGetAllObjects[T any](c *Client, ctx context.Context, key string) (map[string]*T, error) {
	vals, err := c.getSlave().HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("hgetall objects failed: %w", err)
	}

	results := make(map[string]*T)
	for field, val := range vals {
		var obj T
		if err := json.Unmarshal([]byte(val), &obj); err != nil {
			return nil, fmt.Errorf("unmarshal object at field %s failed: %w", field, err)
		}
		results[field] = &obj
	}

	return results, nil
}
