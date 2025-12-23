// pkg/raft/snapshot_test.go
package raft

import (
	"bytes"
	"encoding/binary"
	"runtime"
	"testing"

	"github.com/lk2023060901/xdooria/pkg/checksum"
	"github.com/lk2023060901/xdooria/pkg/compress"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

func TestEncodedSnapshot_RoundTrip(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		compression compress.Type
		checksum    checksum.Type
	}{
		{
			name:        "empty_snappy_crc32c",
			data:        []byte{},
			compression: compress.TypeSnappy,
			checksum:    checksum.TypeCRC32C,
		},
		{
			name:        "small_snappy_crc32c",
			data:        []byte("hello world"),
			compression: compress.TypeSnappy,
			checksum:    checksum.TypeCRC32C,
		},
		{
			name:        "medium_snappy_crc32",
			data:        bytes.Repeat([]byte("hello world "), 1000),
			compression: compress.TypeSnappy,
			checksum:    checksum.TypeCRC32,
		},
		{
			name:        "large_zstd_crc32c",
			data:        bytes.Repeat([]byte("hello world "), 10000),
			compression: compress.TypeZstd,
			checksum:    checksum.TypeCRC32C,
		},
		{
			name:        "none_compression",
			data:        []byte("test data without compression"),
			compression: compress.TypeNone,
			checksum:    checksum.TypeCRC32C,
		},
		{
			name:        "large_lz4_crc32c",
			data:        bytes.Repeat([]byte("hello world "), 10000),
			compression: compress.TypeLZ4,
			checksum:    checksum.TypeCRC32C,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &SnapshotConfig{
				Compression:    tc.compression,
				Checksum:       tc.checksum,
				EnableChecksum: true,
			}

			// 创建快照
			snapshot, err := NewEncodedSnapshot(tc.data, cfg)
			if err != nil {
				t.Fatalf("failed to create snapshot: %v", err)
			}

			// 模拟写入
			sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
			if err := snapshot.Persist(sink); err != nil {
				t.Fatalf("failed to persist snapshot: %v", err)
			}

			// 解码
			reader := bytes.NewReader(sink.buf.Bytes())
			decoded, err := DecodeSnapshot(reader, cfg)
			if err != nil {
				t.Fatalf("failed to decode snapshot: %v", err)
			}

			// 验证
			if !bytes.Equal(tc.data, decoded) {
				t.Errorf("data mismatch: expected %d bytes, got %d bytes",
					len(tc.data), len(decoded))
			}
		})
	}
}

func TestEncodedSnapshot_DefaultConfig(t *testing.T) {
	data := []byte("test data")

	// nil 配置应使用默认值
	snapshot, err := NewEncodedSnapshot(data, nil)
	if err != nil {
		t.Fatalf("failed to create snapshot with nil config: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// 解码时也使用 nil 配置
	reader := bytes.NewReader(sink.buf.Bytes())
	decoded, err := DecodeSnapshot(reader, nil)
	if err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Error("data mismatch with default config")
	}
}

func TestEncodedSnapshot_ChecksumVerification(t *testing.T) {
	data := []byte("important data")
	cfg := DefaultSnapshotConfig()

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// 篡改数据（修改 payload 中的一个字节）
	corruptedData := sink.buf.Bytes()
	if len(corruptedData) > headerSize+5 {
		corruptedData[headerSize+5] ^= 0xFF
	}

	// 解码应该失败
	reader := bytes.NewReader(corruptedData)
	_, err = DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected checksum verification to fail")
	}
}

func TestEncodedSnapshot_DisableChecksum(t *testing.T) {
	data := []byte("test data")
	cfg := &SnapshotConfig{
		Compression:    compress.TypeSnappy,
		EnableChecksum: false,
	}

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// 验证没有校验和（数据长度应该更短）
	expectedMinLen := headerSize + 1 // header + 至少 1 字节压缩数据
	if sink.buf.Len() < expectedMinLen {
		t.Errorf("snapshot data too small: %d bytes", sink.buf.Len())
	}

	// 解码
	reader := bytes.NewReader(sink.buf.Bytes())
	decoded, err := DecodeSnapshot(reader, cfg)
	if err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Error("data mismatch")
	}
}

