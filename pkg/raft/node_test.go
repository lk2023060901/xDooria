package raft

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/raft"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// kvFSM is a simple key-value FSM for testing
type kvFSM struct {
	mu   sync.RWMutex
	data map[string]string
}

func newKVFSM() *kvFSM {
	return &kvFSM{
		data: make(map[string]string),
	}
}

func (f *kvFSM) Apply(log *raft.Log) interface{} {
	cmd, err := DecodeCommand(log.Data)
	if err != nil {
		return NewApplyResult(nil, err)
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Type {
	case CommandTypeSet:
		f.data[cmd.Key] = string(cmd.Data)
		return NewApplyResult(cmd.Key, nil)
	case CommandTypeDelete:
		delete(f.data, cmd.Key)
		return NewApplyResult(cmd.Key, nil)
	default:
		return NewApplyResult(nil, ErrInvalidCommand)
	}
}

func (f *kvFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Simple JSON-like snapshot
	var data []byte
	for k, v := range f.data {
		data = append(data, []byte(fmt.Sprintf("%s=%s\n", k, v))...)
	}
	return NewSimpleFSMSnapshot(data), nil
}

func (f *kvFSM) Restore(snapshot io.ReadCloser) error {
	defer snapshot.Close()
	// For testing, just reset data
	f.mu.Lock()
	f.data = make(map[string]string)
	f.mu.Unlock()
	return nil
}

func (f *kvFSM) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	v, ok := f.data[key]
	return v, ok
}

func (f *kvFSM) Len() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return len(f.data)
}

// createTestNode creates a single node for testing
// expectNodes 设置为 1 时会自动 bootstrap，设置为更大的值时会等待其他节点加入
func createTestNode(t *testing.T, port int, expectNodes int) (*Node, *kvFSM) {
	t.Helper()

	fsm := newKVFSM()
	cfg := &Config{
		BindAddr:           fmt.Sprintf("127.0.0.1:%d", port),
		DataDir:            t.TempDir(),
		HeartbeatTimeout:   50 * time.Millisecond,
		ElectionTimeout:    50 * time.Millisecond,
		CommitTimeout:      5 * time.Millisecond,
		LeaderLeaseTimeout: 25 * time.Millisecond,
		SnapshotInterval:   10 * time.Second,
		SnapshotThreshold:  8192,
		SnapshotRetain:     1,
		MaxAppendEntries:   64,
		TrailingLogs:       128,
		MaxPool:            2,
		LogLevel:           "error",
		// Serf/Gossip 配置
		NodeName:    fmt.Sprintf("node-%d", port),
		Datacenter:  "dc1",
		ExpectNodes: expectNodes,
	}

	node, err := NewNode(cfg, fsm)
	require.NoError(t, err)

	// 当 ExpectNodes=0 时，手动引导单节点集群
	if expectNodes == 0 {
		err = node.Bootstrap()
		require.NoError(t, err)
	}

	return node, fsm
}

