package balancer

import "sync/atomic"

const RoundRobinName = "round_robin"

type roundRobinBuilder struct{}

func NewRoundRobinBuilder() Builder {
	return &roundRobinBuilder{}
}

func (b *roundRobinBuilder) Build() Balancer {
	return &roundRobinBalancer{}
}

func (b *roundRobinBuilder) Name() string {
	return RoundRobinName
}

type roundRobinBalancer struct {
	counter uint64
}

func (b *roundRobinBalancer) Pick(nodes []*Node, _ PickInfo) *Node {
	if len(nodes) == 0 {
		return nil
	}
	idx := atomic.AddUint64(&b.counter, 1) - 1
	return nodes[idx%uint64(len(nodes))]
}