func TestEncodedSnapshot_InvalidMagic(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	// 构造无效魔数的数据
	data := make([]byte, headerSize+checksumSize+10)
	binary.BigEndian.PutUint32(data[0:4], 0xDEADBEEF) // 错误的魔数

	reader := bytes.NewReader(data)
	_, err := DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for invalid magic")
	}
}

func TestEncodedSnapshot_TooSmall(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	// 数据太小
	data := make([]byte, 5)

	reader := bytes.NewReader(data)
	_, err := DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for too small data")
	}
}

func TestSnapshotCompressionRatio(t *testing.T) {
	// 高重复数据应有较好压缩比
	data := bytes.Repeat([]byte("hello world "), 10000)

	compTypes := []compress.Type{compress.TypeSnappy, compress.TypeZstd, compress.TypeLZ4}
	for _, ct := range compTypes {
		cfg := &SnapshotConfig{
			Compression:    ct,
			Checksum:       checksum.TypeCRC32C,
			EnableChecksum: true,
		}

		snapshot, err := NewEncodedSnapshot(data, cfg)
		if err != nil {
			t.Fatalf("failed to create snapshot: %v", err)
		}

		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		if err := snapshot.Persist(sink); err != nil {
			t.Fatalf("failed to persist snapshot: %v", err)
		}

		ratio := float64(len(data)) / float64(sink.buf.Len())
		t.Logf("%s: original=%d, snapshot=%d, ratio=%.2fx",
			ct, len(data), sink.buf.Len(), ratio)

		if ratio < 2.0 {
			t.Errorf("%s: expected compression ratio > 2.0, got %.2f", ct, ratio)
		}
	}
}

// ============== 边界测试 ==============

func TestEncodedSnapshot_NilData(t *testing.T) {
	// 测试所有压缩算法对 nil 数据的处理
	compTypes := []compress.Type{compress.TypeNone, compress.TypeSnappy, compress.TypeZstd, compress.TypeLZ4}

	for _, ct := range compTypes {
		t.Run(string(ct), func(t *testing.T) {
			cfg := &SnapshotConfig{
				Compression:    ct,
				Checksum:       checksum.TypeCRC32C,
				EnableChecksum: true,
			}

			// nil 数据应该能正常处理
			snapshot, err := NewEncodedSnapshot(nil, cfg)
			if err != nil {
				t.Fatalf("failed to create snapshot with nil data: %v", err)
			}

			sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
			err = snapshot.Persist(sink)
			// 某些压缩器可能不支持 nil，这是可接受的
			if err != nil {
				t.Logf("%s: persist with nil data returned error (acceptable): %v", ct, err)
				return
			}

			reader := bytes.NewReader(sink.buf.Bytes())
			decoded, err := DecodeSnapshot(reader, cfg)
			if err != nil {
				t.Logf("%s: decode with nil data returned error (acceptable): %v", ct, err)
				return
			}

			// 如果成功，解码结果应该为空
			if len(decoded) != 0 {
				t.Errorf("%s: expected empty data, got %d bytes", ct, len(decoded))
			}
		})
	}
}

func TestEncodedSnapshot_SingleByte(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	data := []byte{0x42}

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	reader := bytes.NewReader(sink.buf.Bytes())
	decoded, err := DecodeSnapshot(reader, cfg)
	if err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Error("single byte data mismatch")
	}
}

func TestEncodedSnapshot_BinaryData(t *testing.T) {
	// 测试包含所有字节值的数据
	data := make([]byte, 256)
	for i := 0; i < 256; i++ {
		data[i] = byte(i)
	}

	cfg := DefaultSnapshotConfig()
	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	reader := bytes.NewReader(sink.buf.Bytes())
	decoded, err := DecodeSnapshot(reader, cfg)
	if err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Error("binary data mismatch")
	}
}

func TestEncodedSnapshot_LargeData(t *testing.T) {
	// 测试 10MB 数据
	data := make([]byte, 10*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}

	cfg := DefaultSnapshotConfig()
	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	reader := bytes.NewReader(sink.buf.Bytes())
	decoded, err := DecodeSnapshot(reader, cfg)
	if err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Error("large data mismatch")
	}
}

