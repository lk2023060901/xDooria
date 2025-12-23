package raft

import (
	"context"
	"fmt"
	"io"
	"sync"
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
func createTestNode(t *testing.T, port int, bootstrap bool) (*Node, *kvFSM) {
	t.Helper()

	fsm := newKVFSM()
	cfg := &Config{
		BindAddr:           fmt.Sprintf("127.0.0.1:%d", port),
		DataDir:            t.TempDir(),
		Bootstrap:          bootstrap,
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
	}

	node, err := NewNode(cfg, fsm)
	require.NoError(t, err)

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
	node, _ := createTestNode(t, 17001, true)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	// Single bootstrap node should become leader
	assert.True(t, node.IsLeader())
	assert.Equal(t, NodeStateLeader, node.State())
	assert.NotEmpty(t, node.NodeID()) // NodeID is auto-generated
}

func TestSingleNode_Apply(t *testing.T) {
	node, fsm := createTestNode(t, 17002, true)
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
	node, _ := createTestNode(t, 17003, false) // Not bootstrap
	defer node.Close()

	// Node is not a leader (not bootstrapped)
	_, err := node.Apply([]byte("data"), time.Second)
	require.Error(t, err)
	assert.True(t, IsNotLeader(err))
}

func TestSingleNode_Stats(t *testing.T) {
	node, _ := createTestNode(t, 17004, true)
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
	node, _ := createTestNode(t, 17005, true)
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
	node, _ := createTestNode(t, 17006, true)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	servers, err := node.GetConfiguration()
	require.NoError(t, err)
	require.Len(t, servers, 1)
	assert.Equal(t, node.NodeID(), servers[0].ID) // NodeID is auto-generated
	assert.Equal(t, "127.0.0.1:17006", servers[0].Address)
}

func TestSingleNode_VerifyLeader(t *testing.T) {
	node, _ := createTestNode(t, 17007, true)
	defer node.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := node.Start(ctx)
	require.NoError(t, err)

	err = node.VerifyLeader()
	require.NoError(t, err)
}

func TestSingleNode_Barrier(t *testing.T) {
	node, _ := createTestNode(t, 17008, true)
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
	node, _ := createTestNode(t, 17009, true)

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

func TestThreeNode_LeaderElection(t *testing.T) {
	// Create 3 nodes
	node1, _ := createTestNode(t, 17010, true)
	defer node1.Close()

	node2, _ := createTestNode(t, 17011, false)
	defer node2.Close()

	node3, _ := createTestNode(t, 17012, false)
	defer node3.Close()

	// Start node1 first (bootstrap)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node1.Start(ctx)
	require.NoError(t, err)
	require.True(t, node1.IsLeader())

	// Add node2 and node3 to cluster using auto-generated node IDs
	err = node1.AddVoter(node2.NodeID(), "127.0.0.1:17011", 0, 5*time.Second)
	require.NoError(t, err)

	err = node1.AddVoter(node3.NodeID(), "127.0.0.1:17012", 0, 5*time.Second)
	require.NoError(t, err)

	// Verify cluster configuration
	servers, err := node1.GetConfiguration()
	require.NoError(t, err)
	assert.Len(t, servers, 3)

	// Verify exactly one leader
	nodes := []*Node{node1, node2, node3}
	leaderCount := 0
	for _, n := range nodes {
		if n.IsLeader() {
			leaderCount++
		}
	}
	assert.Equal(t, 1, leaderCount)
}

func TestThreeNode_DataReplication(t *testing.T) {
	// Create 3 nodes
	node1, fsm1 := createTestNode(t, 17020, true)
	defer node1.Close()

	node2, fsm2 := createTestNode(t, 17021, false)
	defer node2.Close()

	node3, fsm3 := createTestNode(t, 17022, false)
	defer node3.Close()

	// Start cluster
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node1.Start(ctx)
	require.NoError(t, err)

	err = node1.AddVoter(node2.NodeID(), "127.0.0.1:17021", 0, 5*time.Second)
	require.NoError(t, err)

	err = node1.AddVoter(node3.NodeID(), "127.0.0.1:17022", 0, 5*time.Second)
	require.NoError(t, err)

	// Apply data on leader
	for i := 0; i < 10; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("key%d", i),
			Data: []byte(fmt.Sprintf("value%d", i)),
		}
		_, err := node1.ApplyCommand(cmd, time.Second)
		require.NoError(t, err)
	}

	// Wait for replication
	time.Sleep(200 * time.Millisecond)

	// Verify all FSMs have the same data
	assert.Equal(t, 10, fsm1.Len())
	assert.Equal(t, 10, fsm2.Len())
	assert.Equal(t, 10, fsm3.Len())

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key%d", i)
		expected := fmt.Sprintf("value%d", i)

		v1, ok1 := fsm1.Get(key)
		v2, ok2 := fsm2.Get(key)
		v3, ok3 := fsm3.Get(key)

		assert.True(t, ok1 && ok2 && ok3, "key %s not found in all FSMs", key)
		assert.Equal(t, expected, v1)
		assert.Equal(t, expected, v2)
		assert.Equal(t, expected, v3)
	}
}

