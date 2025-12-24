// pkg/websocket/types.go
package websocket

import "time"

// MessageType 消息类型
type MessageType int

const (
	// MessageTypeText 文本消息
	MessageTypeText MessageType = 1
	// MessageTypeBinary 二进制消息
	MessageTypeBinary MessageType = 2
	// MessageTypePing Ping 消息
	MessageTypePing MessageType = 9
	// MessageTypePong Pong 消息
	MessageTypePong MessageType = 10
)

// String 返回消息类型的字符串表示
func (t MessageType) String() string {
	switch t {
	case MessageTypeText:
		return "text"
	case MessageTypeBinary:
		return "binary"
	case MessageTypePing:
		return "ping"
	case MessageTypePong:
		return "pong"
	default:
		return "unknown"
	}
}

// ConnectionState 连接状态
type ConnectionState int

const (
	// StateDisconnected 未连接
	StateDisconnected ConnectionState = iota
	// StateConnecting 连接中
	StateConnecting
	// StateConnected 已连接
	StateConnected
	// StateReconnecting 重连中
	StateReconnecting
	// StateClosed 已关闭
	StateClosed
)

// String 返回连接状态的字符串表示
func (s ConnectionState) String() string {
	switch s {
	case StateDisconnected:
		return "disconnected"
	case StateConnecting:
		return "connecting"
	case StateConnected:
		return "connected"
	case StateReconnecting:
		return "reconnecting"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// Stats 统计信息
type Stats struct {
	// 连接统计
	TotalConnections  int64          `json:"total_connections"`
	ActiveConnections int64          `json:"active_connections"`
	ConnectionsPerIP  map[string]int `json:"connections_per_ip,omitempty"`

	// 消息统计
	MessagesSent     int64 `json:"messages_sent"`
	MessagesReceived int64 `json:"messages_received"`
	BytesSent        int64 `json:"bytes_sent"`
	BytesReceived    int64 `json:"bytes_received"`

	// 错误统计
	ErrorCount     int64 `json:"error_count"`
	ReconnectCount int64 `json:"reconnect_count"`
}

// ConnectionInfo 连接信息
type ConnectionInfo struct {
	ID          string                 `json:"id"`
	RemoteAddr  string                 `json:"remote_addr"`
	LocalAddr   string                 `json:"local_addr"`
	State       ConnectionState        `json:"state"`
	ConnectedAt time.Time              `json:"connected_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}
