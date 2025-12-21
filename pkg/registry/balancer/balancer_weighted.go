package balancer

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

const (
	// WeightedRoundRobinBalancerName Weighted Round Robin 负载均衡器名称
	WeightedRoundRobinBalancerName = "weighted_round_robin"
	// WeightAttributeKey 用于在 Attributes 中存储权重的 key
	WeightAttributeKey = "weight"
)

func init() {
	// 注册 Weighted Round Robin 负载均衡器
	balancer.Register(newWeightedRoundRobinBuilder())
}

// weightedRoundRobinBuilder 实现 base.PickerBuilder
type weightedRoundRobinBuilder struct{}

func newWeightedRoundRobinBuilder() balancer.Builder {
	return base.NewBalancerBuilder(
		WeightedRoundRobinBalancerName,
		&weightedRoundRobinBuilder{},
		base.Config{HealthCheck: true},
	)
}

// Build 创建 Weighted Round Robin Picker
func (b *weightedRoundRobinBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	// 构建带权重的 SubConn 列表
	var weightedSCs []weightedSubConn
	for sc, scInfo := range info.ReadySCs {
		weight := 1 // 默认权重为 1

		// 从 Attributes 中获取权重（如果有）
		if w, ok := scInfo.Address.Attributes.Value(WeightAttributeKey).(int); ok && w > 0 {
			weight = w
		}

		weightedSCs = append(weightedSCs, weightedSubConn{
			subConn:       sc,
			weight:        weight,
			currentWeight: 0,
		})
	}

	return &weightedRoundRobinPicker{
		subConns: weightedSCs,
	}
}

// weightedSubConn 带权重的 SubConn
type weightedSubConn struct {
	subConn       balancer.SubConn
	weight        int // 配置的权重
	currentWeight int // 当前权重（动态计算）
}

// weightedRoundRobinPicker 实现加权轮询选择器
// 使用平滑加权轮询算法（Smooth Weighted Round-Robin）
// 算法说明：
// 1. 每次选择时，所有节点的 currentWeight += weight
// 2. 选择 currentWeight 最大的节点
// 3. 被选中节点的 currentWeight -= totalWeight
//
// 示例：节点 A(weight=5), B(weight=1), C(weight=1)
// 选择序列：A A A B A C A（7次选择中，A被选5次，B和C各1次）
type weightedRoundRobinPicker struct {
	subConns []weightedSubConn
	mu       sync.Mutex
}

// Pick 根据加权轮询算法选择连接
func (p *weightedRoundRobinPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.subConns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	// 计算总权重
	totalWeight := 0
	for i := range p.subConns {
		totalWeight += p.subConns[i].weight
	}

	// 所有节点的 currentWeight 增加其配置权重
	for i := range p.subConns {
		p.subConns[i].currentWeight += p.subConns[i].weight
	}

	// 找到 currentWeight 最大的节点
	maxIdx := 0
	maxWeight := p.subConns[0].currentWeight
	for i := 1; i < len(p.subConns); i++ {
		if p.subConns[i].currentWeight > maxWeight {
			maxIdx = i
			maxWeight = p.subConns[i].currentWeight
		}
	}

	// 被选中节点的 currentWeight 减去总权重
	p.subConns[maxIdx].currentWeight -= totalWeight

	return balancer.PickResult{SubConn: p.subConns[maxIdx].subConn}, nil
}
