package balancer

// Node 代表一个服务节点
type Node struct {
	// Address 节点地址
	Address string
	// Weight 权重（用于加权算法）
	Weight int
	// Metadata 元数据
	Metadata map[string]string
}

// PickInfo 选择时的上下文信息
type PickInfo struct {
	// Key 用于一致性哈希等需要 key 的算法
	Key string
}

// Balancer 负载均衡器接口（协议无关）
type Balancer interface {
	// Pick 从节点列表中选择一个节点
	Pick(nodes []*Node, info PickInfo) *Node
}

// Builder 负载均衡器构建器
type Builder interface {
	// Build 创建负载均衡器实例
	Build() Balancer
	// Name 返回负载均衡器名称
	Name() string
}
