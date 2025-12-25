// pkg/raft/node.go
package raft

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-raftchunking"
	"github.com/hashicorp/raft"
	autopilotlib "github.com/hashicorp/raft-autopilot"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/raft/consul"
	"github.com/lk2023060901/xdooria/pkg/raft/consul/pool"
	"github.com/lk2023060901/xdooria/pkg/raft/consul/tlsutil"
)

// NodeState 节点状态
type NodeState uint32

const (
	// NodeStateFollower 跟随者
	NodeStateFollower NodeState = iota
	// NodeStateCandidate 候选者
	NodeStateCandidate
	// NodeStateLeader 领导者
	NodeStateLeader
	// NodeStateShutdown 已关闭
	NodeStateShutdown
)

// String 返回状态字符串
func (s NodeState) String() string {
	switch s {
	case NodeStateFollower:
		return "follower"
	case NodeStateCandidate:
		return "candidate"
	case NodeStateLeader:
		return "leader"
	case NodeStateShutdown:
		return "shutdown"
	default:
		return "unknown"
	}
}

// Node Raft 节点封装
type Node struct {
	config *Config
	nodeID string // 自动生成的节点 ID
	raft   *raft.Raft
	fsm    FSM

	// ChunkingFSM 包装器（用于处理大数据分片和快照状态）
	chunker chunkingFSM

	// 存储
	logStore      raft.LogStore
	stableStore   raft.StableStore
	snapshotStore raft.SnapshotStore
	boltStore     *raftboltdb.BoltStore

	// 传输
	transport *raft.NetworkTransport

	// RaftLayer (用于端口复用)
	raftLayer *consul.RaftLayer

	// 主监听器 (Raft 和 RPC 共用)
	listener net.Listener

	// Serf/Gossip 集群发现
	serfLAN *consul.SerfLAN

	// Autopilot 自动集群管理
	autopilot *consul.Autopilot

	// RPC 服务器
	rpcServer *consul.RPCServer

	// 连接池
	connPool *pool.ConnPool

	// TLS 配置器
	tlsConfigurator *tlsutil.Configurator

	// 状态
	mu         sync.RWMutex
	closed     bool
	shutdownCh chan struct{}

	// 选项
	logger   logger.Logger
	hcLogger hclog.Logger // hashicorp 风格的 logger

	// Leader 变更通知
	leaderCh <-chan bool

	// Autopilot 上下文
	autopilotCtx    context.Context
	autopilotCancel context.CancelFunc
}

// NewNode 创建 Raft 节点
func NewNode(cfg *Config, fsm FSM, opts ...Option) (*Node, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	if fsm == nil {
		return nil, fmt.Errorf("%w: fsm is required", ErrInvalidConfig)
	}

	// 自动生成节点 ID（从文件加载或新生成）
	nodeID, err := LoadOrGenerateNodeID(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load or generate node ID: %w", err)
	}

	node := &Node{
		config:     cfg,
		nodeID:     nodeID,
		fsm:        fsm,
		shutdownCh: make(chan struct{}),
	}

	// 应用选项
	for _, opt := range opts {
		opt(node)
	}

	// 初始化存储和传输
	if err := node.setup(); err != nil {
		return nil, err
	}

	return node, nil
}