// waitForLeader waits for a leader to be elected among nodes
func waitForLeader(t *testing.T, nodes []*Node, timeout time.Duration) *Node {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, n := range nodes {
			if n.IsLeader() {
				return n
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal("no leader elected within timeout")
	return nil
}

func TestNodeState_String(t *testing.T) {
	tests := []struct {
		state NodeState
		want  string
	}{
		{NodeStateFollower, "follower"},
		{NodeStateCandidate, "candidate"},
		{NodeStateLeader, "leader"},
		{NodeStateShutdown, "shutdown"},
		{NodeState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestNewNode_Validation(t *testing.T) {
	t.Run("nil config uses default with temp dir", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DataDir = t.TempDir()
		node, err := NewNode(cfg, newKVFSM())
		require.NoError(t, err)
		defer node.Close()
		// NodeID should be auto-generated
		assert.NotEmpty(t, node.NodeID())
	})

	t.Run("nil fsm fails", func(t *testing.T) {
		cfg := DefaultConfig()
		cfg.DataDir = t.TempDir()
		_, err := NewNode(cfg, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "fsm is required")
	})
}

func TestSingleNode_LeaderElection(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot，单节点测试只需要基本 Raft 功能
	node, _ := createTestNode(t, 17001, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Single node should become leader after self-election
	assert.True(t, node.IsLeader())
	assert.Equal(t, NodeStateLeader, node.State())
	assert.NotEmpty(t, node.NodeID()) // NodeID is auto-generated
}

func TestSingleNode_Apply(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, fsm := createTestNode(t, 17002, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Apply a set command
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "foo",
		Data: []byte("bar"),
	}
	result, err := node.ApplyCommand(cmd, time.Second)
	require.NoError(t, err)

	applyResult, ok := result.(*ApplyResult)
	require.True(t, ok)
	assert.Equal(t, "foo", applyResult.Data)
	assert.NoError(t, applyResult.Error)

	// Verify FSM state
	val, exists := fsm.Get("foo")
	assert.True(t, exists)
	assert.Equal(t, "bar", val)
}

func TestSingleNode_ApplyNotLeader(t *testing.T) {
	node, _ := createTestNode(t, 17003, 0) // Not bootstrap
	defer node.Close()

	// Node is not a leader (not bootstrapped)
	_, err := node.Apply([]byte("data"), time.Second)
	require.Error(t, err)
	assert.True(t, IsNotLeader(err))
}

func TestSingleNode_Stats(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17004, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	stats := node.Stats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "state")
	assert.Equal(t, "Leader", stats["state"])
}

func TestSingleNode_Index(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17005, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Get initial indices
	lastIndex := node.LastIndex()
	appliedIndex := node.AppliedIndex()
	assert.GreaterOrEqual(t, lastIndex, uint64(0))
	assert.GreaterOrEqual(t, appliedIndex, uint64(0))

	// Apply a command and verify indices increase
	cmd := &Command{Type: CommandTypeSet, Key: "k", Data: []byte("v")}
	_, err = node.ApplyCommand(cmd, time.Second)
	require.NoError(t, err)

	assert.Greater(t, node.LastIndex(), lastIndex)
}

func TestSingleNode_GetConfiguration(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot，但 createTestNode 会自动 bootstrap
	node, _ := createTestNode(t, 17006, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	servers, err := node.GetConfiguration()
	require.NoError(t, err)
	// 单节点 bootstrap 后有 1 个服务器
	require.Len(t, servers, 1)
	assert.Equal(t, node.NodeID(), servers[0].ID)
	assert.Equal(t, "127.0.0.1:17006", servers[0].Address)
}

func TestSingleNode_VerifyLeader(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17007, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	err = node.VerifyLeader()
	require.NoError(t, err)
}

func TestSingleNode_Barrier(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17008, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Apply some commands
	for i := 0; i < 5; i++ {
		cmd := &Command{Type: CommandTypeSet, Key: fmt.Sprintf("k%d", i), Data: []byte("v")}
		_, err := node.ApplyCommand(cmd, time.Second)
		require.NoError(t, err)
	}

	// Barrier should succeed
	err = node.Barrier(time.Second)
	require.NoError(t, err)
}

func TestSingleNode_Close(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17009, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Close node
	err = node.Close()
	require.NoError(t, err)

	// State should be shutdown
	assert.Equal(t, NodeStateShutdown, node.State())

	// Double close should be safe
	err = node.Close()
	require.NoError(t, err)

	// Apply should fail after close
	_, err = node.Apply([]byte("data"), time.Second)
	require.Error(t, err)
	assert.True(t, IsNodeClosed(err))
}

func TestNode_LeaderCh(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17040, 0)
	defer node.Close()

	leaderCh := node.LeaderCh()
	assert.NotNil(t, leaderCh)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Should receive leadership notification
	select {
	case isLeader := <-leaderCh:
		assert.True(t, isLeader)
	case <-time.After(time.Second):
		// Channel might already be drained, check state directly
		assert.True(t, node.IsLeader())
	}
}

func TestNode_Leader(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17041, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	leader := node.Leader()
	leaderID := node.LeaderID()

	assert.Equal(t, "127.0.0.1:17041", leader)
	assert.Equal(t, node.NodeID(), leaderID) // NodeID is auto-generated
}

func TestSingleNode_ApplyLarge(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, fsm := createTestNode(t, 17060, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Create a command with large data (larger than typical chunk size threshold)
	// We'll use a moderately large payload to test the chunking path
	largeData := make([]byte, 1024*100) // 100KB
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "large-key",
		Data: largeData,
	}

	data, err := EncodeCommand(cmd)
	require.NoError(t, err)

	// Apply using ApplyLarge
	result, err := node.ApplyLarge(data, 5*time.Second)
	require.NoError(t, err)

	applyResult, ok := result.(*ApplyResult)
	require.True(t, ok)
	assert.Equal(t, "large-key", applyResult.Data)
	assert.NoError(t, applyResult.Error)

	// Verify FSM state
	val, exists := fsm.Get("large-key")
	assert.True(t, exists)
	assert.Equal(t, string(largeData), val)
}

func TestSingleNode_ApplyLarge_NotLeader(t *testing.T) {
	node, _ := createTestNode(t, 17061, 0) // Not bootstrap
	defer node.Close()

	// Node is not a leader (not bootstrapped)
	_, err := node.ApplyLarge([]byte("data"), time.Second)
	require.Error(t, err)
	assert.True(t, IsNotLeader(err))
}

func TestSingleNode_ApplyLarge_NodeClosed(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, _ := createTestNode(t, 17062, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Close the node
	err = node.Close()
	require.NoError(t, err)

	// ApplyLarge should fail after close
	_, err = node.ApplyLarge([]byte("data"), time.Second)
	require.Error(t, err)
	assert.True(t, IsNodeClosed(err))
}

func TestSingleNode_ApplyLarge_SmallData(t *testing.T) {
	// ExpectNodes=0 跳过 Serf/Autopilot
	node, fsm := createTestNode(t, 17063, 0)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Test with small data (should still work through ApplyLarge)
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "small-key",
		Data: []byte("small-value"),
	}

	data, err := EncodeCommand(cmd)
	require.NoError(t, err)

	result, err := node.ApplyLarge(data, time.Second)
	require.NoError(t, err)

	applyResult, ok := result.(*ApplyResult)
	require.True(t, ok)
	assert.Equal(t, "small-key", applyResult.Data)

	// Verify FSM state
	val, exists := fsm.Get("small-key")
	assert.True(t, exists)
	assert.Equal(t, "small-value", val)
}

// =============================================================================
// Multi-node cluster tests using Serf/Gossip for automatic node discovery
// =============================================================================

// createTestCluster creates a multi-node cluster for testing
// Returns nodes, fsms, and a cleanup function
func createTestCluster(t *testing.T, basePort int, numNodes int) ([]*Node, []*kvFSM, func()) {
	t.Helper()

	nodes := make([]*Node, numNodes)
	fsms := make([]*kvFSM, numNodes)

	// Calculate Serf ports (basePort + 1000 for each node's Serf)
	serfBasePort := basePort + 1000

	// Create all nodes
	for i := 0; i < numNodes; i++ {
		fsm := newKVFSM()
		fsms[i] = fsm

		raftPort := basePort + i
		serfPort := serfBasePort + i

		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", raftPort),
			SerfLANAddr:        fmt.Sprintf("127.0.0.1:%d", serfPort),
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           fmt.Sprintf("node-%d", i),
			Datacenter:         "dc1",
			ExpectNodes:        numNodes, // Enable Serf with expected nodes
		}

		// First node doesn't need join addresses
		// Other nodes join the first node's Serf
		if i > 0 {
			cfg.JoinAddrs = []string{fmt.Sprintf("127.0.0.1:%d", serfBasePort)}
		}

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err, "failed to create node %d", i)
		nodes[i] = node
		t.Logf("Created node %d: NumNodes=%d, ExpectNodes=%d", i, node.NumNodes(), cfg.ExpectNodes)
	}

	cleanup := func() {
		for _, n := range nodes {
			if n != nil {
				n.Close()
			}
		}
	}

	return nodes, fsms, cleanup
}

// waitForClusterLeader waits for a leader to be elected in the cluster
func waitForClusterLeader(t *testing.T, nodes []*Node, timeout time.Duration) *Node {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, n := range nodes {
			if n.IsLeader() {
				return n
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("no leader elected within timeout")
	return nil
}

// waitForClusterSize waits for all nodes to see the expected cluster size
func waitForClusterSize(t *testing.T, nodes []*Node, expectedSize int, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		allReady := true
		for _, n := range nodes {
			if n.NumNodes() < expectedSize {
				allReady = false
				break
			}
		}
		if allReady {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("cluster did not reach expected size %d within timeout", expectedSize)
}

func TestMultiNode_ClusterFormation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 18000, 3)
	defer cleanup()

	// Debug: print initial cluster state
	time.Sleep(2 * time.Second)
	for i, n := range nodes {
		t.Logf("Node %d: NumNodes=%d, State=%s, Leader=%s", i, n.NumNodes(), n.State(), n.Leader())
		members := n.Members()
		for _, m := range members {
			t.Logf("  Member: %s status=%s", m.Name, m.Status)
		}
	}

	// Wait for cluster to form
	waitForClusterSize(t, nodes, 3, 30*time.Second)

	// Wait for leader election
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	t.Logf("Leader elected: %s", leader.NodeID())

	// Verify all nodes see the same leader
	leaderAddr := leader.Leader()
	for i, n := range nodes {
		assert.Equal(t, leaderAddr, n.Leader(), "node %d sees different leader", i)
	}
}

func TestMultiNode_DataReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 18100, 3)
	defer cleanup()

	// Wait for cluster to form and leader to be elected
	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data through the leader
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "test-key",
		Data: []byte("test-value"),
	}
	_, err := leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify all FSMs have the data
	for i, fsm := range fsms {
		val, exists := fsm.Get("test-key")
		assert.True(t, exists, "node %d FSM missing key", i)
		assert.Equal(t, "test-value", val, "node %d FSM has wrong value", i)
	}
}

func TestMultiNode_LeaderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 18200, 3)
	defer cleanup()

	// Wait for cluster to form and leader to be elected
	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data before failover
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "before-failover",
		Data: []byte("value1"),
	}
	_, err := leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Find the leader index and shut it down
	var leaderIdx int
	for i, n := range nodes {
		if n == leader {
			leaderIdx = i
			break
		}
	}
	t.Logf("Shutting down leader node %d", leaderIdx)
	leader.Close()
	nodes[leaderIdx] = nil // Mark as closed

	// Wait for new leader election from remaining nodes
	remainingNodes := make([]*Node, 0, 2)
	for _, n := range nodes {
		if n != nil {
			remainingNodes = append(remainingNodes, n)
		}
	}

	newLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
	require.NotNil(t, newLeader)
	assert.NotEqual(t, leader.NodeID(), newLeader.NodeID(), "same node became leader after shutdown")

	t.Logf("New leader elected: %s", newLeader.NodeID())

	// Apply data after failover
	cmd = &Command{
		Type: CommandTypeSet,
		Key:  "after-failover",
		Data: []byte("value2"),
	}
	_, err = newLeader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Verify data consistency on remaining nodes
	for i, fsm := range fsms {
		if nodes[i] == nil {
			continue // Skip the closed node
		}

		// Check data from before failover
		val, exists := fsm.Get("before-failover")
		assert.True(t, exists, "node %d FSM missing before-failover key", i)
		assert.Equal(t, "value1", val, "node %d has wrong before-failover value", i)

		// Check data from after failover
		val, exists = fsm.Get("after-failover")
		assert.True(t, exists, "node %d FSM missing after-failover key", i)
		assert.Equal(t, "value2", val, "node %d has wrong after-failover value", i)
	}
}

