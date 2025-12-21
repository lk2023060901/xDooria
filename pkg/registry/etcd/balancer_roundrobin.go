package etcd

import (
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
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
type roundRobinBuilder struct{}

func newRoundRobinBuilder() balancer.Builder {
	return base.NewBalancerBuilder(RoundRobinBalancerName, &roundRobinBuilder{}, base.Config{HealthCheck: true})
}

// Build 创建 Round Robin Picker
func (b *roundRobinBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	scs := make([]balancer.SubConn, 0, len(info.ReadySCs))
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
	}

	return &roundRobinPicker{
		subConns: scs,
		next:     0,
	}
}

// roundRobinPicker 实现轮询选择器
type roundRobinPicker struct {
	subConns []balancer.SubConn
	mu       sync.Mutex
	next     int
}

// Pick 选择下一个连接
func (p *roundRobinPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	sc := p.subConns[p.next]
	p.next = (p.next + 1) % len(p.subConns)
	p.mu.Unlock()

	return balancer.PickResult{SubConn: sc}, nil
}
