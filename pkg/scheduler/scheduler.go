// pkg/scheduler/scheduler.go
package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/robfig/cron/v3"
)

// Scheduler 调度器
type Scheduler struct {
	cron    *cron.Cron
	config  *Config
	logger  logger.Logger
	pool    *conc.Pool[any]
	jobs    map[JobID]*jobEntry
	jobsMu  sync.RWMutex
	running bool
	runMu   sync.RWMutex
}

// New 创建调度器
func New(cfg *Config, opts ...SchedulerOption) (*Scheduler, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// 解析时区
	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone %s: %w", cfg.Timezone, err)
	}

	// 构建 cron 选项
	cronOpts := []cron.Option{
		cron.WithLocation(loc),
	}

	// 秒级精度
	if cfg.WithSeconds {
		cronOpts = append(cronOpts, cron.WithSeconds())
	}

	s := &Scheduler{
		cron:   cron.New(cronOpts...),
		config: cfg,
		logger: logger.Noop(),
		pool:   conc.NewDefaultPool[any](),
		jobs:   make(map[JobID]*jobEntry),
	}

	// 应用选项
	for _, opt := range opts {
		opt(s)
	}

	return s, nil
}

// SchedulerOption 调度器选项
type SchedulerOption func(*Scheduler)

// WithLogger 设置日志记录器
func WithLogger(l logger.Logger) SchedulerOption {
	return func(s *Scheduler) {
		if l != nil {
			s.logger = l
		}
	}
}

// WithPool 设置协程池
func WithPool(pool *conc.Pool[any]) SchedulerOption {
	return func(s *Scheduler) {
		if pool != nil {
			s.pool = pool
		}
	}
}

// AddJob 添加任务
func (s *Scheduler) AddJob(name, spec string, job Job, opts ...JobOption) (JobID, error) {
	entry := newJobEntry(name, spec, s.config.DefaultJobOptions)
	entry.job = job

	// 应用任务选项
	for _, opt := range opts {
		opt(entry)
	}

	return s.addEntry(spec, entry)
}

// AddFunc 添加函数任务
func (s *Scheduler) AddFunc(name, spec string, fn JobFunc, opts ...JobOption) (JobID, error) {
	entry := newJobEntry(name, spec, s.config.DefaultJobOptions)
	entry.fn = fn

	// 应用任务选项
	for _, opt := range opts {
		opt(entry)
	}

	return s.addEntry(spec, entry)
}

// addEntry 添加任务条目到调度器
func (s *Scheduler) addEntry(spec string, entry *jobEntry) (JobID, error) {
	// 包装任务执行
	wrappedJob := s.wrapJob(entry)

	// 添加到 cron
	id, err := s.cron.AddJob(spec, wrappedJob)
	if err != nil {
		return 0, fmt.Errorf("failed to add job %s: %w", entry.name, err)
	}

	entry.SetID(id)

	// 保存到 jobs map
	s.jobsMu.Lock()
	s.jobs[id] = entry
	s.jobsMu.Unlock()

	s.logger.Info("job added",
		"job_id", id,
		"job_name", entry.name,
		"spec", spec,
	)

	return id, nil
}

// wrapJob 包装任务，添加中间件功能
func (s *Scheduler) wrapJob(entry *jobEntry) cron.Job {
	return cron.FuncJob(func() {
		// 跳过正在执行的任务
		if s.config.SkipIfStillRunning && entry.IsRunning() {
			s.logger.Debug("job skipped, still running",
				"job_id", entry.ID(),
				"job_name", entry.Name(),
			)
			return
		}

		entry.running.Store(true)
		defer entry.running.Store(false)

		entry.lastRun = time.Now()

		// 日志记录
		if s.config.Middleware.Logging {
			s.logger.Info("job started",
				"job_id", entry.ID(),
				"job_name", entry.Name(),
			)
		}

		startTime := time.Now()
		var jobErr error

		// 统计和日志记录（放在 defer 中确保 panic 后也能执行）
		defer func() {
			duration := time.Since(startTime)

			// 更新统计
			entry.runCount++
			if jobErr != nil {
				entry.failCount++
			}

			// 日志记录
			if s.config.Middleware.Logging {
				if jobErr != nil {
					s.logger.Error("job failed",
						"job_id", entry.ID(),
						"job_name", entry.Name(),
						"duration", duration,
						"error", jobErr,
					)
				} else {
					s.logger.Info("job completed",
						"job_id", entry.ID(),
						"job_name", entry.Name(),
						"duration", duration,
					)
				}
			}
		}()

		// panic 恢复
		if s.config.Middleware.Recovery {
			defer func() {
				if r := recover(); r != nil {
					jobErr = fmt.Errorf("job panicked: %v", r)
					s.logger.Error("job panicked",
						"job_id", entry.ID(),
						"job_name", entry.Name(),
						"panic", r,
					)
				}
			}()
		}

		// 执行任务（带重试）
		executor := NewRetryExecutor(entry.options)
		jobErr = executor.ExecuteWithCallback(
			func() error {
				if entry.job != nil {
					return entry.job.Run()
				}
				if entry.fn != nil {
					return entry.fn()
				}
				return nil
			},
			func(attempt int, err error, backoff time.Duration) {
				s.logger.Warn("job retry",
					"job_id", entry.ID(),
					"job_name", entry.Name(),
					"attempt", attempt,
					"error", err,
					"backoff", backoff,
				)
			},
		)
	})
}