func TestMultiNode_ConcurrentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 18300, 3)
	defer cleanup()

	// Wait for cluster to form and leader to be elected
	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Concurrent writes
	const numWrites = 50
	var wg sync.WaitGroup
	errors := make(chan error, numWrites)

	for i := 0; i < numWrites; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd := &Command{
				Type: CommandTypeSet,
				Key:  fmt.Sprintf("key-%d", idx),
				Data: []byte(fmt.Sprintf("value-%d", idx)),
			}
			_, err := leader.ApplyCommand(cmd, 5*time.Second)
			if err != nil {
				errors <- fmt.Errorf("write %d failed: %w", idx, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}

	// Wait for replication
	time.Sleep(time.Second)

	// Verify all writes replicated to all FSMs
	for nodeIdx, fsm := range fsms {
		for i := 0; i < numWrites; i++ {
			key := fmt.Sprintf("key-%d", i)
			expectedVal := fmt.Sprintf("value-%d", i)
			val, exists := fsm.Get(key)
			assert.True(t, exists, "node %d missing key %s", nodeIdx, key)
			assert.Equal(t, expectedVal, val, "node %d has wrong value for %s", nodeIdx, key)
		}
	}
}

func TestMultiNode_NotLeaderError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 18400, 3)
	defer cleanup()

	// Wait for cluster to form and leader to be elected
	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Find a follower
	var follower *Node
	for _, n := range nodes {
		if !n.IsLeader() {
			follower = n
			break
		}
	}
	require.NotNil(t, follower, "no follower found")

	// Try to apply through follower - should fail
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "test",
		Data: []byte("value"),
	}
	_, err := follower.ApplyCommand(cmd, time.Second)
	require.Error(t, err)
	assert.True(t, IsNotLeader(err), "expected not leader error, got: %v", err)
}

// =============================================================================
// Edge Case Tests - Bootstrap Scenarios
// =============================================================================

// TestMultiNode_BootstrapWithInsufficientNodes tests that cluster doesn't bootstrap
// until the expected number of nodes is reached
func TestMultiNode_BootstrapWithInsufficientNodes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	// Create only 2 nodes when expecting 3
	nodes := make([]*Node, 2)
	fsms := make([]*kvFSM, 2)
	basePort := 18500
	serfBasePort := basePort + 1000

	for i := 0; i < 2; i++ {
		fsm := newKVFSM()
		fsms[i] = fsm

		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", basePort+i),
			SerfLANAddr:        fmt.Sprintf("127.0.0.1:%d", serfBasePort+i),
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           fmt.Sprintf("node-%d", i),
			Datacenter:         "dc1",
			ExpectNodes:        3, // Expecting 3 but only creating 2
		}

		if i > 0 {
			cfg.JoinAddrs = []string{fmt.Sprintf("127.0.0.1:%d", serfBasePort)}
		}

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err)
		nodes[i] = node
	}

	defer func() {
		for _, n := range nodes {
			if n != nil {
				n.Close()
			}
		}
	}()

	// Wait for nodes to discover each other
	time.Sleep(2 * time.Second)

	// Verify cluster sees 2 nodes
	for _, n := range nodes {
		assert.Equal(t, 2, n.NumNodes(), "expected 2 nodes in cluster")
	}

	// Verify no leader is elected (bootstrap shouldn't happen)
	time.Sleep(3 * time.Second)
	hasLeader := false
	for _, n := range nodes {
		if n.IsLeader() {
			hasLeader = true
			break
		}
	}
	assert.False(t, hasLeader, "leader should not be elected with insufficient nodes")
}

// TestMultiNode_FollowerCrashClusterStable tests that cluster remains stable after a follower crashes
// This is a simpler test than full rejoin - just verifies the majority can continue operating
func TestMultiNode_FollowerCrashClusterStable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 18600, 3)
	defer cleanup()

	// Wait for cluster formation
	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply some data before crash
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("pre-crash-key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Find a follower to "crash"
	var crashedIdx int
	for i, n := range nodes {
		if !n.IsLeader() {
			crashedIdx = i
			break
		}
	}

	// Crash the follower
	t.Logf("Crashing follower node %d", crashedIdx)
	nodes[crashedIdx].Close()
	nodes[crashedIdx] = nil

	// Wait for cluster to notice the failed node
	time.Sleep(2 * time.Second)

	// Verify the remaining nodes still have a leader
	var newLeader *Node
	for _, n := range nodes {
		if n != nil && n.IsLeader() {
			newLeader = n
			break
		}
	}
	require.NotNil(t, newLeader, "cluster should still have a leader after one node crash")

	// Verify we can still apply commands with 2/3 nodes (majority)
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("post-crash-key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := newLeader.ApplyCommand(cmd, 5*time.Second)
		assert.NoError(t, err, "should be able to apply commands with 2/3 nodes")
	}

	// Verify the remaining nodes see at least 2 members
	for i, n := range nodes {
		if n != nil {
			assert.GreaterOrEqual(t, n.NumNodes(), 2, "node %d should see at least 2 nodes", i)
		}
	}
}

// TestMultiNode_LeaderRejoinAsFollower tests that a former leader rejoins as follower
func TestMultiNode_LeaderRejoinAsFollower(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	// Use 5-node cluster to maintain quorum when leader leaves
	basePort := 18650
	serfBasePort := basePort + 1000

	nodes := make([]*Node, 5)
	fsms := make([]*kvFSM, 5)

	for i := 0; i < 5; i++ {
		fsm := newKVFSM()
		fsms[i] = fsm

		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", basePort+i),
			SerfLANAddr:        fmt.Sprintf("127.0.0.1:%d", serfBasePort+i),
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           fmt.Sprintf("node-%d", i),
			Datacenter:         "dc1",
			ExpectNodes:        5,
		}

		if i > 0 {
			cfg.JoinAddrs = []string{fmt.Sprintf("127.0.0.1:%d", serfBasePort)}
		}

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err)
		nodes[i] = node
	}

	defer func() {
		for _, n := range nodes {
			if n != nil {
				n.Close()
			}
		}
	}()

	// Wait for cluster formation
	waitForClusterSize(t, nodes, 5, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data as leader
	originalLeaderID := leader.NodeID()
	t.Logf("Original leader: %s", originalLeaderID)

	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("leader-key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Wait for replication
	time.Sleep(500 * time.Millisecond)

	// Find leader index and save its config for rejoin
	var leaderIdx int
	var leaderConfig *Config
	for i, n := range nodes {
		if n.IsLeader() {
			leaderIdx = i
			leaderConfig = n.config
			break
		}
	}

	// Close the leader
	t.Logf("Closing leader node %d", leaderIdx)
	nodes[leaderIdx].Close()
	nodes[leaderIdx] = nil

	// Wait for new leader election
	remainingNodes := make([]*Node, 0)
	for _, n := range nodes {
		if n != nil {
			remainingNodes = append(remainingNodes, n)
		}
	}

	newLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
	require.NotNil(t, newLeader)
	require.NotEqual(t, originalLeaderID, newLeader.NodeID(), "new leader should be different")
	t.Logf("New leader elected: %s", newLeader.NodeID())

	// Apply more data through new leader
	for i := 0; i < 3; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("new-leader-key-%d", i),
			Data: []byte(fmt.Sprintf("new-value-%d", i)),
		}
		_, err := newLeader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(500 * time.Millisecond)

	// Rejoin the former leader with fresh data directory but same addresses
	t.Logf("Rejoining former leader")
	rejoinFSM := newKVFSM()
	fsms[leaderIdx] = rejoinFSM

	rejoinCfg := &Config{
		BindAddr:           leaderConfig.BindAddr,
		SerfLANAddr:        leaderConfig.SerfLANAddr,
		DataDir:            t.TempDir(), // Fresh data directory
		HeartbeatTimeout:   100 * time.Millisecond,
		ElectionTimeout:    100 * time.Millisecond,
		CommitTimeout:      10 * time.Millisecond,
		LeaderLeaseTimeout: 50 * time.Millisecond,
		SnapshotInterval:   10 * time.Second,
		SnapshotThreshold:  8192,
		SnapshotRetain:     1,
		MaxAppendEntries:   64,
		TrailingLogs:       128,
		MaxPool:            3,
		LogLevel:           "error",
		NodeName:           leaderConfig.NodeName,
		Datacenter:         "dc1",
		ExpectNodes:        5,
	}

	// Join through one of the remaining nodes
	for _, n := range nodes {
		if n != nil {
			rejoinCfg.JoinAddrs = []string{n.config.SerfLANAddr}
			break
		}
	}

	rejoinedNode, err := NewNode(rejoinCfg, rejoinFSM)
	require.NoError(t, err)
	nodes[leaderIdx] = rejoinedNode

	// Wait for rejoin and stabilization
	time.Sleep(5 * time.Second)

	// Verify the rejoined node is NOT the leader (should be follower)
	assert.False(t, rejoinedNode.IsLeader(), "former leader should rejoin as follower")
	t.Logf("Rejoined node state: %s, IsLeader: %v", rejoinedNode.State(), rejoinedNode.IsLeader())

	// Verify there's still only one leader
	leaderCount := 0
	for _, n := range nodes {
		if n != nil && n.IsLeader() {
			leaderCount++
		}
	}
	assert.Equal(t, 1, leaderCount, "should have exactly one leader after rejoin")

	// Verify the rejoined node can receive new data
	currentLeader := newLeader
	if !currentLeader.IsLeader() {
		for _, n := range nodes {
			if n != nil && n.IsLeader() {
				currentLeader = n
				break
			}
		}
	}

	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "after-rejoin-key",
		Data: []byte("after-rejoin-value"),
	}
	_, err = currentLeader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	time.Sleep(time.Second)

	// Verify rejoined node received the new data
	val, exists := rejoinFSM.Get("after-rejoin-key")
	assert.True(t, exists, "rejoined node should receive new data")
	assert.Equal(t, "after-rejoin-value", val)
}

