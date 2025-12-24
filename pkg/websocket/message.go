// pkg/websocket/message.go
package websocket

import (
	"time"

	"github.com/lk2023060901/xdooria/pkg/serializer"
)

// Message WebSocket 消息
type Message struct {
	// Type 消息类型
	Type MessageType `json:"type"`
	// Data 消息数据
	Data []byte `json:"data"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Metadata 元数据（可选）
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewTextMessage 创建文本消息
func NewTextMessage(data []byte) *Message {
	return &Message{
		Type:      MessageTypeText,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewTextMessageString 创建文本消息（字符串）
func NewTextMessageString(data string) *Message {
	return NewTextMessage([]byte(data))
}

// NewBinaryMessage 创建二进制消息
func NewBinaryMessage(data []byte) *Message {
	return &Message{
		Type:      MessageTypeBinary,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// NewMessage 使用序列化器创建消息
func NewMessage(msgType MessageType, v interface{}, s serializer.Serializer) (*Message, error) {
	if s == nil {
		s = serializer.Default()
	}
	data, err := s.Serialize(v)
	if err != nil {
		return nil, err
	}
	return &Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

// NewJSONMessage 创建 JSON 消息
func NewJSONMessage(v interface{}) (*Message, error) {
	return NewMessage(MessageTypeText, v, serializer.NewJSON())
}

// Unmarshal 反序列化消息数据
func (m *Message) Unmarshal(v interface{}, s serializer.Serializer) error {
	if s == nil {
		s = serializer.Default()
	}
	return s.Deserialize(m.Data, v)
}

// UnmarshalJSON 反序列化 JSON 消息
func (m *Message) UnmarshalJSON(v interface{}) error {
	return m.Unmarshal(v, serializer.NewJSON())
}

// String 返回消息数据的字符串表示
func (m *Message) String() string {
	return string(m.Data)
}

// Len 返回消息数据长度
func (m *Message) Len() int {
	return len(m.Data)
}

// WithMetadata 设置元数据
func (m *Message) WithMetadata(key string, value interface{}) *Message {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
	return m
}

// GetMetadata 获取元数据
func (m *Message) GetMetadata(key string) (interface{}, bool) {
	if m.Metadata == nil {
		return nil, false
	}
	v, ok := m.Metadata[key]
	return v, ok
}

// Clone 克隆消息
func (m *Message) Clone() *Message {
	clone := &Message{
		Type:      m.Type,
		Data:      make([]byte, len(m.Data)),
		Timestamp: m.Timestamp,
	}
	copy(clone.Data, m.Data)
	if m.Metadata != nil {
		clone.Metadata = make(map[string]interface{}, len(m.Metadata))
		for k, v := range m.Metadata {
			clone.Metadata[k] = v
		}
	}
	return clone
}
