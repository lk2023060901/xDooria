// pkg/checksum/xxhash.go
package checksum

import (
	"github.com/cespare/xxhash/v2"
)

// xxhashHasher XXHash 校验实现
// XXHash 是一种极快的非加密哈希算法
type xxhashHasher struct{}

// newXXHashHasher 创建 XXHash 校验器
func newXXHashHasher() *xxhashHasher {
	return &xxhashHasher{}
}

// Sum 计算 XXHash 校验和（取低 32 位）
func (h *xxhashHasher) Sum(data []byte) uint32 {
	if data == nil {
		return 0
	}
	// XXHash64 取低 32 位
	return uint32(xxhash.Sum64(data))
}

// Verify 验证 XXHash 校验和
func (h *xxhashHasher) Verify(data []byte, expected uint32) bool {
	return h.Sum(data) == expected
}

// Name 返回校验算法名称
func (h *xxhashHasher) Name() string {
	return string(TypeXXHash)
}