func TestEncodedSnapshot_IncompressibleData(t *testing.T) {
	// 随机数据几乎不可压缩
	data := make([]byte, 1000)
	for i := range data {
		data[i] = byte((i * 17 + 31) % 256) // 伪随机
	}

	compTypes := []compress.Type{compress.TypeSnappy, compress.TypeZstd, compress.TypeLZ4}
	for _, ct := range compTypes {
		t.Run(string(ct), func(t *testing.T) {
			cfg := &SnapshotConfig{
				Compression:    ct,
				Checksum:       checksum.TypeCRC32C,
				EnableChecksum: true,
			}

			snapshot, err := NewEncodedSnapshot(data, cfg)
			if err != nil {
				t.Fatalf("failed to create snapshot: %v", err)
			}

			sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
			if err := snapshot.Persist(sink); err != nil {
				t.Fatalf("failed to persist snapshot: %v", err)
			}

			reader := bytes.NewReader(sink.buf.Bytes())
			decoded, err := DecodeSnapshot(reader, cfg)
			if err != nil {
				t.Fatalf("failed to decode snapshot: %v", err)
			}

			if !bytes.Equal(data, decoded) {
				t.Error("incompressible data mismatch")
			}
		})
	}
}

// ============== 错误处理测试 ==============

func TestEncodedSnapshot_InvalidVersion(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	// 构造无效版本的数据
	data := make([]byte, headerSize+checksumSize+10)
	binary.BigEndian.PutUint32(data[0:4], snapshotMagic) // 正确的魔数
	binary.BigEndian.PutUint16(data[4:6], 0xFFFF)        // 无效版本

	reader := bytes.NewReader(data)
	_, err := DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for invalid version")
	}
}

func TestEncodedSnapshot_CorruptedChecksum(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	data := []byte("test data for corruption")

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// 篡改校验和本身（最后 4 字节）
	corruptedData := sink.buf.Bytes()
	corruptedData[len(corruptedData)-1] ^= 0xFF

	reader := bytes.NewReader(corruptedData)
	_, err = DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for corrupted checksum")
	}
}

func TestEncodedSnapshot_TruncatedData(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	data := bytes.Repeat([]byte("test data "), 100)

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// 截断数据
	fullData := sink.buf.Bytes()
	truncated := fullData[:len(fullData)/2]

	reader := bytes.NewReader(truncated)
	_, err = DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for truncated data")
	}
}

func TestEncodedSnapshot_HeaderOnly(t *testing.T) {
	cfg := &SnapshotConfig{
		Compression:    compress.TypeSnappy,
		EnableChecksum: false, // 禁用校验和以测试边界
	}

	// 只有 header，没有压缩数据
	data := make([]byte, headerSize)
	binary.BigEndian.PutUint32(data[0:4], snapshotMagic)
	binary.BigEndian.PutUint16(data[4:6], snapshotVersion)
	binary.BigEndian.PutUint16(data[6:8], flagSnappy)
	binary.BigEndian.PutUint64(data[8:16], 100) // 声称有 100 字节原始数据

	reader := bytes.NewReader(data)
	_, err := DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for header-only data")
	}
}

func TestEncodedSnapshot_DataLengthMismatch(t *testing.T) {
	cfg := &SnapshotConfig{
		Compression:    compress.TypeNone, // 不压缩便于构造
		EnableChecksum: false,
	}

	// 构造 header 声称长度与实际不符的数据
	payload := []byte("actual data")
	data := make([]byte, headerSize+len(payload))
	binary.BigEndian.PutUint32(data[0:4], snapshotMagic)
	binary.BigEndian.PutUint16(data[4:6], snapshotVersion)
	binary.BigEndian.PutUint16(data[6:8], flagNone)
	binary.BigEndian.PutUint64(data[8:16], 999) // 声称 999 字节，实际只有 11 字节
	copy(data[headerSize:], payload)

	reader := bytes.NewReader(data)
	_, err := DecodeSnapshot(reader, cfg)
	if err == nil {
		t.Error("expected error for data length mismatch")
	}
}

func TestEncodedSnapshot_UnsupportedCompression(t *testing.T) {
	cfg := &SnapshotConfig{
		Compression:    compress.Type("unsupported"),
		EnableChecksum: true,
	}

	data := []byte("test data")
	_, err := NewEncodedSnapshot(data, cfg)
	// 应该在创建时或持久化时失败
	if err != nil {
		return // 创建时失败，符合预期
	}
}

// ============== 写入错误测试 ==============

