// pkg/compress/compress.go
package compress

import (
	"fmt"
	"sync"
)

// Compressor 压缩器接口
type Compressor interface {
	// Compress 压缩数据
	Compress(src []byte) ([]byte, error)

	// Decompress 解压数据
	Decompress(src []byte) ([]byte, error)

	// Name 返回压缩算法名称
	Name() string
}

// Factory 压缩器工厂函数类型
type Factory func() (Compressor, error)

// Type 压缩算法类型
type Type string

const (
	// TypeNone 不压缩
	TypeNone Type = "none"
	// TypeSnappy Snappy 压缩算法
	TypeSnappy Type = "snappy"
	// TypeZstd Zstd 压缩算法
	TypeZstd Type = "zstd"
	// TypeLZ4 LZ4 压缩算法
	TypeLZ4 Type = "lz4"
)

var (
	mu        sync.RWMutex
	factories = make(map[Type]Factory)
)

func init() {
	// 注册默认支持的压缩算法
	Register(TypeNone, func() (Compressor, error) {
		return &noneCompressor{}, nil
	})
	Register(TypeSnappy, func() (Compressor, error) {
		return &snappyCompressor{}, nil
	})
	Register(TypeZstd, func() (Compressor, error) {
		return newZstdCompressor()
	})
	Register(TypeLZ4, func() (Compressor, error) {
		return &lz4Compressor{}, nil
	})
}

// Register 注册压缩器工厂
func Register(t Type, factory Factory) {
	mu.Lock()
	defer mu.Unlock()
	factories[t] = factory
}

// Unregister 注销压缩器工厂
func Unregister(t Type) {
	mu.Lock()
	defer mu.Unlock()
	delete(factories, t)
}

// New 创建压缩器
func New(t Type) (Compressor, error) {
	mu.RLock()
	factory, ok := factories[t]
	mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported compression type: %s", t)
	}
	return factory()
}

// MustNew 创建压缩器，失败时 panic
func MustNew(t Type) Compressor {
	c, err := New(t)
	if err != nil {
		panic(err)
	}
	return c
}

// List 返回所有已注册的压缩算法类型
func List() []Type {
	mu.RLock()
	defer mu.RUnlock()

	types := make([]Type, 0, len(factories))
	for t := range factories {
		types = append(types, t)
	}
	return types
}

// IsRegistered 检查压缩算法是否已注册
func IsRegistered(t Type) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := factories[t]
	return ok
}
