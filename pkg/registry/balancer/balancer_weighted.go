package balancer

import (
	"strconv"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"

	genericbalancer "github.com/lk2023060901/xdooria/pkg/balancer"
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

	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	nodes := make([]*genericbalancer.Node, 0, len(info.ReadySCs))
	idx := 0
	for sc, scInfo := range info.ReadySCs {
		weight := 1 // 默认权重为 1

		// 从 Attributes 中获取权重（如果有）
		if w, ok := scInfo.Address.Attributes.Value(WeightAttributeKey).(int); ok && w > 0 {
			weight = w
		}

		scs = append(scs, sc)
		nodes = append(nodes, &genericbalancer.Node{
			Address: strconv.Itoa(idx),
			Weight:  weight,
		})
		idx++
	}

	return &weightedRoundRobinPicker{
		subConns: scs,
		nodes:    nodes,
		balancer: genericbalancer.New(genericbalancer.WeightedName),
	}
}

// weightedRoundRobinPicker 实现加权轮询选择器
type weightedRoundRobinPicker struct {
	subConns []balancer.SubConn
	nodes    []*genericbalancer.Node
	balancer genericbalancer.Balancer
}

// Pick 根据加权轮询算法选择连接
func (p *weightedRoundRobinPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	node := p.balancer.Pick(p.nodes, genericbalancer.PickInfo{})
	if node == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	idx, _ := strconv.Atoi(node.Address)
	return balancer.PickResult{SubConn: p.subConns[idx]}, nil
}
