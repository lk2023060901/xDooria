package kafka

import "errors"

var (
	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("kafka: invalid config")

	// ErrNoBrokers 无 broker 地址
	ErrNoBrokers = errors.New("kafka: no brokers configured")

	// ErrEmptyTopic 空主题
	ErrEmptyTopic = errors.New("kafka: empty topic")

	// ErrEmptyGroupID 空消费者组 ID
	ErrEmptyGroupID = errors.New("kafka: empty group id")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("kafka: client is closed")

	// ErrProducerClosed 生产者已关闭
	ErrProducerClosed = errors.New("kafka: producer is closed")

	// ErrConsumerClosed 消费者已关闭
	ErrConsumerClosed = errors.New("kafka: consumer is closed")

	// ErrConsumerAlreadyRunning 消费者已在运行
	ErrConsumerAlreadyRunning = errors.New("kafka: consumer is already running")

	// ErrConsumerNotRunning 消费者未运行
	ErrConsumerNotRunning = errors.New("kafka: consumer is not running")

	// ErrInvalidProtoMessage 无效的 Protobuf 消息
	ErrInvalidProtoMessage = errors.New("kafka: invalid protobuf message")

	// ErrNoHandler 无消息处理器
	ErrNoHandler = errors.New("kafka: no handler provided")

	// ErrNoTopics 无订阅主题
	ErrNoTopics = errors.New("kafka: no topics provided")

	// ErrConsumerPanic 消费者 panic
	ErrConsumerPanic = errors.New("kafka: consumer panic")

	// ErrProducerPanic 生产者 panic
	ErrProducerPanic = errors.New("kafka: producer panic")
)
