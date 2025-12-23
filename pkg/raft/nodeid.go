// pkg/raft/nodeid.go
package raft

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const (
	// nodeIDFileName 节点 ID 持久化文件名
	nodeIDFileName = "node-id"
)

// GenerateNodeID 生成新的节点 ID (UUID v4)
func GenerateNodeID() string {
	return uuid.New().String()
}

// LoadOrGenerateNodeID 加载或生成节点 ID
// 如果数据目录中存在 node-id 文件，则读取；否则生成新 ID 并持久化
func LoadOrGenerateNodeID(dataDir string) (string, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create data dir: %w", err)
	}

	nodeIDPath := filepath.Join(dataDir, nodeIDFileName)

	// 尝试读取已有的节点 ID
	data, err := os.ReadFile(nodeIDPath)
	if err == nil {
		nodeID := strings.TrimSpace(string(data))
		if nodeID != "" {
			return nodeID, nil
		}
	}

	// 生成新的节点 ID
	nodeID := GenerateNodeID()

	// 持久化到文件
	if err := os.WriteFile(nodeIDPath, []byte(nodeID+"\n"), 0644); err != nil {
		return "", fmt.Errorf("failed to persist node ID: %w", err)
	}

	return nodeID, nil
}
