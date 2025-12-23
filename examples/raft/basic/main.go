// examples/raft/basic/main.go
// 基础示例：演示 3 节点 Raft 集群的启动、数据写入和读取
package main

import (
	"context"
	"encoding/json"
	"flag"
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

// kvStore 简单的内存 KV 存储，实现 raft.FSM 接口
type kvStore struct {
	mu   sync.RWMutex
	data map[string]string
}

func newKVStore() *kvStore {
	return &kvStore{
		data: make(map[string]string),
	}
}

// Apply 应用日志到状态机
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
		return raft.NewApplyResult(nil, fmt.Errorf("unknown command type: %d", cmd.Type))
	}
}

// Snapshot 创建快照
func (s *kvStore) Snapshot() (hraft.FSMSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(s.data)
	if err != nil {
		return nil, err
	}
	return raft.NewSimpleFSMSnapshot(data), nil
}

// Restore 从快照恢复
func (s *kvStore) Restore(snapshot io.ReadCloser) error {
	defer snapshot.Close()

	data, err := io.ReadAll(snapshot)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	return json.Unmarshal(data, &s.data)
}

// Get 读取值
func (s *kvStore) Get(key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	v, ok := s.data[key]
	return v, ok
}

// All 返回所有数据
func (s *kvStore) All() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]string, len(s.data))
	for k, v := range s.data {
		result[k] = v
	}
	return result
}

func main() {
	// 命令行参数
	bindAddr := flag.String("bind", "127.0.0.1:7000", "绑定地址")
	dataDir := flag.String("data", "./raft-data", "数据目录")
	bootstrap := flag.Bool("bootstrap", false, "是否 bootstrap 集群")
	join := flag.String("join", "", "加入已有集群的 Leader 地址 (node_id=host:port)")
	flag.Parse()

	// 创建 logger
	log, err := logger.New(&logger.Config{
		Level:  "info",
		Format: "text",
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}

	// 创建 FSM
	store := newKVStore()

	// 创建 Raft 节点配置
	// NodeID 会在节点创建时自动生成并持久化到 data_dir/node-id 文件
	cfg := &raft.Config{
		BindAddr:           *bindAddr,
		DataDir:            *dataDir,
		Bootstrap:          *bootstrap,
		HeartbeatTimeout:   1000 * time.Millisecond,
		ElectionTimeout:    1000 * time.Millisecond,
		CommitTimeout:      50 * time.Millisecond,
		LeaderLeaseTimeout: 500 * time.Millisecond,
		SnapshotInterval:   5 * time.Minute,
		SnapshotThreshold:  8192,
		SnapshotRetain:     3,
		MaxAppendEntries:   64,
		TrailingLogs:       10240,
		MaxPool:            3,
		LogLevel:           "warn",
	}

	// 如果指定了 join，解析 peers
	if *join != "" {
		cfg.Peers = []string{*join}
	}

	// 创建 Raft 节点
	node, err := raft.NewNode(cfg, store, raft.WithLogger(log))
	if err != nil {
		log.Error("failed to create raft node", "error", err)
		os.Exit(1)
	}
	defer node.Close()

	// 启动节点
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = node.Start(ctx)
	cancel()
	if err != nil {
		log.Error("failed to start raft node", "error", err)
		os.Exit(1)
	}

	log.Info("raft node started",
		"id", node.NodeID(),
		"state", node.State(),
		"leader", node.LeaderID(),
	)

	// 监听 Leader 变化
	conc.Go(func() (struct{}, error) {
		for isLeader := range node.LeaderCh() {
			if isLeader {
				log.Info("became leader")
			} else {
				log.Info("lost leadership", "new_leader", node.LeaderID())
			}
		}
		return struct{}{}, nil
	})

	// 如果是 Leader，写入一些测试数据
	conc.Go(func() (struct{}, error) {
		// 等待成为 Leader
		time.Sleep(2 * time.Second)

		for i := 0; i < 10; i++ {
			if !node.IsLeader() {
				time.Sleep(time.Second)
				continue
			}

			key := fmt.Sprintf("key-%d", i)
			value := fmt.Sprintf("value-%d @ %s", i, time.Now().Format(time.RFC3339))

			cmd := &raft.Command{
				Type: raft.CommandTypeSet,
				Key:  key,
				Data: []byte(value),
			}

			result, err := node.ApplyCommand(cmd, 5*time.Second)
			if err != nil {
				log.Error("failed to apply command", "error", err)
				continue
			}

			applyResult := result.(*raft.ApplyResult)
			if applyResult.Error != nil {
				log.Error("apply result error", "error", applyResult.Error)
				continue
			}

			log.Info("applied command", "key", key)
			time.Sleep(time.Second)
		}

		log.Info("finished writing test data")
		return struct{}{}, nil
	})

	// 定期打印状态
	conc.Go(func() (struct{}, error) {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			log.Info("node status",
				"state", node.State(),
				"leader", node.LeaderID(),
				"last_index", node.LastIndex(),
				"applied_index", node.AppliedIndex(),
				"data_count", len(store.All()),
			)
		}
		return struct{}{}, nil
	})

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("shutting down...")

	// 打印最终状态
	log.Info("final state", "data", store.All())
}
