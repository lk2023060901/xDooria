package actions

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/app/robot/internal/bt"
	"github.com/lk2023060901/xdooria/app/robot/internal/client"
)

// AcceptQuest 接受任务
type AcceptQuest struct {
	bt.BaseNode
	robot   *client.Robot
	questID int32
}

// NewAcceptQuest 创建接受任务节点
func NewAcceptQuest(robot *client.Robot, questID int32) *AcceptQuest {
	return &AcceptQuest{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		questID:  questID,
	}
}

func (a *AcceptQuest) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// TODO: 实现接受任务逻辑
	// 1. 构造接受任务请求
	// 2. 发送请求
	// 3. 等待响应

	bb.Set("current_quest", a.questID)
	bb.Set("quest_status", "accepted")

	return bt.StatusSuccess
}

func (a *AcceptQuest) Name() string {
	return fmt.Sprintf("AcceptQuest(%d)", a.questID)
}

// CompleteQuest 完成任务
type CompleteQuest struct {
	bt.BaseNode
	robot *client.Robot
}

// NewCompleteQuest 创建完成任务节点
func NewCompleteQuest(robot *client.Robot) *CompleteQuest {
	return &CompleteQuest{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (c *CompleteQuest) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 从黑板获取当前任务
	questID, ok := bb.GetInt("current_quest")
	if !ok {
		bb.Set("error", "no current quest")
		return bt.StatusFailure
	}

	// TODO: 实现完成任务逻辑
	// 1. 检查任务是否可完成
	// 2. 提交任务
	// 3. 领取奖励

	bb.Delete("current_quest")
	bb.Delete("quest_status")
	bb.Set("completed_quests", append(bb.Get("completed_quests").([]int), questID))

	return bt.StatusSuccess
}

func (c *CompleteQuest) Name() string {
	return "CompleteQuest"
}

// CheckQuestProgress 检查任务进度
type CheckQuestProgress struct {
	bt.BaseNode
	robot *client.Robot
}

// NewCheckQuestProgress 创建检查任务进度节点
func NewCheckQuestProgress(robot *client.Robot) *CheckQuestProgress {
	return &CheckQuestProgress{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (c *CheckQuestProgress) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 检查任务状态
	status, ok := bb.GetString("quest_status")
	if !ok {
		return bt.StatusFailure
	}

	// 如果任务已完成，返回成功
	if status == "completed" {
		return bt.StatusSuccess
	}

	// 否则返回运行中
	return bt.StatusRunning
}

func (c *CheckQuestProgress) Name() string {
	return "CheckQuestProgress"
}