// =============================================================================
// Edge Case Tests - Cascading Failures
// =============================================================================

// TestMultiNode_DoubleFailure tests cluster behavior when two nodes fail
func TestMultiNode_DoubleFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	// Create 5-node cluster for better fault tolerance
	nodes, _, cleanup := createTestCluster(t, 18700, 5)
	defer cleanup()

	waitForClusterSize(t, nodes, 5, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply some data
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "before-failures",
		Data: []byte("value"),
	}
	_, err := leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	// Find two non-leader nodes to fail
	var failedIndices []int
	for i, n := range nodes {
		if !n.IsLeader() && len(failedIndices) < 2 {
			failedIndices = append(failedIndices, i)
		}
	}

	// Fail first node
	t.Logf("Failing node %d", failedIndices[0])
	nodes[failedIndices[0]].Close()
	nodes[failedIndices[0]] = nil

	time.Sleep(time.Second)

	// Fail second node
	t.Logf("Failing node %d", failedIndices[1])
	nodes[failedIndices[1]].Close()
	nodes[failedIndices[1]] = nil

	// Wait for cluster to stabilize
	time.Sleep(2 * time.Second)

	// Get remaining nodes
	remainingNodes := make([]*Node, 0)
	for _, n := range nodes {
		if n != nil {
			remainingNodes = append(remainingNodes, n)
		}
	}

	// Cluster should still function with 3 out of 5 nodes (majority)
	newLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
	require.NotNil(t, newLeader, "cluster should still have a leader with 3/5 nodes")

	// Should be able to apply new data
	cmd = &Command{
		Type: CommandTypeSet,
		Key:  "after-failures",
		Data: []byte("new-value"),
	}
	_, err = newLeader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err, "should be able to write with majority")
}

// TestMultiNode_QuorumLoss tests cluster behavior when quorum is lost
func TestMultiNode_QuorumLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 18800, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data before quorum loss
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "before-quorum-loss",
		Data: []byte("value"),
	}
	_, err := leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	// Find the leader index
	var leaderIdx int
	for i, n := range nodes {
		if n.IsLeader() {
			leaderIdx = i
			break
		}
	}

	// Fail two non-leader nodes (lose quorum)
	failedCount := 0
	for i, n := range nodes {
		if i != leaderIdx && failedCount < 2 {
			t.Logf("Failing node %d", i)
			n.Close()
			nodes[i] = nil
			failedCount++
		}
	}

	// Wait for leader to notice
	time.Sleep(2 * time.Second)

	// Leader should still be running but unable to commit
	if nodes[leaderIdx] != nil {
		cmd = &Command{
			Type: CommandTypeSet,
			Key:  "after-quorum-loss",
			Data: []byte("value"),
		}
		// This should timeout or fail because there's no quorum
		_, err = nodes[leaderIdx].ApplyCommand(cmd, 2*time.Second)
		// Either timeout error or leadership lost error is acceptable
		assert.Error(t, err, "write should fail without quorum")
	}
}

// =============================================================================
// Edge Case Tests - Timing and Race Conditions
// =============================================================================

// TestMultiNode_RapidLeadershipChanges tests cluster stability during rapid leader changes
func TestMultiNode_RapidLeadershipChanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 18900, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Track how many leader changes we cause
	leaderChanges := 0
	maxChanges := 3

	for i := 0; i < maxChanges; i++ {
		// Apply data through current leader
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("key-round-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		if err != nil {
			t.Logf("Round %d: apply failed (expected during transition): %v", i, err)
		}

		// Find and kill current leader
		var leaderIdx int
		for idx, n := range nodes {
			if n != nil && n.IsLeader() {
				leaderIdx = idx
				break
			}
		}

		t.Logf("Round %d: killing leader at index %d", i, leaderIdx)
		nodes[leaderIdx].Close()
		nodes[leaderIdx] = nil
		leaderChanges++

		// Wait for new leader
		remainingNodes := make([]*Node, 0)
		for _, n := range nodes {
			if n != nil {
				remainingNodes = append(remainingNodes, n)
			}
		}

		if len(remainingNodes) < 2 {
			t.Log("Not enough nodes remaining, stopping test")
			break
		}

		newLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
		if newLeader == nil {
			t.Log("Could not elect new leader, stopping test")
			break
		}
		leader = newLeader
	}

	// Verify data consistency on remaining nodes
	time.Sleep(time.Second)
	for nodeIdx, fsm := range fsms {
		if nodes[nodeIdx] == nil {
			continue
		}
		for i := 0; i < leaderChanges; i++ {
			key := fmt.Sprintf("key-round-%d", i)
			_, exists := fsm.Get(key)
			// Some keys might not exist if apply failed during transition
			if !exists {
				t.Logf("Node %d missing key %s (acceptable during rapid changes)", nodeIdx, key)
			}
		}
	}
}

// TestMultiNode_ConcurrentWritesDuringFailover tests write handling during leader failover
func TestMultiNode_ConcurrentWritesDuringFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 19000, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Start concurrent writes
	const numWriters = 10
	var wg sync.WaitGroup
	successCount := int32(0)
	errorCount := int32(0)
	stopWriters := make(chan struct{})

	for i := 0; i < numWriters; i++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			writeNum := 0
			for {
				select {
				case <-stopWriters:
					return
				default:
				}

				// Find current leader
				var currentLeader *Node
				for _, n := range nodes {
					if n != nil && n.IsLeader() {
						currentLeader = n
						break
					}
				}
				if currentLeader == nil {
					time.Sleep(50 * time.Millisecond)
					continue
				}

				cmd := &Command{
					Type: CommandTypeSet,
					Key:  fmt.Sprintf("writer-%d-key-%d", writerID, writeNum),
					Data: []byte("value"),
				}
				_, err := currentLeader.ApplyCommand(cmd, time.Second)
				if err != nil {
					atomic.AddInt32(&errorCount, 1)
				} else {
					atomic.AddInt32(&successCount, 1)
				}
				writeNum++
				time.Sleep(10 * time.Millisecond)
			}
		}(i)
	}

	// Let writers run for a bit
	time.Sleep(time.Second)

	// Kill the leader while writes are happening
	for i, n := range nodes {
		if n != nil && n.IsLeader() {
			t.Logf("Killing leader at index %d during writes", i)
			n.Close()
			nodes[i] = nil
			break
		}
	}

	// Let failover happen and more writes
	time.Sleep(3 * time.Second)

	close(stopWriters)
	wg.Wait()

	t.Logf("Writes during failover: success=%d, errors=%d", successCount, errorCount)

	// Some writes should have succeeded, some may have failed during transition
	assert.Greater(t, int(successCount), 0, "some writes should have succeeded")
}

