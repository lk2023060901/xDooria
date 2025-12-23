// pkg/compress/none.go
package compress

// noneCompressor 不压缩实现
type noneCompressor struct{}

// Compress 返回数据副本（不压缩）
func (c *noneCompressor) Compress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst, nil
}

// Decompress 返回数据副本（不解压）
func (c *noneCompressor) Decompress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst, nil
}

// Name 返回压缩算法名称
func (c *noneCompressor) Name() string {
	return string(TypeNone)
}
