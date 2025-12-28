package bt

import (
	"context"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Tree 行为树
type Tree struct {
	root       Node
	blackboard *Blackboard
	logger     logger.Logger
	interval   time.Duration
	running    bool
}

// NewTree 创建行为树
func NewTree(root Node, interval time.Duration, l logger.Logger) *Tree {
	return &Tree{
		root:       root,
		blackboard: NewBlackboard(),
		logger:     l.Named("bt"),
		interval:   interval,
	}
}

// Start 启动行为树
func (t *Tree) Start(ctx context.Context) {
	t.running = true
	ticker := time.NewTicker(t.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.logger.Info("behavior tree stopped")
			return

		case <-ticker.C:
			if !t.running {
				return
			}

			// 执行行为树
			status := t.root.Tick(ctx, t.blackboard)

			t.logger.Debug("tick completed",
				"status", status.String(),
			)

			// 如果根节点完成（成功或失败），重置树
			if status != StatusRunning {
				t.root.Reset()
			}
		}
	}
}

// Stop 停止行为树
func (t *Tree) Stop() {
	t.running = false
}

// GetBlackboard 获取黑板
func (t *Tree) GetBlackboard() *Blackboard {
	return t.blackboard
}

// Reset 重置行为树
func (t *Tree) Reset() {
	t.root.Reset()
	t.blackboard.Clear()
}