// setup 初始化 Raft 节点
func (n *Node) setup() error {
	// 确保数据目录存在
	if err := os.MkdirAll(n.config.DataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data dir: %w", err)
	}

	// 创建 hclog logger（使用适配器将我们的 logger 转为 hclog.Logger）
	if n.logger != nil {
		n.hcLogger = NewHclogAdapter(n.logger, "raft", n.config.LogLevel)
	} else {
		// 如果没有提供 logger，使用默认的 hclog
		n.hcLogger = hclog.New(&hclog.LoggerOptions{
			Name:   "raft",
			Level:  hclog.LevelFromString(n.config.LogLevel),
			Output: os.Stderr,
		})
	}

	// 创建 BoltDB 存储（同时用于 LogStore 和 StableStore）
	boltPath := filepath.Join(n.config.DataDir, "raft.db")
	boltStore, err := raftboltdb.NewBoltStore(boltPath)
	if err != nil {
		return fmt.Errorf("failed to create bolt store: %w", err)
	}
	n.boltStore = boltStore
	n.logStore = boltStore
	n.stableStore = boltStore

	// 创建快照存储
	snapshotPath := filepath.Join(n.config.DataDir, "snapshots")
	snapshotStore, err := raft.NewFileSnapshotStore(snapshotPath, n.config.SnapshotRetain, os.Stderr)
	if err != nil {
		boltStore.Close()
		return fmt.Errorf("failed to create snapshot store: %w", err)
	}
	n.snapshotStore = snapshotStore

	// 创建主监听器 (Raft 和 RPC 共用)
	listener, err := net.Listen("tcp", n.config.BindAddr)
	if err != nil {
		boltStore.Close()
		return fmt.Errorf("failed to create listener: %w", err)
	}
	n.listener = listener

	// 解析监听地址
	addr, err := net.ResolveTCPAddr("tcp", n.config.BindAddr)
	if err != nil {
		listener.Close()
		boltStore.Close()
		return fmt.Errorf("failed to resolve bind addr: %w", err)
	}

	// 创建 RaftLayer (用于端口复用)
	n.raftLayer = consul.NewRaftLayer(addr, listener.Addr(), nil, nil)

	// 创建传输层 (使用 RaftLayer)
	transConfig := &raft.NetworkTransportConfig{
		Stream:  n.raftLayer,
		MaxPool: n.config.MaxPool,
		Timeout: 10 * time.Second,
		Logger:  n.hcLogger,
	}
	n.transport = raft.NewNetworkTransportWithConfig(transConfig)

	// 创建 Raft 实例
	raftConfig := n.config.ToRaftConfig(n.nodeID)

	// 用 ChunkingFSM 包装用户 FSM，支持大数据分片
	wrapper := newFSMWrapper(n.fsm)
	n.chunker = newChunkingFSMWrapper(wrapper)

	r, err := raft.NewRaft(raftConfig, n.chunker, n.logStore, n.stableStore, n.snapshotStore, n.transport)
	if err != nil {
		n.transport.Close()
		n.raftLayer.Close()
		n.listener.Close()
		n.boltStore.Close()
		return fmt.Errorf("failed to create raft: %w", err)
	}
	n.raft = r
	n.leaderCh = r.LeaderCh()

	// 初始化 TLS 配置器
	if err := n.setupTLS(); err != nil {
		n.raft.Shutdown()
		n.transport.Close()
		n.raftLayer.Close()
		n.listener.Close()
		n.boltStore.Close()
		return fmt.Errorf("failed to setup TLS: %w", err)
	}

	// 初始化连接池
	if err := n.setupConnPool(); err != nil {
		n.raft.Shutdown()
		n.transport.Close()
		n.raftLayer.Close()
		n.listener.Close()
		n.boltStore.Close()
		return fmt.Errorf("failed to setup connection pool: %w", err)
	}

	// 初始化 RPC 服务器 (使用主监听器)
	if err := n.setupRPCServer(); err != nil {
		n.connPool.Shutdown()
		n.raft.Shutdown()
		n.transport.Close()
		n.raftLayer.Close()
		n.listener.Close()
		n.boltStore.Close()
		return fmt.Errorf("failed to setup RPC server: %w", err)
	}

	// 启动连接处理 (路由 Raft/RPC 连接)
	go n.listen()

	// 如果 ExpectNodes > 0，启用 Serf 和 Autopilot
	// ExpectNodes = 0 表示跳过集群发现（单节点测试模式）
	if n.config.ExpectNodes > 0 {
		// 初始化 Serf LAN（用于节点发现和自动 Bootstrap）
		if err := n.setupSerfLAN(); err != nil {
			n.rpcServer.Shutdown()
			n.connPool.Shutdown()
			n.raft.Shutdown()
			n.transport.Close()
			n.raftLayer.Close()
			n.listener.Close()
			n.boltStore.Close()
			return fmt.Errorf("failed to setup serf LAN: %w", err)
		}

		// 初始化 Autopilot（用于自动集群管理）
		if err := n.setupAutopilot(); err != nil {
			n.serfLAN.Shutdown()
			n.rpcServer.Shutdown()
			n.connPool.Shutdown()
			n.raft.Shutdown()
			n.transport.Close()
			n.raftLayer.Close()
			n.listener.Close()
			n.boltStore.Close()
			return fmt.Errorf("failed to setup autopilot: %w", err)
		}
	}

	return nil
}

