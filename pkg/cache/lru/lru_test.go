package lru

import (
	"testing"
	"time"
)

func TestLRU_Basic(t *testing.T) {
	cache := New[string, int](&Config{
		MaxSize:         100,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Second,
	})
	defer cache.Close()

	// Test Set and Get
	cache.Set("key1", 100)
	val, ok := cache.Get("key1")
	if !ok {
		t.Error("expected key1 to exist")
	}
	if val != 100 {
		t.Errorf("expected 100, got %d", val)
	}

	// Test non-existent key
	_, ok = cache.Get("nonexistent")
	if ok {
		t.Error("expected nonexistent key to not exist")
	}

	// Test Delete
	cache.Delete("key1")
	_, ok = cache.Get("key1")
	if ok {
		t.Error("expected key1 to be deleted")
	}

	// Test Len
	cache.Set("a", 1)
	cache.Set("b", 2)
	if cache.Len() != 2 {
		t.Errorf("expected len 2, got %d", cache.Len())
	}

	// Test Clear
	cache.Clear()
	if cache.Len() != 0 {
		t.Errorf("expected len 0 after clear, got %d", cache.Len())
	}
}

func TestLRU_MaxSize(t *testing.T) {
	cache := New[string, int](&Config{
		MaxSize:         3,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Second,
	})
	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3)
	cache.Set("d", 4) // should evict "a"

	if cache.Len() != 3 {
		t.Errorf("expected len 3, got %d", cache.Len())
	}

	_, ok := cache.Get("a")
	if ok {
		t.Error("expected 'a' to be evicted")
	}

	_, ok = cache.Get("d")
	if !ok {
		t.Error("expected 'd' to exist")
	}
}

func TestLRU_TTL(t *testing.T) {
	cache := New[string, int](&Config{
		MaxSize:         100,
		DefaultTTL:      50 * time.Millisecond,
		CleanupInterval: 10 * time.Millisecond,
	})
	defer cache.Close()

	cache.Set("key1", 100)

	// Should exist immediately
	_, ok := cache.Get("key1")
	if !ok {
		t.Error("expected key1 to exist immediately")
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, ok = cache.Get("key1")
	if ok {
		t.Error("expected key1 to be expired")
	}
}

func TestLRU_GetOrCreate(t *testing.T) {
	cache := New[string, int](&Config{
		MaxSize:         100,
		DefaultTTL:      time.Minute,
		CleanupInterval: time.Second,
	})
	defer cache.Close()

	callCount := 0
	create := func() int {
		callCount++
		return 42
	}

	// First call should create
	val := cache.GetOrCreate("key", create)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
	if callCount != 1 {
		t.Errorf("expected 1 call, got %d", callCount)
	}

	// Second call should return cached value
	val = cache.GetOrCreate("key", create)
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
	if callCount != 1 {
		t.Errorf("expected still 1 call, got %d", callCount)
	}
}

func TestLRU_OnEvict(t *testing.T) {
	evicted := make(map[string]int)
	cache := New[string, int](
		&Config{
			MaxSize:         2,
			DefaultTTL:      time.Minute,
			CleanupInterval: time.Second,
		},
		WithOnEvict(func(key string, value int) {
			evicted[key] = value
		}),
	)
	defer cache.Close()

	cache.Set("a", 1)
	cache.Set("b", 2)
	cache.Set("c", 3) // should evict "a"

	if len(evicted) != 1 {
		t.Errorf("expected 1 eviction, got %d", len(evicted))
	}
	if evicted["a"] != 1 {
		t.Error("expected 'a' to be evicted with value 1")
	}
}