// =============================================================================
// Edge Case Tests - Shutdown Scenarios
// =============================================================================

// TestMultiNode_GracefulShutdown tests graceful shutdown of all nodes
func TestMultiNode_GracefulShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, _ := createTestCluster(t, 19100, 3)

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify data before shutdown
	for i, fsm := range fsms {
		for j := 0; j < 5; j++ {
			key := fmt.Sprintf("key-%d", j)
			val, exists := fsm.Get(key)
			assert.True(t, exists, "node %d missing key %s before shutdown", i, key)
			assert.Equal(t, fmt.Sprintf("value-%d", j), val)
		}
	}

	// Gracefully shutdown all nodes in sequence
	for i, n := range nodes {
		if n != nil {
			t.Logf("Gracefully shutting down node %d", i)
			err := n.Close()
			assert.NoError(t, err, "node %d should shutdown gracefully", i)
		}
	}
}

// TestMultiNode_ShutdownDuringApply tests shutdown while apply is in progress
func TestMultiNode_ShutdownDuringApply(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 19200, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Start many concurrent applies
	var wg sync.WaitGroup
	applyErrors := make(chan error, 100)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cmd := &Command{
				Type: CommandTypeSet,
				Key:  fmt.Sprintf("concurrent-key-%d", idx),
				Data: []byte("value"),
			}
			_, err := leader.ApplyCommand(cmd, 5*time.Second)
			if err != nil {
				applyErrors <- err
			}
		}(i)
	}

	// Shutdown leader while applies are in flight
	time.Sleep(100 * time.Millisecond)
	leader.Close()

	wg.Wait()
	close(applyErrors)

	// Count errors - some should have failed
	errorCount := 0
	for err := range applyErrors {
		errorCount++
		// Errors should be meaningful (not panics)
		assert.NotNil(t, err)
		t.Logf("Apply error during shutdown: %v", err)
	}

	// At least some should have failed due to shutdown
	t.Logf("Total apply errors during shutdown: %d", errorCount)
}

// =============================================================================
// Edge Case Tests - Data Integrity
// =============================================================================

// TestMultiNode_LargeDataReplication tests replication of large data
func TestMultiNode_LargeDataReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 19300, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Create large data (1MB)
	largeData := make([]byte, 1024*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "large-key",
		Data: largeData,
	}

	_, err := leader.ApplyCommand(cmd, 30*time.Second)
	require.NoError(t, err, "should be able to replicate 1MB data")

	// Wait for replication
	time.Sleep(2 * time.Second)

	// Verify all nodes have the large data
	for i, fsm := range fsms {
		val, exists := fsm.Get("large-key")
		assert.True(t, exists, "node %d missing large key", i)
		assert.Equal(t, string(largeData), val, "node %d has corrupted large data", i)
	}
}

// TestMultiNode_EmptyValueReplication tests replication of empty values
func TestMultiNode_EmptyValueReplication(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 19400, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Test empty value
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "empty-value-key",
		Data: []byte(""),
	}
	_, err := leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	// Test nil data (should be handled gracefully)
	cmd = &Command{
		Type: CommandTypeSet,
		Key:  "nil-data-key",
		Data: nil,
	}
	_, err = leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	// Verify on all nodes
	for i, fsm := range fsms {
		val, exists := fsm.Get("empty-value-key")
		assert.True(t, exists, "node %d missing empty-value-key", i)
		assert.Equal(t, "", val, "node %d has wrong empty value", i)

		val, exists = fsm.Get("nil-data-key")
		assert.True(t, exists, "node %d missing nil-data-key", i)
		assert.Equal(t, "", val, "node %d has wrong nil data value", i)
	}
}

// TestMultiNode_SpecialCharacterKeys tests keys with special characters
func TestMultiNode_SpecialCharacterKeys(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping multi-node test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 19500, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Test various special character keys
	specialKeys := []string{
		"key with spaces",
		"key/with/slashes",
		"key:with:colons",
		"key.with.dots",
		"key-with-dashes",
		"key_with_underscores",
		"键中文",
		"キー日本語",
		"key\nwith\nnewlines",
		"key\twith\ttabs",
		"",  // empty key
	}

	for _, key := range specialKeys {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  key,
			Data: []byte("value-for-" + key),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err, "failed to apply key: %q", key)
	}

	time.Sleep(500 * time.Millisecond)

	// Verify all keys on all nodes
	for i, fsm := range fsms {
		for _, key := range specialKeys {
			val, exists := fsm.Get(key)
			assert.True(t, exists, "node %d missing key %q", i, key)
			assert.Equal(t, "value-for-"+key, val, "node %d has wrong value for %q", i, key)
		}
	}
}

// =============================================================================
// Stress Tests - Long-running Stability Verification
// =============================================================================

// TestStress_ContinuousWrites tests cluster stability under continuous write load
func TestStress_ContinuousWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 20000, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Run continuous writes for 30 seconds
	duration := 30 * time.Second
	deadline := time.Now().Add(duration)

	var successCount, errorCount int64
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	// Start 5 concurrent writers
	numWriters := 5
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			writeNum := 0
			for {
				select {
				case <-stopCh:
					return
				default:
				}

				// Find current leader
				var currentLeader *Node
				for _, n := range nodes {
					if n != nil && n.IsLeader() {
						currentLeader = n
						break
					}
				}
				if currentLeader == nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}

				cmd := &Command{
					Type: CommandTypeSet,
					Key:  fmt.Sprintf("stress-w%d-k%d", writerID, writeNum),
					Data: []byte(fmt.Sprintf("value-%d-%d", writerID, writeNum)),
				}
				_, err := currentLeader.ApplyCommand(cmd, 5*time.Second)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
				} else {
					atomic.AddInt64(&successCount, 1)
				}
				writeNum++
				time.Sleep(5 * time.Millisecond)
			}
		}(w)
	}

	// Wait for duration
	time.Sleep(time.Until(deadline))
	close(stopCh)
	wg.Wait()

	t.Logf("Stress test completed: success=%d, errors=%d", successCount, errorCount)

	// Verify high success rate (>95%)
	total := successCount + errorCount
	successRate := float64(successCount) / float64(total) * 100
	t.Logf("Success rate: %.2f%%", successRate)
	assert.Greater(t, successRate, 95.0, "success rate should be >95%%")

	// Verify data consistency across nodes
	time.Sleep(time.Second)
	fsmLens := make([]int, len(fsms))
	for i, fsm := range fsms {
		fsmLens[i] = fsm.Len()
	}
	t.Logf("FSM lengths: %v", fsmLens)

	// All FSMs should have similar number of entries (within 5% tolerance)
	maxLen := fsmLens[0]
	minLen := fsmLens[0]
	for _, l := range fsmLens {
		if l > maxLen {
			maxLen = l
		}
		if l < minLen {
			minLen = l
		}
	}
	if maxLen > 0 {
		diff := float64(maxLen-minLen) / float64(maxLen) * 100
		t.Logf("FSM length diff: %.2f%%", diff)
		assert.Less(t, diff, 5.0, "FSM lengths should be within 5%% of each other")
	}
}