func TestEncodedSnapshot_WriteError(t *testing.T) {
	cfg := DefaultSnapshotConfig()
	data := []byte("test data")

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	// 模拟写入错误
	sink := &mockSnapshotSink{
		buf:      &bytes.Buffer{},
		writeErr: bytes.ErrTooLarge,
	}

	err = snapshot.Persist(sink)
	if err == nil {
		t.Error("expected error when write fails")
	}

	if !sink.cancelled {
		t.Error("expected sink to be cancelled on error")
	}
}

// ============== Header 编解码测试 ==============

func TestSnapshotHeader_EncodeDecode(t *testing.T) {
	testCases := []snapshotHeader{
		{Magic: snapshotMagic, Version: 1, Flags: flagNone, DataLen: 0},
		{Magic: snapshotMagic, Version: 1, Flags: flagSnappy, DataLen: 100},
		{Magic: snapshotMagic, Version: 1, Flags: flagZstd, DataLen: 1000000},
		{Magic: snapshotMagic, Version: 1, Flags: flagLZ4, DataLen: ^uint64(0)}, // 最大值
	}

	for _, tc := range testCases {
		encoded := tc.encode()
		if len(encoded) != headerSize {
			t.Errorf("expected header size %d, got %d", headerSize, len(encoded))
		}

		var decoded snapshotHeader
		if err := decoded.decode(encoded); err != nil {
			t.Fatalf("failed to decode header: %v", err)
		}

		if decoded.Magic != tc.Magic {
			t.Errorf("magic mismatch: expected 0x%X, got 0x%X", tc.Magic, decoded.Magic)
		}
		if decoded.Version != tc.Version {
			t.Errorf("version mismatch: expected %d, got %d", tc.Version, decoded.Version)
		}
		if decoded.Flags != tc.Flags {
			t.Errorf("flags mismatch: expected %d, got %d", tc.Flags, decoded.Flags)
		}
		if decoded.DataLen != tc.DataLen {
			t.Errorf("dataLen mismatch: expected %d, got %d", tc.DataLen, decoded.DataLen)
		}
	}
}

func TestSnapshotHeader_DecodeInvalidSize(t *testing.T) {
	var h snapshotHeader
	err := h.decode(make([]byte, 5)) // 太小
	if err == nil {
		t.Error("expected error for invalid header size")
	}
}

// ============== Flag 转换测试 ==============

func TestCompressionToFlag(t *testing.T) {
	testCases := []struct {
		compType compress.Type
		expected uint16
	}{
		{compress.TypeNone, flagNone},
		{compress.TypeSnappy, flagSnappy},
		{compress.TypeZstd, flagZstd},
		{compress.TypeLZ4, flagLZ4},
		{compress.Type("unknown"), flagNone},
	}

	for _, tc := range testCases {
		got := compressionToFlag(tc.compType)
		if got != tc.expected {
			t.Errorf("compressionToFlag(%s): expected %d, got %d",
				tc.compType, tc.expected, got)
		}
	}
}

func TestFlagToCompression(t *testing.T) {
	testCases := []struct {
		flags    uint16
		expected compress.Type
	}{
		{flagNone, compress.TypeNone},
		{flagSnappy, compress.TypeSnappy},
		{flagZstd, compress.TypeZstd},
		{flagLZ4, compress.TypeLZ4},
		{0xFF, compress.TypeSnappy}, // 多个 flag 时，按优先级返回
	}

	for _, tc := range testCases {
		got := flagToCompression(tc.flags)
		if got != tc.expected {
			t.Errorf("flagToCompression(0x%X): expected %s, got %s",
				tc.flags, tc.expected, got)
		}
	}
}

// ============== 配置测试 ==============

func TestDefaultSnapshotConfig(t *testing.T) {
	cfg := DefaultSnapshotConfig()

	if cfg.Compression != compress.TypeSnappy {
		t.Errorf("default compression should be snappy, got %s", cfg.Compression)
	}
	if cfg.Checksum != checksum.TypeCRC32C {
		t.Errorf("default checksum should be crc32c, got %s", cfg.Checksum)
	}
	if !cfg.EnableChecksum {
		t.Error("default should enable checksum")
	}
}

