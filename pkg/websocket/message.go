// pkg/websocket/message.go
package websocket

import "time"

// Message WebSocket 消息（传输层）
// 固定使用 BinaryMessage 传输，业务层编解码由 Framer 处理
type Message struct {
	// Data 消息数据
	Data []byte `json:"data"`
	// Timestamp 时间戳
	Timestamp time.Time `json:"timestamp"`
	// Metadata 元数据（可选，用于连接级别的上下文传递）
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewMessage 创建消息
func NewMessage(data []byte) *Message {
	return &Message{
		Data:      data,
		Timestamp: time.Now(),
	}
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