func TestThreeNode_RemoveServer(t *testing.T) {
	// Create 3 nodes
	node1, _ := createTestNode(t, 17030, true)
	defer node1.Close()

	node2, _ := createTestNode(t, 17031, false)
	defer node2.Close()

	node3, _ := createTestNode(t, 17032, false)
	defer node3.Close()

	// Start cluster
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := node1.Start(ctx)
	require.NoError(t, err)

	err = node1.AddVoter(node2.NodeID(), "127.0.0.1:17031", 0, 5*time.Second)
	require.NoError(t, err)

	err = node1.AddVoter(node3.NodeID(), "127.0.0.1:17032", 0, 5*time.Second)
	require.NoError(t, err)

	// Verify 3 nodes
	servers, _ := node1.GetConfiguration()
	assert.Len(t, servers, 3)

	// Remove node3
	err = node1.RemoveServer(node3.NodeID(), 0, 5*time.Second)
	require.NoError(t, err)

	// Verify 2 nodes remain
	servers, _ = node1.GetConfiguration()
	assert.Len(t, servers, 2)
}

func TestNode_LeaderCh(t *testing.T) {
	node, _ := createTestNode(t, 17040, true)
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
	node, _ := createTestNode(t, 17041, true)
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

// TestThreeNode_ClusterRestart tests that a 3-node cluster can restart and reform
// after all nodes are shut down. This verifies:
// 1. NodeID persistence works correctly
// 2. Raft state is persisted and restored
// 3. Cluster can re-elect a leader after restart
func TestThreeNode_ClusterRestart(t *testing.T) {
	// Use fixed data directories that persist across restart
	dataDir1 := t.TempDir()
	dataDir2 := t.TempDir()
	dataDir3 := t.TempDir()

	// Helper to create node with specific data directory
	createNodeWithDir := func(dataDir string, port int, bootstrap bool) (*Node, *kvFSM) {
		fsm := newKVFSM()
		cfg := &Config{
			BindAddr:           fmt.Sprintf("127.0.0.1:%d", port),
			DataDir:            dataDir,
			Bootstrap:          bootstrap,
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
		}

		node, err := NewNode(cfg, fsm)
		require.NoError(t, err)
		return node, fsm
	}

	// === Phase 1: Create initial cluster ===
	t.Log("Phase 1: Creating initial 3-node cluster...")

	node1, fsm1 := createNodeWithDir(dataDir1, 17050, true)
	node2, _ := createNodeWithDir(dataDir2, 17051, false)
	node3, _ := createNodeWithDir(dataDir3, 17052, false)

	// Record node IDs for verification after restart
	nodeID1 := node1.NodeID()
	nodeID2 := node2.NodeID()
	nodeID3 := node3.NodeID()

	t.Logf("Node IDs: node1=%s, node2=%s, node3=%s", nodeID1, nodeID2, nodeID3)

	// Start node1 as bootstrap leader
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	err := node1.Start(ctx)
	cancel()
	require.NoError(t, err)
	require.True(t, node1.IsLeader())

	// Add node2 and node3 to cluster
	err = node1.AddVoter(nodeID2, "127.0.0.1:17051", 0, 5*time.Second)
	require.NoError(t, err)

	err = node1.AddVoter(nodeID3, "127.0.0.1:17052", 0, 5*time.Second)
	require.NoError(t, err)

	// Write some data
	for i := 0; i < 5; i++ {
		cmd := &Command{
			Type: CommandTypeSet,
			Key:  fmt.Sprintf("key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		_, err := node1.ApplyCommand(cmd, 5*time.Second)
		require.NoError(t, err)
	}

	// Verify data on leader
	assert.Equal(t, 5, fsm1.Len())
	t.Log("Phase 1 complete: Cluster formed with 5 keys")

	// === Phase 2: Shutdown all nodes ===
	t.Log("Phase 2: Shutting down all nodes...")

	node1.Close()
	node2.Close()
	node3.Close()

	// Wait for cleanup
	time.Sleep(500 * time.Millisecond)
	t.Log("Phase 2 complete: All nodes shut down")

	// === Phase 3: Restart all nodes ===
	t.Log("Phase 3: Restarting all nodes...")

	// Recreate nodes with same data directories (Bootstrap=false for all since cluster exists)
	node1Restart, fsm1Restart := createNodeWithDir(dataDir1, 17050, false)
	defer node1Restart.Close()

	node2Restart, fsm2Restart := createNodeWithDir(dataDir2, 17051, false)
	defer node2Restart.Close()

	node3Restart, fsm3Restart := createNodeWithDir(dataDir3, 17052, false)
	defer node3Restart.Close()

	// Verify node IDs are preserved
	assert.Equal(t, nodeID1, node1Restart.NodeID(), "node1 ID should be preserved after restart")
	assert.Equal(t, nodeID2, node2Restart.NodeID(), "node2 ID should be preserved after restart")
	assert.Equal(t, nodeID3, node3Restart.NodeID(), "node3 ID should be preserved after restart")
	t.Logf("Node IDs preserved: node1=%s, node2=%s, node3=%s",
		node1Restart.NodeID(), node2Restart.NodeID(), node3Restart.NodeID())

	// Wait for leader election
	nodes := []*Node{node1Restart, node2Restart, node3Restart}
	leader := waitForLeader(t, nodes, 10*time.Second)
	require.NotNil(t, leader)
	t.Logf("Phase 3 complete: New leader elected: %s", leader.NodeID())

	// === Phase 4: Verify cluster state ===
	t.Log("Phase 4: Verifying cluster state...")

	// Verify cluster configuration
	servers, err := leader.GetConfiguration()
	require.NoError(t, err)
	assert.Len(t, servers, 3, "cluster should have 3 nodes")

	// Wait for FSM to restore from logs
	time.Sleep(500 * time.Millisecond)

	// Check data is restored on the leader's FSM
	var leaderFSM *kvFSM
	switch leader.NodeID() {
	case nodeID1:
		leaderFSM = fsm1Restart
	case nodeID2:
		leaderFSM = fsm2Restart
	case nodeID3:
		leaderFSM = fsm3Restart
	}

	// Note: FSM state is restored from Raft logs, should have the 5 keys
	t.Logf("Leader FSM has %d keys", leaderFSM.Len())

	// Verify we can write new data
	cmd := &Command{
		Type: CommandTypeSet,
		Key:  "after-restart-key",
		Data: []byte("after-restart-value"),
	}
	_, err = leader.ApplyCommand(cmd, 5*time.Second)
	require.NoError(t, err)

	t.Log("Phase 4 complete: Cluster fully operational after restart")
}

func TestSingleNode_ApplyLarge(t *testing.T) {
	node, fsm := createTestNode(t, 17060, true)
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
	node, _ := createTestNode(t, 17061, false) // Not bootstrap
	defer node.Close()

	// Node is not a leader (not bootstrapped)
	_, err := node.ApplyLarge([]byte("data"), time.Second)
	require.Error(t, err)
	assert.True(t, IsNotLeader(err))
}

func TestSingleNode_ApplyLarge_NodeClosed(t *testing.T) {
	node, _ := createTestNode(t, 17062, true)

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
	node, fsm := createTestNode(t, 17063, true)
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
