package bt

import (
	"context"
	"time"
)

// Inverter 反转节点：反转子节点的结果
type Inverter struct {
	BaseNode
	child Node
}

// NewInverter 创建反转节点
func NewInverter(name string, child Node) *Inverter {
	return &Inverter{
		BaseNode: BaseNode{name: name},
		child:    child,
	}
}

func (i *Inverter) Tick(ctx context.Context, bb *Blackboard) Status {
	status := i.child.Tick(ctx, bb)

	switch status {
	case StatusSuccess:
		return StatusFailure
	case StatusFailure:
		return StatusSuccess
	default:
		return status
	}
}

func (i *Inverter) Reset() {
	i.child.Reset()
}

// Repeater 重复节点：重复执行子节点指定次数
type Repeater struct {
	BaseNode
	child     Node
	maxCount  int // 最大重复次数，-1 表示无限
	count     int // 当前已重复次数
}

// NewRepeater 创建重复节点
func NewRepeater(name string, maxCount int, child Node) *Repeater {
	return &Repeater{
		BaseNode: BaseNode{name: name},
		child:    child,
		maxCount: maxCount,
	}
}

func (r *Repeater) Tick(ctx context.Context, bb *Blackboard) Status {
	if r.maxCount != -1 && r.count >= r.maxCount {
		r.Reset()
		return StatusSuccess
	}

	status := r.child.Tick(ctx, bb)

	if status != StatusRunning {
		r.count++
		r.child.Reset()

		if r.maxCount != -1 && r.count >= r.maxCount {
			r.Reset()
			return StatusSuccess
		}

		return StatusRunning
	}

	return StatusRunning
}

func (r *Repeater) Reset() {
	r.count = 0
	r.child.Reset()
}

// UntilSuccess 直到成功节点：重复执行直到子节点成功
type UntilSuccess struct {
	BaseNode
	child Node
}

// NewUntilSuccess 创建直到成功节点
func NewUntilSuccess(name string, child Node) *UntilSuccess {
	return &UntilSuccess{
		BaseNode: BaseNode{name: name},
		child:    child,
	}
}

func (u *UntilSuccess) Tick(ctx context.Context, bb *Blackboard) Status {
	status := u.child.Tick(ctx, bb)

	if status == StatusSuccess {
		u.Reset()
		return StatusSuccess
	}

	if status != StatusRunning {
		u.child.Reset()
	}

	return StatusRunning
}

func (u *UntilSuccess) Reset() {
	u.child.Reset()
}

// UntilFailure 直到失败节点：重复执行直到子节点失败
type UntilFailure struct {
	BaseNode
	child Node
}

// NewUntilFailure 创建直到失败节点
func NewUntilFailure(name string, child Node) *UntilFailure {
	return &UntilFailure{
		BaseNode: BaseNode{name: name},
		child:    child,
	}
}

func (u *UntilFailure) Tick(ctx context.Context, bb *Blackboard) Status {
	status := u.child.Tick(ctx, bb)

	if status == StatusFailure {
		u.Reset()
		return StatusSuccess
	}

	if status != StatusRunning {
		u.child.Reset()
	}

	return StatusRunning
}

func (u *UntilFailure) Reset() {
	u.child.Reset()
}

// Delay 延迟节点：延迟指定时间后再执行子节点
type Delay struct {
	BaseNode
	child     Node
	duration  time.Duration
	startTime time.Time
	started   bool
}

// NewDelay 创建延迟节点
func NewDelay(name string, duration time.Duration, child Node) *Delay {
	return &Delay{
		BaseNode: BaseNode{name: name},
		child:    child,
		duration: duration,
	}
}

func (d *Delay) Tick(ctx context.Context, bb *Blackboard) Status {
	if !d.started {
		d.startTime = time.Now()
		d.started = true
	}

	if time.Since(d.startTime) < d.duration {
		return StatusRunning
	}

	status := d.child.Tick(ctx, bb)

	if status != StatusRunning {
		d.Reset()
	}

	return status
}

func (d *Delay) Reset() {
	d.started = false
	d.child.Reset()
}

// Timeout 超时节点：子节点执行超时则失败
type Timeout struct {
	BaseNode
	child     Node
	duration  time.Duration
	startTime time.Time
	started   bool
}

// NewTimeout 创建超时节点
func NewTimeout(name string, duration time.Duration, child Node) *Timeout {
	return &Timeout{
		BaseNode: BaseNode{name: name},
		child:    child,
		duration: duration,
	}
}

func (t *Timeout) Tick(ctx context.Context, bb *Blackboard) Status {
	if !t.started {
		t.startTime = time.Now()
		t.started = true
	}

	if time.Since(t.startTime) >= t.duration {
		t.Reset()
		return StatusFailure
	}

	status := t.child.Tick(ctx, bb)

	if status != StatusRunning {
		t.Reset()
	}

	return status
}

func (t *Timeout) Reset() {
	t.started = false
	t.child.Reset()
}
