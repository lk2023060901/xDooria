package actions

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/lk2023060901/xdooria/app/robot/internal/bt"
	"github.com/lk2023060901/xdooria/app/robot/internal/client"
)

// FindMonster 寻找怪物
type FindMonster struct {
	bt.BaseNode
	robot      *client.Robot
	monsterType string
}

// NewFindMonster 创建寻找怪物节点
func NewFindMonster(robot *client.Robot, monsterType string) *FindMonster {
	return &FindMonster{
		BaseNode:   bt.BaseNode{},
		robot:      robot,
		monsterType: monsterType,
	}
}

func (f *FindMonster) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// TODO: 实现寻找怪物逻辑
	// 1. 查询周围怪物
	// 2. 筛选目标类型
	// 3. 选择最近的怪物

	// 模拟找到怪物
	monsterID := rand.Int63n(10000) + 1000
	bb.Set("target_monster", monsterID)
	bb.Set("target_monster_type", f.monsterType)

	return bt.StatusSuccess
}

func (f *FindMonster) Name() string {
	return fmt.Sprintf("FindMonster(%s)", f.monsterType)
}

// AttackMonster 攻击怪物
type AttackMonster struct {
	bt.BaseNode
	robot *client.Robot
}

// NewAttackMonster 创建攻击怪物节点
func NewAttackMonster(robot *client.Robot) *AttackMonster {
	return &AttackMonster{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (a *AttackMonster) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 从黑板获取目标怪物
	monsterID, ok := bb.GetInt64("target_monster")
	if !ok {
		bb.Set("error", "no target monster")
		return bt.StatusFailure
	}

	// TODO: 实现攻击逻辑
	// 1. 构造攻击请求
	// 2. 发送攻击
	// 3. 等待结果

	// 模拟攻击
	_ = monsterID
	bb.Set("combat_status", "attacking")

	return bt.StatusRunning
}

func (a *AttackMonster) Name() string {
	return "AttackMonster"
}

// CheckMonsterDead 检查怪物是否死亡
type CheckMonsterDead struct {
	bt.BaseNode
	robot *client.Robot
}

// NewCheckMonsterDead 创建检查怪物死亡节点
func NewCheckMonsterDead(robot *client.Robot) *CheckMonsterDead {
	return &CheckMonsterDead{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (c *CheckMonsterDead) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// TODO: 实现检查怪物状态逻辑
	// 1. 查询怪物信息
	// 2. 检查 HP 是否为 0

	// 模拟检查（随机结果）
	if rand.Float32() < 0.3 { // 30% 概率怪物死亡
		bb.Delete("target_monster")
		bb.Delete("combat_status")
		return bt.StatusSuccess
	}

	return bt.StatusFailure
}

func (c *CheckMonsterDead) Name() string {
	return "CheckMonsterDead"
}

// PickupLoot 拾取掉落
type PickupLoot struct {
	bt.BaseNode
	robot *client.Robot
}

// NewPickupLoot 创建拾取掉落节点
func NewPickupLoot(robot *client.Robot) *PickupLoot {
	return &PickupLoot{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (p *PickupLoot) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// TODO: 实现拾取掉落逻辑
	// 1. 查询周围掉落物
	// 2. 拾取所有物品

	// 模拟拾取
	lootCount, _ := bb.GetInt("loot_count")
	bb.Set("loot_count", lootCount+1)

	return bt.StatusSuccess
}

func (p *PickupLoot) Name() string {
	return "PickupLoot"
}

// UseSkill 使用技能
type UseSkill struct {
	bt.BaseNode
	robot   *client.Robot
	skillID int32
}

// NewUseSkill 创建使用技能节点
func NewUseSkill(robot *client.Robot, skillID int32) *UseSkill {
	return &UseSkill{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		skillID:  skillID,
	}
}

func (u *UseSkill) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// TODO: 实现使用技能逻辑
	// 1. 检查技能CD
	// 2. 检查MP是否足够
	// 3. 发送使用技能请求

	// 模拟使用技能
	_ = u.skillID

	return bt.StatusSuccess
}

func (u *UseSkill) Name() string {
	return fmt.Sprintf("UseSkill(%d)", u.skillID)
}
