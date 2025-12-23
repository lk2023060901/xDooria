package serializer

import (
	"encoding/json"
	"errors"

	"google.golang.org/protobuf/proto"
)

// 错误定义
var (
	ErrInvalidProtoMessage = errors.New("serializer: invalid protobuf message")
)

// Serializer 序列化器接口
type Serializer interface {
	// Serialize 序列化
	Serialize(v any) ([]byte, error)
	// Deserialize 反序列化
	Deserialize(data []byte, v any) error
	// ContentType 内容类型（用于日志追踪）
	ContentType() string
}

// ===============================
// JSON 序列化器
// ===============================

// JSON JSON 序列化器
type JSON struct{}

// NewJSON 创建 JSON 序列化器
func NewJSON() *JSON {
	return &JSON{}
}

// Serialize 序列化为 JSON
func (s *JSON) Serialize(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Deserialize 从 JSON 反序列化
func (s *JSON) Deserialize(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// ContentType 返回内容类型
func (s *JSON) ContentType() string {
	return "application/json"
}

// ===============================
// Protobuf 序列化器
// ===============================

// Proto Protobuf 序列化器
type Proto struct{}

// NewProto 创建 Protobuf 序列化器
func NewProto() *Proto {
	return &Proto{}
}

// Serialize 序列化为 Protobuf
func (s *Proto) Serialize(v any) ([]byte, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return nil, ErrInvalidProtoMessage
	}
	return proto.Marshal(msg)
}

// Deserialize 从 Protobuf 反序列化
func (s *Proto) Deserialize(data []byte, v any) error {
	msg, ok := v.(proto.Message)
	if !ok {
		return ErrInvalidProtoMessage
	}
	return proto.Unmarshal(data, msg)
}

// ContentType 返回内容类型
func (s *Proto) ContentType() string {
	return "application/protobuf"
}

// ===============================
// 原始字节序列化器
// ===============================

// Raw 原始字节序列化器（不做任何转换）
type Raw struct{}

// NewRaw 创建原始字节序列化器
func NewRaw() *Raw {
	return &Raw{}
}

// Serialize 序列化（期望输入 []byte 或 string）
func (s *Raw) Serialize(v any) ([]byte, error) {
	switch val := v.(type) {
	case []byte:
		return val, nil
	case string:
		return []byte(val), nil
	default:
		return json.Marshal(v)
	}
}

// Deserialize 反序列化
func (s *Raw) Deserialize(data []byte, v any) error {
	switch ptr := v.(type) {
	case *[]byte:
		*ptr = make([]byte, len(data))
		copy(*ptr, data)
		return nil
	case *string:
		*ptr = string(data)
		return nil
	default:
		return json.Unmarshal(data, v)
	}
}

// ContentType 返回内容类型
func (s *Raw) ContentType() string {
	return "application/octet-stream"
}

// ===============================
// 默认序列化器
// ===============================

var defaultSerializer Serializer = NewJSON()

// SetDefault 设置默认序列化器
func SetDefault(s Serializer) {
	if s != nil {
		defaultSerializer = s
	}
}

// Default 获取默认序列化器
func Default() Serializer {
	return defaultSerializer
}
