// pkg/compress/compress_test.go
package compress

import (
	"bytes"
	"testing"
)

func TestNoneCompressor(t *testing.T) {
	c, err := New(TypeNone)
	if err != nil {
		t.Fatalf("failed to create none compressor: %v", err)
	}

	testCompressor(t, c)
}

func TestSnappyCompressor(t *testing.T) {
	c, err := New(TypeSnappy)
	if err != nil {
		t.Fatalf("failed to create snappy compressor: %v", err)
	}

	testCompressor(t, c)
}

func TestZstdCompressor(t *testing.T) {
	c, err := New(TypeZstd)
	if err != nil {
		t.Fatalf("failed to create zstd compressor: %v", err)
	}

	testCompressor(t, c)
}

func TestLZ4Compressor(t *testing.T) {
	c, err := New(TypeLZ4)
	if err != nil {
		t.Fatalf("failed to create lz4 compressor: %v", err)
	}

	testCompressor(t, c)
}

func testCompressor(t *testing.T, c Compressor) {
	t.Helper()

	testCases := []struct {
		name string
		data []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"small", []byte("hello world")},
		{"medium", bytes.Repeat([]byte("hello world "), 1000)},
		{"large", bytes.Repeat([]byte("hello world "), 100000)},
		{"random", generateRandomBytes(10000)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			compressed, err := c.Compress(tc.data)
			if err != nil {
				t.Fatalf("compress failed: %v", err)
			}

			decompressed, err := c.Decompress(compressed)
			if err != nil {
				t.Fatalf("decompress failed: %v", err)
			}

			if !bytes.Equal(tc.data, decompressed) {
				t.Errorf("data mismatch: original len=%d, decompressed len=%d",
					len(tc.data), len(decompressed))
			}
		})
	}
}

func TestCompressionRatio(t *testing.T) {
	// 高重复数据应该有较好的压缩比
	data := bytes.Repeat([]byte("hello world "), 10000)

	compressors := []Type{TypeSnappy, TypeZstd, TypeLZ4}
	for _, ct := range compressors {
		c, err := New(ct)
		if err != nil {
			t.Fatalf("failed to create %s compressor: %v", ct, err)
		}

		compressed, err := c.Compress(data)
		if err != nil {
			t.Fatalf("%s compress failed: %v", ct, err)
		}

		ratio := float64(len(data)) / float64(len(compressed))
		t.Logf("%s: original=%d, compressed=%d, ratio=%.2fx",
			ct, len(data), len(compressed), ratio)

		if ratio < 2.0 {
			t.Errorf("%s: expected compression ratio > 2.0, got %.2f", ct, ratio)
		}
	}
}

func TestRegister(t *testing.T) {
	// 测试自定义压缩器注册
	customType := Type("custom")

	// 注册自定义压缩器
	Register(customType, func() (Compressor, error) {
		return &noneCompressor{}, nil
	})
	defer Unregister(customType)

	if !IsRegistered(customType) {
		t.Error("custom compressor should be registered")
	}

	c, err := New(customType)
	if err != nil {
		t.Fatalf("failed to create custom compressor: %v", err)
	}

	if c == nil {
		t.Error("compressor should not be nil")
	}
}

func TestUnregister(t *testing.T) {
	customType := Type("temp")

	Register(customType, func() (Compressor, error) {
		return &noneCompressor{}, nil
	})

	if !IsRegistered(customType) {
		t.Error("temp compressor should be registered")
	}

	Unregister(customType)

	if IsRegistered(customType) {
		t.Error("temp compressor should be unregistered")
	}

	_, err := New(customType)
	if err == nil {
		t.Error("expected error for unregistered compressor")
	}
}

func TestList(t *testing.T) {
	types := List()

	// 至少应该有 4 个默认注册的压缩器
	if len(types) < 4 {
		t.Errorf("expected at least 4 registered compressors, got %d", len(types))
	}

	// 检查默认压缩器是否存在
	typeMap := make(map[Type]bool)
	for _, tp := range types {
		typeMap[tp] = true
	}

	for _, expected := range []Type{TypeNone, TypeSnappy, TypeZstd, TypeLZ4} {
		if !typeMap[expected] {
			t.Errorf("expected %s to be registered", expected)
		}
	}
}

func TestMustNew(t *testing.T) {
	// 正常情况
	c := MustNew(TypeNone)
	if c == nil {
		t.Error("compressor should not be nil")
	}

	// panic 情况
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown compressor type")
		}
	}()
	MustNew(Type("unknown"))
}

func TestUnsupportedType(t *testing.T) {
	_, err := New(Type("unknown"))
	if err == nil {
		t.Error("expected error for unsupported compression type")
	}
}

func generateRandomBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i % 256)
	}
	return b
}

// 基准测试
func BenchmarkSnappyCompress(b *testing.B) {
	c := MustNew(TypeSnappy)
	data := bytes.Repeat([]byte("hello world "), 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(data)
	}
}

func BenchmarkSnappyDecompress(b *testing.B) {
	c := MustNew(TypeSnappy)
	data := bytes.Repeat([]byte("hello world "), 10000)
	compressed, _ := c.Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Decompress(compressed)
	}
}

func BenchmarkZstdCompress(b *testing.B) {
	c := MustNew(TypeZstd)
	data := bytes.Repeat([]byte("hello world "), 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(data)
	}
}

func BenchmarkZstdDecompress(b *testing.B) {
	c := MustNew(TypeZstd)
	data := bytes.Repeat([]byte("hello world "), 10000)
	compressed, _ := c.Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Decompress(compressed)
	}
}

func BenchmarkLZ4Compress(b *testing.B) {
	c := MustNew(TypeLZ4)
	data := bytes.Repeat([]byte("hello world "), 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Compress(data)
	}
}

func BenchmarkLZ4Decompress(b *testing.B) {
	c := MustNew(TypeLZ4)
	data := bytes.Repeat([]byte("hello world "), 10000)
	compressed, _ := c.Compress(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = c.Decompress(compressed)
	}
}
