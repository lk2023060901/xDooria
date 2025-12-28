package bt

import "context"

// Status 节点执行状态
type Status int

const (
	StatusInvalid Status = iota
	StatusSuccess        // 成功
	StatusFailure        // 失败
	StatusRunning        // 运行中
)

func (s Status) String() string {
	switch s {
	case StatusSuccess:
		return "Success"
	case StatusFailure:
		return "Failure"
	case StatusRunning:
		return "Running"
	default:
		return "Invalid"
	}
}

// Node 行为树节点接口
type Node interface {
	// Tick 执行节点
	Tick(ctx context.Context, bb *Blackboard) Status

	// Reset 重置节点状态
	Reset()

	// Name 返回节点名称
	Name() string
}

// BaseNode 基础节点
type BaseNode struct {
	name string
}

func (n *BaseNode) Name() string {
	return n.name
}

func (n *BaseNode) Reset() {}