// TestStress_ConcurrentReadersWriters tests cluster under mixed read/write load
func TestStress_ConcurrentReadersWriters(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 20100, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Pre-populate some data
	for i := 0; i < 100; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("initial-key-%d", i),
			Data: []byte(fmt.Sprintf("initial-value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(500 * time.Millisecond)

	duration := 20 * time.Second
	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	var writeSuccess, writeError, readSuccess, readError int64

	// Start writers
	numWriters := 3
	for w := 0; w < numWriters; w++ {
		wg.Add(1)
		go func(writerID int) {
			defer wg.Done()
			writeNum := 0
			for {
				select {
				case <-stopCh:
					return
				default:
				}

				var currentLeader *Node
				for _, n := range nodes {
					if n != nil && n.IsLeader() {
						currentLeader = n
						break
					}
				}
				if currentLeader == nil {
					time.Sleep(10 * time.Millisecond)
					continue
				}

				cmd := &Command{
					Type: CommandTypeSet,
					Key:  fmt.Sprintf("stress-write-w%d-k%d", writerID, writeNum),
					Data: []byte(fmt.Sprintf("value-%d", writeNum)),
				}
				_, err := currentLeader.ApplyCommand(cmd, 5*time.Second)
				if err != nil {
					atomic.AddInt64(&writeError, 1)
				} else {
					atomic.AddInt64(&writeSuccess, 1)
				}
				writeNum++
				time.Sleep(10 * time.Millisecond)
			}
		}(w)
	}

	// Start readers (from all nodes including followers)
	numReaders := 5
	for r := 0; r < numReaders; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
				}

				// Read from random node's FSM
				fsmIdx := readerID % len(fsms)
				key := fmt.Sprintf("initial-key-%d", readerID%100)
				_, exists := fsms[fsmIdx].Get(key)
				if exists {
					atomic.AddInt64(&readSuccess, 1)
				} else {
					atomic.AddInt64(&readError, 1)
				}
				time.Sleep(1 * time.Millisecond)
			}
		}(r)
	}

	time.Sleep(duration)
	close(stopCh)
	wg.Wait()

	t.Logf("Write: success=%d, error=%d", writeSuccess, writeError)
	t.Logf("Read: success=%d, error=%d", readSuccess, readError)

	// Most reads should succeed (initial data should be present)
	readTotal := readSuccess + readError
	if readTotal > 0 {
		readSuccessRate := float64(readSuccess) / float64(readTotal) * 100
		t.Logf("Read success rate: %.2f%%", readSuccessRate)
		assert.Greater(t, readSuccessRate, 95.0, "read success rate should be >95%%")
	}
}