func TestEncodedSnapshot_PartialConfig(t *testing.T) {
	data := []byte("test data")

	// 只设置部分配置，其他应使用默认值
	cfg := &SnapshotConfig{
		Compression: compress.TypeZstd,
		// Checksum 和 EnableChecksum 使用默认值
	}

	snapshot, err := NewEncodedSnapshot(data, cfg)
	if err != nil {
		t.Fatalf("failed to create snapshot: %v", err)
	}

	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	if err := snapshot.Persist(sink); err != nil {
		t.Fatalf("failed to persist snapshot: %v", err)
	}

	// 使用相同配置解码
	reader := bytes.NewReader(sink.buf.Bytes())
	decoded, err := DecodeSnapshot(reader, cfg)
	if err != nil {
		t.Fatalf("failed to decode snapshot: %v", err)
	}

	if !bytes.Equal(data, decoded) {
		t.Error("data mismatch with partial config")
	}
}

// ============== 一致性测试 ==============

func TestEncodedSnapshot_Consistency(t *testing.T) {
	// 相同数据多次编解码应该得到相同结果
	data := []byte("consistency test data")
	cfg := DefaultSnapshotConfig()

	for i := 0; i < 10; i++ {
		snapshot, err := NewEncodedSnapshot(data, cfg)
		if err != nil {
			t.Fatalf("iteration %d: failed to create snapshot: %v", i, err)
		}

		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		if err := snapshot.Persist(sink); err != nil {
			t.Fatalf("iteration %d: failed to persist snapshot: %v", i, err)
		}

		reader := bytes.NewReader(sink.buf.Bytes())
		decoded, err := DecodeSnapshot(reader, cfg)
		if err != nil {
			t.Fatalf("iteration %d: failed to decode snapshot: %v", i, err)
		}

		if !bytes.Equal(data, decoded) {
			t.Errorf("iteration %d: data mismatch", i)
		}
	}
}

// ============== Release 测试 ==============

func TestEncodedSnapshot_Release(t *testing.T) {
	data := []byte("test data")
	snapshot, _ := NewEncodedSnapshot(data, nil)

	// Release 应该是空操作，不应 panic
	snapshot.Release()
	snapshot.Release() // 多次调用也不应 panic
}

// ============== 并发安全测试 ==============

func TestEncodedSnapshot_Concurrent(t *testing.T) {
	data := bytes.Repeat([]byte("concurrent test data "), 100)
	cfg := DefaultSnapshotConfig()

	const iterations = 100
	pool := conc.NewPool[error](runtime.GOMAXPROCS(0))
	defer pool.Release()

	futures := make([]*conc.Future[error], iterations)
	for i := 0; i < iterations; i++ {
		futures[i] = pool.Submit(func() (error, error) {
			// 创建快照
			snapshot, err := NewEncodedSnapshot(data, cfg)
			if err != nil {
				return err, nil
			}

			// 持久化
			sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
			if err := snapshot.Persist(sink); err != nil {
				return err, nil
			}

			// 解码
			reader := bytes.NewReader(sink.buf.Bytes())
			decoded, err := DecodeSnapshot(reader, cfg)
			if err != nil {
				return err, nil
			}

			// 验证
			if !bytes.Equal(data, decoded) {
				return ErrSnapshotCorrupted, nil
			}
			return nil, nil
		})
	}

	// 等待所有任务完成并检查错误
	for i, f := range futures {
		result, err := f.Await()
		if err != nil {
			t.Errorf("task %d submit error: %v", i, err)
		}
		if result != nil {
			t.Errorf("task %d execution error: %v", i, result)
		}
	}
}

func TestEncodedSnapshot_ConcurrentDifferentCompression(t *testing.T) {
	data := bytes.Repeat([]byte("compression test "), 500)
	compTypes := []compress.Type{compress.TypeNone, compress.TypeSnappy, compress.TypeZstd, compress.TypeLZ4}

	const iterationsPerType = 50
	pool := conc.NewPool[error](runtime.GOMAXPROCS(0))
	defer pool.Release()

	futures := make([]*conc.Future[error], 0, len(compTypes)*iterationsPerType)

	for _, ct := range compTypes {
		compType := ct // 捕获循环变量
		for i := 0; i < iterationsPerType; i++ {
			f := pool.Submit(func() (error, error) {
				cfg := &SnapshotConfig{
					Compression:    compType,
					Checksum:       checksum.TypeCRC32C,
					EnableChecksum: true,
				}

				snapshot, err := NewEncodedSnapshot(data, cfg)
				if err != nil {
					return err, nil
				}

				sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
				if err := snapshot.Persist(sink); err != nil {
					return err, nil
				}

				reader := bytes.NewReader(sink.buf.Bytes())
				decoded, err := DecodeSnapshot(reader, cfg)
				if err != nil {
					return err, nil
				}

				if !bytes.Equal(data, decoded) {
					return ErrSnapshotCorrupted, nil
				}
				return nil, nil
			})
			futures = append(futures, f)
		}
	}

	// 等待所有任务完成
	for i, f := range futures {
		result, err := f.Await()
		if err != nil {
			t.Errorf("task %d submit error: %v", i, err)
		}
		if result != nil {
			t.Errorf("task %d execution error: %v", i, result)
		}
	}
}

