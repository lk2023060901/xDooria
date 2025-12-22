// pkg/scheduler/job.go
package scheduler

import (
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
)

// JobID 任务唯一标识
type JobID = cron.EntryID

// Job 任务接口
type Job interface {
	// Run 执行任务，返回错误用于判断是否重试
	Run() error
	// Name 返回任务名称
	Name() string
}

// JobFunc 函数类型任务
type JobFunc func() error

// JobInfo 任务信息
type JobInfo struct {
	// ID 任务唯一标识
	ID JobID

	// Name 任务名称
	Name string

	// Spec Cron 表达式
	Spec string

	// LastRun 上次执行时间
	LastRun time.Time

	// NextRun 下次执行时间
	NextRun time.Time

	// RunCount 执行次数
	RunCount int64

	// FailCount 失败次数
	FailCount int64

	// Running 是否正在执行
	Running bool

	// Options 任务选项
	Options JobOptions
}

// jobEntry 内部任务条目
type jobEntry struct {
	id        JobID
	name      string
	spec      string
	job       Job
	fn        JobFunc
	options   JobOptions
	runCount  int64
	failCount int64
	running   atomic.Bool
	lastRun   time.Time
}

// newJobEntry 创建任务条目
func newJobEntry(name, spec string, options JobOptions) *jobEntry {
	return &jobEntry{
		name:    name,
		spec:    spec,
		options: options,
	}
}

// Run 实现 cron.Job 接口
func (e *jobEntry) Run() {
	e.running.Store(true)
	defer e.running.Store(false)

	e.lastRun = time.Now()
	atomic.AddInt64(&e.runCount, 1)

	var err error
	if e.job != nil {
		err = e.job.Run()
	} else if e.fn != nil {
		err = e.fn()
	}

	if err != nil {
		atomic.AddInt64(&e.failCount, 1)
	}
}

// Name 返回任务名称
func (e *jobEntry) Name() string {
	return e.name
}

// Spec 返回 Cron 表达式
func (e *jobEntry) Spec() string {
	return e.spec
}

// RunCount 返回执行次数
func (e *jobEntry) RunCount() int64 {
	return atomic.LoadInt64(&e.runCount)
}

// FailCount 返回失败次数
func (e *jobEntry) FailCount() int64 {
	return atomic.LoadInt64(&e.failCount)
}

// IsRunning 返回是否正在执行
func (e *jobEntry) IsRunning() bool {
	return e.running.Load()
}

// LastRun 返回上次执行时间
func (e *jobEntry) LastRun() time.Time {
	return e.lastRun
}

// Options 返回任务选项
func (e *jobEntry) Options() JobOptions {
	return e.options
}

// SetID 设置任务 ID
func (e *jobEntry) SetID(id JobID) {
	e.id = id
}

// ID 返回任务 ID
func (e *jobEntry) ID() JobID {
	return e.id
}

// JobOption 任务选项函数
type JobOption func(*jobEntry)

// WithMaxRetries 设置最大重试次数
func WithMaxRetries(n int) JobOption {
	return func(e *jobEntry) {
		e.options.MaxRetries = n
	}
}

// WithBackoffStrategy 设置退避策略
func WithBackoffStrategy(strategy BackoffStrategy) JobOption {
	return func(e *jobEntry) {
		e.options.BackoffStrategy = strategy
	}
}

// WithInitialBackoff 设置初始退避时间
func WithInitialBackoff(d time.Duration) JobOption {
	return func(e *jobEntry) {
		e.options.InitialBackoff = d
	}
}

// WithMaxBackoff 设置最大退避时间
func WithMaxBackoff(d time.Duration) JobOption {
	return func(e *jobEntry) {
		e.options.MaxBackoff = d
	}
}

// WithBackoffMultiplier 设置退避乘数
func WithBackoffMultiplier(m float64) JobOption {
	return func(e *jobEntry) {
		e.options.BackoffMultiplier = m
	}
}

// WithJobOptions 设置完整的任务选项
func WithJobOptions(opts JobOptions) JobOption {
	return func(e *jobEntry) {
		e.options = opts
	}
}

// WithNoRetry 禁用重试
func WithNoRetry() JobOption {
	return func(e *jobEntry) {
		e.options.MaxRetries = 0
		e.options.BackoffStrategy = BackoffNone
	}
}
