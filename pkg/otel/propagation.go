package otel

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// TextMapCarrier 文本映射载体接口
type TextMapCarrier = propagation.TextMapCarrier

// TextMapPropagator 文本映射传播器接口
type TextMapPropagator = propagation.TextMapPropagator

// MapCarrier map[string]string 类型的载体适配器
type MapCarrier propagation.MapCarrier

// Get 获取键值
func (c MapCarrier) Get(key string) string {
	return propagation.MapCarrier(c).Get(key)
}

// Set 设置键值
func (c MapCarrier) Set(key, value string) {
	propagation.MapCarrier(c).Set(key, value)
}

// Keys 获取所有键
func (c MapCarrier) Keys() []string {
	return propagation.MapCarrier(c).Keys()
}

// SetTextMapPropagator 设置全局文本传播器
func SetTextMapPropagator(p propagation.TextMapPropagator) {
	otel.SetTextMapPropagator(p)
}

// GetTextMapPropagator 获取全局文本传播器
func GetTextMapPropagator() propagation.TextMapPropagator {
	return otel.GetTextMapPropagator()
}

// NewCompositeTextMapPropagator 创建组合传播器
func NewCompositeTextMapPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// TraceContext 返回 W3C Trace Context 传播器
func TraceContext() propagation.TextMapPropagator {
	return propagation.TraceContext{}
}

// Baggage 返回 W3C Baggage 传播器
func Baggage() propagation.TextMapPropagator {
	return propagation.Baggage{}
}
