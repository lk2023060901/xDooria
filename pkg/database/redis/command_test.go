package redis

import (
	"context"
	"testing"
	"time"
)

// TestStringCommands 测试 String 操作
func TestStringCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:string:key"
	value := "hello world"

	// 清理测试数据
	defer client.Del(ctx, key)

	// Test Set
	if err := client.Set(ctx, key, value, 10*time.Second); err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// Test Get
	got, err := client.Get(ctx, key)
	if err != nil {
		t.Errorf("Get() error = %v", err)
	}
	if got != value {
		t.Errorf("Get() = %v, want %v", got, value)
	}

	// Test SetNX (should fail, key exists)
	ok, err := client.SetNX(ctx, key, "new value", 10*time.Second)
	if err != nil {
		t.Errorf("SetNX() error = %v", err)
	}
	if ok {
		t.Error("SetNX() should return false for existing key")
	}

	// Test SetEX
	if err := client.SetEX(ctx, key, "updated", 5*time.Second); err != nil {
		t.Errorf("SetEX() error = %v", err)
	}

	// Test Exists
	count, err := client.Exists(ctx, key)
	if err != nil {
		t.Errorf("Exists() error = %v", err)
	}
	if count != 1 {
		t.Errorf("Exists() = %v, want 1", count)
	}

	// Test TTL
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Errorf("TTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 5*time.Second {
		t.Errorf("TTL() = %v, want > 0 and <= 5s", ttl)
	}

	// Test Incr/Decr
	counterKey := "test:counter"
	defer client.Del(ctx, counterKey)

	val, err := client.Incr(ctx, counterKey)
	if err != nil {
		t.Errorf("Incr() error = %v", err)
	}
	if val != 1 {
		t.Errorf("Incr() = %v, want 1", val)
	}

	val, err = client.IncrBy(ctx, counterKey, 5)
	if err != nil {
		t.Errorf("IncrBy() error = %v", err)
	}
	if val != 6 {
		t.Errorf("IncrBy() = %v, want 6", val)
	}

	val, err = client.Decr(ctx, counterKey)
	if err != nil {
		t.Errorf("Decr() error = %v", err)
	}
	if val != 5 {
		t.Errorf("Decr() = %v, want 5", val)
	}

	// Test Del
	deleted, err := client.Del(ctx, key, counterKey)
	if err != nil {
		t.Errorf("Del() error = %v", err)
	}
	if deleted != 2 {
		t.Errorf("Del() = %v, want 2", deleted)
	}
}

// TestHashCommands 测试 Hash 操作
func TestHashCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:hash"

	defer client.Del(ctx, key)

	// Test HSet
	count, err := client.HSet(ctx, key, "field1", "value1", "field2", "value2")
	if err != nil {
		t.Errorf("HSet() error = %v", err)
	}
	if count != 2 {
		t.Errorf("HSet() = %v, want 2", count)
	}

	// Test HGet
	val, err := client.HGet(ctx, key, "field1")
	if err != nil {
		t.Errorf("HGet() error = %v", err)
	}
	if val != "value1" {
		t.Errorf("HGet() = %v, want value1", val)
	}

	// Test HGetAll
	all, err := client.HGetAll(ctx, key)
	if err != nil {
		t.Errorf("HGetAll() error = %v", err)
	}
	if len(all) != 2 {
		t.Errorf("HGetAll() returned %v fields, want 2", len(all))
	}

	// Test HExists
	exists, err := client.HExists(ctx, key, "field1")
	if err != nil {
		t.Errorf("HExists() error = %v", err)
	}
	if !exists {
		t.Error("HExists() = false, want true")
	}

	// Test HLen
	length, err := client.HLen(ctx, key)
	if err != nil {
		t.Errorf("HLen() error = %v", err)
	}
	if length != 2 {
		t.Errorf("HLen() = %v, want 2", length)
	}

	// Test HDel
	deleted, err := client.HDel(ctx, key, "field1")
	if err != nil {
		t.Errorf("HDel() error = %v", err)
	}
	if deleted != 1 {
		t.Errorf("HDel() = %v, want 1", deleted)
	}
}

// TestListCommands 测试 List 操作
func TestListCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:list"

	defer client.Del(ctx, key)

	// Test RPush
	length, err := client.RPush(ctx, key, "item1", "item2", "item3")
	if err != nil {
		t.Errorf("RPush() error = %v", err)
	}
	if length != 3 {
		t.Errorf("RPush() = %v, want 3", length)
	}

	// Test LPush
	length, err = client.LPush(ctx, key, "item0")
	if err != nil {
		t.Errorf("LPush() error = %v", err)
	}
	if length != 4 {
		t.Errorf("LPush() = %v, want 4", length)
	}

	// Test LLen
	length, err = client.LLen(ctx, key)
	if err != nil {
		t.Errorf("LLen() error = %v", err)
	}
	if length != 4 {
		t.Errorf("LLen() = %v, want 4", length)
	}

	// Test LRange
	items, err := client.LRange(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("LRange() error = %v", err)
	}
	if len(items) != 4 {
		t.Errorf("LRange() returned %v items, want 4", len(items))
	}

	// Test LPop
	val, err := client.LPop(ctx, key)
	if err != nil {
		t.Errorf("LPop() error = %v", err)
	}
	if val != "item0" {
		t.Errorf("LPop() = %v, want item0", val)
	}

	// Test RPop
	val, err = client.RPop(ctx, key)
	if err != nil {
		t.Errorf("RPop() error = %v", err)
	}
	if val != "item3" {
		t.Errorf("RPop() = %v, want item3", val)
	}
}

