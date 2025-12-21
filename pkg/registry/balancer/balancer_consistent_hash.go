package balancer

import (
	"fmt"
	"hash/crc32"
	"sort"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"
)

const (
	// ConsistentHashBalancerName 一致性哈希负载均衡器名称
	ConsistentHashBalancerName = "consistent_hash"
	// HashKeyHeader 用于传递哈希 key 的 metadata header
	HashKeyHeader = "hash-key"
)

func init() {
	// 注册一致性哈希负载均衡器
	balancer.Register(&consistentHashBalancerBuilder{virtualNodes: 150})
}

// consistentHashBalancerBuilder 实现 balancer.Builder
type consistentHashBalancerBuilder struct {
	virtualNodes int
}

func (b *consistentHashBalancerBuilder) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	return &consistentHashBalancer{
		cc:           cc,
		subConns:     make(map[string]balancer.SubConn),
		scStates:     make(map[balancer.SubConn]connectivity.State),
		scAddrs:      make(map[balancer.SubConn]string),
		virtualNodes: b.virtualNodes,
	}
}

func (b *consistentHashBalancerBuilder) Name() string {
	return ConsistentHashBalancerName
}

// consistentHashBalancer 实现 balancer.Balancer 接口
// 确保只有当所有 SubConn 都 Ready 时才提供 Picker，避免哈希环不稳定
type consistentHashBalancer struct {
	cc             balancer.ClientConn
	mu             sync.RWMutex
	subConns       map[string]balancer.SubConn           // 地址字符串 -> SubConn
	scStates       map[balancer.SubConn]connectivity.State
	scAddrs        map[balancer.SubConn]string           // SubConn -> 地址字符串（用于反查）
	addrCount      int  // 总地址数
	virtualNodes   int
	firstPickerSet bool // 是否已经提供过第一个 Picker（用于区分初始化和运行阶段）
}

func (b *consistentHashBalancer) UpdateClientConnState(state balancer.ClientConnState) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 记录总地址数
	b.addrCount = len(state.ResolverState.Addresses)

	// 构建新的 SubConn 映射（使用地址字符串作为 key，避免 map 不可哈希问题）
	addrsSet := make(map[string]resolver.Address, len(state.ResolverState.Addresses))
	for _, addr := range state.ResolverState.Addresses {
		addrsSet[addr.Addr] = addr

		// 如果这个地址还没有 SubConn，创建一个
		if _, ok := b.subConns[addr.Addr]; !ok {
			sc, err := b.cc.NewSubConn([]resolver.Address{addr}, balancer.NewSubConnOptions{
				HealthCheckEnabled: true,
			})
			if err != nil {
				return fmt.Errorf("failed to create SubConn for %v: %w", addr, err)
			}
			b.subConns[addr.Addr] = sc
			b.scStates[sc] = connectivity.Idle
			b.scAddrs[sc] = addr.Addr

			// 触发连接
			sc.Connect()
		}
	}

	// 移除不在新地址列表中的 SubConn
	for addrStr, sc := range b.subConns {
		if _, ok := addrsSet[addrStr]; !ok {
			sc.Shutdown()
			delete(b.subConns, addrStr)
			delete(b.scStates, sc)
			delete(b.scAddrs, sc)
		}
	}

	// 更新 Picker
	b.regeneratePicker()
	return nil
}

func (b *consistentHashBalancer) ResolverError(err error) {
	// 记录错误，但不影响现有连接
}

func (b *consistentHashBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.SubConnState) {
	b.mu.Lock()
	defer b.mu.Unlock()

	oldState, ok := b.scStates[sc]
	if !ok {
		return
	}

	b.scStates[sc] = state.ConnectivityState

	// 状态变化时重新生成 Picker
	if oldState != state.ConnectivityState {
		b.regeneratePicker()
	}

	// 如果连接失败，触发重连
	if state.ConnectivityState == connectivity.Idle {
		sc.Connect()
	}
}

func (b *consistentHashBalancer) Close() {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, sc := range b.subConns {
		sc.Shutdown()
	}
	b.subConns = nil
	b.scStates = nil
}

func (b *consistentHashBalancer) ExitIdle() {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// 触发所有 SubConn 连接
	for _, sc := range b.subConns {
		sc.Connect()
	}
}

// regeneratePicker 重新生成 Picker
// 关键逻辑：
// 1. 初始化阶段（第一次提供 Picker 前）：必须等待所有 SubConn Ready，确保哈希环完整
// 2. 运行阶段（已提供过 Picker 后）：只要有至少 1 个 Ready，就继续提供服务（容忍节点故障/扩缩容）
func (b *consistentHashBalancer) regeneratePicker() {
	// 统计 Ready 的 SubConn
	readySCs := make(map[balancer.SubConn]string)
	for sc, state := range b.scStates {
		if state == connectivity.Ready {
			if addrStr, ok := b.scAddrs[sc]; ok {
				readySCs[sc] = addrStr
			}
		}
	}

	readyCount := len(readySCs)

	// 如果没有任何 Ready 的 SubConn，返回错误
	if readyCount == 0 {
		b.cc.UpdateState(balancer.State{
			ConnectivityState: connectivity.TransientFailure,
			Picker:            &errPicker{err: balancer.ErrNoSubConnAvailable},
		})
		return
	}

	// 初始化阶段：必须等待所有 SubConn Ready
	// 这确保第一次构建的哈希环是完整的，避免首次请求的路由不稳定
	if !b.firstPickerSet && readyCount < b.addrCount {
		b.cc.UpdateState(balancer.State{
			ConnectivityState: connectivity.Connecting,
			Picker:            &errPicker{err: balancer.ErrNoSubConnAvailable},
		})
		return
	}

	// 运行阶段：基于当前 Ready 的 SubConn 构建哈希环
	// 即使部分节点故障或正在扩缩容，也继续提供服务
	ring := &hashRing{
		nodes:        make(map[uint32]balancer.SubConn),
		sortedHashes: make([]uint32, 0),
	}

	for sc, addrStr := range readySCs {
		for i := 0; i < b.virtualNodes; i++ {
			virtualKey := fmt.Sprintf("%s#%d", addrStr, i)
			hash := crc32.ChecksumIEEE([]byte(virtualKey))
			ring.nodes[hash] = sc
			ring.sortedHashes = append(ring.sortedHashes, hash)
		}
	}

	sort.Slice(ring.sortedHashes, func(i, j int) bool {
		return ring.sortedHashes[i] < ring.sortedHashes[j]
	})

	b.cc.UpdateState(balancer.State{
		ConnectivityState: connectivity.Ready,
		Picker:            &consistentHashPicker{ring: ring},
	})

	// 标记已经提供过第一个 Picker
	b.firstPickerSet = true
}

// errPicker 用于返回错误的 Picker
type errPicker struct {
	err error
}

func (p *errPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	return balancer.PickResult{}, p.err
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
