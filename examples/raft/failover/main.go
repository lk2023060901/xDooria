// examples/raft/failover/main.go
// 故障转移示例：演示 Leader 宕机后的自动选举和恢复
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	hraft "github.com/hashicorp/raft"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/raft"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// kvStore 简单的内存 KV 存储
type kvStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func newKVStore() *kvStore {
	return &kvStore{data: make(map[string]string)}
}

func (s *kvStore) Apply(log *hraft.Log) interface{} {
	cmd, err := raft.DecodeCommand(log.Data)
	if err != nil {
		return raft.NewApplyResult(nil, err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch cmd.Type {
	case raft.CommandTypeSet:
		s.data[cmd.Key] = string(cmd.Data)
		return raft.NewApplyResult(cmd.Key, nil)
	case raft.CommandTypeDelete:
		delete(s.data, cmd.Key)
		return raft.NewApplyResult(cmd.Key, nil)
	default:
		return raft.NewApplyResult(nil, fmt.Errorf("unknown command type"))
	}
}

func (s *kvStore) Snapshot() (hraft.FSMSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, _ := json.Marshal(s.data)
	return raft.NewSimpleFSMSnapshot(data), nil
}

func (s *kvStore) Restore(snapshot io.ReadCloser) error {
	defer snapshot.Close()
	data, _ := io.ReadAll(snapshot)
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Unmarshal(data, &s.data)
}

func (s *kvStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.data)
}

// createNode 创建一个 Raft 节点
// NodeID 会自动生成并持久化到 data_dir/node-id 文件
func createNode(log logger.Logger, dataSubDir string, port int) (*raft.Node, *kvStore, error) {
	store := newKVStore()
	cfg := &raft.Config{
		BindAddr:           fmt.Sprintf("127.0.0.1:%d", port),
		DataDir:            fmt.Sprintf("./raft-failover-data/%s", dataSubDir),
		ExpectNodes:        0, // 跳过 Serf，手动管理集群
		HeartbeatTimeout:   200 * time.Millisecond,
		ElectionTimeout:    200 * time.Millisecond,
		CommitTimeout:      10 * time.Millisecond,
		LeaderLeaseTimeout: 100 * time.Millisecond,
		SnapshotInterval:   1 * time.Minute,
		SnapshotThreshold:  100,
		SnapshotRetain:     2,
		MaxAppendEntries:   64,
		TrailingLogs:       128,
		MaxPool:            3,
		LogLevel:           "error",
	}

	node, err := raft.NewNode(cfg, store, raft.WithLogger(log))
	if err != nil {
		return nil, nil, err
	}

	return node, store, nil
}

func main() {
	// 创建 logger
	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "text",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// 清理旧数据
	os.RemoveAll("./raft-failover-data")

	log.Info("=== Raft Failover Demo ===")
	log.Info("This demo shows automatic leader election after leader failure")

	// 创建 3 个节点
	log.Info("Step 1: Creating 3-node cluster...")

	node1, store1, err := createNode(log, "node1", 18001)
	if err != nil {
		log.Error("failed to create node1", "error", err)
		os.Exit(1)
	}

	node2, store2, err := createNode(log, "node2", 18002)
	if err != nil {
		log.Error("failed to create node2", "error", err)
		os.Exit(1)
	}

	node3, store3, err := createNode(log, "node3", 18003)
	if err != nil {
		log.Error("failed to create node3", "error", err)
		os.Exit(1)
	}

	nodes := []*raft.Node{node1, node2, node3}
	stores := []*kvStore{store1, store2, store3}

	// 启动 node1 作为初始 Leader
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := node1.Start(ctx); err != nil {
		log.Error("failed to start node1", "error", err)
		os.Exit(1)
	}
	cancel()

	// Bootstrap node1 作为初始集群
	if err := node1.Bootstrap(); err != nil {
		log.Warn("bootstrap failed", "error", err)
	}

	// 启动 node2 和 node3
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	if err := node2.Start(ctx2); err != nil {
		log.Error("failed to start node2", "error", err)
	}
	cancel2()

	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	if err := node3.Start(ctx3); err != nil {
		log.Error("failed to start node3", "error", err)
	}
	cancel3()

	log.Info("node1 started as leader", "state", node1.State())

	// 添加 node2 和 node3 到集群 (使用自动生成的 NodeID)
	if err := node1.AddVoter(node2.NodeID(), "127.0.0.1:18002", 0, 5*time.Second); err != nil {
		log.Error("failed to add node2", "error", err)
	}
	if err := node1.AddVoter(node3.NodeID(), "127.0.0.1:18003", 0, 5*time.Second); err != nil {
		log.Error("failed to add node3", "error", err)
	}

	log.Info("Step 2: Cluster formed with 3 nodes")

	// 写入一些数据
	log.Info("Step 3: Writing test data...")
	for i := 0; i < 5; i++ {
		cmd := &raft.Command{
			Type: raft.CommandTypeSet,
			Key:  fmt.Sprintf("key-%d", i),
			Data: []byte(fmt.Sprintf("value-%d", i)),
		}
		if _, err := node1.ApplyCommand(cmd, 5*time.Second); err != nil {
			log.Error("failed to apply", "error", err)
		}
	}

	time.Sleep(500 * time.Millisecond) // 等待复制

	log.Info("Data replicated to all nodes",
		"node1_count", store1.Len(),
		"node2_count", store2.Len(),
		"node3_count", store3.Len(),
	)

	// 找到当前 Leader
	var leaderNode *raft.Node
	var leaderIdx int
	for i, n := range nodes {
		if n.IsLeader() {
			leaderNode = n
			leaderIdx = i
			break
		}
	}

	log.Info("Step 4: Simulating leader failure...", "leader", leaderNode.NodeID())

	// 关闭 Leader
	leaderNode.Close()
	log.Info("Leader node closed", "node", leaderNode.NodeID())

	// 等待新 Leader 选举
	log.Info("Step 5: Waiting for new leader election...")

	time.Sleep(2 * time.Second)

	// 找到新 Leader
	var newLeader *raft.Node
	for i, n := range nodes {
		if i == leaderIdx {
			continue // 跳过已关闭的节点
		}
		if n.IsLeader() {
			newLeader = n
			break
		}
	}

	if newLeader != nil {
		log.Info("New leader elected!", "new_leader", newLeader.NodeID())

		// 在新 Leader 上继续写入数据
		log.Info("Step 6: Writing more data on new leader...")
		for i := 5; i < 10; i++ {
			cmd := &raft.Command{
				Type: raft.CommandTypeSet,
				Key:  fmt.Sprintf("key-%d", i),
				Data: []byte(fmt.Sprintf("value-%d-after-failover", i)),
			}
			if _, err := newLeader.ApplyCommand(cmd, 5*time.Second); err != nil {
				log.Error("failed to apply on new leader", "error", err)
			}
		}

		time.Sleep(500 * time.Millisecond)

		// 打印各节点数据量
		for i, s := range stores {
			if i == leaderIdx {
				log.Info("node data (closed)", "node", fmt.Sprintf("node%d", i+1), "count", s.Len())
			} else {
				log.Info("node data", "node", fmt.Sprintf("node%d", i+1), "count", s.Len())
			}
		}
	} else {
		log.Warn("No new leader elected!")
	}

	log.Info("=== Demo Complete ===")
	log.Info("Press Ctrl+C to exit")

	// 定期打印状态
	conc.Go(func() (struct{}, error) {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			for i, n := range nodes {
				if i == leaderIdx {
					continue
				}
				log.Info("node status",
					"node", n.NodeID(),
					"state", n.State(),
					"leader", n.LeaderID(),
				)
			}
		}
		return struct{}{}, nil
	})

	// 等待退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 清理
	for i, n := range nodes {
		if i != leaderIdx {
			n.Close()
		}
	}

	log.Info("Shutdown complete")
}
