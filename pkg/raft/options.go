// pkg/raft/options.go
package raft

import (
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Option 配置选项
type Option func(*Node)

// WithLogger 设置日志记录器
func WithLogger(l logger.Logger) Option {
	return func(n *Node) {
		n.logger = l
	}
}
