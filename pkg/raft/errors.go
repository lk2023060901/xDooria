// pkg/raft/errors.go
package raft

import (
	"errors"
)

// 定义 Raft 相关错误
var (
	// ErrNotLeader 当前节点不是 Leader
	ErrNotLeader = errors.New("raft: not leader")

	// ErrLeadershipLost Leader 身份丢失
	ErrLeadershipLost = errors.New("raft: leadership lost")

	// ErrNodeClosed 节点已关闭
	ErrNodeClosed = errors.New("raft: node closed")

	// ErrNodeNotReady 节点未就绪
	ErrNodeNotReady = errors.New("raft: node not ready")

	// ErrApplyTimeout 应用命令超时
	ErrApplyTimeout = errors.New("raft: apply timeout")

	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("raft: invalid config")

	// ErrNoLeader 集群没有 Leader
	ErrNoLeader = errors.New("raft: no leader")

	// ErrSnapshotFailed 快照失败
	ErrSnapshotFailed = errors.New("raft: snapshot failed")

	// ErrRestoreFailed 恢复快照失败
	ErrRestoreFailed = errors.New("raft: restore failed")

	// ErrInvalidCommand 无效命令
	ErrInvalidCommand = errors.New("raft: invalid command")

	// ErrAlreadyBootstrapped 集群已启动
	ErrAlreadyBootstrapped = errors.New("raft: cluster already bootstrapped")

	// ErrMembershipChangePending 成员变更进行中
	ErrMembershipChangePending = errors.New("raft: membership change pending")

	// ErrServerNotFound 服务器未找到
	ErrServerNotFound = errors.New("raft: server not found")
)

// IsNotLeader 检查是否是非 Leader 错误
func IsNotLeader(err error) bool {
	return errors.Is(err, ErrNotLeader)
}

// IsLeadershipLost 检查是否是 Leader 身份丢失错误
func IsLeadershipLost(err error) bool {
	return errors.Is(err, ErrLeadershipLost)
}

// IsNodeClosed 检查是否是节点关闭错误
func IsNodeClosed(err error) bool {
	return errors.Is(err, ErrNodeClosed)
}

// IsRetryable 检查错误是否可重试
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// 非 Leader 错误可以重试（找到新 Leader）
	if IsNotLeader(err) || IsLeadershipLost(err) {
		return true
	}

	// 超时错误可以重试
	if errors.Is(err, ErrApplyTimeout) {
		return true
	}

	return false
}
