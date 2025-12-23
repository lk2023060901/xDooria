// pkg/raft/node.go
package raft

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
	"github.com/lk2023060901/xdooria/pkg/logger"
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

	// 存储
	logStore      raft.LogStore
	stableStore   raft.StableStore
	snapshotStore raft.SnapshotStore
	boltStore     *raftboltdb.BoltStore

	// 传输
	transport *raft.NetworkTransport

	// 状态
	mu         sync.RWMutex
	closed     bool
	shutdownCh chan struct{}

	// 选项
	logger logger.Logger

	// Leader 变更通知
	leaderCh <-chan bool
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

	// 创建传输层
	addr, err := net.ResolveTCPAddr("tcp", n.config.BindAddr)
	if err != nil {
		boltStore.Close()
		return fmt.Errorf("failed to resolve bind addr: %w", err)
	}

	transport, err := raft.NewTCPTransport(
		n.config.BindAddr,
		addr,
		n.config.MaxPool,
		10*time.Second,
		os.Stderr,
	)
	if err != nil {
		boltStore.Close()
		return fmt.Errorf("failed to create transport: %w", err)
	}
	n.transport = transport

	// 创建 Raft 实例
	raftConfig := n.config.ToRaftConfig(n.nodeID)
	fsmWrapper := newFSMWrapper(n.fsm)

	r, err := raft.NewRaft(raftConfig, fsmWrapper, n.logStore, n.stableStore, n.snapshotStore, n.transport)
	if err != nil {
		transport.Close()
		boltStore.Close()
		return fmt.Errorf("failed to create raft: %w", err)
	}
	n.raft = r
	n.leaderCh = r.LeaderCh()

	// Bootstrap 集群（如果是第一个节点）
	if n.config.Bootstrap {
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(n.nodeID),
					Address: raft.ServerAddress(n.config.BindAddr),
				},
			},
		}

		// 添加其他对等节点
		peers, err := ParsePeers(n.config.Peers)
		if err != nil {
			n.log("warn", "failed to parse peers", "error", err)
		} else {
			for _, p := range peers {
				configuration.Servers = append(configuration.Servers, raft.Server{
					ID:      raft.ServerID(p.ID),
					Address: raft.ServerAddress(p.Address),
				})
			}
		}

		future := n.raft.BootstrapCluster(configuration)
		if err := future.Error(); err != nil {
			if err != raft.ErrCantBootstrap {
				n.log("warn", "bootstrap failed", "error", err)
			}
		}
	}

	return nil
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

// AddVoter 添加投票节点
func (n *Node) AddVoter(id, address string, prevIndex uint64, timeout time.Duration) error {
	if n.State() != NodeStateLeader {
		return ErrNotLeader
	}

	future := n.raft.AddVoter(raft.ServerID(id), raft.ServerAddress(address), prevIndex, timeout)
	return future.Error()
}

// AddNonvoter 添加非投票节点（只读副本）
func (n *Node) AddNonvoter(id, address string, prevIndex uint64, timeout time.Duration) error {
	if n.State() != NodeStateLeader {
		return ErrNotLeader
	}

	future := n.raft.AddNonvoter(raft.ServerID(id), raft.ServerAddress(address), prevIndex, timeout)
	return future.Error()
}

// RemoveServer 移除节点
func (n *Node) RemoveServer(id string, prevIndex uint64, timeout time.Duration) error {
	if n.State() != NodeStateLeader {
		return ErrNotLeader
	}

	future := n.raft.RemoveServer(raft.ServerID(id), prevIndex, timeout)
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
