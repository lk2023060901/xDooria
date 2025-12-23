// pkg/checksum/checksum.go
package checksum

import (
	"fmt"
	"sync"
)

// Hasher 校验和计算器接口
type Hasher interface {
	// Sum 计算数据的校验和
	Sum(data []byte) uint32

	// Verify 验证数据的校验和
	Verify(data []byte, expected uint32) bool

	// Name 返回校验算法名称
	Name() string
}

// Factory 校验器工厂函数类型
type Factory func() (Hasher, error)

// Type 校验算法类型
type Type string

const (
	// TypeCRC32 CRC32 校验算法 (IEEE 多项式)
	TypeCRC32 Type = "crc32"
	// TypeCRC32C CRC32C 校验算法 (Castagnoli 多项式，硬件加速)
	TypeCRC32C Type = "crc32c"
	// TypeXXHash XXHash 校验算法（高性能）
	TypeXXHash Type = "xxhash"
)

var (
	mu        sync.RWMutex
	factories = make(map[Type]Factory)
)

func init() {
	// 注册默认支持的校验算法
	Register(TypeCRC32, func() (Hasher, error) {
		return newCRC32Hasher(), nil
	})
	Register(TypeCRC32C, func() (Hasher, error) {
		return newCRC32CHasher(), nil
	})
	Register(TypeXXHash, func() (Hasher, error) {
		return newXXHashHasher(), nil
	})
}

// Register 注册校验器工厂
func Register(t Type, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[t] = factory
}

// Unregister 注销校验器工厂
func Unregister(t Type) {
	mu.Lock()
	defer mu.Unlock()
	delete(factories, t)
}

// New 创建校验器
func New(t Type) (Hasher, error) {
	mu.RLock()
	factory, ok := factories[t]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported checksum type: %s", t)
	}
	return factory()
}

// MustNew 创建校验器，失败时 panic
func MustNew(t Type) Hasher {
	h, err := New(t)
	if err != nil {
		panic(err)
	}
	return h
}

// List 返回所有已注册的校验算法类型
func List() []Type {
	mu.RLock()
	defer mu.RUnlock()

	types := make([]Type, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

// IsRegistered 检查校验算法是否已注册
func IsRegistered(t Type) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := factories[t]
	return ok
}

// Default 返回默认校验器 (CRC32C，支持硬件加速)
func Default() Hasher {
	return MustNew(TypeCRC32C)
}
