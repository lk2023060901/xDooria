// pkg/framer/framer.go
// Envelope 消息帧的编码和解码
package framer

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/compress"
	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/crypto"
	"github.com/lk2023060901/xdooria/pkg/framer/seqid"
	"github.com/lk2023060901/xdooria/pkg/framer/signer"
	"github.com/lk2023060901/xdooria/pkg/pool/bytebuff"
	"google.golang.org/protobuf/proto"

	pb "github.com/lk2023060901/xdooria-proto-common"
)

// Framer 消息帧处理器接口
type Framer interface {
	// Encode 编码消息为 Envelope
	Encode(op uint32, payload []byte) (*pb.Envelope, error)

	// Decode 解码 Envelope 并验证
	Decode(envelope *pb.Envelope) (op uint32, payload []byte, err error)
}

// Config Framer 配置
type Config struct {
	// 签名密钥
	SignKey []byte

	// 加密密钥（AES-256，32字节）
	EncryptKey []byte

	// 压缩算法类型
	CompressType compress.Type

	// SeqId 管理器配置
	SeqIdConfig *seqid.Config

	// 是否启用加密
	EnableEncrypt bool

	// 是否启用压缩
	EnableCompress bool

	// 压缩最小字节数（小于此值不压缩）
	CompressMinBytes int

	// 时间戳容差（秒），防重放攻击
	TimestampTolerance time.Duration
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		CompressType:       compress.TypeNone,
		EnableEncrypt:      true,
		EnableCompress:     true,
		CompressMinBytes:   256, // 小于 1KB 不压缩
		TimestampTolerance: 5 * time.Minute,
		SeqIdConfig:        seqid.DefaultConfig(),
	}
}

// frameImpl Framer 实现
type frameImpl struct {
	config *Config

	// 签名器
	signer signer.Signer

	// 加密器
	aes *crypto.AES

	// 压缩器
	compressor compress.Compressor

	// SeqId 管理器
	seqIdMgr seqid.Manager
}

// New 创建新的 Framer
func New(cfg *Config) (Framer, error) {
	// 使用 MergeConfig 确保配置完整
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	f := &frameImpl{
		config: newCfg,
	}

	// 初始化签名器
	if len(newCfg.SignKey) > 0 {
		hmac := crypto.NewHMACHasher(newCfg.SignKey)
		f.signer = signer.NewHMACSigner(hmac)
	}

	// 初始化加密器
	if newCfg.EnableEncrypt && len(newCfg.EncryptKey) > 0 {
		aes, err := crypto.NewAES(newCfg.EncryptKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create AES: %w", err)
		}
		f.aes = aes
	}

	// 初始化压缩器（始终初始化，使用 TypeNone 作为透传）
	compressor, err := compress.New(newCfg.CompressType)
	if err != nil {
		return nil, fmt.Errorf("failed to create compressor: %w", err)
	}
	f.compressor = compressor

	// 初始化 SeqId 管理器
	seqIdMgr, err := seqid.New(newCfg.SeqIdConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create seqid manager: %w", err)
	}
	f.seqIdMgr = seqIdMgr

	return f, nil
}

// Encode 编码消息为 Envelope
func (f *frameImpl) Encode(op uint32, payload []byte) (*pb.Envelope, error) {
	processedPayload := payload
	var flags uint32 = uint32(pb.MessageFlags_MESSAGE_FLAGS_NONE)

	// 1. 压缩（如果启用且满足最小字节数）
	if f.config.EnableCompress && len(processedPayload) >= f.config.CompressMinBytes {
		compressed, compressErr := f.compressor.Compress(processedPayload)
		if compressErr != nil {
			return nil, fmt.Errorf("compress failed: %w", compressErr)
		}
		processedPayload = compressed
		flags |= uint32(pb.MessageFlags_MESSAGE_FLAGS_COMPRESSED) // 压缩成功才设置标志位
	}

	// 2. 加密（如果启用）
	if f.config.EnableEncrypt && f.aes != nil {
		encrypted, err := f.aes.EncryptBytes(processedPayload)
		if err != nil {
			return nil, fmt.Errorf("encrypt failed: %w", err)
		}
		processedPayload = encrypted
		flags |= uint32(pb.MessageFlags_MESSAGE_FLAGS_ENCRYPTED) // 加密成功才设置标志位
	}

	// 3. 生成 SeqId
	seqId := f.seqIdMgr.Next()

	// 4. 构建 Header（不含签名）
	header := &pb.MessageHeader{
		Op:        op,
		SeqId:     seqId,
		Size:      uint32(len(processedPayload)),
		Flags:     flags,
		Timestamp: uint64(time.Now().Unix()),
		Sign:      nil, // 先不设置签名
	}

	// 5. 计算签名
	if f.signer != nil {
		signBuf, err := f.marshalHeaderWithoutSign(header, processedPayload)
		if err != nil {
			return nil, fmt.Errorf("marshal header for sign failed: %w", err)
		}
		signature, err := f.signer.Sign(signBuf.Bytes())
		bytebuff.Put(signBuf)
		if err != nil {
			return nil, fmt.Errorf("sign failed: %w", err)
		}
		header.Sign = signature
	}

	// 6. 构建 Envelope
	envelope := &pb.Envelope{
		Header:  header,
		Payload: processedPayload,
	}

	return envelope, nil
}

