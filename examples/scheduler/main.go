// examples/scheduler/main.go
package main

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/scheduler"
)

func main() {
	// 初始化日志
	log, err := logger.New(&logger.Config{
		Level:         logger.InfoLevel,
		Format:        logger.ConsoleFormat,
		EnableConsole: true,
	})
	if err != nil {
		panic(err)
	}

	// 创建调度器配置
	cfg := &scheduler.Config{
		Timezone:           "Asia/Shanghai",
		WithSeconds:        true, // 启用秒级精度
		SkipIfStillRunning: true,
		Middleware: scheduler.MiddlewareConfig{
			Logging:  true,
			Recovery: true,
			Metrics:  false,
		},
		DefaultJobOptions: scheduler.JobOptions{
			MaxRetries:        3,
			BackoffStrategy:   scheduler.BackoffExponential,
			InitialBackoff:    100 * time.Millisecond,
			MaxBackoff:        5 * time.Second,
			BackoffMultiplier: 2.0,
		},
	}

	// 创建调度器
	s, err := scheduler.New(cfg, scheduler.WithLogger(log))
	if err != nil {
		log.Error("failed to create scheduler", "error", err)
		return
	}
	defer s.Release()

	// 示例 1: 简单的定时任务（每 5 秒执行）
	_, err = s.AddFunc("simple-job", "*/5 * * * * *", func() error {
		fmt.Println("[SimpleJob] Hello, World!")
		return nil
	})
	if err != nil {
		log.Error("failed to add simple job", "error", err)
		return
	}

	// 示例 2: 带状态的任务（使用 Job 接口）
	counterJob := &CounterJob{name: "counter-job"}
	_, err = s.AddJob("counter-job", "*/3 * * * * *", counterJob)
	if err != nil {
		log.Error("failed to add counter job", "error", err)
		return
	}

	// 示例 3: 会失败并重试的任务
	var failCount int32
	_, err = s.AddFunc("retry-job", "*/10 * * * * *", func() error {
		count := atomic.AddInt32(&failCount, 1)
		if count%4 != 0 { // 每 4 次成功一次
			fmt.Printf("[RetryJob] Attempt %d failed\n", count)
			return errors.New("simulated failure")
		}
		fmt.Printf("[RetryJob] Attempt %d succeeded!\n", count)
		return nil
	}, scheduler.WithMaxRetries(3), scheduler.WithBackoffStrategy(scheduler.BackoffFixed), scheduler.WithInitialBackoff(500*time.Millisecond))
	if err != nil {
		log.Error("failed to add retry job", "error", err)
		return
	}

	// 示例 4: 不重试的任务
	_, err = s.AddFunc("no-retry-job", "*/15 * * * * *", func() error {
		fmt.Println("[NoRetryJob] This job never retries")
		return nil
	}, scheduler.WithNoRetry())
	if err != nil {
		log.Error("failed to add no-retry job", "error", err)
		return
	}

	// 启动调度器
	s.Start()
	log.Info("scheduler started, press Ctrl+C to stop")

	// 打印所有任务
	jobs := s.ListJobs()
	fmt.Println("\nRegistered jobs:")
	for _, job := range jobs {
		fmt.Printf("  - ID: %d, Name: %s, Spec: %s, NextRun: %s\n",
			job.ID, job.Name, job.Spec, job.NextRun.Format("15:04:05"))
	}
	fmt.Println()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down scheduler...")
	ctx := s.Stop()
	<-ctx.Done()
	log.Info("scheduler stopped")

	// 打印最终统计
	fmt.Println("\nFinal job statistics:")
	for _, job := range s.ListJobs() {
		fmt.Printf("  - %s: RunCount=%d, FailCount=%d\n",
			job.Name, job.RunCount, job.FailCount)
	}
}

// CounterJob 计数器任务示例
type CounterJob struct {
	name  string
	count int32
}

func (j *CounterJob) Run() error {
	count := atomic.AddInt32(&j.count, 1)
	fmt.Printf("[CounterJob] Executed %d times\n", count)
	return nil
}

func (j *CounterJob) Name() string {
	return j.name
}