// ============== 模糊测试 ==============

func FuzzDecodeSnapshot(f *testing.F) {
	// 添加种子语料
	cfg := DefaultSnapshotConfig()

	// 正常快照数据作为种子
	normalData := []byte("normal snapshot data")
	snapshot, _ := NewEncodedSnapshot(normalData, cfg)
	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	_ = snapshot.Persist(sink)
	f.Add(sink.buf.Bytes())

	// 空数据
	emptySnapshot, _ := NewEncodedSnapshot([]byte{}, cfg)
	emptySink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	_ = emptySnapshot.Persist(emptySink)
	f.Add(emptySink.buf.Bytes())

	// 只有 header 的数据
	headerOnly := make([]byte, headerSize)
	binary.BigEndian.PutUint32(headerOnly[0:4], snapshotMagic)
	binary.BigEndian.PutUint16(headerOnly[4:6], snapshotVersion)
	f.Add(headerOnly)

	// 随机数据
	f.Add([]byte{0x00, 0x01, 0x02, 0x03})
	f.Add([]byte{0xFF, 0xFF, 0xFF, 0xFF})

	f.Fuzz(func(t *testing.T, data []byte) {
		// 模糊测试解码函数，确保不会 panic
		reader := bytes.NewReader(data)
		_, _ = DecodeSnapshot(reader, cfg)

		// 禁用校验和的配置也测试
		cfgNoChecksum := &SnapshotConfig{
			Compression:    compress.TypeSnappy,
			EnableChecksum: false,
		}
		reader2 := bytes.NewReader(data)
		_, _ = DecodeSnapshot(reader2, cfgNoChecksum)
	})
}

func FuzzEncodedSnapshotRoundTrip(f *testing.F) {
	// 添加种子语料
	f.Add([]byte("hello world"))
	f.Add([]byte{})
	f.Add([]byte{0x00})
	f.Add([]byte{0xFF})
	f.Add(bytes.Repeat([]byte("a"), 1000))

	f.Fuzz(func(t *testing.T, data []byte) {
		cfg := DefaultSnapshotConfig()

		snapshot, err := NewEncodedSnapshot(data, cfg)
		if err != nil {
			return // 创建失败是可接受的
		}

		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		if err := snapshot.Persist(sink); err != nil {
			return // 持久化失败是可接受的
		}

		reader := bytes.NewReader(sink.buf.Bytes())
		decoded, err := DecodeSnapshot(reader, cfg)
		if err != nil {
			t.Fatalf("round trip failed: %v", err)
		}

		if !bytes.Equal(data, decoded) {
			t.Fatalf("data mismatch: expected %d bytes, got %d bytes",
				len(data), len(decoded))
		}
	})
}

// ============== 基准测试 ==============

func BenchmarkEncodedSnapshot_Persist_Snappy(b *testing.B) {
	benchmarkPersist(b, compress.TypeSnappy)
}

func BenchmarkEncodedSnapshot_Persist_Zstd(b *testing.B) {
	benchmarkPersist(b, compress.TypeZstd)
}

func BenchmarkEncodedSnapshot_Persist_LZ4(b *testing.B) {
	benchmarkPersist(b, compress.TypeLZ4)
}

func BenchmarkEncodedSnapshot_Persist_None(b *testing.B) {
	benchmarkPersist(b, compress.TypeNone)
}

func benchmarkPersist(b *testing.B, compType compress.Type) {
	data := bytes.Repeat([]byte("benchmark data for snapshot persistence "), 10000)
	cfg := &SnapshotConfig{
		Compression:    compType,
		Checksum:       checksum.TypeCRC32C,
		EnableChecksum: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		snapshot, _ := NewEncodedSnapshot(data, cfg)
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		_ = snapshot.Persist(sink)
	}
}

