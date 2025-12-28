package bt

import "sync"

// Blackboard 黑板，用于节点间共享数据
type Blackboard struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewBlackboard 创建黑板
func NewBlackboard() *Blackboard {
	return &Blackboard{
		data: make(map[string]interface{}),
	}
}

// Set 设置数据
func (bb *Blackboard) Set(key string, value interface{}) {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	bb.data[key] = value
}

// Get 获取数据
func (bb *Blackboard) Get(key string) (interface{}, bool) {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	val, ok := bb.data[key]
	return val, ok
}

// GetString 获取字符串
func (bb *Blackboard) GetString(key string) (string, bool) {
	val, ok := bb.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt 获取整数
func (bb *Blackboard) GetInt(key string) (int, bool) {
	val, ok := bb.Get(key)
	if !ok {
		return 0, false
	}
	i, ok := val.(int)
	return i, ok
}

// GetInt64 获取 int64
func (bb *Blackboard) GetInt64(key string) (int64, bool) {
	val, ok := bb.Get(key)
	if !ok {
		return 0, false
	}
	i, ok := val.(int64)
	return i, ok
}

// GetBool 获取布尔值
func (bb *Blackboard) GetBool(key string) (bool, bool) {
	val, ok := bb.Get(key)
	if !ok {
		return false, false
	}
	b, ok := val.(bool)
	return b, ok
}

// Has 检查是否存在
func (bb *Blackboard) Has(key string) bool {
	bb.mu.RLock()
	defer bb.mu.RUnlock()
	_, ok := bb.data[key]
	return ok
}

// Delete 删除数据
func (bb *Blackboard) Delete(key string) {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	delete(bb.data, key)
}

// Clear 清空黑板
func (bb *Blackboard) Clear() {
	bb.mu.Lock()
	defer bb.mu.Unlock()
	bb.data = make(map[string]interface{})
}
