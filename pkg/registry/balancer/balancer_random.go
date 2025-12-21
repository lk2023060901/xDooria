package balancer

import (
	"math/rand"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
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
	for sc := range info.ReadySCs {
		scs = append(scs, sc)
	}

	return &randomPicker{
		subConns: scs,
	}
}

// randomPicker 实现随机选择器
type randomPicker struct {
	subConns []balancer.SubConn
	mu       sync.Mutex
}

// Pick 随机选择一个连接
func (p *randomPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.subConns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	// 随机选择一个索引
	idx := rand.Intn(len(p.subConns))

	return balancer.PickResult{SubConn: p.subConns[idx]}, nil
}
