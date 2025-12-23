package kafka

import (
	"context"
	"time"
)

// Message 消息结构
type Message struct {
	// Topic 主题
	Topic string

	// Key 消息键（用于分区路由，同一 Key 的消息会路由到同一分区）
	Key []byte

	// Value 消息值
	Value []byte

	// Headers 消息头（元数据，如 trace_id、event_type 等）
	Headers map[string]string

	// Partition 分区（发送时可选，消费时填充）
	Partition int

	// Offset 偏移量（消费时填充）
	Offset int64

	// Timestamp 时间戳
	Timestamp time.Time
}

// Handler 消息处理器
type Handler func(ctx context.Context, msg *Message) error

// BatchHandler 批量消息处理器
type BatchHandler func(ctx context.Context, msgs []*Message) error

// Middleware 消费者中间件
type Middleware func(Handler) Handler

// ProducerMiddleware 生产者中间件
type ProducerMiddleware func(ctx context.Context, msg *Message, next func(context.Context, *Message) error) error

// ConsumerState 消费者状态
type ConsumerState int32

const (
	// ConsumerStateIdle 空闲状态
	ConsumerStateIdle ConsumerState = iota

	// ConsumerStateRunning 运行中
	ConsumerStateRunning

	// ConsumerStateStopping 停止中
	ConsumerStateStopping

	// ConsumerStateStopped 已停止
	ConsumerStateStopped
)

// String 返回状态字符串
func (s ConsumerState) String() string {
	switch s {
	case ConsumerStateIdle:
		return "idle"
	case ConsumerStateRunning:
		return "running"
	case ConsumerStateStopping:
		return "stopping"
	case ConsumerStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// PublishResult 发布结果
type PublishResult struct {
	Topic     string
	Partition int
	Offset    int64
	Error     error
}

// TopicPartition 主题分区
type TopicPartition struct {
	Topic     string
	Partition int
}

// PartitionOffset 分区偏移量
type PartitionOffset struct {
	Topic     string
	Partition int
	Offset    int64
}

// ConsumerStats 消费者统计
type ConsumerStats struct {
	// MessagesConsumed 消费的消息数
	MessagesConsumed int64

	// MessagesSucceeded 处理成功的消息数
	MessagesSucceeded int64

	// MessagesFailed 处理失败的消息数
	MessagesFailed int64

	// LastMessageTime 最后一条消息时间
	LastMessageTime time.Time
}

// ProducerStats 生产者统计
type ProducerStats struct {
	// MessagesProduced 发送的消息数
	MessagesProduced int64

	// MessagesSucceeded 发送成功的消息数
	MessagesSucceeded int64

	// MessagesFailed 发送失败的消息数
	MessagesFailed int64

	// LastMessageTime 最后一条消息时间
	LastMessageTime time.Time
}
