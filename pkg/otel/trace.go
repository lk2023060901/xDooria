package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// 重导出常用类型，避免使用者直接依赖 go.opentelemetry.io/otel
type (
	// Span 表示一个追踪 span
	Span = trace.Span

	// SpanKind 表示 span 的类型
	SpanKind = trace.SpanKind

	// SpanStartOption span 启动选项
	SpanStartOption = trace.SpanStartOption

	// TracerOption tracer 选项
	TracerOption = trace.TracerOption

	// Attribute 属性键值对
	Attribute = attribute.KeyValue

	// Code 状态码
	Code = codes.Code
)

// SpanKind 常量
const (
	SpanKindUnspecified = trace.SpanKindUnspecified
	SpanKindInternal    = trace.SpanKindInternal
	SpanKindServer      = trace.SpanKindServer
	SpanKindClient      = trace.SpanKindClient
	SpanKindProducer    = trace.SpanKindProducer
	SpanKindConsumer    = trace.SpanKindConsumer
)

// Code 常量
const (
	CodeUnset = codes.Unset
	CodeError = codes.Error
	CodeOk    = codes.Ok
)

// Tracer 便捷函数：获取全局 Tracer
func Tracer(name string, opts ...trace.TracerOption) trace.Tracer {
	return otel.Tracer(name, opts...)
}

// GetTracerProvider 获取全局 TracerProvider
func GetTracerProvider() trace.TracerProvider {
	return otel.GetTracerProvider()
}

// WithSpanKind 设置 span 类型
func WithSpanKind(kind SpanKind) SpanStartOption {
	return trace.WithSpanKind(kind)
}

// WithAttributes 设置 span 属性
func WithAttributes(attrs ...Attribute) SpanStartOption {
	return trace.WithAttributes(attrs...)
}

// 属性构造函数
var (
	// String 创建字符串属性
	String = attribute.String

	// Int 创建整数属性
	Int = attribute.Int

	// Int64 创建 int64 属性
	Int64 = attribute.Int64

	// Float64 创建浮点数属性
	Float64 = attribute.Float64

	// Bool 创建布尔属性
	Bool = attribute.Bool

	// StringSlice 创建字符串切片属性
	StringSlice = attribute.StringSlice
)

// 常用的语义属性键（Messaging 相关）
const (
	// MessagingSystemKey 消息系统
	MessagingSystemKey = "messaging.system"

	// MessagingDestinationKey 消息目标
	MessagingDestinationKey = "messaging.destination"

	// MessagingDestinationKindKey 消息目标类型
	MessagingDestinationKindKey = "messaging.destination_kind"

	// MessagingMessageIDKey 消息 ID
	MessagingMessageIDKey = "messaging.message_id"

	// MessagingOperationKey 消息操作
	MessagingOperationKey = "messaging.operation"
)

// Kafka 相关属性键
const (
	// MessagingKafkaPartitionKey Kafka 分区
	MessagingKafkaPartitionKey = "messaging.kafka.partition"

	// MessagingKafkaOffsetKey Kafka 偏移量
	MessagingKafkaOffsetKey = "messaging.kafka.offset"

	// MessagingKafkaMessageKeyKey Kafka 消息键
	MessagingKafkaMessageKeyKey = "messaging.kafka.message_key"

	// MessagingKafkaConsumerGroupKey Kafka 消费者组
	MessagingKafkaConsumerGroupKey = "messaging.kafka.consumer_group"
)

// RPC 相关属性键
const (
	// RPCSystemKey RPC 系统
	RPCSystemKey = "rpc.system"

	// RPCServiceKey RPC 服务
	RPCServiceKey = "rpc.service"

	// RPCMethodKey RPC 方法
	RPCMethodKey = "rpc.method"

	// RPCGRPCStatusCodeKey gRPC 状态码
	RPCGRPCStatusCodeKey = "rpc.grpc.status_code"
)
