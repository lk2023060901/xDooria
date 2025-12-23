// pkg/checksum/crc32.go
package checksum

import (
	"hash/crc32"
)

// crc32Hasher CRC32 校验实现 (IEEE 多项式)
type crc32Hasher struct {
	table *crc32.Table
}

// newCRC32Hasher 创建 CRC32 校验器
func newCRC32Hasher() *crc32Hasher {
	return &crc32Hasher{
		table: crc32.IEEETable,
	}
}

// Sum 计算 CRC32 校验和
func (h *crc32Hasher) Sum(data []byte) uint32 {
	if data == nil {
		return 0
	}
	return crc32.Checksum(data, h.table)
}

// Verify 验证 CRC32 校验和
func (h *crc32Hasher) Verify(data []byte, expected uint32) bool {
	return h.Sum(data) == expected
}

// Name 返回校验算法名称
func (h *crc32Hasher) Name() string {
	return string(TypeCRC32)
}

// crc32cHasher CRC32C 校验实现 (Castagnoli 多项式)
// CRC32C 在现代 CPU 上有硬件加速支持 (SSE4.2)
type crc32cHasher struct {
	table *crc32.Table
}

// newCRC32CHasher 创建 CRC32C 校验器
func newCRC32CHasher() *crc32cHasher {
	return &crc32cHasher{
		table: crc32.MakeTable(crc32.Castagnoli),
	}
}

// Sum 计算 CRC32C 校验和
func (h *crc32cHasher) Sum(data []byte) uint32 {
	if data == nil {
		return 0
	}
	return crc32.Checksum(data, h.table)
}

// Verify 验证 CRC32C 校验和
func (h *crc32cHasher) Verify(data []byte, expected uint32) bool {
	return h.Sum(data) == expected
}

// Name 返回校验算法名称
func (h *crc32cHasher) Name() string {
	return string(TypeCRC32C)
}
