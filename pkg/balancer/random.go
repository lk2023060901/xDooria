package balancer

import "math/rand"

const RandomName = "random"

type randomBuilder struct{}

func NewRandomBuilder() Builder {
	return &randomBuilder{}
}

func (b *randomBuilder) Build() Balancer {
	return &randomBalancer{}
}

func (b *randomBuilder) Name() string {
	return RandomName
}

type randomBalancer struct{}

func (b *randomBalancer) Pick(nodes []*Node, _ PickInfo) *Node {
	if len(nodes) == 0 {
		return nil
	}
	return nodes[rand.Intn(len(nodes))]
}
