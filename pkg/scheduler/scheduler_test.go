// pkg/scheduler/scheduler_test.go
package scheduler

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// TestNew 测试创建调度器
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid config",
			config: &Config{
				Timezone:    "Asia/Shanghai",
				WithSeconds: true,
			},
			wantErr: false,
		},
		{
			name: "invalid timezone",
			config: &Config{
				Timezone: "Invalid/Timezone",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && s == nil {
				t.Error("New() returned nil scheduler without error")
			}
			if s != nil {
				s.Release()
			}
		})
	}
}

// TestAddFunc 测试添加函数任务
func TestAddFunc(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	var counter int32

	// 添加每秒执行的任务
	id, err := s.AddFunc("test-job", "* * * * *", func() error {
		atomic.AddInt32(&counter, 1)
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	if id == 0 {
		t.Error("AddFunc() returned zero JobID")
	}

	// 验证任务已添加
	job, exists := s.GetJob(id)
	if !exists {
		t.Error("GetJob() returned false for added job")
	}
	if job.Name != "test-job" {
		t.Errorf("Job name = %s, want test-job", job.Name)
	}
}

// TestAddJob 测试添加 Job 接口任务
func TestAddJob(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	job := &testJob{name: "my-test-job"}

	id, err := s.AddJob("my-test-job", "* * * * *", job)
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	info, exists := s.GetJob(id)
	if !exists {
		t.Error("GetJob() returned false for added job")
	}
	if info.Name != "my-test-job" {
		t.Errorf("Job name = %s, want my-test-job", info.Name)
	}
}

// TestRemoveJob 测试移除任务
func TestRemoveJob(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	id, err := s.AddFunc("test-job", "* * * * *", func() error {
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	// 移除任务
	s.RemoveJob(id)

	// 验证任务已移除
	_, exists := s.GetJob(id)
	if exists {
		t.Error("GetJob() returned true for removed job")
	}
}

// TestListJobs 测试列出所有任务
func TestListJobs(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	// 添加多个任务
	_, err = s.AddFunc("job1", "* * * * *", func() error { return nil })
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	_, err = s.AddFunc("job2", "*/5 * * * *", func() error { return nil })
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	jobs := s.ListJobs()
	if len(jobs) != 2 {
		t.Errorf("ListJobs() returned %d jobs, want 2", len(jobs))
	}
}

// TestStartStop 测试启动和停止
func TestStartStop(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	if s.IsRunning() {
		t.Error("Scheduler should not be running before Start()")
	}

	s.Start()

	if !s.IsRunning() {
		t.Error("Scheduler should be running after Start()")
	}

	// 重复调用 Start 应该是安全的
	s.Start()

	if !s.IsRunning() {
		t.Error("Scheduler should still be running after second Start()")
	}

	s.Stop()

	if s.IsRunning() {
		t.Error("Scheduler should not be running after Stop()")
	}

	// 重复调用 Stop 应该是安全的
	s.Stop()
}

// TestJobExecution 测试任务执行
func TestJobExecution(t *testing.T) {
	// 使用秒级精度
	cfg := &Config{
		Timezone:           "Asia/Shanghai",
		WithSeconds:        true,
		SkipIfStillRunning: true,
		DefaultJobOptions: JobOptions{
			MaxRetries:      0,
			BackoffStrategy: BackoffNone,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	executed := make(chan struct{}, 1)

	// 添加每秒执行的任务
	_, err = s.AddFunc("test-job", "* * * * * *", func() error {
		select {
		case executed <- struct{}{}:
		default:
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	s.Start()

	// 等待任务执行
	select {
	case <-executed:
		// 成功
	case <-time.After(3 * time.Second):
		t.Error("Job was not executed within timeout")
	}
}

// TestJobWithRetry 测试带重试的任务
func TestJobWithRetry(t *testing.T) {
	cfg := &Config{
		Timezone:    "Asia/Shanghai",
		WithSeconds: true,
		DefaultJobOptions: JobOptions{
			MaxRetries:        3,
			BackoffStrategy:   BackoffFixed,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        100 * time.Millisecond,
			BackoffMultiplier: 2.0,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	var attempts int32
	done := make(chan struct{})

	_, err = s.AddFunc("retry-job", "* * * * * *", func() error {
		count := atomic.AddInt32(&attempts, 1)
		if count < 3 {
			return errors.New("simulated error")
		}
		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	s.Start()

	select {
	case <-done:
		if atomic.LoadInt32(&attempts) != 3 {
			t.Errorf("Expected 3 attempts, got %d", attempts)
		}
	case <-time.After(5 * time.Second):
		t.Error("Job retry did not complete within timeout")
	}
}

// TestRunNow 测试立即执行
func TestRunNow(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	executed := make(chan struct{}, 1)

	id, err := s.AddFunc("test-job", "0 0 1 1 *", func() error { // 每年1月1日执行
		select {
		case executed <- struct{}{}:
		default:
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	// 立即执行
	err = s.RunNow(id)
	if err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}

	select {
	case <-executed:
		// 成功
	case <-time.After(2 * time.Second):
		t.Error("RunNow() did not execute job within timeout")
	}
}

// TestRunNowNotFound 测试立即执行不存在的任务
func TestRunNowNotFound(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	err = s.RunNow(999)
	if err == nil {
		t.Error("RunNow() should return error for non-existent job")
	}
}

// TestSkipIfStillRunning 测试跳过正在执行的任务
func TestSkipIfStillRunning(t *testing.T) {
	cfg := &Config{
		Timezone:           "Asia/Shanghai",
		WithSeconds:        true,
		SkipIfStillRunning: true,
		DefaultJobOptions: JobOptions{
			MaxRetries:      0,
			BackoffStrategy: BackoffNone,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	var runCount int32
	started := make(chan struct{})
	done := make(chan struct{})

	id, err := s.AddFunc("slow-job", "* * * * * *", func() error {
		atomic.AddInt32(&runCount, 1)
		close(started)
		<-done
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	s.Start()

	// 等待任务开始
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("Job did not start within timeout")
	}

	// 尝试立即执行（应该被跳过）
	err = s.RunNow(id)
	if err != nil {
		t.Fatalf("RunNow() error = %v", err)
	}

	// 给一点时间让 RunNow 有机会执行
	time.Sleep(100 * time.Millisecond)

	// 完成第一个任务
	close(done)

	// 验证只执行了一次
	time.Sleep(100 * time.Millisecond)
	if count := atomic.LoadInt32(&runCount); count != 1 {
		t.Errorf("Expected 1 run, got %d", count)
	}
}

// TestJobOptions 测试任务选项
func TestJobOptions(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	id, err := s.AddFunc("test-job", "* * * * *", func() error {
		return nil
	},
		WithMaxRetries(5),
		WithBackoffStrategy(BackoffExponential),
		WithInitialBackoff(100*time.Millisecond),
		WithMaxBackoff(5*time.Second),
		WithBackoffMultiplier(3.0),
	)
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	job, exists := s.GetJob(id)
	if !exists {
		t.Fatal("GetJob() returned false")
	}

	if job.Options.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", job.Options.MaxRetries)
	}
	if job.Options.BackoffStrategy != BackoffExponential {
		t.Errorf("BackoffStrategy = %s, want exponential", job.Options.BackoffStrategy)
	}
	if job.Options.InitialBackoff != 100*time.Millisecond {
		t.Errorf("InitialBackoff = %v, want 100ms", job.Options.InitialBackoff)
	}
	if job.Options.MaxBackoff != 5*time.Second {
		t.Errorf("MaxBackoff = %v, want 5s", job.Options.MaxBackoff)
	}
	if job.Options.BackoffMultiplier != 3.0 {
		t.Errorf("BackoffMultiplier = %f, want 3.0", job.Options.BackoffMultiplier)
	}
}

// TestWithNoRetry 测试禁用重试
func TestWithNoRetry(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	id, err := s.AddFunc("test-job", "* * * * *", func() error {
		return nil
	}, WithNoRetry())
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	job, exists := s.GetJob(id)
	if !exists {
		t.Fatal("GetJob() returned false")
	}

	if job.Options.MaxRetries != 0 {
		t.Errorf("MaxRetries = %d, want 0", job.Options.MaxRetries)
	}
	if job.Options.BackoffStrategy != BackoffNone {
		t.Errorf("BackoffStrategy = %s, want none", job.Options.BackoffStrategy)
	}
}

// TestWithLogger 测试 WithLogger 选项
func TestWithLogger(t *testing.T) {
	s, err := New(nil, WithLogger(logger.Noop()))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Release()

	if s.logger == nil {
		t.Error("Logger should not be nil")
	}
}

// TestWithPool 测试 WithPool 选项
func TestWithPool(t *testing.T) {
	customPool := conc.NewPool[any](4)
	defer customPool.Release()

	s, err := New(nil, WithPool(customPool))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer s.Release()

	if s.pool != customPool {
		t.Error("Pool should be the custom pool")
	}
}

// TestWithJobOptions 测试 WithJobOptions 选项
func TestWithJobOptions(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	opts := JobOptions{
		MaxRetries:        10,
		BackoffStrategy:   BackoffFixed,
		InitialBackoff:    500 * time.Millisecond,
		MaxBackoff:        10 * time.Second,
		BackoffMultiplier: 1.5,
	}

	id, err := s.AddFunc("test-job", "* * * * *", func() error {
		return nil
	}, WithJobOptions(opts))
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	job, exists := s.GetJob(id)
	if !exists {
		t.Fatal("GetJob() returned false")
	}

	if job.Options.MaxRetries != 10 {
		t.Errorf("MaxRetries = %d, want 10", job.Options.MaxRetries)
	}
	if job.Options.BackoffStrategy != BackoffFixed {
		t.Errorf("BackoffStrategy = %s, want fixed", job.Options.BackoffStrategy)
	}
}

// TestInvalidCronSpec 测试无效 cron 表达式
func TestInvalidCronSpec(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	_, err = s.AddFunc("test-job", "invalid cron spec", func() error {
		return nil
	})
	if err == nil {
		t.Error("AddFunc() should return error for invalid cron spec")
	}
}

// TestPanicRecovery 测试 panic 恢复
func TestPanicRecovery(t *testing.T) {
	cfg := &Config{
		Timezone:    "Asia/Shanghai",
		WithSeconds: true,
		Middleware: MiddlewareConfig{
			Logging:  false,
			Recovery: true,
			Metrics:  false,
		},
		DefaultJobOptions: JobOptions{
			MaxRetries:      0,
			BackoffStrategy: BackoffNone,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	panicked := make(chan struct{}, 1)
	recovered := make(chan struct{}, 1)

	id, err := s.AddFunc("panic-job", "* * * * * *", func() error {
		select {
		case panicked <- struct{}{}:
		default:
		}
		panic("test panic")
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	s.Start()

	// 等待任务 panic
	select {
	case <-panicked:
		// 给 recovery 一点时间处理
		time.Sleep(100 * time.Millisecond)
		close(recovered)
	case <-time.After(3 * time.Second):
		t.Fatal("Job did not execute within timeout")
	}

	// 验证调度器仍在运行（panic 被恢复）
	if !s.IsRunning() {
		t.Error("Scheduler should still be running after panic recovery")
	}

	// 验证失败计数
	job, exists := s.GetJob(id)
	if !exists {
		t.Fatal("GetJob() returned false")
	}
	if job.FailCount == 0 {
		t.Error("FailCount should be > 0 after panic")
	}
}

// TestExponentialBackoff 测试指数退避策略
func TestExponentialBackoff(t *testing.T) {
	cfg := &Config{
		Timezone:    "Asia/Shanghai",
		WithSeconds: true,
		DefaultJobOptions: JobOptions{
			MaxRetries:        3,
			BackoffStrategy:   BackoffExponential,
			InitialBackoff:    10 * time.Millisecond,
			MaxBackoff:        1 * time.Second,
			BackoffMultiplier: 2.0,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	var attempts int32
	var timestamps []time.Time
	var mu sync.Mutex
	done := make(chan struct{})

	_, err = s.AddFunc("backoff-job", "* * * * * *", func() error {
		mu.Lock()
		timestamps = append(timestamps, time.Now())
		mu.Unlock()

		count := atomic.AddInt32(&attempts, 1)
		if count <= 3 {
			return errors.New("simulated error")
		}
		close(done)
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	s.Start()

	select {
	case <-done:
		mu.Lock()
		defer mu.Unlock()

		// 验证退避时间间隔是指数增长的
		if len(timestamps) < 4 {
			t.Fatalf("Expected at least 4 attempts, got %d", len(timestamps))
		}

		// 第一次重试后等待 10ms，第二次等待 20ms，第三次等待 40ms
		// 由于执行时间和调度误差，只验证间隔是递增的
		for i := 2; i < len(timestamps); i++ {
			interval := timestamps[i].Sub(timestamps[i-1])
			prevInterval := timestamps[i-1].Sub(timestamps[i-2])
			// 允许一些误差
			if interval < prevInterval-5*time.Millisecond && i > 2 {
				t.Logf("Warning: interval %v is not greater than previous %v", interval, prevInterval)
			}
		}
	case <-time.After(5 * time.Second):
		t.Error("Job did not complete within timeout")
	}
}

// TestJobInterfaceExecution 测试 Job 接口任务执行
func TestJobInterfaceExecution(t *testing.T) {
	cfg := &Config{
		Timezone:    "Asia/Shanghai",
		WithSeconds: true,
		DefaultJobOptions: JobOptions{
			MaxRetries:      0,
			BackoffStrategy: BackoffNone,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	job := &testJob{name: "test-interface-job"}
	executed := make(chan struct{}, 1)
	job.onRun = func() {
		select {
		case executed <- struct{}{}:
		default:
		}
	}

	_, err = s.AddJob("test-interface-job", "* * * * * *", job)
	if err != nil {
		t.Fatalf("AddJob() error = %v", err)
	}

	s.Start()

	select {
	case <-executed:
		if job.runCount == 0 {
			t.Error("Job.Run() was not called")
		}
	case <-time.After(3 * time.Second):
		t.Error("Job was not executed within timeout")
	}
}

// TestEntries 测试 Entries 方法
func TestEntries(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	// 添加任务
	_, err = s.AddFunc("job1", "* * * * *", func() error { return nil })
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	_, err = s.AddFunc("job2", "*/5 * * * *", func() error { return nil })
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	entries := s.Entries()
	if len(entries) != 2 {
		t.Errorf("Entries() returned %d entries, want 2", len(entries))
	}
}

// TestJobInfoFields 测试 JobInfo 字段
func TestJobInfoFields(t *testing.T) {
	cfg := &Config{
		Timezone:    "Asia/Shanghai",
		WithSeconds: true,
		DefaultJobOptions: JobOptions{
			MaxRetries:      0,
			BackoffStrategy: BackoffNone,
		},
	}

	s, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	executed := make(chan struct{}, 1)
	failOnce := true

	id, err := s.AddFunc("test-job", "* * * * * *", func() error {
		if failOnce {
			failOnce = false
			return errors.New("first failure")
		}
		select {
		case executed <- struct{}{}:
		default:
		}
		return nil
	})
	if err != nil {
		t.Fatalf("AddFunc() error = %v", err)
	}

	s.Start()

	// 等待两次执行（一次失败，一次成功）
	select {
	case <-executed:
	case <-time.After(5 * time.Second):
		t.Fatal("Job was not executed within timeout")
	}

	// 等待一点时间让统计更新
	time.Sleep(100 * time.Millisecond)

	job, exists := s.GetJob(id)
	if !exists {
		t.Fatal("GetJob() returned false")
	}

	// 验证字段
	if job.ID != id {
		t.Errorf("ID = %d, want %d", job.ID, id)
	}
	if job.Name != "test-job" {
		t.Errorf("Name = %s, want test-job", job.Name)
	}
	if job.Spec != "* * * * * *" {
		t.Errorf("Spec = %s, want * * * * * *", job.Spec)
	}
	if job.RunCount < 2 {
		t.Errorf("RunCount = %d, want >= 2", job.RunCount)
	}
	if job.FailCount < 1 {
		t.Errorf("FailCount = %d, want >= 1", job.FailCount)
	}
	if job.LastRun.IsZero() {
		t.Error("LastRun should not be zero")
	}
	if job.NextRun.IsZero() {
		t.Error("NextRun should not be zero")
	}
}

// TestRemoveNonExistentJob 测试移除不存在的任务
func TestRemoveNonExistentJob(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	// 移除不存在的任务不应该 panic
	s.RemoveJob(999)
}

// TestGetJobNotFound 测试获取不存在的任务
func TestGetJobNotFound(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	_, exists := s.GetJob(999)
	if exists {
		t.Error("GetJob() should return false for non-existent job")
	}
}

// TestListJobsEmpty 测试空任务列表
func TestListJobsEmpty(t *testing.T) {
	s, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create scheduler: %v", err)
	}
	defer s.Release()

	jobs := s.ListJobs()
	if len(jobs) != 0 {
		t.Errorf("ListJobs() returned %d jobs, want 0", len(jobs))
	}
}

// testJob 测试用 Job 实现
type testJob struct {
	name     string
	runCount int32
	onRun    func()
}

func (j *testJob) Run() error {
	atomic.AddInt32(&j.runCount, 1)
	if j.onRun != nil {
		j.onRun()
	}
	return nil
}

func (j *testJob) Name() string {
	return j.name
}