// TestSetCommands 测试 Set 操作
func TestSetCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:set"

	defer client.Del(ctx, key)

	// Test SAdd
	count, err := client.SAdd(ctx, key, "member1", "member2", "member3")
	if err != nil {
		t.Errorf("SAdd() error = %v", err)
	}
	if count != 3 {
		t.Errorf("SAdd() = %v, want 3", count)
	}

	// Test SCard
	size, err := client.SCard(ctx, key)
	if err != nil {
		t.Errorf("SCard() error = %v", err)
	}
	if size != 3 {
		t.Errorf("SCard() = %v, want 3", size)
	}

	// Test SIsMember
	isMember, err := client.SIsMember(ctx, key, "member1")
	if err != nil {
		t.Errorf("SIsMember() error = %v", err)
	}
	if !isMember {
		t.Error("SIsMember() = false, want true")
	}

	// Test SMembers
	members, err := client.SMembers(ctx, key)
	if err != nil {
		t.Errorf("SMembers() error = %v", err)
	}
	if len(members) != 3 {
		t.Errorf("SMembers() returned %v members, want 3", len(members))
	}

	// Test SRem
	removed, err := client.SRem(ctx, key, "member1")
	if err != nil {
		t.Errorf("SRem() error = %v", err)
	}
	if removed != 1 {
		t.Errorf("SRem() = %v, want 1", removed)
	}
}

// TestSortedSetCommands 测试 Sorted Set 操作
func TestSortedSetCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:zset"

	defer client.Del(ctx, key)

	// Test ZAdd
	count, err := client.ZAdd(ctx, key,
		ZItem{Member: "player1", Score: 100},
		ZItem{Member: "player2", Score: 95},
		ZItem{Member: "player3", Score: 90},
	)
	if err != nil {
		t.Errorf("ZAdd() error = %v", err)
	}
	if count != 3 {
		t.Errorf("ZAdd() = %v, want 3", count)
	}

	// Test ZCard
	size, err := client.ZCard(ctx, key)
	if err != nil {
		t.Errorf("ZCard() error = %v", err)
	}
	if size != 3 {
		t.Errorf("ZCard() = %v, want 3", size)
	}

	// Test ZScore
	score, err := client.ZScore(ctx, key, "player1")
	if err != nil {
		t.Errorf("ZScore() error = %v", err)
	}
	if score != 100 {
		t.Errorf("ZScore() = %v, want 100", score)
	}

	// Test ZRange
	members, err := client.ZRange(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("ZRange() error = %v", err)
	}
	if len(members) != 3 {
		t.Errorf("ZRange() returned %v members, want 3", len(members))
	}
	// 检查顺序（从小到大）
	if len(members) > 0 && members[0] != "player3" {
		t.Errorf("ZRange()[0] = %v, want player3", members[0])
	}

	// Test ZRevRange
	members, err = client.ZRevRange(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("ZRevRange() error = %v", err)
	}
	// 检查顺序（从大到小）
	if len(members) > 0 && members[0] != "player1" {
		t.Errorf("ZRevRange()[0] = %v, want player1", members[0])
	}

	// Test ZRangeWithScores
	items, err := client.ZRangeWithScores(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("ZRangeWithScores() error = %v", err)
	}
	if len(items) != 3 {
		t.Errorf("ZRangeWithScores() returned %v items, want 3", len(items))
	}

	// Test ZRevRangeWithScores
	items, err = client.ZRevRangeWithScores(ctx, key, 0, -1)
	if err != nil {
		t.Errorf("ZRevRangeWithScores() error = %v", err)
	}
	if len(items) == 0 {
		t.Errorf("ZRevRangeWithScores() returned empty items")
	} else if items[0].Score != 100 {
		t.Errorf("ZRevRangeWithScores()[0].Score = %v, want 100", items[0].Score)
	}

	// Test ZRangeByScore
	members, err = client.ZRangeByScore(ctx, key, "90", "100", 0, -1)
	if err != nil {
		t.Errorf("ZRangeByScore() error = %v", err)
	}
	if len(members) != 3 {
		t.Errorf("ZRangeByScore() returned %v members, want 3", len(members))
	}

	// Test ZRem
	removed, err := client.ZRem(ctx, key, "player1")
	if err != nil {
		t.Errorf("ZRem() error = %v", err)
	}
	if removed != 1 {
		t.Errorf("ZRem() = %v, want 1", removed)
	}
}

// TestExpireCommands 测试过期时间相关命令
func TestExpireCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:expire"

	defer client.Del(ctx, key)

	// 设置键
	if err := client.Set(ctx, key, "value", 0); err != nil {
		t.Errorf("Set() error = %v", err)
	}

	// 设置过期时间
	ok, err := client.Expire(ctx, key, 10*time.Second)
	if err != nil {
		t.Errorf("Expire() error = %v", err)
	}
	if !ok {
		t.Error("Expire() = false, want true")
	}

	// 检查 TTL
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Errorf("TTL() error = %v", err)
	}
	if ttl <= 0 || ttl > 10*time.Second {
		t.Errorf("TTL() = %v, want > 0 and <= 10s", ttl)
	}
}
