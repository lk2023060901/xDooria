package balancer

import (
	"google.golang.org/grpc/attributes"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/resolver"
)

// mockSubConn 模拟 SubConn，用于测试
type mockSubConn struct {
	balancer.SubConn // 嵌入接口以满足 enforceSubConnEmbedding 要求
	id               string
}

func (m *mockSubConn) UpdateAddresses([]resolver.Address) {}
func (m *mockSubConn) Connect()                           {}
func (m *mockSubConn) GetOrBuildProducer(balancer.ProducerBuilder) (balancer.Producer, func()) {
	return nil, nil
}
func (m *mockSubConn) Shutdown()                                     {}
func (m *mockSubConn) RegisterHealthListener(func(balancer.SubConnState)) {}

// newAttributesWithWeight 创建带权重的 Attributes
func newAttributesWithWeight(weight int) *attributes.Attributes {
	return attributes.New(WeightAttributeKey, weight)
}

// abs 计算绝对值
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
