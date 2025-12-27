// Package codec 提供通用的消息编解码框架，适配 WebSocket、TCP、gRPC 等传输层。
package codec

import (
	"github.com/lk2023060901/xdooria/pkg/network/framer"

	pb "github.com/lk2023060901/xdooria-proto-common"
)

// Codec 消息编解码器，封装 framer 提供统一的编解码接口。
type Codec interface {
	// Encode 编码消息为 Envelope
	Encode(op uint32, payload []byte) (*pb.Envelope, error)

	// Decode 解码 Envelope，返回 op 和 payload
	Decode(envelope *pb.Envelope) (op uint32, payload []byte, err error)

	// Marshal 序列化 Envelope 为字节数组
	Marshal(envelope *pb.Envelope) ([]byte, error)

	// Unmarshal 反序列化字节数组为 Envelope
	Unmarshal(data []byte) (*pb.Envelope, error)
}

// codec Codec 实现，封装 framer
type codec struct {
	framer framer.Framer
}

// New 创建 Codec
func New(f framer.Framer) Codec {
	return &codec{framer: f}
}

// NewWithConfig 使用配置创建 Codec
func NewWithConfig(cfg *framer.Config) (Codec, error) {
	f, err := framer.New(cfg)
	if err != nil {
		return nil, err
	}
	return &codec{framer: f}, nil
}

// Encode 编码消息
func (c *codec) Encode(op uint32, payload []byte) (*pb.Envelope, error) {
	return c.framer.Encode(op, payload)
}

// Decode 解码消息
func (c *codec) Decode(envelope *pb.Envelope) (op uint32, payload []byte, err error) {
	return c.framer.Decode(envelope)
}

// Marshal 序列化 Envelope
func (c *codec) Marshal(envelope *pb.Envelope) ([]byte, error) {
	return framer.Marshal(envelope)
}

// Unmarshal 反序列化 Envelope
func (c *codec) Unmarshal(data []byte) (*pb.Envelope, error) {
	return framer.Unmarshal(data)
}
