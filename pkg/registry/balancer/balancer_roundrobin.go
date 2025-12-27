package balancer

import (
	"strconv"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"

	genericbalancer "github.com/lk2023060901/xdooria/pkg/balancer"
)

const (
	// RoundRobinBalancerName Round Robin 负载均衡器名称
	RoundRobinBalancerName = "round_robin"
)

func init() {
	// 注册 Round Robin 负载均衡器
	balancer.Register(newRoundRobinBuilder())
}

// roundRobinBuilder 实现 base.PickerBuilder
type roundRobinBuilder struct {
	balancer genericbalancer.Balancer
}

func newRoundRobinBuilder() balancer.Builder {
	return base.NewBalancerBuilder(
		RoundRobinBalancerName,
		&roundRobinBuilder{
			balancer: genericbalancer.New(genericbalancer.RoundRobinName),
		},
		base.Config{HealthCheck: true},
	)
}

// Build 创建 Round Robin Picker
func (b *roundRobinBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	nodes := make([]*genericbalancer.Node, 0, len(info.ReadySCs))
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
		nodes = append(nodes, &genericbalancer.Node{Address: strconv.Itoa(len(scs) - 1)})
	}

	return &roundRobinPicker{
		subConns: scs,
		nodes:    nodes,
		balancer: b.balancer,
	}
}

// roundRobinPicker 实现轮询选择器
type roundRobinPicker struct {
	subConns []balancer.SubConn
	nodes    []*genericbalancer.Node
	balancer genericbalancer.Balancer
}

// Pick 选择下一个连接
func (p *roundRobinPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	node := p.balancer.Pick(p.nodes, genericbalancer.PickInfo{})
	if node == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	idx, _ := strconv.Atoi(node.Address)
	return balancer.PickResult{SubConn: p.subConns[idx]}, nil
}
