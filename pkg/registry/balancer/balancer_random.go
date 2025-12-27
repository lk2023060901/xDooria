package balancer

import (
	"strconv"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"

	genericbalancer "github.com/lk2023060901/xdooria/pkg/balancer"
)

const (
	// RandomBalancerName Random 负载均衡器名称
	RandomBalancerName = "random"
)

func init() {
	// 注册 Random 负载均衡器
	balancer.Register(newRandomBuilder())
}

// randomBuilder 实现 base.PickerBuilder
type randomBuilder struct{}

func newRandomBuilder() balancer.Builder {
	return base.NewBalancerBuilder(
		RandomBalancerName,
		&randomBuilder{},
		base.Config{HealthCheck: true},
	)
}

// Build 创建 Random Picker
func (b *randomBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	nodes := make([]*genericbalancer.Node, 0, len(info.ReadySCs))
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
		nodes = append(nodes, &genericbalancer.Node{Address: strconv.Itoa(len(scs) - 1)})
	}

	return &randomPicker{
		subConns: scs,
		nodes:    nodes,
		balancer: genericbalancer.New(genericbalancer.RandomName),
	}
}

// randomPicker 实现随机选择器
type randomPicker struct {
	subConns []balancer.SubConn
	nodes    []*genericbalancer.Node
	balancer genericbalancer.Balancer
}

// Pick 随机选择一个连接
func (p *randomPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	node := p.balancer.Pick(p.nodes, genericbalancer.PickInfo{})
	if node == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	idx, _ := strconv.Atoi(node.Address)
	return balancer.PickResult{SubConn: p.subConns[idx]}, nil
}