// setupTLS 初始化 TLS 配置器
func (n *Node) setupTLS() error {
	tlsConfig := tlsutil.Config{
		VerifyIncoming:       n.config.TLSVerify,
		VerifyOutgoing:       n.config.TLSVerify,
		VerifyServerHostname: n.config.TLSVerify,
		CAFile:               n.config.TLSCAFile,
		CertFile:             n.config.TLSCertFile,
		KeyFile:              n.config.TLSKeyFile,
		NodeName:             n.getNodeName(),
		Datacenter:           n.config.Datacenter,
		InternalRPC: tlsutil.ProtocolConfig{
			VerifyIncoming: n.config.TLSVerify,
			VerifyOutgoing: n.config.TLSEnabled,
			CAFile:         n.config.TLSCAFile,
			CertFile:       n.config.TLSCertFile,
			KeyFile:        n.config.TLSKeyFile,
		},
	}

	configurator, err := tlsutil.NewConfigurator(tlsConfig)
	if err != nil {
		return err
	}
	n.tlsConfigurator = configurator
	return nil
}

// setupConnPool 初始化连接池
func (n *Node) setupConnPool() error {
	n.connPool = &pool.ConnPool{
		Datacenter:      n.config.Datacenter,
		MaxStreams:      n.config.MaxPool,
		MaxTime:         10 * time.Minute,
		TLSConfigurator: n.tlsConfigurator,
		Server:          true,
	}
	return nil
}

// setupRPCServer 初始化 RPC 服务器 (不创建监听器，连接由主监听器路由)
func (n *Node) setupRPCServer() error {
	rpcServer, err := consul.NewRPCServer(&consul.RPCServerConfig{
		Raft:           n.raft,
		Logger:         n.hcLogger,
		MaxConnections: 256,
	})
	if err != nil {
		return err
	}
	n.rpcServer = rpcServer
	return nil
}

// listen 处理主监听器上的连接，根据首字节路由到 Raft 或 RPC
func (n *Node) listen() {
	for {
		conn, err := n.listener.Accept()
		if err != nil {
			n.mu.RLock()
			closed := n.closed
			n.mu.RUnlock()
			if closed {
				return
			}
			n.hcLogger.Error("failed to accept conn", "error", err)
			continue
		}

		go n.handleConn(conn)
	}
}

// handleConn 根据首字节确定连接类型并路由
func (n *Node) handleConn(conn net.Conn) {
	// 检查是否已关闭
	n.mu.RLock()
	closed := n.closed
	n.mu.RUnlock()
	if closed {
		conn.Close()
		return
	}

	// 设置读取超时，防止恶意客户端占用连接
	if err := conn.SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		n.hcLogger.Error("failed to set read deadline", "error", err)
		conn.Close()
		return
	}

	// 读取首字节确定连接类型
	buf := make([]byte, 1)
	if _, err := conn.Read(buf); err != nil {
		if err != io.EOF && !strings.Contains(err.Error(), "closed") {
			n.hcLogger.Error("failed to read byte", "error", err)
		}
		conn.Close()
		return
	}

	// 重置 deadline，后续由各处理器自行管理
	if err := conn.SetReadDeadline(time.Time{}); err != nil {
		n.hcLogger.Error("failed to reset read deadline", "error", err)
		conn.Close()
		return
	}

	typ := pool.RPCType(buf[0])

	switch typ {
	case pool.RPCRaft:
		// Raft 连接，交给 RaftLayer 处理
		n.raftLayer.Handoff(conn)

	case pool.RPCConsul:
		// RPC 连接
		n.rpcServer.HandleConn(conn)

	case pool.RPCMultiplexV2:
		// 多路复用 RPC 连接
		n.rpcServer.HandleMultiplexV2(conn)

	case pool.RPCTLS:
		// TLS 连接，当前简单处理为普通 RPC
		n.rpcServer.HandleConn(conn)

	case pool.RPCTLSInsecure:
		n.rpcServer.HandleConn(conn)

	default:
		n.hcLogger.Error("unrecognized RPC byte", "byte", typ)
		conn.Close()
	}
}