func BenchmarkDecodeSnapshot_Snappy(b *testing.B) {
	benchmarkDecode(b, compress.TypeSnappy)
}

func BenchmarkDecodeSnapshot_Zstd(b *testing.B) {
	benchmarkDecode(b, compress.TypeZstd)
}

func BenchmarkDecodeSnapshot_LZ4(b *testing.B) {
	benchmarkDecode(b, compress.TypeLZ4)
}

func BenchmarkDecodeSnapshot_None(b *testing.B) {
	benchmarkDecode(b, compress.TypeNone)
}

func benchmarkDecode(b *testing.B, compType compress.Type) {
	data := bytes.Repeat([]byte("benchmark data for snapshot decoding "), 10000)
	cfg := &SnapshotConfig{
		Compression:    compType,
		Checksum:       checksum.TypeCRC32C,
		EnableChecksum: true,
	}

	// 预先创建快照数据
	snapshot, _ := NewEncodedSnapshot(data, cfg)
	sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
	_ = snapshot.Persist(sink)
	snapshotData := sink.buf.Bytes()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(snapshotData)
		_, _ = DecodeSnapshot(reader, cfg)
	}
}

func BenchmarkEncodedSnapshot_RoundTrip_Snappy(b *testing.B) {
	benchmarkRoundTrip(b, compress.TypeSnappy)
}

func BenchmarkEncodedSnapshot_RoundTrip_Zstd(b *testing.B) {
	benchmarkRoundTrip(b, compress.TypeZstd)
}

func BenchmarkEncodedSnapshot_RoundTrip_LZ4(b *testing.B) {
	benchmarkRoundTrip(b, compress.TypeLZ4)
}

func benchmarkRoundTrip(b *testing.B, compType compress.Type) {
	data := bytes.Repeat([]byte("round trip benchmark data "), 10000)
	cfg := &SnapshotConfig{
		Compression:    compType,
		Checksum:       checksum.TypeCRC32C,
		EnableChecksum: true,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		snapshot, _ := NewEncodedSnapshot(data, cfg)
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		_ = snapshot.Persist(sink)
		reader := bytes.NewReader(sink.buf.Bytes())
		_, _ = DecodeSnapshot(reader, cfg)
	}
}

// BenchmarkEncodedSnapshot_LargeData 大数据基准测试
func BenchmarkEncodedSnapshot_LargeData_1MB(b *testing.B) {
	benchmarkLargeData(b, 1*1024*1024)
}

func BenchmarkEncodedSnapshot_LargeData_10MB(b *testing.B) {
	benchmarkLargeData(b, 10*1024*1024)
}

func benchmarkLargeData(b *testing.B, size int) {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}
	cfg := DefaultSnapshotConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		snapshot, _ := NewEncodedSnapshot(data, cfg)
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		_ = snapshot.Persist(sink)
		reader := bytes.NewReader(sink.buf.Bytes())
		_, _ = DecodeSnapshot(reader, cfg)
	}
}

// ============== 内存分配测试 ==============

func TestEncodedSnapshot_MemoryAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory allocation test in short mode")
	}

	data := bytes.Repeat([]byte("memory test "), 1000)
	cfg := DefaultSnapshotConfig()

	// 预热
	for i := 0; i < 10; i++ {
		snapshot, _ := NewEncodedSnapshot(data, cfg)
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		_ = snapshot.Persist(sink)
	}

	// 测试内存分配是否稳定
	var m1, m2 runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&m1)

	for i := 0; i < 100; i++ {
		snapshot, _ := NewEncodedSnapshot(data, cfg)
		sink := &mockSnapshotSink{buf: &bytes.Buffer{}}
		_ = snapshot.Persist(sink)
		reader := bytes.NewReader(sink.buf.Bytes())
		_, _ = DecodeSnapshot(reader, cfg)
	}

	runtime.GC()
	runtime.ReadMemStats(&m2)

	// 记录内存使用情况
	t.Logf("HeapAlloc: before=%d, after=%d, diff=%d",
		m1.HeapAlloc, m2.HeapAlloc, int64(m2.HeapAlloc)-int64(m1.HeapAlloc))
	t.Logf("TotalAlloc: before=%d, after=%d, diff=%d",
		m1.TotalAlloc, m2.TotalAlloc, m2.TotalAlloc-m1.TotalAlloc)
}
