package balancer

import "sync"

var (
	mu       sync.RWMutex
	builders = make(map[string]Builder)
)

func init() {
	// 注册内置负载均衡器
	Register(NewRandomBuilder())
	Register(NewRoundRobinBuilder())
	Register(NewWeightedBuilder())
	Register(NewConsistentHashBuilder())
}

// Register 注册负载均衡器构建器
func Register(b Builder) {
	mu.Lock()
	defer mu.Unlock()
	builders[b.Name()] = b
}

// Get 获取负载均衡器构建器
func Get(name string) Builder {
	mu.RLock()
	defer mu.RUnlock()
	return builders[name]
}

// New 创建负载均衡器实例
func New(name string) Balancer {
	b := Get(name)
	if b == nil {
		return nil
	}
	return b.Build()
}
