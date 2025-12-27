package balancer

import (
	"fmt"
	"strconv"
	"sync"

	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/resolver"

	genericbalancer "github.com/lk2023060901/xdooria/pkg/balancer"
)

const (
	// ConsistentHashBalancerName 一致性哈希负载均衡器名称
	ConsistentHashBalancerName = "consistent_hash"
	// HashKeyHeader 用于传递哈希 key 的 metadata header
	HashKeyHeader = "hash-key"
)

func init() {
	// 注册一致性哈希负载均衡器
	balancer.Register(&consistentHashBalancerBuilder{})
}

// consistentHashBalancerBuilder 实现 balancer.Builder
type consistentHashBalancerBuilder struct{}

func (b *consistentHashBalancerBuilder) Build(cc balancer.ClientConn, opts balancer.BuildOptions) balancer.Balancer {
	return &consistentHashBalancer{
		cc:       cc,
		subConns: make(map[string]balancer.SubConn),
		scStates: make(map[balancer.SubConn]connectivity.State),
		scAddrs:  make(map[balancer.SubConn]string),
	}
}

func (b *consistentHashBalancerBuilder) Name() string {
	return ConsistentHashBalancerName
}

// consistentHashBalancer 实现 balancer.Balancer 接口
type consistentHashBalancer struct {
	cc             balancer.ClientConn
	mu             sync.RWMutex
	subConns       map[string]balancer.SubConn
	scStates       map[balancer.SubConn]connectivity.State
	scAddrs        map[balancer.SubConn]string
	addrCount      int
	firstPickerSet bool
}

func (b *consistentHashBalancer) UpdateClientConnState(state balancer.ClientConnState) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.addrCount = len(state.ResolverState.Addresses)

	addrsSet := make(map[string]resolver.Address, len(state.ResolverState.Addresses))
	for _, addr := range state.ResolverState.Addresses {
		addrsSet[addr.Addr] = addr

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
			sc.Connect()
		}
	}

	for addrStr, sc := range b.subConns {
		if _, ok := addrsSet[addrStr]; !ok {
			sc.Shutdown()
			delete(b.subConns, addrStr)
			delete(b.scStates, sc)
			delete(b.scAddrs, sc)
		}
	}

	b.regeneratePicker()
	return nil
}

func (b *consistentHashBalancer) ResolverError(err error) {}

func (b *consistentHashBalancer) UpdateSubConnState(sc balancer.SubConn, state balancer.SubConnState) {
	b.mu.Lock()
	defer b.mu.Unlock()

	oldState, ok := b.scStates[sc]
	if !ok {
		return
	}

	b.scStates[sc] = state.ConnectivityState

	if oldState != state.ConnectivityState {
		b.regeneratePicker()
	}

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

	for _, sc := range b.subConns {
		sc.Connect()
	}
}

func (b *consistentHashBalancer) regeneratePicker() {
	// 收集 Ready 的 SubConn
	readySCs := make([]balancer.SubConn, 0)
	readyAddrs := make([]string, 0)
	for sc, state := range b.scStates {
		if state == connectivity.Ready {
			if addrStr, ok := b.scAddrs[sc]; ok {
				readySCs = append(readySCs, sc)
				readyAddrs = append(readyAddrs, addrStr)
			}
		}
	}

	readyCount := len(readySCs)

	if readyCount == 0 {
		b.cc.UpdateState(balancer.State{
			ConnectivityState: connectivity.TransientFailure,
			Picker:            &errPicker{err: balancer.ErrNoSubConnAvailable},
		})
		return
	}

	if !b.firstPickerSet && readyCount < b.addrCount {
		b.cc.UpdateState(balancer.State{
			ConnectivityState: connectivity.Connecting,
			Picker:            &errPicker{err: balancer.ErrNoSubConnAvailable},
		})
		return
	}

	// 构建通用节点列表
	nodes := make([]*genericbalancer.Node, len(readySCs))
	for i, addr := range readyAddrs {
		nodes[i] = &genericbalancer.Node{
			Address: strconv.Itoa(i),
			Metadata: map[string]string{
				"real_addr": addr,
			},
		}
	}

	b.cc.UpdateState(balancer.State{
		ConnectivityState: connectivity.Ready,
		Picker: &consistentHashPicker{
			subConns: readySCs,
			nodes:    nodes,
			balancer: genericbalancer.New(genericbalancer.ConsistentHashName),
		},
	})

	b.firstPickerSet = true
}

// errPicker 用于返回错误的 Picker
type errPicker struct {
	err error
}

func (p *errPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	return balancer.PickResult{}, p.err
}

// consistentHashPicker 实现一致性哈希选择器
type consistentHashPicker struct {
	subConns []balancer.SubConn
	nodes    []*genericbalancer.Node
	balancer genericbalancer.Balancer
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

	if hashKey == "" {
		return balancer.PickResult{}, fmt.Errorf("consistent hash requires %s in metadata", HashKeyHeader)
	}

	node := p.balancer.Pick(p.nodes, genericbalancer.PickInfo{Key: hashKey})
	if node == nil {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	idx, _ := strconv.Atoi(node.Address)
	return balancer.PickResult{SubConn: p.subConns[idx]}, nil
}
