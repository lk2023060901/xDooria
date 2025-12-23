// pkg/raft/snapshot.go
package raft

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
	"github.com/lk2023060901/xdooria/pkg/checksum"
	"github.com/lk2023060901/xdooria/pkg/compress"
	"github.com/lk2023060901/xdooria/pkg/config"
)

const (
	// snapshotMagic 快照文件魔数 "RAFT"
	snapshotMagic uint32 = 0x52414654

	// snapshotVersion 快照格式版本
	snapshotVersion uint16 = 1

	// headerSize 快照头部大小
	headerSize = 16 // magic(4) + version(2) + flags(2) + dataLen(8)

	// checksumSize 校验和大小
	checksumSize = 4
)

// 压缩标志位
const (
	flagNone   uint16 = 0
	flagSnappy uint16 = 1 << 0
	flagZstd   uint16 = 1 << 1
	flagLZ4    uint16 = 1 << 2
)

// SnapshotConfig 快照配置
type SnapshotConfig struct {
	// Compression 压缩算法类型
	Compression compress.Type `mapstructure:"compression"`

	// Checksum 校验算法类型
	Checksum checksum.Type `mapstructure:"checksum"`

	// EnableChecksum 是否启用校验和
	EnableChecksum bool `mapstructure:"enable_checksum"`
}

// DefaultSnapshotConfig 返回默认快照配置
func DefaultSnapshotConfig() *SnapshotConfig {
	return &SnapshotConfig{
		Compression:    compress.TypeSnappy,
		Checksum:       checksum.TypeCRC32C,
		EnableChecksum: true,
	}
}

// snapshotHeader 快照头部结构
type snapshotHeader struct {
	Magic   uint32 // 魔数
	Version uint16 // 版本
	Flags   uint16 // 标志位（压缩类型等）
	DataLen uint64 // 原始数据长度
}

// encode 编码头部
func (h *snapshotHeader) encode() []byte {
	buf := make([]byte, headerSize)
	binary.BigEndian.PutUint32(buf[0:4], h.Magic)
	binary.BigEndian.PutUint16(buf[4:6], h.Version)
	binary.BigEndian.PutUint16(buf[6:8], h.Flags)
	binary.BigEndian.PutUint64(buf[8:16], h.DataLen)
	return buf
}

// decode 解码头部
func (h *snapshotHeader) decode(buf []byte) error {
	if len(buf) < headerSize {
		return fmt.Errorf("invalid header size: %d", len(buf))
	}
	h.Magic = binary.BigEndian.Uint32(buf[0:4])
	h.Version = binary.BigEndian.Uint16(buf[4:6])
	h.Flags = binary.BigEndian.Uint16(buf[6:8])
	h.DataLen = binary.BigEndian.Uint64(buf[8:16])
	return nil
}

// EncodedSnapshot 带编码的快照实现
type EncodedSnapshot struct {
	data   []byte
	config *SnapshotConfig
}

// NewEncodedSnapshot 创建编码快照
func NewEncodedSnapshot(data []byte, cfg *SnapshotConfig) (*EncodedSnapshot, error) {
	newCfg, err := config.MergeConfig(DefaultSnapshotConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge snapshot config: %w", err)
	}
	return &EncodedSnapshot{
		data:   data,
		config: newCfg,
	}, nil
}

