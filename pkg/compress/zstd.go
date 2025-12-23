// pkg/compress/zstd.go
package compress

import (
	"github.com/klauspost/compress/zstd"
)

// zstdCompressor Zstd 压缩实现
type zstdCompressor struct {
	encoder *zstd.Encoder
	decoder *zstd.Decoder
}

// newZstdCompressor 创建 Zstd 压缩器
func newZstdCompressor() (*zstdCompressor, error) {
	encoder, err := zstd.NewWriter(nil,
		zstd.WithEncoderLevel(zstd.SpeedDefault),
	)
	if err != nil {
		return nil, err
	}

	decoder, err := zstd.NewReader(nil)
	if err != nil {
		encoder.Close()
		return nil, err
	}

	return &zstdCompressor{
		encoder: encoder,
		decoder: decoder,
	}, nil
}

// Compress 使用 Zstd 压缩数据
func (c *zstdCompressor) Compress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	return c.encoder.EncodeAll(src, nil), nil
}

// Decompress 使用 Zstd 解压数据
func (c *zstdCompressor) Decompress(src []byte) ([]byte, error) {
	if src == nil {
		return nil, nil
	}
	return c.decoder.DecodeAll(src, nil)
}

// Name 返回压缩算法名称
func (c *zstdCompressor) Name() string {
	return string(TypeZstd)
}