// setupSerfLAN 初始化 Serf LAN 集群
func (n *Node) setupSerfLAN() error {
	// 解析 Serf 绑定地址
	serfBindAddr, serfBindPort, err := n.getSerfBindAddr()
	if err != nil {
		return err
	}

	// 解析 Raft 地址
	raftHost, raftPortStr, err := net.SplitHostPort(n.config.BindAddr)
	if err != nil {
		return fmt.Errorf("invalid raft bind addr: %w", err)
	}
	raftPort, _ := strconv.Atoi(raftPortStr)

	// 创建 SerfLAN 配置
	serfConfig := &consul.SerfLANConfig{
		NodeName:         n.getNodeName(),
		NodeID:           n.nodeID, // Must match Raft server ID for bootstrap to work
		Datacenter:       n.config.Datacenter,
		BindAddr:         serfBindAddr,
		BindPort:         serfBindPort,
		RaftAddr:         n.config.BindAddr,
		RaftPort:         raftPort,
		Bootstrap:        false, // 使用 BootstrapExpect 自动 bootstrap
		BootstrapExpect:  n.config.ExpectNodes,
		ReadReplica:      false,
		UseTLS:           n.config.TLSEnabled,
		Build:            "1.0.0",
		ProtocolVersion:  3,
		RaftVersion:      3,
		DataDir:          n.config.DataDir,
		RejoinAfterLeave: true,
		Logger:           n.hcLogger,
	}

	// 如果 Raft 地址的 host 不是绑定地址，使用它作为广播地址
	if raftHost != "" && raftHost != "0.0.0.0" {
		serfConfig.BindAddr = raftHost
	}

	// 创建 StatsFetcher
	statsFetcher := consul.NewStatsFetcher(n.hcLogger.Named("stats_fetcher"), n.connPool, n.config.Datacenter)

	// 创建 SerfLAN
	serfLAN, err := consul.NewSerfLAN(serfConfig, n.raft, n.logStore, n.connPool, n.IsLeader)
	if err != nil {
		return err
	}
	n.serfLAN = serfLAN

	// 加入种子节点
	if len(n.config.JoinAddrs) > 0 {
		go func() {
			// 等待一小段时间让本地 Serf 初始化完成
			time.Sleep(100 * time.Millisecond)
			joined, err := n.serfLAN.Join(n.config.JoinAddrs)
			if err != nil {
				n.log("warn", "failed to join some nodes", "joined", joined, "error", err)
			} else {
				n.log("info", "joined cluster", "joined", joined)
			}
		}()
	}

	// 保存 statsFetcher 供 autopilot 使用（通过闭包）
	_ = statsFetcher

	return nil
}

// setupAutopilot 初始化 Autopilot
func (n *Node) setupAutopilot() error {
	// 创建 StatsFetcher
	statsFetcher := consul.NewStatsFetcher(n.hcLogger.Named("stats_fetcher"), n.connPool, n.config.Datacenter)

	// 创建 Autopilot 配置
	autopilotConfig := consul.DefaultAutopilotConfig()

	// 创建 Autopilot
	ap, err := consul.NewAutopilot(consul.AutopilotOptions{
		Config:       autopilotConfig,
		Raft:         n.raft,
		SerfLAN:      n.serfLAN,
		StatsFetcher: statsFetcher,
		Logger:       n.hcLogger,
		RemoveFailedServerFunc: func(serverID string, serverName string) error {
			n.log("info", "removing failed server", "id", serverID, "name", serverName)
			future := n.raft.RemoveServer(raft.ServerID(serverID), 0, 0)
			return future.Error()
		},
		StateNotifyFunc: func(state *autopilotlib.State) {
			if state.Healthy {
				n.log("debug", "cluster healthy", "failure_tolerance", state.FailureTolerance)
			} else {
				n.log("warn", "cluster unhealthy", "failure_tolerance", state.FailureTolerance)
			}
		},
	})
	if err != nil {
		return err
	}
	n.autopilot = ap

	// 创建 Autopilot 上下文
	n.autopilotCtx, n.autopilotCancel = context.WithCancel(context.Background())

	// 启动 Autopilot
	n.autopilot.Start(n.autopilotCtx)

	// 当成为 Leader 时启用 Autopilot 的 Raft 调和
	go n.watchLeadershipForAutopilot()

	return nil
}

// watchLeadershipForAutopilot 监听 leadership 变化，控制 Autopilot 调和
func (n *Node) watchLeadershipForAutopilot() {
	for {
		select {
		case <-n.shutdownCh:
			return
		case isLeader := <-n.leaderCh:
			if isLeader {
				n.log("info", "became leader, enabling autopilot reconciliation")
				n.autopilot.EnableReconciliation()
			} else {
				n.log("info", "lost leadership, disabling autopilot reconciliation")
				n.autopilot.DisableReconciliation()
			}
		}
	}
}