// RemoveJob 移除任务
func (s *Scheduler) RemoveJob(id JobID) {
	s.cron.Remove(id)

	s.jobsMu.Lock()
	entry, exists := s.jobs[id]
	delete(s.jobs, id)
	s.jobsMu.Unlock()

	if exists {
		s.logger.Info("job removed",
			"job_id", id,
			"job_name", entry.name,
		)
	}
}

// Start 启动调度器
func (s *Scheduler) Start() {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	if s.running {
		return
	}

	s.cron.Start()
	s.running = true

	s.logger.Info("scheduler started")
}

// Stop 停止调度器
func (s *Scheduler) Stop() context.Context {
	s.runMu.Lock()
	defer s.runMu.Unlock()

	if !s.running {
		return context.Background()
	}

	ctx := s.cron.Stop()
	s.running = false

	s.logger.Info("scheduler stopped")

	return ctx
}

// IsRunning 返回调度器是否正在运行
func (s *Scheduler) IsRunning() bool {
	s.runMu.RLock()
	defer s.runMu.RUnlock()
	return s.running
}

// GetJob 获取任务信息
func (s *Scheduler) GetJob(id JobID) (*JobInfo, bool) {
	s.jobsMu.RLock()
	entry, exists := s.jobs[id]
	s.jobsMu.RUnlock()

	if !exists {
		return nil, false
	}

	// 获取 cron entry 信息
	cronEntry := s.cron.Entry(id)

	return &JobInfo{
		ID:        id,
		Name:      entry.name,
		Spec:      entry.spec,
		LastRun:   entry.lastRun,
		NextRun:   cronEntry.Next,
		RunCount:  entry.runCount,
		FailCount: entry.failCount,
		Running:   entry.IsRunning(),
		Options:   entry.options,
	}, true
}

// ListJobs 列出所有任务
func (s *Scheduler) ListJobs() []*JobInfo {
	s.jobsMu.RLock()
	defer s.jobsMu.RUnlock()

	jobs := make([]*JobInfo, 0, len(s.jobs))
	for id, entry := range s.jobs {
		cronEntry := s.cron.Entry(id)
		jobs = append(jobs, &JobInfo{
			ID:        id,
			Name:      entry.name,
			Spec:      entry.spec,
			LastRun:   entry.lastRun,
			NextRun:   cronEntry.Next,
			RunCount:  entry.runCount,
			FailCount: entry.failCount,
			Running:   entry.IsRunning(),
			Options:   entry.options,
		})
	}

	return jobs
}

// RunNow 立即执行任务（不影响调度）
func (s *Scheduler) RunNow(id JobID) error {
	s.jobsMu.RLock()
	entry, exists := s.jobs[id]
	s.jobsMu.RUnlock()

	if !exists {
		return fmt.Errorf("job %d not found", id)
	}

	// 使用协程池执行
	s.pool.Submit(func() (any, error) {
		s.wrapJob(entry).Run()
		return nil, nil
	})

	return nil
}

// Entries 返回底层 cron entries（用于调试）
func (s *Scheduler) Entries() []cron.Entry {
	return s.cron.Entries()
}

// Release 释放调度器资源
func (s *Scheduler) Release() {
	s.Stop()
	s.pool.Release()
}
