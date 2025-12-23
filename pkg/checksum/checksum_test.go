// pkg/checksum/checksum_test.go
package checksum

import (
	"bytes"
	"testing"
)

func TestCRC32Hasher(t *testing.T) {
	h, err := New(TypeCRC32)
	if err != nil {
		t.Fatalf("failed to create CRC32 hasher: %v", err)
	}

	testHasher(t, h)
}

func TestCRC32CHasher(t *testing.T) {
	h, err := New(TypeCRC32C)
	if err != nil {
		t.Fatalf("failed to create CRC32C hasher: %v", err)
	}

	testHasher(t, h)
}

func TestXXHashHasher(t *testing.T) {
	h, err := New(TypeXXHash)
	if err != nil {
		t.Fatalf("failed to create XXHash hasher: %v", err)
	}

	testHasher(t, h)
}

func testHasher(t *testing.T, h Hasher) {
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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sum := h.Sum(tc.data)

			// 验证应该通过
			if !h.Verify(tc.data, sum) {
				t.Errorf("verify failed for correct checksum")
			}

			// 错误的校验和应该失败（除非是 nil/empty 且 sum 恰好为 0）
			if tc.data != nil && len(tc.data) > 0 {
				wrongSum := sum ^ 0xFFFFFFFF
				if h.Verify(tc.data, wrongSum) {
					t.Errorf("verify should fail for wrong checksum")
				}
			}
		})
	}
}

func TestConsistency(t *testing.T) {
	// 相同数据多次计算应该得到相同结果
	data := []byte("test data for consistency check")

	hashers := []Type{TypeCRC32, TypeCRC32C, TypeXXHash}
	for _, ht := range hashers {
		h, err := New(ht)
		if err != nil {
			t.Fatalf("failed to create %s hasher: %v", ht, err)
		}

		sum1 := h.Sum(data)
		sum2 := h.Sum(data)
		sum3 := h.Sum(data)

		if sum1 != sum2 || sum2 != sum3 {
			t.Errorf("%s: inconsistent hash results: %d, %d, %d", ht, sum1, sum2, sum3)
		}
	}
}

func TestDifferentHashers(t *testing.T) {
	// 不同的哈希算法对同一数据应该产生不同的结果
	data := []byte("test data")

	crc32, _ := New(TypeCRC32)
	crc32c, _ := New(TypeCRC32C)
	xxhash, _ := New(TypeXXHash)

	sum1 := crc32.Sum(data)
	sum2 := crc32c.Sum(data)
	sum3 := xxhash.Sum(data)

	// 至少有两个不同（理论上三个都应该不同）
	if sum1 == sum2 && sum2 == sum3 {
		t.Logf("CRC32=%d, CRC32C=%d, XXHash=%d", sum1, sum2, sum3)
		t.Error("different hash algorithms should produce different results")
	}
}

func TestDataIntegrity(t *testing.T) {
	// 测试数据完整性检测
	h := Default()
	original := []byte("important data that must not be corrupted")

	sum := h.Sum(original)

	// 修改一个字节
	corrupted := make([]byte, len(original))
	copy(corrupted, original)
	corrupted[10] ^= 0x01 // 翻转一位

	if h.Verify(corrupted, sum) {
		t.Error("should detect data corruption")
	}
}

func TestRegister(t *testing.T) {
	customType := Type("custom")

	Register(customType, func() (Hasher, error) {
		return newCRC32Hasher(), nil
	})
	defer Unregister(customType)

	if !IsRegistered(customType) {
		t.Error("custom hasher should be registered")
	}

	h, err := New(customType)
	if err != nil {
		t.Fatalf("failed to create custom hasher: %v", err)
	}

	if h == nil {
		t.Error("hasher should not be nil")
	}
}

func TestUnregister(t *testing.T) {
	customType := Type("temp")

	Register(customType, func() (Hasher, error) {
		return newCRC32Hasher(), nil
	})

	if !IsRegistered(customType) {
		t.Error("temp hasher should be registered")
	}

	Unregister(customType)

	if IsRegistered(customType) {
		t.Error("temp hasher should be unregistered")
	}

	_, err := New(customType)
	if err == nil {
		t.Error("expected error for unregistered hasher")
	}
}

func TestList(t *testing.T) {
	types := List()

	if len(types) < 3 {
		t.Errorf("expected at least 3 registered hashers, got %d", len(types))
	}

	typeMap := make(map[Type]bool)
	for _, tp := range types {
		typeMap[tp] = true
	}

	for _, expected := range []Type{TypeCRC32, TypeCRC32C, TypeXXHash} {
		if !typeMap[expected] {
			t.Errorf("expected %s to be registered", expected)
		}
	}
}

func TestMustNew(t *testing.T) {
	h := MustNew(TypeCRC32)
	if h == nil {
		t.Error("hasher should not be nil")
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for unknown hasher type")
		}
	}()
	MustNew(Type("unknown"))
}

func TestDefault(t *testing.T) {
	h := Default()
	if h == nil {
		t.Error("default hasher should not be nil")
	}

	if h.Name() != string(TypeCRC32C) {
		t.Errorf("default hasher should be CRC32C, got %s", h.Name())
	}
}

func TestUnsupportedType(t *testing.T) {
	_, err := New(Type("unknown"))
	if err == nil {
		t.Error("expected error for unsupported checksum type")
	}
}

// 基准测试
func BenchmarkCRC32(b *testing.B) {
	h := MustNew(TypeCRC32)
	data := bytes.Repeat([]byte("hello world "), 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Sum(data)
	}
}

func BenchmarkCRC32C(b *testing.B) {
	h := MustNew(TypeCRC32C)
	data := bytes.Repeat([]byte("hello world "), 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Sum(data)
	}
}

func BenchmarkXXHash(b *testing.B) {
	h := MustNew(TypeXXHash)
	data := bytes.Repeat([]byte("hello world "), 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Sum(data)
	}
}