// getSerfBindAddr 获取 Serf 绑定地址
func (n *Node) getSerfBindAddr() (string, int, error) {
	if n.config.SerfLANAddr != "" {
		host, portStr, err := net.SplitHostPort(n.config.SerfLANAddr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid serf_lan_addr: %w", err)
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return "", 0, fmt.Errorf("invalid serf port: %w", err)
		}
		return host, port, nil
	}

	// 默认使用 Raft 地址的 host + 8301 端口（Consul 默认 Serf LAN 端口）
	host, _, err := net.SplitHostPort(n.config.BindAddr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid bind_addr: %w", err)
	}
	return host, 8301, nil
}

// getNodeName 获取节点名称
func (n *Node) getNodeName() string {
	if n.config.NodeName != "" {
		return n.config.NodeName
	}
	// 使用节点 ID 作为名称
	return n.nodeID
}

// Start 启动节点（节点在 NewNode 时已经启动，此方法主要用于等待就绪）
func (n *Node) Start(ctx context.Context) error {
	n.mu.RLock()
	if n.closed {
		n.mu.RUnlock()
		return ErrNodeClosed
	}
	n.mu.RUnlock()

	// 等待节点就绪
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if n.raft.Leader() != "" || n.State() == NodeStateLeader {
				n.log("info", "node started", "id", n.nodeID, "state", n.State())
				return nil
			}
		}
	}
}

// Close 关闭节点
func (n *Node) Close() error {
	n.mu.Lock()
	if n.closed {
		n.mu.Unlock()
		return nil
	}
	n.closed = true
	close(n.shutdownCh)
	n.mu.Unlock()

	// 停止 Autopilot
	if n.autopilotCancel != nil {
		n.autopilotCancel()
	}

	// 优雅离开 Serf 集群
	if n.serfLAN != nil {
		if err := n.serfLAN.Leave(); err != nil {
			n.log("error", "serf leave failed", "error", err)
		}
		if err := n.serfLAN.Shutdown(); err != nil {
			n.log("error", "shutdown serf failed", "error", err)
		}
	}

	// 关闭 RPC 服务器
	if n.rpcServer != nil {
		if err := n.rpcServer.Shutdown(); err != nil {
			n.log("error", "shutdown rpc server failed", "error", err)
		}
	}

	// 关闭连接池
	if n.connPool != nil {
		n.connPool.Shutdown()
	}

	// 关闭 Raft
	if n.raft != nil {
		future := n.raft.Shutdown()
		if err := future.Error(); err != nil {
			n.log("error", "shutdown raft failed", "error", err)
		}
	}

	// 关闭传输
	if n.transport != nil {
		if err := n.transport.Close(); err != nil {
			n.log("error", "close transport failed", "error", err)
		}
	}

	// 关闭 RaftLayer
	if n.raftLayer != nil {
		if err := n.raftLayer.Close(); err != nil {
			n.log("error", "close raft layer failed", "error", err)
		}
	}

	// 关闭主监听器
	if n.listener != nil {
		if err := n.listener.Close(); err != nil {
			n.log("error", "close listener failed", "error", err)
		}
	}

	// 关闭存储
	if n.boltStore != nil {
		if err := n.boltStore.Close(); err != nil {
			n.log("error", "close bolt store failed", "error", err)
		}
	}

	return nil
}

// Apply 应用命令到状态机
func (n *Node) Apply(data []byte, timeout time.Duration) (interface{}, error) {
	n.mu.RLock()
	if n.closed {
		n.mu.RUnlock()
		return nil, ErrNodeClosed
	}
	n.mu.RUnlock()

	if n.State() != NodeStateLeader {
		return nil, ErrNotLeader
	}

	future := n.raft.Apply(data, timeout)
	if err := future.Error(); err != nil {
		if err == raft.ErrLeadershipLost {
			return nil, ErrLeadershipLost
		}
		return nil, err
	}

	return future.Response(), nil
}

// ApplyCommand 应用命令（封装版）
func (n *Node) ApplyCommand(cmd *Command, timeout time.Duration) (interface{}, error) {
	data, err := EncodeCommand(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to encode command: %w", err)
	}
	return n.Apply(data, timeout)
}

