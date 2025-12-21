package etcd

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
	"google.golang.org/grpc/metadata"
)

const (
	// ConsistentHashBalancerName 一致性哈希负载均衡器名称
	ConsistentHashBalancerName = "consistent_hash"
	// HashKeyHeader 用于传递哈希 key 的 metadata header
	HashKeyHeader = "hash-key"
)

func init() {
	// 注册一致性哈希负载均衡器
	balancer.Register(newConsistentHashBuilder())
}

// consistentHashBuilder 实现 base.PickerBuilder
type consistentHashBuilder struct {
	virtualNodes int
}

func newConsistentHashBuilder() balancer.Builder {
	return base.NewBalancerBuilder(
		ConsistentHashBalancerName,
		&consistentHashBuilder{virtualNodes: 150}, // 默认虚拟节点数
		base.Config{HealthCheck: true},
	)
}

// Build 创建 Consistent Hash Picker
func (b *consistentHashBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	ring := &hashRing{
		nodes:        make(map[uint32]balancer.SubConn),
		sortedHashes: make([]uint32, 0),
	}

	// 为每个 SubConn 创建虚拟节点
	for sc, scInfo := range info.ReadySCs {
		addr := scInfo.Address.Addr
		for i := 0; i < b.virtualNodes; i++ {
			virtualKey := fmt.Sprintf("%s#%d", addr, i)
			hash := crc32.ChecksumIEEE([]byte(virtualKey))
			ring.nodes[hash] = sc
			ring.sortedHashes = append(ring.sortedHashes, hash)
		}
	}

	// 排序哈希值
	sort.Slice(ring.sortedHashes, func(i, j int) bool {
		return ring.sortedHashes[i] < ring.sortedHashes[j]
	})

	return &consistentHashPicker{
		ring: ring,
	}
}

// hashRing 哈希环
type hashRing struct {
	nodes        map[uint32]balancer.SubConn
	sortedHashes []uint32
	mu           sync.RWMutex
}

// get 根据 key 获取对应的 SubConn
func (r *hashRing) get(key string) (balancer.SubConn, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.sortedHashes) == 0 {
		return nil, balancer.ErrNoSubConnAvailable
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

	return r.nodes[r.sortedHashes[idx]], nil
}

// consistentHashPicker 实现一致性哈希选择器
type consistentHashPicker struct {
	ring *hashRing
}

// Pick 根据 hash-key metadata 选择连接
func (p *consistentHashPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	// 从 metadata 中获取 hash key
	var hashKey string
	if md, ok := metadata.FromOutgoingContext(info.Ctx); ok {
		if keys := md.Get(HashKeyHeader); len(keys) > 0 {
			hashKey = keys[0]
		}
	}

	// 如果没有提供 hash key，返回错误
	if hashKey == "" {
		return balancer.PickResult{}, fmt.Errorf("consistent hash requires %s in metadata", HashKeyHeader)
	}

	sc, err := p.ring.get(hashKey)
	if err != nil {
		return balancer.PickResult{}, err
	}

	return balancer.PickResult{SubConn: sc}, nil
}
