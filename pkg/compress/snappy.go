// pkg/compress/snappy.go
package compress

import (
	"github.com/golang/snappy"
)

// snappyCompressor Snappy 压缩实现
type snappyCompressor struct{}

// Compress 使用 Snappy 压缩数据
func (c *snappyCompressor) Compress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	return snappy.Encode(nil, src), nil
}

// Decompress 使用 Snappy 解压数据
func (c *snappyCompressor) Decompress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	return snappy.Decode(nil, src)
}

// Name 返回压缩算法名称
func (c *snappyCompressor) Name() string {
	return string(TypeSnappy)
}