// ApplyLarge 应用大数据命令（自动分片）
// 当数据超过 Raft 建议的最大数据大小时，会自动分片处理
// timeout 是每个分片的超时时间，不是总超时时间
func (n *Node) ApplyLarge(data []byte, timeout time.Duration) (interface{}, error) {
	n.mu.RLock()
	if n.closed {
		n.mu.RUnlock()
		return nil, ErrNodeClosed
	}
	n.mu.RUnlock()

	if n.State() != NodeStateLeader {
		return nil, ErrNotLeader
	}

	// 使用 ChunkingApply 处理大数据分片
	applyFunc := func(log raft.Log, t time.Duration) raft.ApplyFuture {
		return n.raft.Apply(log.Data, t)
	}

	future := raftchunking.ChunkingApply(data, nil, timeout, applyFunc)
	if err := future.Error(); err != nil {
		if err == raft.ErrLeadershipLost {
			return nil, ErrLeadershipLost
		}
		return nil, err
	}

	// 处理 ChunkingSuccess 包装
	resp := future.Response()
	if cs, ok := resp.(raftchunking.ChunkingSuccess); ok {
		return cs.Response, nil
	}
	return resp, nil
}

// State 返回当前节点状态
func (n *Node) State() NodeState {
	if n.raft == nil {
		return NodeStateShutdown
	}

	switch n.raft.State() {
	case raft.Follower:
		return NodeStateFollower
	case raft.Candidate:
		return NodeStateCandidate
	case raft.Leader:
		return NodeStateLeader
	case raft.Shutdown:
		return NodeStateShutdown
	default:
		return NodeStateFollower
	}
}

// IsLeader 是否是 Leader
func (n *Node) IsLeader() bool {
	return n.State() == NodeStateLeader
}

// Leader 返回当前 Leader 地址
func (n *Node) Leader() string {
	addr, _ := n.raft.LeaderWithID()
	return string(addr)
}

// LeaderID 返回当前 Leader ID
func (n *Node) LeaderID() string {
	_, id := n.raft.LeaderWithID()
	return string(id)
}

// LeaderCh 返回 Leader 变更通知通道
func (n *Node) LeaderCh() <-chan bool {
	return n.leaderCh
}

// NodeID 返回节点 ID
func (n *Node) NodeID() string {
	return n.nodeID
}

// Stats 返回 Raft 统计信息
func (n *Node) Stats() map[string]string {
	if n.raft == nil {
		return nil
	}
	return n.raft.Stats()
}

// Snapshot 手动触发快照
func (n *Node) Snapshot() error {
	future := n.raft.Snapshot()
	return future.Error()
}


// GetConfiguration 获取集群配置
func (n *Node) GetConfiguration() ([]Server, error) {
	future := n.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return nil, err
	}

	config := future.Configuration()
	servers := make([]Server, 0, len(config.Servers))
	for _, s := range config.Servers {
		servers = append(servers, Server{
			ID:       string(s.ID),
			Address:  string(s.Address),
			Suffrage: s.Suffrage.String(),
		})
	}

	return servers, nil
}

// Server 服务器信息
type Server struct {
	ID       string
	Address  string
	Suffrage string // Voter, Nonvoter, Staging
}

// LastIndex 返回最后的日志索引
func (n *Node) LastIndex() uint64 {
	return n.raft.LastIndex()
}

// AppliedIndex 返回已应用的日志索引
func (n *Node) AppliedIndex() uint64 {
	return n.raft.AppliedIndex()
}

// Barrier 等待所有已提交的日志被应用
func (n *Node) Barrier(timeout time.Duration) error {
	future := n.raft.Barrier(timeout)
	return future.Error()
}

// VerifyLeader 验证当前节点是否仍是 Leader
func (n *Node) VerifyLeader() error {
	future := n.raft.VerifyLeader()
	return future.Error()
}

// log 记录日志
func (n *Node) log(level string, msg string, keysAndValues ...interface{}) {
	if n.logger == nil {
		return
	}

	switch level {
	case "debug":
		n.logger.Debug(msg, keysAndValues...)
	case "info":
		n.logger.Info(msg, keysAndValues...)
	case "warn":
		n.logger.Warn(msg, keysAndValues...)
	case "error":
		n.logger.Error(msg, keysAndValues...)
	}
}

