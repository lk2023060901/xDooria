// examples/raft/membership/main.go
// 成员变更示例：演示运行时动态添加和移除节点
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
		DataDir:            fmt.Sprintf("./raft-membership-data/%s", dataSubDir),
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

// printConfiguration 打印集群配置
func printConfiguration(log logger.Logger, node *raft.Node) {
	servers, err := node.GetConfiguration()
	if err != nil {
		log.Error("failed to get configuration", "error", err)
		return
	}

	log.Info("current cluster configuration", "count", len(servers))
	for _, s := range servers {
		log.Info("  server", "id", s.ID, "address", s.Address, "suffrage", s.Suffrage)
	}
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
	os.RemoveAll("./raft-membership-data")

	log.Info("=== Raft Dynamic Membership Demo ===")
	log.Info("This demo shows how to add and remove nodes at runtime")

	// Step 1: 创建单节点集群
	log.Info("")
	log.Info("Step 1: Starting with single node cluster...")

	node1, store1, err := createNode(log, "node1", 19001)
	if err != nil {
		log.Error("failed to create node1", "error", err)
		os.Exit(1)
	}
	defer node1.Close()

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

	log.Info("node1 started", "state", node1.State())
	printConfiguration(log, node1)

	// 写入初始数据
	for i := 0; i < 3; i++ {
		cmd := &raft.Command{
			Type: raft.CommandTypeSet,
			Key:  fmt.Sprintf("initial-key-%d", i),
			Data: []byte(fmt.Sprintf("initial-value-%d", i)),
		}
		node1.ApplyCommand(cmd, 5*time.Second)
	}
	log.Info("initial data written", "count", store1.Len())

	// Step 2: 添加 node2
	log.Info("")
	log.Info("Step 2: Adding node2 to cluster...")

	node2, store2, err := createNode(log, "node2", 19002)
	if err != nil {
		log.Error("failed to create node2", "error", err)
		os.Exit(1)
	}
	defer node2.Close()

	// 启动 node2
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	if err := node2.Start(ctx2); err != nil {
		log.Error("failed to start node2", "error", err)
	}
	cancel2()

	if err := node1.AddVoter(node2.NodeID(), "127.0.0.1:19002", 0, 5*time.Second); err != nil {
		log.Error("failed to add node2", "error", err)
	} else {
		log.Info("node2 added successfully", "node_id", node2.NodeID())
	}

	time.Sleep(500 * time.Millisecond)
	printConfiguration(log, node1)
	log.Info("node2 data synced", "count", store2.Len())

	// Step 3: 添加 node3
	log.Info("")
	log.Info("Step 3: Adding node3 to cluster...")

	node3, store3, err := createNode(log, "node3", 19003)
	if err != nil {
		log.Error("failed to create node3", "error", err)
		os.Exit(1)
	}
	defer node3.Close()

	// 启动 node3
	ctx3, cancel3 := context.WithTimeout(context.Background(), 10*time.Second)
	if err := node3.Start(ctx3); err != nil {
		log.Error("failed to start node3", "error", err)
	}
	cancel3()

	if err := node1.AddVoter(node3.NodeID(), "127.0.0.1:19003", 0, 5*time.Second); err != nil {
		log.Error("failed to add node3", "error", err)
	} else {
		log.Info("node3 added successfully", "node_id", node3.NodeID())
	}

	time.Sleep(500 * time.Millisecond)
	printConfiguration(log, node1)
	log.Info("node3 data synced", "count", store3.Len())

	// Step 4: 写入更多数据
	log.Info("")
	log.Info("Step 4: Writing more data to 3-node cluster...")

	for i := 0; i < 3; i++ {
		cmd := &raft.Command{
			Type: raft.CommandTypeSet,
			Key:  fmt.Sprintf("cluster-key-%d", i),
			Data: []byte(fmt.Sprintf("cluster-value-%d", i)),
		}
		node1.ApplyCommand(cmd, 5*time.Second)
	}

	time.Sleep(500 * time.Millisecond)
	log.Info("data after cluster write",
		"node1", store1.Len(),
		"node2", store2.Len(),
		"node3", store3.Len(),
	)

	// Step 5: 添加只读副本 (Nonvoter)
	log.Info("")
	log.Info("Step 5: Adding node4 as non-voter (read replica)...")

	node4, store4, err := createNode(log, "node4", 19004)
	if err != nil {
		log.Error("failed to create node4", "error", err)
		os.Exit(1)
	}
	defer node4.Close()

	// 启动 node4
	ctx4, cancel4 := context.WithTimeout(context.Background(), 10*time.Second)
	if err := node4.Start(ctx4); err != nil {
		log.Error("failed to start node4", "error", err)
	}
	cancel4()

	if err := node1.AddNonvoter(node4.NodeID(), "127.0.0.1:19004", 0, 5*time.Second); err != nil {
		log.Error("failed to add node4 as nonvoter", "error", err)
	} else {
		log.Info("node4 added as non-voter", "node_id", node4.NodeID())
	}

	time.Sleep(500 * time.Millisecond)
	printConfiguration(log, node1)
	log.Info("node4 data synced", "count", store4.Len())

	// Step 6: 移除 node3
	log.Info("")
	log.Info("Step 6: Removing node3 from cluster...")

	if err := node1.RemoveServer(node3.NodeID(), 0, 5*time.Second); err != nil {
		log.Error("failed to remove node3", "error", err)
	} else {
		log.Info("node3 removed successfully", "node_id", node3.NodeID())
	}

	time.Sleep(500 * time.Millisecond)
	printConfiguration(log, node1)

	// Step 7: 验证集群仍然正常工作
	log.Info("")
	log.Info("Step 7: Verifying cluster still works...")

	for i := 0; i < 3; i++ {
		cmd := &raft.Command{
			Type: raft.CommandTypeSet,
			Key:  fmt.Sprintf("after-remove-key-%d", i),
			Data: []byte(fmt.Sprintf("after-remove-value-%d", i)),
		}
		if _, err := node1.ApplyCommand(cmd, 5*time.Second); err != nil {
			log.Error("failed to apply after remove", "error", err)
		}
	}

	time.Sleep(500 * time.Millisecond)
	log.Info("final data count",
		"node1", store1.Len(),
		"node2", store2.Len(),
		"node4", store4.Len(),
	)

	log.Info("")
	log.Info("=== Demo Complete ===")
	log.Info("Summary:")
	log.Info("  - Started with 1 node")
	log.Info("  - Scaled up to 3 voters + 1 non-voter")
	log.Info("  - Removed 1 voter")
	log.Info("  - Final: 2 voters + 1 non-voter")
	log.Info("")
	log.Info("Press Ctrl+C to exit")

	// 定期打印状态
	conc.Go(func() (struct{}, error) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			log.Info("cluster status",
				"leader", node1.LeaderID(),
				"node1_state", node1.State(),
				"node2_state", node2.State(),
				"node4_state", node4.State(),
			)
		}
		return struct{}{}, nil
	})

	// 等待退出
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("Shutdown complete")
}