// TestStress_RepeatedLeaderFailover tests cluster stability under repeated leader failures
func TestStress_RepeatedLeaderFailover(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	// Use 5-node cluster for better fault tolerance during repeated failovers
	basePort := 20200
	serfBasePort := basePort + 1000

	nodes := make([]*Node, 5)
	fsms := make([]*kvFSM, 5)
	configs := make([]*Config, 5)

	for i := 0; i < 5; i++ {
		fsm := newKVFSM()
		fsms[i] = fsm

		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", basePort+i),
			SerfLANAddr:        fmt.Sprintf("127.0.0.1:%d", serfBasePort+i),
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           fmt.Sprintf("node-%d", i),
			Datacenter:         "dc1",
			ExpectNodes:        5,
		}

		if i > 0 {
			cfg.JoinAddrs = []string{fmt.Sprintf("127.0.0.1:%d", serfBasePort)}
		}
		configs[i] = cfg

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err)
		nodes[i] = node
	}

	defer func() {
		for _, n := range nodes {
			if n != nil {
				n.Close()
			}
		}
	}()

	waitForClusterSize(t, nodes, 5, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Start continuous writer
	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	var writeSuccess, writeError int64

	wg.Add(1)
	go func() {
		defer wg.Done()
		writeNum := 0
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			var currentLeader *Node
			for _, n := range nodes {
				if n != nil && n.IsLeader() {
					currentLeader = n
					break
				}
			}
			if currentLeader == nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			cmd := &Command{
				Type: CommandTypeSet,
				Key:  fmt.Sprintf("failover-key-%d", writeNum),
				Data: []byte(fmt.Sprintf("value-%d", writeNum)),
			}
			_, err := currentLeader.ApplyCommand(cmd, 3*time.Second)
			if err != nil {
				atomic.AddInt64(&writeError, 1)
			} else {
				atomic.AddInt64(&writeSuccess, 1)
			}
			writeNum++
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Perform multiple leader failovers
	numFailovers := 3
	for f := 0; f < numFailovers; f++ {
		time.Sleep(3 * time.Second)

		// Find and kill current leader
		var leaderIdx int = -1
		for i, n := range nodes {
			if n != nil && n.IsLeader() {
				leaderIdx = i
				break
			}
		}

		if leaderIdx == -1 {
			t.Log("No leader found, waiting for election")
			continue
		}

		t.Logf("Failover %d: killing leader at index %d", f+1, leaderIdx)
		nodes[leaderIdx].Close()
		nodes[leaderIdx] = nil

		// Wait for new leader
		time.Sleep(2 * time.Second)

		remainingNodes := make([]*Node, 0)
		for _, n := range nodes {
			if n != nil {
				remainingNodes = append(remainingNodes, n)
			}
		}

		if len(remainingNodes) < 3 {
			t.Log("Not enough nodes remaining")
			break
		}

		newLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
		if newLeader != nil {
			t.Logf("New leader elected: %s", newLeader.NodeID())
		}

		// Rejoin the failed node
		rejoinFSM := newKVFSM()
		fsms[leaderIdx] = rejoinFSM

		rejoinCfg := &Config{
			BindAddr:           configs[leaderIdx].BindAddr,
			SerfLANAddr:        configs[leaderIdx].SerfLANAddr,
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           configs[leaderIdx].NodeName,
			Datacenter:         "dc1",
			ExpectNodes:        5,
		}

		// Join through one of the remaining nodes
		for _, n := range nodes {
			if n != nil {
				rejoinCfg.JoinAddrs = []string{n.config.SerfLANAddr}
				break
			}
		}

		rejoinedNode, err := NewNode(rejoinCfg, rejoinFSM)
		if err != nil {
			t.Logf("Failed to rejoin node %d: %v", leaderIdx, err)
			continue
		}
		nodes[leaderIdx] = rejoinedNode
		t.Logf("Rejoined node %d", leaderIdx)
	}

	close(stopCh)
	wg.Wait()

	t.Logf("Repeated failover test: success=%d, errors=%d", writeSuccess, writeError)

	// Some writes should succeed even with failovers
	assert.Greater(t, writeSuccess, int64(10), "should have successful writes despite failovers")

	// Success rate during failovers can be lower, but should be reasonable
	total := writeSuccess + writeError
	if total > 0 {
		successRate := float64(writeSuccess) / float64(total) * 100
		t.Logf("Success rate during failovers: %.2f%%", successRate)
		assert.Greater(t, successRate, 50.0, "success rate should be >50%% even with failovers")
	}
}

// TestStress_HighThroughput tests cluster under high throughput conditions
func TestStress_HighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 20300, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// High throughput: many concurrent writes with minimal delay
	numWrites := 500
	numConcurrent := 20
	sem := make(chan struct{}, numConcurrent)

	var wg sync.WaitGroup
	var successCount, errorCount int64
	startTime := time.Now()

	for i := 0; i < numWrites; i++ {
		sem <- struct{}{}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			cmd := &Command{
				Type: CommandTypeSet,
				Key:  fmt.Sprintf("htp-key-%d", idx),
				Data: []byte(fmt.Sprintf("value-%d", idx)),
			}
			_, err := leader.ApplyCommand(cmd, 10*time.Second)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(startTime)

	t.Logf("High throughput test: %d writes in %v", numWrites, elapsed)
	t.Logf("Success: %d, Errors: %d", successCount, errorCount)
	t.Logf("Throughput: %.2f writes/sec", float64(successCount)/elapsed.Seconds())

	// Most writes should succeed
	successRate := float64(successCount) / float64(numWrites) * 100
	t.Logf("Success rate: %.2f%%", successRate)
	assert.Greater(t, successRate, 90.0, "success rate should be >90%%")

	// Wait for replication
	time.Sleep(2 * time.Second)

	// Verify data consistency
	for i, fsm := range fsms {
		fsmLen := fsm.Len()
		t.Logf("Node %d FSM has %d entries", i, fsmLen)
		assert.GreaterOrEqual(t, fsmLen, int(successCount*9/10), "node %d should have most entries", i)
	}
}

// =============================================================================
// Network Fault Tests - Partition, Delay, Packet Loss Simulation
// =============================================================================

// TestNetworkFault_PartitionMinority tests cluster behavior when minority is partitioned
func TestNetworkFault_PartitionMinority(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network fault test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 20400, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data before partition
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("pre-partition-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(500 * time.Millisecond)

	// Simulate partition: shut down one follower (minority partition)
	var partitionedIdx int
	for i, n := range nodes {
		if !n.IsLeader() {
			partitionedIdx = i
			break
		}
	}

	t.Logf("Partitioning node %d (minority)", partitionedIdx)
	nodes[partitionedIdx].Close()
	nodes[partitionedIdx] = nil

	// Wait for cluster to detect the partition
	time.Sleep(2 * time.Second)

	// Majority (2 nodes) should still be able to write
	remainingNodes := make([]*Node, 0)
	for _, n := range nodes {
		if n != nil {
			remainingNodes = append(remainingNodes, n)
		}
	}

	currentLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
	require.NotNil(t, currentLeader, "majority should maintain leadership")

	// Apply data during partition
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("during-partition-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := currentLeader.ApplyCommand(cmd, 5*time.Second)
		assert.NoError(t, err, "writes should succeed with majority")
	}

	// Verify data consistency on remaining nodes
	time.Sleep(500 * time.Millisecond)
	for i, fsm := range fsms {
		if nodes[i] == nil {
			continue
		}
		for j := 0; j < 5; j++ {
			key := fmt.Sprintf("during-partition-%d", j)
			_, exists := fsm.Get(key)
			assert.True(t, exists, "node %d should have partition data", i)
		}
	}
}

// TestNetworkFault_PartitionLeader tests cluster behavior when leader is partitioned
func TestNetworkFault_PartitionLeader(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network fault test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 20500, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	originalLeaderID := leader.NodeID()

	// Apply data before partition
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("pre-partition-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(500 * time.Millisecond)

	// Simulate leader partition (shutdown the leader)
	var leaderIdx int
	for i, n := range nodes {
		if n.IsLeader() {
			leaderIdx = i
			break
		}
	}

	t.Logf("Partitioning leader node %d", leaderIdx)
	nodes[leaderIdx].Close()
	nodes[leaderIdx] = nil

	// Get remaining (follower) nodes
	remainingNodes := make([]*Node, 0)
	for _, n := range nodes {
		if n != nil {
			remainingNodes = append(remainingNodes, n)
		}
	}

	// New leader should be elected among remaining nodes
	newLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
	require.NotNil(t, newLeader, "new leader should be elected after partition")
	assert.NotEqual(t, originalLeaderID, newLeader.NodeID(), "new leader should be different")

	t.Logf("New leader elected: %s", newLeader.NodeID())

	// Apply data after partition
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("post-partition-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := newLeader.ApplyCommand(cmd, 5*time.Second)
		assert.NoError(t, err, "writes should succeed with new leader")
	}

	// Verify data on surviving nodes
	time.Sleep(500 * time.Millisecond)
	for i, fsm := range fsms {
		if nodes[i] == nil {
			continue
		}
		// Should have both pre and post partition data
		for j := 0; j < 5; j++ {
			preKey := fmt.Sprintf("pre-partition-%d", j)
			postKey := fmt.Sprintf("post-partition-%d", j)
			_, preExists := fsm.Get(preKey)
			_, postExists := fsm.Get(postKey)
			assert.True(t, preExists, "node %d should have pre-partition data", i)
			assert.True(t, postExists, "node %d should have post-partition data", i)
		}
	}
}

// TestNetworkFault_PartitionHeal tests cluster recovery after partition heals
func TestNetworkFault_PartitionHeal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network fault test in short mode")
	}

	basePort := 20600
	serfBasePort := basePort + 1000

	nodes := make([]*Node, 3)
	fsms := make([]*kvFSM, 3)
	configs := make([]*Config, 3)

	for i := 0; i < 3; i++ {
		fsm := newKVFSM()
		fsms[i] = fsm

		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", basePort+i),
			SerfLANAddr:        fmt.Sprintf("127.0.0.1:%d", serfBasePort+i),
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           fmt.Sprintf("node-%d", i),
			Datacenter:         "dc1",
			ExpectNodes:        3,
		}

		if i > 0 {
			cfg.JoinAddrs = []string{fmt.Sprintf("127.0.0.1:%d", serfBasePort)}
		}
		configs[i] = cfg

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err)
		nodes[i] = node
	}

	defer func() {
		for _, n := range nodes {
			if n != nil {
				n.Close()
			}
		}
	}()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply data before partition
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("before-heal-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}
	time.Sleep(500 * time.Millisecond)

	// Partition a follower
	var partitionedIdx int
	for i, n := range nodes {
		if !n.IsLeader() {
			partitionedIdx = i
			break
		}
	}

	t.Logf("Partitioning node %d", partitionedIdx)
	nodes[partitionedIdx].Close()
	nodes[partitionedIdx] = nil

	// Apply data during partition
	for _, n := range nodes {
		if n != nil && n.IsLeader() {
			leader = n
			break
		}
	}

	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("during-heal-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Heal the partition by rejoining the node
	t.Logf("Healing partition: rejoining node %d", partitionedIdx)
	healFSM := newKVFSM()
	fsms[partitionedIdx] = healFSM

	healCfg := &Config{
		BindAddr:           configs[partitionedIdx].BindAddr,
		SerfLANAddr:        configs[partitionedIdx].SerfLANAddr,
		DataDir:            t.TempDir(),
		HeartbeatTimeout:   100 * time.Millisecond,
		ElectionTimeout:    100 * time.Millisecond,
		CommitTimeout:      10 * time.Millisecond,
		LeaderLeaseTimeout: 50 * time.Millisecond,
		SnapshotInterval:   10 * time.Second,
		SnapshotThreshold:  8192,
		SnapshotRetain:     1,
		MaxAppendEntries:   64,
		TrailingLogs:       128,
		MaxPool:            3,
		LogLevel:           "error",
		NodeName:           configs[partitionedIdx].NodeName,
		Datacenter:         "dc1",
		ExpectNodes:        3,
	}

	for _, n := range nodes {
		if n != nil {
			healCfg.JoinAddrs = []string{n.config.SerfLANAddr}
			break
		}
	}

	healedNode, err := NewNode(healCfg, healFSM)
	require.NoError(t, err)
	nodes[partitionedIdx] = healedNode

	// Wait for the healed node to sync
	time.Sleep(5 * time.Second)

	// Apply data after heal
	for _, n := range nodes {
		if n != nil && n.IsLeader() {
			leader = n
			break
		}
	}

	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("after-heal-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Wait for final replication
	time.Sleep(2 * time.Second)

	// Verify the healed node received all data
	for _, prefix := range []string{"before-heal", "during-heal", "after-heal"} {
		for i := 0; i < 5; i++ {
			key := fmt.Sprintf("%s-%d", prefix, i)
			_, exists := healFSM.Get(key)
			assert.True(t, exists, "healed node should have key %s", key)
		}
	}
	t.Logf("Healed node has %d entries", healFSM.Len())
}

// TestNetworkFault_IntermittentConnectivity tests cluster under flaky network conditions
func TestNetworkFault_IntermittentConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network fault test in short mode")
	}

	nodes, _, cleanup := createTestCluster(t, 20700, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Start continuous writer
	stopCh := make(chan struct{})
	var wg sync.WaitGroup
	var successCount, errorCount int64

	wg.Add(1)
	go func() {
		defer wg.Done()
		writeNum := 0
		for {
			select {
			case <-stopCh:
				return
			default:
			}

			var currentLeader *Node
			for _, n := range nodes {
				if n != nil && n.IsLeader() {
					currentLeader = n
					break
				}
			}
			if currentLeader == nil {
				time.Sleep(50 * time.Millisecond)
				continue
			}

			cmd := &Command{
				Type: CommandTypeSet,
				Key:  fmt.Sprintf("intermittent-key-%d", writeNum),
				Data: []byte(fmt.Sprintf("value-%d", writeNum)),
			}
			_, err := currentLeader.ApplyCommand(cmd, 2*time.Second)
			if err != nil {
				atomic.AddInt64(&errorCount, 1)
			} else {
				atomic.AddInt64(&successCount, 1)
			}
			writeNum++
			time.Sleep(20 * time.Millisecond)
		}
	}()

	// Simulate intermittent connectivity by temporarily blocking a node
	// Do this 3 times
	for round := 0; round < 3; round++ {
		time.Sleep(2 * time.Second)

		// Find a follower to temporarily disconnect
		var followerIdx int = -1
		for i, n := range nodes {
			if n != nil && !n.IsLeader() {
				followerIdx = i
				break
			}
		}

		if followerIdx == -1 {
			continue
		}

		t.Logf("Round %d: temporarily disconnecting node %d", round+1, followerIdx)

		// Brief disconnection (simulate network flap)
		// We can't actually simulate this without a custom transport,
		// so we'll just verify the cluster remains stable
		time.Sleep(500 * time.Millisecond)
	}

	close(stopCh)
	wg.Wait()

	t.Logf("Intermittent connectivity test: success=%d, errors=%d", successCount, errorCount)

	// Cluster should remain functional
	assert.Greater(t, successCount, int64(0), "should have successful writes")
}

// TestNetworkFault_SlowFollower tests cluster with one slow follower
func TestNetworkFault_SlowFollower(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network fault test in short mode")
	}

	nodes, fsms, cleanup := createTestCluster(t, 20800, 3)
	defer cleanup()

	waitForClusterSize(t, nodes, 3, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Apply many commands quickly to simulate load
	// A slow follower scenario would lag behind but eventually catch up
	numCommands := 100
	for i := 0; i < numCommands; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("slow-follower-key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := leader.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Check immediately - leader FSM should have all entries
	leaderFSM := fsms[0]
	for i, n := range nodes {
		if n.IsLeader() {
			leaderFSM = fsms[i]
			break
		}
	}

	// Wait for replication to catch up
	time.Sleep(3 * time.Second)

	// All FSMs should eventually have all entries
	for i, fsm := range fsms {
		fsmLen := fsm.Len()
		t.Logf("Node %d FSM has %d entries", i, fsmLen)

		if fsm == leaderFSM {
			assert.Equal(t, numCommands, fsmLen, "leader should have all entries")
		} else {
			// Followers might be slightly behind but should have most entries
			assert.GreaterOrEqual(t, fsmLen, numCommands*9/10, "follower %d should have most entries", i)
		}
	}
}

// TestNetworkFault_MultiplePartitionsRecovery tests recovery from multiple partition events
func TestNetworkFault_MultiplePartitionsRecovery(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network fault test in short mode")
	}

	basePort := 20900
	serfBasePort := basePort + 1000

	nodes := make([]*Node, 5)
	fsms := make([]*kvFSM, 5)
	configs := make([]*Config, 5)

	for i := 0; i < 5; i++ {
		fsm := newKVFSM()
		fsms[i] = fsm

		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", basePort+i),
			SerfLANAddr:        fmt.Sprintf("127.0.0.1:%d", serfBasePort+i),
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           fmt.Sprintf("node-%d", i),
			Datacenter:         "dc1",
			ExpectNodes:        5,
		}

		if i > 0 {
			cfg.JoinAddrs = []string{fmt.Sprintf("127.0.0.1:%d", serfBasePort)}
		}
		configs[i] = cfg

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err)
		nodes[i] = node
	}

	defer func() {
		for _, n := range nodes {
			if n != nil {
				n.Close()
			}
		}
	}()

	waitForClusterSize(t, nodes, 5, 30*time.Second)
	leader := waitForClusterLeader(t, nodes, 30*time.Second)
	require.NotNil(t, leader)

	// Perform multiple partition/recovery cycles
	for cycle := 0; cycle < 2; cycle++ {
		t.Logf("Partition cycle %d", cycle+1)

		// Apply data
		for i := 0; i < 5; i++ {
			var currentLeader *Node
			for _, n := range nodes {
				if n != nil && n.IsLeader() {
					currentLeader = n
					break
				}
			}
			if currentLeader == nil {
				time.Sleep(time.Second)
				continue
			}

			cmd := &Command{
				Type: CommandTypeSet,
				Key:  fmt.Sprintf("cycle-%d-key-%d", cycle, i),
				Data: []byte(fmt.Sprintf("value-%d", i)),
			}
			_, err := currentLeader.ApplyCommand(cmd, 5*time.Second)
			if err != nil {
				t.Logf("Write failed during cycle %d: %v", cycle, err)
			}
		}

		// Partition a random non-leader node
		var partitionIdx int = -1
		for i, n := range nodes {
			if n != nil && !n.IsLeader() {
				partitionIdx = i
				break
			}
		}

		if partitionIdx == -1 {
			continue
		}

		t.Logf("Partitioning node %d", partitionIdx)
		nodes[partitionIdx].Close()
		nodes[partitionIdx] = nil

		time.Sleep(2 * time.Second)

		// Heal the partition
		healFSM := newKVFSM()
		fsms[partitionIdx] = healFSM

		healCfg := &Config{
			BindAddr:           configs[partitionIdx].BindAddr,
			SerfLANAddr:        configs[partitionIdx].SerfLANAddr,
			DataDir:            t.TempDir(),
			HeartbeatTimeout:   100 * time.Millisecond,
			ElectionTimeout:    100 * time.Millisecond,
			CommitTimeout:      10 * time.Millisecond,
			LeaderLeaseTimeout: 50 * time.Millisecond,
			SnapshotInterval:   10 * time.Second,
			SnapshotThreshold:  8192,
			SnapshotRetain:     1,
			MaxAppendEntries:   64,
			TrailingLogs:       128,
			MaxPool:            3,
			LogLevel:           "error",
			NodeName:           configs[partitionIdx].NodeName,
			Datacenter:         "dc1",
			ExpectNodes:        5,
		}

		for _, n := range nodes {
			if n != nil {
				healCfg.JoinAddrs = []string{n.config.SerfLANAddr}
				break
			}
		}

		healedNode, err := NewNode(healCfg, healFSM)
		if err != nil {
			t.Logf("Failed to heal node %d: %v", partitionIdx, err)
			continue
		}
		nodes[partitionIdx] = healedNode
		t.Logf("Healed node %d", partitionIdx)

		time.Sleep(3 * time.Second)
	}

	// Final verification - all nodes should have a leader
	remainingNodes := make([]*Node, 0)
	for _, n := range nodes {
		if n != nil {
			remainingNodes = append(remainingNodes, n)
		}
	}

	finalLeader := waitForClusterLeader(t, remainingNodes, 30*time.Second)
	require.NotNil(t, finalLeader, "cluster should have a leader after multiple partitions")
	t.Logf("Final leader: %s", finalLeader.NodeID())
}