// LeadershipTransfer 主动转移 Leadership
func (n *Node) LeadershipTransfer() error {
	if n.State() != NodeStateLeader {
		return ErrNotLeader
	}

	future := n.raft.LeadershipTransfer()
	return future.Error()
}

// Bootstrap 手动引导单节点集群
// 当 ExpectNodes=0 时（跳过 Serf），需要调用此方法来引导集群
func (n *Node) Bootstrap() error {
	configuration := raft.Configuration{
		Servers: []raft.Server{
			{
				ID:       raft.ServerID(n.nodeID),
				Address:  raft.ServerAddress(n.config.BindAddr),
				Suffrage: raft.Voter,
			},
		},
	}
	future := n.raft.BootstrapCluster(configuration)
	return future.Error()
}

// Join 加入集群（通过 Serf gossip）
func (n *Node) Join(addrs []string) (int, error) {
	if n.serfLAN == nil {
		return 0, fmt.Errorf("serf not initialized")
	}
	return n.serfLAN.Join(addrs)
}

// Members 返回集群成员列表
func (n *Node) Members() []ClusterMember {
	if n.serfLAN == nil {
		return nil
	}

	serfMembers := n.serfLAN.Members()
	members := make([]ClusterMember, 0, len(serfMembers))
	for _, m := range serfMembers {
		members = append(members, ClusterMember{
			Name:   m.Name,
			Addr:   m.Addr.String(),
			Port:   int(m.Port),
			Status: m.Status.String(),
			Tags:   m.Tags,
		})
	}
	return members
}

// ClusterMember 集群成员信息
type ClusterMember struct {
	Name   string
	Addr   string
	Port   int
	Status string
	Tags   map[string]string
}

// NumNodes 返回集群节点数量
func (n *Node) NumNodes() int {
	if n.serfLAN == nil {
		return 0
	}
	return n.serfLAN.NumNodes()
}

// IsClusterHealthy 检查集群是否健康
func (n *Node) IsClusterHealthy() bool {
	if n.autopilot == nil {
		return false
	}
	return n.autopilot.IsHealthy()
}

// ClusterFailureTolerance 返回集群容错能力（可以失去多少个节点仍保持正常）
func (n *Node) ClusterFailureTolerance() int {
	if n.autopilot == nil {
		return 0
	}
	return n.autopilot.FailureTolerance()
}

// AddVoter 添加一个投票节点到集群
// serverID: 节点 ID
// address: 节点地址 (host:port)
// prevIndex: 用于检测过时请求，传 0 表示不检查
// timeout: 操作超时时间
func (n *Node) AddVoter(serverID string, address string, prevIndex uint64, timeout time.Duration) error {
	if n.raft == nil {
		return ErrNotStarted
	}
	future := n.raft.AddVoter(raft.ServerID(serverID), raft.ServerAddress(address), prevIndex, timeout)
	return future.Error()
}

// AddNonvoter 添加一个非投票节点（只读副本）到集群
// serverID: 节点 ID
// address: 节点地址 (host:port)
// prevIndex: 用于检测过时请求，传 0 表示不检查
// timeout: 操作超时时间
func (n *Node) AddNonvoter(serverID string, address string, prevIndex uint64, timeout time.Duration) error {
	if n.raft == nil {
		return ErrNotStarted
	}
	future := n.raft.AddNonvoter(raft.ServerID(serverID), raft.ServerAddress(address), prevIndex, timeout)
	return future.Error()
}

// RemoveServer 从集群中移除一个节点
// serverID: 要移除的节点 ID
// prevIndex: 用于检测过时请求，传 0 表示不检查
// timeout: 操作超时时间
func (n *Node) RemoveServer(serverID string, prevIndex uint64, timeout time.Duration) error {
	if n.raft == nil {
		return ErrNotStarted
	}
	future := n.raft.RemoveServer(raft.ServerID(serverID), prevIndex, timeout)
	return future.Error()
}

// DemoteVoter 将投票节点降级为非投票节点
// serverID: 节点 ID
// prevIndex: 用于检测过时请求，传 0 表示不检查
// timeout: 操作超时时间
func (n *Node) DemoteVoter(serverID string, prevIndex uint64, timeout time.Duration) error {
	if n.raft == nil {
		return ErrNotStarted
	}
	future := n.raft.DemoteVoter(raft.ServerID(serverID), prevIndex, timeout)
	return future.Error()
}
