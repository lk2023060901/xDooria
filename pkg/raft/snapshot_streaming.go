// pkg/raft/snapshot_streaming.go
package raft

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/hashicorp/raft"
	"github.com/lk2023060901/xdooria/pkg/checksum"
	"github.com/lk2023060901/xdooria/pkg/compress"
	"github.com/lk2023060901/xdooria/pkg/config"
)

// MessageType 快照消息类型
type MessageType uint8

const (
	// MessageTypeKV KV 数据
	MessageTypeKV MessageType = iota + 1
	// MessageTypeCustom 自定义数据
	MessageTypeCustom

	// MessageTypeEOF 快照结束标记
	MessageTypeEOF MessageType = 0xFF
)

// SnapshotWriter 快照写入器
type SnapshotWriter struct {
	sink       raft.SnapshotSink
	compressor compress.Compressor
	hasher     checksum.Hasher
	config     *SnapshotConfig

	// 用于计算整体校验和
	written []byte
}

// NewSnapshotWriter 创建快照写入器
func NewSnapshotWriter(sink raft.SnapshotSink, cfg *SnapshotConfig) (*SnapshotWriter, error) {
	newCfg, err := config.MergeConfig(DefaultSnapshotConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	compressor, err := compress.New(newCfg.Compression)
	if err != nil {
		return nil, fmt.Errorf("failed to create compressor: %w", err)
	}

	var hasher checksum.Hasher
	if newCfg.EnableChecksum {
		hasher, err = checksum.New(newCfg.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to create hasher: %w", err)
		}
	}

	return &SnapshotWriter{
		sink:       sink,
		compressor: compressor,
		hasher:     hasher,
		config:     newCfg,
		written:    make([]byte, 0, 1024*1024),
	}, nil
}

// WriteRecord 写入一条记录
// 格式: type(1) + origLen(4) + compLen(4) + data
func (w *SnapshotWriter) WriteRecord(msgType MessageType, data []byte) error {
	// 压缩数据
	compressed, err := w.compressor.Compress(data)
	if err != nil {
		return fmt.Errorf("failed to compress record: %w", err)
	}

	// 构建记录
	record := make([]byte, 1+4+4+len(compressed))
	record[0] = byte(msgType)
	binary.BigEndian.PutUint32(record[1:5], uint32(len(data)))
	binary.BigEndian.PutUint32(record[5:9], uint32(len(compressed)))
	copy(record[9:], compressed)

	// 写入 sink
	if _, err := w.sink.Write(record); err != nil {
		return fmt.Errorf("failed to write record: %w", err)
	}

	// 累积用于校验和计算
	if w.config.EnableChecksum {
		w.written = append(w.written, record...)
	}

	return nil
}

// WriteKV 写入 KV 数据
func (w *SnapshotWriter) WriteKV(key string, value []byte) error {
	keyBytes := []byte(key)
	data := make([]byte, 4+len(keyBytes)+len(value))
	binary.BigEndian.PutUint32(data[0:4], uint32(len(keyBytes)))
	copy(data[4:4+len(keyBytes)], keyBytes)
	copy(data[4+len(keyBytes):], value)

	return w.WriteRecord(MessageTypeKV, data)
}

// Close 完成写入并关闭
func (w *SnapshotWriter) Close() error {
	// 写入 EOF 标记
	eof := []byte{byte(MessageTypeEOF)}
	if _, err := w.sink.Write(eof); err != nil {
		w.sink.Cancel()
		return fmt.Errorf("failed to write EOF: %w", err)
	}

	// 写入校验和
	if w.config.EnableChecksum {
		w.written = append(w.written, eof...)
		crc := w.hasher.Sum(w.written)
		crcBuf := make([]byte, 4)
		binary.BigEndian.PutUint32(crcBuf, crc)
		if _, err := w.sink.Write(crcBuf); err != nil {
			w.sink.Cancel()
			return fmt.Errorf("failed to write checksum: %w", err)
		}
	}

	return w.sink.Close()
}

// Cancel 取消写入
func (w *SnapshotWriter) Cancel() error {
	return w.sink.Cancel()
}

// SnapshotReader 快照读取器
type SnapshotReader struct {
	reader     io.Reader
	compressor compress.Compressor
	hasher     checksum.Hasher
	config     *SnapshotConfig

	readData []byte
	eof      bool
}

// NewSnapshotReader 创建快照读取器
func NewSnapshotReader(reader io.Reader, cfg *SnapshotConfig) (*SnapshotReader, error) {
	newCfg, err := config.MergeConfig(DefaultSnapshotConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	compressor, err := compress.New(newCfg.Compression)
	if err != nil {
		return nil, fmt.Errorf("failed to create compressor: %w", err)
	}

	var hasher checksum.Hasher
	if newCfg.EnableChecksum {
		hasher, err = checksum.New(newCfg.Checksum)
		if err != nil {
			return nil, fmt.Errorf("failed to create hasher: %w", err)
		}
	}

	return &SnapshotReader{
		reader:     reader,
		compressor: compressor,
		hasher:     hasher,
		config:     newCfg,
		readData:   make([]byte, 0, 1024*1024),
	}, nil
}

// SnapshotRecord 快照记录
type SnapshotRecord struct {
	Type MessageType
	Data []byte
}

// ReadRecord 读取一条记录
func (r *SnapshotReader) ReadRecord() (*SnapshotRecord, error) {
	if r.eof {
		return nil, io.EOF
	}

	// 读取类型字节
	typeBuf := make([]byte, 1)
	if _, err := io.ReadFull(r.reader, typeBuf); err != nil {
		return nil, fmt.Errorf("failed to read record type: %w", err)
	}

	msgType := MessageType(typeBuf[0])

	// 检查 EOF
	if msgType == MessageTypeEOF {
		r.eof = true
		r.readData = append(r.readData, typeBuf...)

		// 验证校验和
		if r.config.EnableChecksum {
			crcBuf := make([]byte, 4)
			if _, err := io.ReadFull(r.reader, crcBuf); err != nil {
				return nil, fmt.Errorf("failed to read checksum: %w", err)
			}
			storedCRC := binary.BigEndian.Uint32(crcBuf)
			if !r.hasher.Verify(r.readData, storedCRC) {
				return nil, ErrSnapshotCorrupted
			}
		}

		return &SnapshotRecord{Type: MessageTypeEOF}, nil
	}

	// 读取长度信息
	lenBuf := make([]byte, 8)
	if _, err := io.ReadFull(r.reader, lenBuf); err != nil {
		return nil, fmt.Errorf("failed to read record length: %w", err)
	}

	origLen := binary.BigEndian.Uint32(lenBuf[0:4])
	compLen := binary.BigEndian.Uint32(lenBuf[4:8])

	// 读取压缩数据
	compressed := make([]byte, compLen)
	if _, err := io.ReadFull(r.reader, compressed); err != nil {
		return nil, fmt.Errorf("failed to read record data: %w", err)
	}

	// 累积用于校验和验证
	if r.config.EnableChecksum {
		r.readData = append(r.readData, typeBuf...)
		r.readData = append(r.readData, lenBuf...)
		r.readData = append(r.readData, compressed...)
	}

	// 解压数据
	decompressed, err := r.compressor.Decompress(compressed)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress record: %w", err)
	}

	if uint32(len(decompressed)) != origLen {
		return nil, fmt.Errorf("record length mismatch: expected %d, got %d", origLen, len(decompressed))
	}

	return &SnapshotRecord{
		Type: msgType,
		Data: decompressed,
	}, nil
}

// ParseKV 从 KV 记录中解析 key 和 value
func ParseKV(record *SnapshotRecord) (key string, value []byte, err error) {
	if record.Type != MessageTypeKV {
		return "", nil, fmt.Errorf("expected KV record, got type %d", record.Type)
	}

	if len(record.Data) < 4 {
		return "", nil, fmt.Errorf("KV record too short")
	}

	keyLen := binary.BigEndian.Uint32(record.Data[0:4])
	if len(record.Data) < int(4+keyLen) {
		return "", nil, fmt.Errorf("KV record data incomplete")
	}

	key = string(record.Data[4 : 4+keyLen])
	value = record.Data[4+keyLen:]
	return key, value, nil
}

// SimpleStreamingSnapshot 简单的流式快照实现
type SimpleStreamingSnapshot struct {
	data   map[string][]byte
	config *SnapshotConfig
}

// NewSimpleStreamingSnapshot 创建简单流式快照
func NewSimpleStreamingSnapshot(data map[string][]byte, cfg *SnapshotConfig) *SimpleStreamingSnapshot {
	return &SimpleStreamingSnapshot{
		data:   data,
		config: cfg,
	}
}

// Persist 实现 raft.FSMSnapshot 接口
func (s *SimpleStreamingSnapshot) Persist(sink raft.SnapshotSink) error {
	writer, err := NewSnapshotWriter(sink, s.config)
	if err != nil {
		sink.Cancel()
		return err
	}

	// 流式写入每个 KV
	for key, value := range s.data {
		if err := writer.WriteKV(key, value); err != nil {
			writer.Cancel()
			return err
		}
	}

	return writer.Close()
}

// Release 实现 raft.FSMSnapshot 接口
func (s *SimpleStreamingSnapshot) Release() {}

// RestoreStreamingSnapshot 从流式快照恢复数据
func RestoreStreamingSnapshot(reader io.Reader, cfg *SnapshotConfig) (map[string][]byte, error) {
	snapReader, err := NewSnapshotReader(reader, cfg)
	if err != nil {
		return nil, err
	}

	data := make(map[string][]byte)
	for {
		record, err := snapReader.ReadRecord()
		if err != nil {
			return nil, err
		}

		if record.Type == MessageTypeEOF {
			break
		}

		if record.Type == MessageTypeKV {
			key, value, err := ParseKV(record)
			if err != nil {
				return nil, fmt.Errorf("failed to parse KV record: %w", err)
			}
			data[key] = value
		}
	}

	return data, nil
}
