package actions

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lk2023060901/xdooria/app/robot/internal/bt"
	"github.com/lk2023060901/xdooria/app/robot/internal/client"
)

// MoveTo 移动到指定位置
type MoveTo struct {
	bt.BaseNode
	robot *client.Robot
	x, y, z float32
}

// NewMoveTo 创建移动到指定位置节点
func NewMoveTo(robot *client.Robot, x, y, z float32) *MoveTo {
	return &MoveTo{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		x:        x,
		y:        y,
		z:        z,
	}
}

func (m *MoveTo) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// TODO: 实现移动逻辑
	// 1. 构造移动请求
	// 2. 发送移动指令
	// 3. 等待到达

	// 更新黑板中的位置
	bb.Set("position_x", m.x)
	bb.Set("position_y", m.y)
	bb.Set("position_z", m.z)

	return bt.StatusSuccess
}

func (m *MoveTo) Name() string {
	return fmt.Sprintf("MoveTo(%.1f,%.1f,%.1f)", m.x, m.y, m.z)
}

// MoveRandom 随机移动
type MoveRandom struct {
	bt.BaseNode
	robot  *client.Robot
	radius float32
}

// NewMoveRandom 创建随机移动节点
func NewMoveRandom(robot *client.Robot, radius float32) *MoveRandom {
	return &MoveRandom{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		radius:   radius,
	}
}

func (m *MoveRandom) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 获取当前位置
	currentX, _ := bb.Get("position_x")
	currentZ, _ := bb.Get("position_z")

	x := currentX.(float32)
	z := currentZ.(float32)

	// 生成随机偏移
	offsetX := (rand.Float32()*2 - 1) * m.radius
	offsetZ := (rand.Float32()*2 - 1) * m.radius

	newX := x + offsetX
	newZ := z + offsetZ

	// TODO: 发送移动请求

	// 更新位置
	bb.Set("position_x", newX)
	bb.Set("position_z", newZ)

	return bt.StatusSuccess
}

func (m *MoveRandom) Name() string {
	return fmt.Sprintf("MoveRandom(%.1f)", m.radius)
}

// MoveToTarget 移动到目标
type MoveToTarget struct {
	bt.BaseNode
	robot      *client.Robot
	targetKey  string  // 黑板中目标的键名
	range_     float32 // 停止距离
}

// NewMoveToTarget 创建移动到目标节点
func NewMoveToTarget(robot *client.Robot, targetKey string, range_ float32) *MoveToTarget {
	return &MoveToTarget{
		BaseNode:  bt.BaseNode{},
		robot:     robot,
		targetKey: targetKey,
		range_:    range_,
	}
}

func (m *MoveToTarget) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 从黑板获取目标
	target, ok := bb.Get(m.targetKey)
	if !ok {
		bb.Set("error", fmt.Sprintf("target %s not found", m.targetKey))
		return bt.StatusFailure
	}

	// TODO: 实现移动到目标逻辑
	// 1. 计算到目标的距离
	// 2. 如果在范围内，返回成功
	// 3. 否则移动靠近目标

	_ = target

	return bt.StatusSuccess
}

func (m *MoveToTarget) Name() string {
	return fmt.Sprintf("MoveToTarget(%s)", m.targetKey)
}

// Patrol 巡逻
type Patrol struct {
	bt.BaseNode
	robot     *client.Robot
	points    []Position
	current   int
}

// Position 位置
type Position struct {
	X, Y, Z float32
}

// NewPatrol 创建巡逻节点
func NewPatrol(robot *client.Robot, points []Position) *Patrol {
	return &Patrol{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		points:   points,
	}
}

func (p *Patrol) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	if len(p.points) == 0 {
		return bt.StatusFailure
	}

	// 获取当前巡逻点
	target := p.points[p.current]

	// TODO: 移动到巡逻点

	// 更新位置
	bb.Set("position_x", target.X)
	bb.Set("position_y", target.Y)
	bb.Set("position_z", target.Z)

	// 移动到下一个巡逻点
	p.current = (p.current + 1) % len(p.points)

	return bt.StatusSuccess
}

func (p *Patrol) Name() string {
	return fmt.Sprintf("Patrol(%d points)", len(p.points))
}
