package bt

import "context"

// Sequence 顺序节点：按顺序执行子节点，全部成功才成功
type Sequence struct {
	BaseNode
	children []Node
	current  int
}

// NewSequence 创建顺序节点
func NewSequence(name string, children ...Node) *Sequence {
	return &Sequence{
		BaseNode: BaseNode{name: name},
		children: children,
	}
}

func (s *Sequence) Tick(ctx context.Context, bb *Blackboard) Status {
	for s.current < len(s.children) {
		status := s.children[s.current].Tick(ctx, bb)

		switch status {
		case StatusSuccess:
			s.current++
			continue
		case StatusRunning:
			return StatusRunning
		case StatusFailure:
			s.Reset()
			return StatusFailure
		}
	}

	s.Reset()
	return StatusSuccess
}

func (s *Sequence) Reset() {
	s.current = 0
	for _, child := range s.children {
		child.Reset()
	}
}

// Selector 选择节点：按顺序执行子节点，有一个成功就成功
type Selector struct {
	BaseNode
	children []Node
	current  int
}

// NewSelector 创建选择节点
func NewSelector(name string, children ...Node) *Selector {
	return &Selector{
		BaseNode: BaseNode{name: name},
		children: children,
	}
}

func (s *Selector) Tick(ctx context.Context, bb *Blackboard) Status {
	for s.current < len(s.children) {
		status := s.children[s.current].Tick(ctx, bb)

		switch status {
		case StatusSuccess:
			s.Reset()
			return StatusSuccess
		case StatusRunning:
			return StatusRunning
		case StatusFailure:
			s.current++
			continue
		}
	}

	s.Reset()
	return StatusFailure
}

func (s *Selector) Reset() {
	s.current = 0
	for _, child := range s.children {
		child.Reset()
	}
}

// Parallel 并行节点：同时执行所有子节点
type Parallel struct {
	BaseNode
	children       []Node
	successPolicy  int // 需要多少个成功才算成功
	failurePolicy  int // 需要多少个失败才算失败
	successCount   int
	failureCount   int
}

// NewParallel 创建并行节点
func NewParallel(name string, successPolicy, failurePolicy int, children ...Node) *Parallel {
	return &Parallel{
		BaseNode:      BaseNode{name: name},
		children:      children,
		successPolicy: successPolicy,
		failurePolicy: failurePolicy,
	}
}

func (p *Parallel) Tick(ctx context.Context, bb *Blackboard) Status {
	p.successCount = 0
	p.failureCount = 0

	for _, child := range p.children {
		status := child.Tick(ctx, bb)

		switch status {
		case StatusSuccess:
			p.successCount++
		case StatusFailure:
			p.failureCount++
		}
	}

	if p.successCount >= p.successPolicy {
		p.Reset()
		return StatusSuccess
	}

	if p.failureCount >= p.failurePolicy {
		p.Reset()
		return StatusFailure
	}

	return StatusRunning
}

func (p *Parallel) Reset() {
	p.successCount = 0
	p.failureCount = 0
	for _, child := range p.children {
		child.Reset()
	}
}

// RandomSelector 随机选择节点：随机执行一个子节点
type RandomSelector struct {
	BaseNode
	children []Node
	selected int
}

// NewRandomSelector 创建随机选择节点
func NewRandomSelector(name string, children ...Node) *RandomSelector {
	return &RandomSelector{
		BaseNode: BaseNode{name: name},
		children: children,
		selected: -1,
	}
}

func (r *RandomSelector) Tick(ctx context.Context, bb *Blackboard) Status {
	if r.selected == -1 {
		// 首次执行，随机选择一个子节点
		r.selected = bb.data["_random"].(func() int)() % len(r.children)
	}

	status := r.children[r.selected].Tick(ctx, bb)

	if status != StatusRunning {
		r.Reset()
	}

	return status
}

func (r *RandomSelector) Reset() {
	r.selected = -1
	for _, child := range r.children {
		child.Reset()
	}
}
