package redis

import (
	"context"
	"testing"
)

// TestPipeline 测试 Pipeline 基本操作
func TestPipeline(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	keys := []string{"pipe:key1", "pipe:key2", "pipe:key3"}

	defer client.Del(ctx, keys...)

	// 创建 Pipeline
	pipe := client.Pipeline()

	// 添加多个命令
	pipe.Set("pipe:key1", "value1", 0)
	pipe.Set("pipe:key2", "value2", 0)
	pipe.Set("pipe:key3", "value3", 0)
	pipe.Incr("pipe:counter")
	pipe.Incr("pipe:counter")

	// 执行 Pipeline
	results, err := pipe.Exec(ctx)
	if err != nil {
		t.Errorf("Exec() error = %v", err)
	}

	if len(results) != 5 {
		t.Errorf("Exec() returned %v results, want 5", len(results))
	}

	// 验证数据
	val, err := client.Get(ctx, "pipe:key1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if val != "value1" {
		t.Errorf("Get(pipe:key1) = %v, want value1", val)
	}
}

// TestPipelined 测试 Pipelined 便捷方法
func TestPipelined(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	keys := []string{"pipelined:key1", "pipelined:key2"}

	defer client.Del(ctx, keys...)

	// 使用 Pipelined
	results, err := client.Pipelined(ctx, func(p *Pipeline) error {
		p.Set("pipelined:key1", "value1", 0)
		p.Set("pipelined:key2", "value2", 0)
		p.Incr("pipelined:counter")
		return nil
	})

	if err != nil {
		t.Errorf("Pipelined() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Pipelined() returned %v results, want 3", len(results))
	}

	// 验证数据
	val, err := client.Get(ctx, "pipelined:key1")
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if val != "value1" {
		t.Errorf("Get(pipelined:key1) = %v, want value1", val)
	}
}

// TestPipelineHash 测试 Pipeline Hash 操作
func TestPipelineHash(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "pipe:hash"

	defer client.Del(ctx, key)

	pipe := client.Pipeline()
	pipe.HSet(key, "field1", "value1")
	pipe.HSet(key, "field2", "value2")
	pipe.HGet(key, "field1")

	results, err := pipe.Exec(ctx)
	if err != nil {
		t.Errorf("Exec() error = %v", err)
	}

	if len(results) != 3 {
		t.Errorf("Exec() returned %v results, want 3", len(results))
	}
}

// TestPipelineZSet 测试 Pipeline Sorted Set 操作
func TestPipelineZSet(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "pipe:zset"

	defer client.Del(ctx, key)

	pipe := client.Pipeline()
	pipe.ZAdd(key,
		ZItem{Member: "player1", Score: 100},
		ZItem{Member: "player2", Score: 95},
	)
	pipe.ZRange(key, 0, -1)

	results, err := pipe.Exec(ctx)
	if err != nil {
		t.Errorf("Exec() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Exec() returned %v results, want 2", len(results))
	}
}

// BenchmarkPipeline Benchmark Pipeline 操作
func BenchmarkPipeline(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pipe := client.Pipeline()
		pipe.Set("bench:key1", "value1", 0)
		pipe.Set("bench:key2", "value2", 0)
		pipe.Set("bench:key3", "value3", 0)
		_, err := pipe.Exec(ctx)
		if err != nil {
			b.Errorf("Exec() error = %v", err)
		}
	}
}

// BenchmarkPipelined Benchmark Pipelined 便捷方法
func BenchmarkPipelined(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.Pipelined(ctx, func(p *Pipeline) error {
			p.Set("bench:key1", "value1", 0)
			p.Set("bench:key2", "value2", 0)
			p.Set("bench:key3", "value3", 0)
			return nil
		})
		if err != nil {
			b.Errorf("Pipelined() error = %v", err)
		}
	}
}
