// pkg/raft/fsm.go
package raft

import (
	"encoding/json"
	"io"

	"github.com/hashicorp/raft"
)

// CommandType 命令类型
type CommandType uint8

const (
	// CommandTypeSet 设置命令
	CommandTypeSet CommandType = iota + 1
	// CommandTypeDelete 删除命令
	CommandTypeDelete
	// CommandTypeCustom 自定义命令（由用户定义具体语义）
	CommandTypeCustom
)

// Command 表示一个要应用到 FSM 的命令
type Command struct {
	Type CommandType `json:"type"`
	Key  string      `json:"key,omitempty"`
	Data []byte      `json:"data,omitempty"`
}

// EncodeCommand 编码命令
func EncodeCommand(cmd *Command) ([]byte, error) {
	return json.Marshal(cmd)
}

// DecodeCommand 解码命令
func DecodeCommand(data []byte) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		return nil, err
	}
	return &cmd, nil
}

// FSM 状态机接口
// 用户需要实现这个接口来定义业务逻辑
type FSM interface {
	// Apply 应用日志条目到状态机
	// 返回的 interface{} 会作为 Apply 的结果返回给调用者
	Apply(log *raft.Log) interface{}

	// Snapshot 创建状态机快照
	Snapshot() (raft.FSMSnapshot, error)

	// Restore 从快照恢复状态机
	Restore(snapshot io.ReadCloser) error
}

// fsmWrapper 包装用户 FSM 以适配 HashiCorp Raft 接口
type fsmWrapper struct {
	fsm FSM
}

// newFSMWrapper 创建 FSM 包装器
func newFSMWrapper(fsm FSM) *fsmWrapper {
	return &fsmWrapper{fsm: fsm}
}

// Apply 实现 raft.FSM 接口
func (f *fsmWrapper) Apply(log *raft.Log) interface{} {
	return f.fsm.Apply(log)
}

// Snapshot 实现 raft.FSM 接口
func (f *fsmWrapper) Snapshot() (raft.FSMSnapshot, error) {
	return f.fsm.Snapshot()
}

// Restore 实现 raft.FSM 接口
func (f *fsmWrapper) Restore(snapshot io.ReadCloser) error {
	return f.fsm.Restore(snapshot)
}

// ApplyResult Apply 操作的结果
type ApplyResult struct {
	Data  interface{}
	Error error
}

// NewApplyResult 创建应用结果
func NewApplyResult(data interface{}, err error) *ApplyResult {
	return &ApplyResult{
		Data:  data,
		Error: err,
	}
}

// SimpleFSMSnapshot 简单的快照实现
type SimpleFSMSnapshot struct {
	data []byte
}

// NewSimpleFSMSnapshot 创建简单快照
func NewSimpleFSMSnapshot(data []byte) *SimpleFSMSnapshot {
	return &SimpleFSMSnapshot{data: data}
}

// Persist 实现 raft.FSMSnapshot 接口
func (s *SimpleFSMSnapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := sink.Write(s.data); err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// Release 实现 raft.FSMSnapshot 接口
func (s *SimpleFSMSnapshot) Release() {}
