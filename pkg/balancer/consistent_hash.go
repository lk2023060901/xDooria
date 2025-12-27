package balancer

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"
)

const ConsistentHashName = "consistent_hash"

type consistentHashBuilder struct {
	virtualNodes int
}

func NewConsistentHashBuilder() Builder {
	return &consistentHashBuilder{virtualNodes: 150}
}

func (b *consistentHashBuilder) Build() Balancer {
	return &consistentHashBalancer{
		virtualNodes: b.virtualNodes,
	}
}

func (b *consistentHashBuilder) Name() string {
	return ConsistentHashName
}

// consistentHashBalancer 实现一致性哈希算法
type consistentHashBalancer struct {
	virtualNodes int
	mu           sync.RWMutex
	ring         *hashRing
	lastNodes    string // 用于检测节点列表是否变化
}

func (b *consistentHashBalancer) Pick(nodes []*Node, info PickInfo) *Node {
	if len(nodes) == 0 {
		return nil
	}

	if info.Key == "" {
		return nil
	}

	b.mu.Lock()
	// 检测节点列表是否变化，需要重建哈希环
	nodeKey := b.buildNodeKey(nodes)
	if b.ring == nil || b.lastNodes != nodeKey {
		b.ring = b.buildRing(nodes)
		b.lastNodes = nodeKey
	}
	ring := b.ring
	b.mu.Unlock()

	return ring.get(info.Key)
}

func (b *consistentHashBalancer) buildNodeKey(nodes []*Node) string {
	key := ""
	for _, n := range nodes {
		key += n.Address + ","
	}
	return key
}

func (b *consistentHashBalancer) buildRing(nodes []*Node) *hashRing {
	ring := &hashRing{
		nodes:        make(map[uint32]*Node),
		sortedHashes: make([]uint32, 0, len(nodes)*b.virtualNodes),
	}

	for _, node := range nodes {
		for i := 0; i < b.virtualNodes; i++ {
			virtualKey := fmt.Sprintf("%s#%d", node.Address, i)
			hash := crc32.ChecksumIEEE([]byte(virtualKey))
			ring.nodes[hash] = node
			ring.sortedHashes = append(ring.sortedHashes, hash)
		}
	}

	sort.Slice(ring.sortedHashes, func(i, j int) bool {
		return ring.sortedHashes[i] < ring.sortedHashes[j]
	})

	return ring
}

// hashRing 哈希环
type hashRing struct {
	nodes        map[uint32]*Node
	sortedHashes []uint32
}

// get 根据 key 获取对应的节点
func (r *hashRing) get(key string) *Node {
	if len(r.sortedHashes) == 0 {
		return nil
	}

	hash := crc32.ChecksumIEEE([]byte(key))

	// 二分查找第一个 >= hash 的节点
	idx := sort.Search(len(r.sortedHashes), func(i int) bool {
		return r.sortedHashes[i] >= hash
	})

	// 如果没找到，使用第一个节点（环形）
	if idx == len(r.sortedHashes) {
		idx = 0
	}

	return r.nodes[r.sortedHashes[idx]]
}