// Decode 解码 Envelope 并验证
func (f *frameImpl) Decode(envelope *pb.Envelope) (op uint32, payload []byte, err error) {
	if envelope == nil || envelope.Header == nil {
		return 0, nil, fmt.Errorf("invalid envelope: nil envelope or header")
	}

	header := envelope.Header
	processedPayload := envelope.Payload

	// 1. 验证时间戳（防重放攻击）
	now := time.Now().Unix()
	msgTime := int64(header.Timestamp)
	tolerance := int64(f.config.TimestampTolerance.Seconds())

	if now-msgTime > tolerance || msgTime-now > tolerance {
		return 0, nil, fmt.Errorf("timestamp out of tolerance: msg=%d, now=%d, tolerance=%d",
			msgTime, now, tolerance)
	}

	// 2. 验证签名
	if f.signer != nil {
		if len(header.Sign) == 0 {
			return 0, nil, fmt.Errorf("signature required but not present")
		}

		signBuf, err := f.marshalHeaderWithoutSign(header, processedPayload)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal header for verify failed: %w", err)
		}

		verifyErr := f.signer.Verify(signBuf.Bytes(), header.Sign)
		bytebuff.Put(signBuf)
		if verifyErr != nil {
			return 0, nil, fmt.Errorf("signature verification failed: %w", verifyErr)
		}
	}

	// 3. 验证 SeqId（防重放）
	if !f.seqIdMgr.Validate(header.SeqId, header.Timestamp) {
		return 0, nil, fmt.Errorf("invalid or duplicate seqId: %d", header.SeqId)
	}

	// 4. 解密（如果配置启用且消息有加密标志）
	if f.config.EnableEncrypt && header.Flags&uint32(pb.MessageFlags_MESSAGE_FLAGS_ENCRYPTED) != 0 {
		if f.aes == nil {
			return 0, nil, fmt.Errorf("message is encrypted but no key configured")
		}
		decrypted, err := f.aes.DecryptBytes(processedPayload)
		if err != nil {
			return 0, nil, fmt.Errorf("decrypt failed: %w", err)
		}
		processedPayload = decrypted
	}

	// 5. 解压（如果配置启用且消息有压缩标志）
	if f.config.EnableCompress && header.Flags&uint32(pb.MessageFlags_MESSAGE_FLAGS_COMPRESSED) != 0 {
		decompressed, err := f.compressor.Decompress(processedPayload)
		if err != nil {
			return 0, nil, fmt.Errorf("decompress failed: %w", err)
		}
		processedPayload = decompressed
	}

	return header.Op, processedPayload, nil
}

// signHeaderSize 签名 header 固定大小: op(4) + seqId(8) + size(4) + flags(4) + timestamp(8) = 28 bytes
const signHeaderSize = 28

// marshalHeaderWithoutSign 将 Header（不含签名）和 Payload 序列化用于签名
// 返回 buffer 需要调用方通过 bytebuff.Put 归还
// 格式: BigEndian(op) + BigEndian(seqId) + BigEndian(size) + BigEndian(flags) + BigEndian(timestamp) + payload
func (f *frameImpl) marshalHeaderWithoutSign(header *pb.MessageHeader, payload []byte) (*bytes.Buffer, error) {
	buf := bytebuff.Get(signHeaderSize + len(payload))

	// 写入字段（BigEndian）
	if err := binary.Write(buf, binary.BigEndian, header.Op); err != nil {
		bytebuff.Put(buf)
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, header.SeqId); err != nil {
		bytebuff.Put(buf)
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, header.Size); err != nil {
		bytebuff.Put(buf)
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, header.Flags); err != nil {
		bytebuff.Put(buf)
		return nil, err
	}
	if err := binary.Write(buf, binary.BigEndian, header.Timestamp); err != nil {
		bytebuff.Put(buf)
		return nil, err
	}

	// 写入 payload
	if _, err := buf.Write(payload); err != nil {
		bytebuff.Put(buf)
		return nil, err
	}

	return buf, nil
}

// Marshal 将 Envelope 序列化为字节数组（使用 protobuf）
func Marshal(envelope *pb.Envelope) ([]byte, error) {
	return proto.Marshal(envelope)
}

// Unmarshal 从字节数组反序列化 Envelope
func Unmarshal(data []byte) (*pb.Envelope, error) {
	envelope := &pb.Envelope{}
	if err := proto.Unmarshal(data, envelope); err != nil {
		return nil, err
	}
	return envelope, nil
}