// Persist 实现 raft.FSMSnapshot 接口
func (s *EncodedSnapshot) Persist(sink raft.SnapshotSink) error {
	// 获取压缩器
	compressor, err := compress.New(s.config.Compression)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to create compressor: %w", err)
	}

	// 压缩数据
	compressed, err := compressor.Compress(s.data)
	if err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to compress data: %w", err)
	}

	// 构建头部
	header := &snapshotHeader{
		Magic:   snapshotMagic,
		Version: snapshotVersion,
		Flags:   compressionToFlag(s.config.Compression),
		DataLen: uint64(len(s.data)),
	}

	// 构建完整数据：header + compressed
	var buf bytes.Buffer
	buf.Write(header.encode())
	buf.Write(compressed)

	payload := buf.Bytes()

	// 计算校验和
	var crc uint32
	if s.config.EnableChecksum {
		hasher, err := checksum.New(s.config.Checksum)
		if err != nil {
			sink.Cancel()
			return fmt.Errorf("failed to create hasher: %w", err)
		}
		crc = hasher.Sum(payload)
	}

	// 写入 payload
	if _, err := sink.Write(payload); err != nil {
		sink.Cancel()
		return fmt.Errorf("failed to write payload: %w", err)
	}

	// 写入校验和
	if s.config.EnableChecksum {
		crcBuf := make([]byte, checksumSize)
		binary.BigEndian.PutUint32(crcBuf, crc)
		if _, err := sink.Write(crcBuf); err != nil {
			sink.Cancel()
			return fmt.Errorf("failed to write checksum: %w", err)
		}
	}

	return sink.Close()
}

// Release 实现 raft.FSMSnapshot 接口
func (s *EncodedSnapshot) Release() {}

// DecodeSnapshot 解码快照数据
func DecodeSnapshot(reader io.Reader, cfg *SnapshotConfig) ([]byte, error) {
	newCfg, err := config.MergeConfig(DefaultSnapshotConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge snapshot config: %w", err)
	}

	// 读取全部数据
	allData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read snapshot data: %w", err)
	}

	// 检查最小长度
	minLen := headerSize
	if newCfg.EnableChecksum {
		minLen += checksumSize
	}
	if len(allData) < minLen {
		return nil, fmt.Errorf("snapshot data too small: %d bytes", len(allData))
	}

	// 分离 payload 和校验和
	var payload []byte
	var storedCRC uint32
	if newCfg.EnableChecksum {
		payload = allData[:len(allData)-checksumSize]
		storedCRC = binary.BigEndian.Uint32(allData[len(allData)-checksumSize:])
	} else {
		payload = allData
	}

	// 验证校验和
	if newCfg.EnableChecksum {
		hasher, err := checksum.New(newCfg.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to create hasher: %w", err)
		}
		if !hasher.Verify(payload, storedCRC) {
			return nil, ErrSnapshotCorrupted
		}
	}

	// 解析头部
	var header snapshotHeader
	if err := header.decode(payload[:headerSize]); err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	// 验证魔数
	if header.Magic != snapshotMagic {
		return nil, fmt.Errorf("invalid snapshot magic: 0x%X", header.Magic)
	}

	// 验证版本
	if header.Version != snapshotVersion {
		return nil, fmt.Errorf("unsupported snapshot version: %d", header.Version)
	}

	// 获取压缩数据
	compressed := payload[headerSize:]

	// 解压数据
	compType := flagToCompression(header.Flags)
	compressor, err := compress.New(compType)
	if err != nil {
		return nil, fmt.Errorf("failed to create decompressor: %w", err)
	}

	decompressed, err := compressor.Decompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress data: %w", err)
	}

	// 验证数据长度
	if uint64(len(decompressed)) != header.DataLen {
		return nil, fmt.Errorf("data length mismatch: expected %d, got %d",
			header.DataLen, len(decompressed))
	}

	return decompressed, nil
}

// compressionToFlag 压缩类型转标志位
func compressionToFlag(t compress.Type) uint16 {
	switch t {
	case compress.TypeSnappy:
		return flagSnappy
	case compress.TypeZstd:
		return flagZstd
	case compress.TypeLZ4:
		return flagLZ4
	default:
		return flagNone
	}
}

// flagToCompression 标志位转压缩类型
func flagToCompression(flags uint16) compress.Type {
	switch {
	case flags&flagSnappy != 0:
		return compress.TypeSnappy
	case flags&flagZstd != 0:
		return compress.TypeZstd
	case flags&flagLZ4 != 0:
		return compress.TypeLZ4
	default:
		return compress.TypeNone
	}
}
