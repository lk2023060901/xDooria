package balancer

import "sync"

const WeightedName = "weighted"

type weightedBuilder struct{}

func NewWeightedBuilder() Builder {
	return &weightedBuilder{}
}

func (b *weightedBuilder) Build() Balancer {
	return &weightedBalancer{
		weights: make(map[string]int),
	}
}

func (b *weightedBuilder) Name() string {
	return WeightedName
}

// weightedBalancer 实现平滑加权轮询算法（Smooth Weighted Round-Robin）
// 算法说明：
// 1. 每次选择时，所有节点的 currentWeight += weight
// 2. 选择 currentWeight 最大的节点
// 3. 被选中节点的 currentWeight -= totalWeight
//
// 示例：节点 A(weight=5), B(weight=1), C(weight=1)
// 选择序列：A A A B A C A（7次选择中，A被选5次，B和C各1次）
type weightedBalancer struct {
	mu      sync.Mutex
	weights map[string]int // address -> currentWeight
}

func (b *weightedBalancer) Pick(nodes []*Node, _ PickInfo) *Node {
	if len(nodes) == 0 {
		return nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	// 计算总权重，并更新 currentWeight
	totalWeight := 0
	for _, node := range nodes {
		w := node.Weight
		if w <= 0 {
			w = 1
		}
		totalWeight += w

		// 增加当前权重
		b.weights[node.Address] += w
	}

	// 找到 currentWeight 最大的节点
	var maxNode *Node
	maxWeight := 0
	for _, node := range nodes {
		if cw := b.weights[node.Address]; maxNode == nil || cw > maxWeight {
			maxNode = node
			maxWeight = cw
		}
	}

	// 被选中节点的 currentWeight 减去总权重
	if maxNode != nil {
		b.weights[maxNode.Address] -= totalWeight
	}

	return maxNode
}
