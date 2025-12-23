// pkg/compress/lz4.go
package compress

import (
	"github.com/pierrec/lz4/v4"
)

// lz4Compressor LZ4 压缩实现
type lz4Compressor struct{}

// Compress 使用 LZ4 压缩数据
func (c *lz4Compressor) Compress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	if len(src) == 0 {
		return []byte{}, nil
	}

	// 预分配足够大的缓冲区
	dst := make([]byte, lz4.CompressBlockBound(len(src)))
	n, err := lz4.CompressBlock(src, dst, nil)
	if err != nil {
		return nil, err
	}

	// 如果压缩后反而更大，返回原始数据标记
	if n == 0 {
		// LZ4 返回 0 表示数据不可压缩
		// 使用带长度前缀的格式存储原始数据
		result := make([]byte, len(src)+4)
		// 前 4 字节存储原始长度（负数表示未压缩）
		result[0] = 0xFF
		result[1] = 0xFF
		result[2] = 0xFF
		result[3] = 0xFF
		copy(result[4:], src)
		return result, nil
	}

	return dst[:n], nil
}

// Decompress 使用 LZ4 解压数据
func (c *lz4Compressor) Decompress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	if len(src) == 0 {
		return []byte{}, nil
	}

	// 检查是否是未压缩标记
	if len(src) >= 4 && src[0] == 0xFF && src[1] == 0xFF && src[2] == 0xFF && src[3] == 0xFF {
		dst := make([]byte, len(src)-4)
		copy(dst, src[4:])
		return dst, nil
	}

	// 尝试解压，逐步增大缓冲区
	for dstSize := len(src) * 4; dstSize <= len(src)*256; dstSize *= 2 {
		dst := make([]byte, dstSize)
		n, err := lz4.UncompressBlock(src, dst)
		if err == nil && n > 0 {
			return dst[:n], nil
		}
	}

	// 最后尝试一个较大的缓冲区
	dst := make([]byte, len(src)*256)
	n, err := lz4.UncompressBlock(src, dst)
	if err != nil {
		return nil, err
	}
	return dst[:n], nil
}

// Name 返回压缩算法名称
func (c *lz4Compressor) Name() string {
	return string(TypeLZ4)
}
