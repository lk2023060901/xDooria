package actions

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/app/robot/internal/bt"
	"github.com/lk2023060901/xdooria/app/robot/internal/client"
)

// Wait 等待指定时间
type Wait struct {
	bt.BaseNode
	robot     *client.Robot
	duration  time.Duration
	startTime time.Time
	started   bool
}

// NewWait 创建等待节点
func NewWait(robot *client.Robot, duration time.Duration) *Wait {
	return &Wait{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		duration: duration,
	}
}

func (w *Wait) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	if !w.started {
		w.startTime = time.Now()
		w.started = true
	}

	if time.Since(w.startTime) >= w.duration {
		w.Reset()
		return bt.StatusSuccess
	}

	return bt.StatusRunning
}

func (w *Wait) Reset() {
	w.started = false
}

func (w *Wait) Name() string {
	return fmt.Sprintf("Wait(%s)", w.duration)
}

// Idle 空闲（什么都不做）
type Idle struct {
	bt.BaseNode
	robot *client.Robot
}

// NewIdle 创建空闲节点
func NewIdle(robot *client.Robot) *Idle {
	return &Idle{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (i *Idle) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 什么都不做，直接返回成功
	return bt.StatusSuccess
}

func (i *Idle) Name() string {
	return "Idle"
}
