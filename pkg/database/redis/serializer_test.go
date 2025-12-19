package redis

import (
	"context"
	"reflect"
	"testing"
	"time"
)

// 测试用数据结构
type Player struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level int    `json:"level"`
	Exp   int64  `json:"exp"`
}

type ComplexData struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Tags      []string          `json:"tags"`
	Metadata  map[string]string `json:"metadata"`
	Timestamp time.Time         `json:"timestamp"`
}

// TestSetObjectGetObject 测试对象序列化和反序列化
func TestSetObjectGetObject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:object:player"

	defer client.Del(ctx, key)

	// 测试数据
	player := &Player{
		ID:    "p12345",
		Name:  "Alice",
		Level: 10,
		Exp:   5000,
	}

	// Test SetObject
	if err := SetObject(client, ctx, key, player, 10*time.Second); err != nil {
		t.Errorf("SetObject() error = %v", err)
	}

	// Test GetObject
	got, err := GetObject[Player](client, ctx, key)
	if err != nil {
		t.Errorf("GetObject() error = %v", err)
	}

	if !reflect.DeepEqual(got, player) {
		t.Errorf("GetObject() = %+v, want %+v", got, player)
	}
}

// TestSetObjectGetObject_Complex 测试复杂对象序列化
func TestSetObjectGetObject_Complex(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:object:complex"

	defer client.Del(ctx, key)

	// 测试数据
	now := time.Now().Round(time.Second) // 去除纳秒精度
	data := &ComplexData{
		ID:   "c001",
		Name: "Test Data",
		Tags: []string{"tag1", "tag2", "tag3"},
		Metadata: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Timestamp: now,
	}

	// Test SetObject
	if err := SetObject(client, ctx, key, data, 10*time.Second); err != nil {
		t.Errorf("SetObject() error = %v", err)
	}

	// Test GetObject
	got, err := GetObject[ComplexData](client, ctx, key)
	if err != nil {
		t.Errorf("GetObject() error = %v", err)
	}

	// 比较时间字段（去除纳秒差异）
	got.Timestamp = got.Timestamp.Round(time.Second)

	if !reflect.DeepEqual(got, data) {
		t.Errorf("GetObject() = %+v, want %+v", got, data)
	}
}

// TestGetObject_NotFound 测试获取不存在的对象
func TestGetObject_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:object:notfound"

	// Test GetObject on non-existent key
	_, err = GetObject[Player](client, ctx, key)
	if err != ErrNil {
		t.Errorf("GetObject() error = %v, want ErrNil", err)
	}
}

// TestHSetObjectHGetObject 测试 Hash 对象序列化
func TestHSetObjectHGetObject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:hash:object"
	field := "player1"

	defer client.Del(ctx, key)

	// 测试数据
	player := &Player{
		ID:    "p12345",
		Name:  "Bob",
		Level: 20,
		Exp:   10000,
	}

	// Test HSetObject
	if err := HSetObject(client, ctx, key, field, player); err != nil {
		t.Errorf("HSetObject() error = %v", err)
	}

	// Test HGetObject
	got, err := HGetObject[Player](client, ctx, key, field)
	if err != nil {
		t.Errorf("HGetObject() error = %v", err)
	}

	if !reflect.DeepEqual(got, player) {
		t.Errorf("HGetObject() = %+v, want %+v", got, player)
	}
}

// TestHGetAllObjects 测试批量获取 Hash 对象
func TestHGetAllObjects(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:hash:objects"

	defer client.Del(ctx, key)

	// 测试数据
	players := map[string]*Player{
		"player1": {ID: "p1", Name: "Alice", Level: 10, Exp: 5000},
		"player2": {ID: "p2", Name: "Bob", Level: 20, Exp: 10000},
		"player3": {ID: "p3", Name: "Charlie", Level: 30, Exp: 15000},
	}

	// 设置多个对象
	for field, player := range players {
		if err := HSetObject(client, ctx, key, field, player); err != nil {
			t.Errorf("HSetObject(%s) error = %v", field, err)
		}
	}

	// Test HGetAllObjects
	got, err := HGetAllObjects[Player](client, ctx, key)
	if err != nil {
		t.Errorf("HGetAllObjects() error = %v", err)
	}

	if len(got) != len(players) {
		t.Errorf("HGetAllObjects() returned %v objects, want %v", len(got), len(players))
	}

	// 验证每个对象
	for field, want := range players {
		gotPlayer, ok := got[field]
		if !ok {
			t.Errorf("HGetAllObjects() missing field %s", field)
			continue
		}
		if !reflect.DeepEqual(gotPlayer, want) {
			t.Errorf("HGetAllObjects()[%s] = %+v, want %+v", field, gotPlayer, want)
		}
	}
}

// TestObjectSerialization_EmptySlice 测试空切片序列化
func TestObjectSerialization_EmptySlice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:object:emptyslice"

	defer client.Del(ctx, key)

	// 测试空切片
	data := &ComplexData{
		ID:       "c001",
		Name:     "Empty Tags",
		Tags:     []string{},
		Metadata: map[string]string{},
	}

	if err := SetObject(client, ctx, key, data, 10*time.Second); err != nil {
		t.Errorf("SetObject() error = %v", err)
	}

	got, err := GetObject[ComplexData](client, ctx, key)
	if err != nil {
		t.Errorf("GetObject() error = %v", err)
	}

	// 空切片和 nil 在 JSON 序列化后可能不同
	// 确保至少长度为 0
	if got.Tags == nil || len(got.Tags) != 0 {
		t.Errorf("GetObject().Tags should be empty slice, got %v", got.Tags)
	}
}

// BenchmarkSetObject Benchmark 对象序列化写入
func BenchmarkSetObject(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "bench:object"

	player := &Player{
		ID:    "p12345",
		Name:  "Alice",
		Level: 10,
		Exp:   5000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := SetObject(client, ctx, key, player, 10*time.Second); err != nil {
			b.Errorf("SetObject() error = %v", err)
		}
	}
}

// BenchmarkGetObject Benchmark 对象反序列化读取
func BenchmarkGetObject(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "bench:object"

	player := &Player{
		ID:    "p12345",
		Name:  "Alice",
		Level: 10,
		Exp:   5000,
	}

	// 预先设置数据
	if err := SetObject(client, ctx, key, player, 10*time.Minute); err != nil {
		b.Fatalf("SetObject() error = %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetObject[Player](client, ctx, key)
		if err != nil {
			b.Errorf("GetObject() error = %v", err)
		}
	}
}
