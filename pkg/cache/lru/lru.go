package lru

import (
	"container/list"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// Cache 通用缓存接口
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	SetWithTTL(key K, value V, ttl time.Duration)
	Delete(key K)
	Len() int
	Clear()
	Close() error
}

// Config LRU 配置
type Config struct {
	// MaxSize 最大容量
	MaxSize int
	// DefaultTTL 默认过期时间
	DefaultTTL time.Duration
	// CleanupInterval 清理间隔
	CleanupInterval time.Duration
}

// LRU 基于内存的 LRU 缓存实现
type LRU[K comparable, V any] struct {
	config *Config
	cache  *list.List
	items  map[K]*list.Element
	mu     sync.RWMutex
	pool   *conc.Pool[struct{}]
	stopCh chan struct{}

	onEvict func(key K, value V)
}

type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
}

// Option LRU 配置选项
type Option[K comparable, V any] func(*LRU[K, V])

// WithOnEvict 设置淘汰回调
func WithOnEvict[K comparable, V any](fn func(key K, value V)) Option[K, V] {
	return func(c *LRU[K, V]) {
		c.onEvict = fn
	}
}

// New 创建 LRU 缓存
func New[K comparable, V any](cfg *Config, opts ...Option[K, V]) *LRU[K, V] {
	c := &LRU[K, V]{
		config: cfg,
		cache:  list.New(),
		items:  make(map[K]*list.Element),
		pool:   conc.NewPool[struct{}](1),
		stopCh: make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	c.startCleanup()
	return c
}

// startCleanup 启动后台清理协程
func (c *LRU[K, V]) startCleanup() {
	c.pool.Submit(func() (struct{}, error) {
		ticker := time.NewTicker(c.config.CleanupInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				c.removeExpired()
			case <-c.stopCh:
				return struct{}{}, nil
			}
		}
	})
}

// removeExpired 移除过期条目
func (c *LRU[K, V]) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for e := c.cache.Back(); e != nil; {
		prev := e.Prev()
		ent := e.Value.(*entry[K, V])
		if now.After(ent.expiresAt) {
			c.removeElement(e)
		}
		e = prev
	}
}

// Get 获取值
func (c *LRU[K, V]) Get(key K) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		ent := elem.Value.(*entry[K, V])
		if time.Now().After(ent.expiresAt) {
			c.removeElement(elem)
			var zero V
			return zero, false
		}
		c.cache.MoveToFront(elem)
		return ent.value, true
	}

	var zero V
	return zero, false
}

// Set 设置值（使用默认 TTL）
func (c *LRU[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.config.DefaultTTL)
}

// SetWithTTL 设置值（自定义 TTL）
func (c *LRU[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := time.Now().Add(ttl)

	if elem, ok := c.items[key]; ok {
		c.cache.MoveToFront(elem)
		ent := elem.Value.(*entry[K, V])
		ent.value = value
		ent.expiresAt = expiresAt
		return
	}

	ent := &entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}
	elem := c.cache.PushFront(ent)
	c.items[key] = elem

	for c.cache.Len() > c.config.MaxSize {
		c.removeOldest()
	}
}

// GetOrCreate 原子获取或创建
func (c *LRU[K, V]) GetOrCreate(key K, create func() V) V {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		ent := elem.Value.(*entry[K, V])
		if time.Now().Before(ent.expiresAt) {
			c.cache.MoveToFront(elem)
			return ent.value
		}
		c.removeElement(elem)
	}

	value := create()
	expiresAt := time.Now().Add(c.config.DefaultTTL)

	ent := &entry[K, V]{
		key:       key,
		value:     value,
		expiresAt: expiresAt,
	}
	elem := c.cache.PushFront(ent)
	c.items[key] = elem

	for c.cache.Len() > c.config.MaxSize {
		c.removeOldest()
	}

	return value
}

// Delete 删除
func (c *LRU[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.items[key]; ok {
		c.removeElement(elem)
	}
}

// Len 返回当前缓存大小
func (c *LRU[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cache.Len()
}

// Clear 清空缓存
func (c *LRU[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache.Init()
	c.items = make(map[K]*list.Element)
}

// Close 关闭缓存
func (c *LRU[K, V]) Close() error {
	close(c.stopCh)
	c.pool.Release()
	return nil
}

// removeOldest 移除最老的条目
func (c *LRU[K, V]) removeOldest() {
	elem := c.cache.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement 移除元素
func (c *LRU[K, V]) removeElement(elem *list.Element) {
	c.cache.Remove(elem)
	ent := elem.Value.(*entry[K, V])
	delete(c.items, ent.key)
	if c.onEvict != nil {
		c.onEvict(ent.key, ent.value)
	}
}
