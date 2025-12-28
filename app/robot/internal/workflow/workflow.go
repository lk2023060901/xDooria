package workflow

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Status 工作流状态
type Status int

const (
	StatusPending Status = iota
	StatusRunning
	StatusSuccess
	StatusFailure
	StatusRetrying
)

func (s Status) String() string {
	switch s {
	case StatusPending:
		return "Pending"
	case StatusRunning:
		return "Running"
	case StatusSuccess:
		return "Success"
	case StatusFailure:
		return "Failure"
	case StatusRetrying:
		return "Retrying"
	default:
		return "Unknown"
	}
}

// StepFunc 步骤执行函数
type StepFunc func(ctx context.Context, data *Data) error

// Step 工作流步骤
type Step struct {
	Name         string
	Func         StepFunc
	MaxRetries   int           // 最大重试次数
	RetryDelay   time.Duration // 重试延迟
	Timeout      time.Duration // 超时时间
	retryCount   int
	status       Status
	err          error
	startTime    time.Time
	completeTime time.Time
}

// Workflow 工作流
type Workflow struct {
	id      string
	steps   []*Step
	current int
	data    *Data
	logger  logger.Logger
	status  Status
}

// Data 工作流数据（类似黑板）
type Data struct {
	values map[string]interface{}
}

// NewData 创建数据容器
func NewData() *Data {
	return &Data{
		values: make(map[string]interface{}),
	}
}

// Set 设置数据
func (d *Data) Set(key string, value interface{}) {
	d.values[key] = value
}

// Get 获取数据
func (d *Data) Get(key string) (interface{}, bool) {
	val, ok := d.values[key]
	return val, ok
}

// GetString 获取字符串
func (d *Data) GetString(key string) (string, bool) {
	val, ok := d.Get(key)
	if !ok {
		return "", false
	}
	str, ok := val.(string)
	return str, ok
}

// GetInt64 获取 int64
func (d *Data) GetInt64(key string) (int64, bool) {
	val, ok := d.Get(key)
	if !ok {
		return 0, false
	}
	i, ok := val.(int64)
	return i, ok
}

// NewWorkflow 创建工作流
func NewWorkflow(id string, l logger.Logger) *Workflow {
	return &Workflow{
		id:     id,
		steps:  make([]*Step, 0),
		data:   NewData(),
		logger: l.Named(fmt.Sprintf("workflow.%s", id)),
		status: StatusPending,
	}
}

// AddStep 添加步骤
func (w *Workflow) AddStep(name string, fn StepFunc) *Workflow {
	step := &Step{
		Name:       name,
		Func:       fn,
		MaxRetries: 3,
		RetryDelay: time.Second,
		Timeout:    30 * time.Second,
		status:     StatusPending,
	}
	w.steps = append(w.steps, step)
	return w
}

// AddStepWithOptions 添加带配置的步骤
func (w *Workflow) AddStepWithOptions(name string, fn StepFunc, maxRetries int, retryDelay, timeout time.Duration) *Workflow {
	step := &Step{
		Name:       name,
		Func:       fn,
		MaxRetries: maxRetries,
		RetryDelay: retryDelay,
		Timeout:    timeout,
		status:     StatusPending,
	}
	w.steps = append(w.steps, step)
	return w
}

// Run 运行工作流
func (w *Workflow) Run(ctx context.Context) error {
	w.status = StatusRunning
	w.logger.Info("workflow started", "total_steps", len(w.steps))

	for w.current < len(w.steps) {
		step := w.steps[w.current]
		w.logger.Info("executing step",
			"step", step.Name,
			"index", w.current+1,
			"total", len(w.steps),
		)

		// 执行步骤
		err := w.executeStep(ctx, step)
		if err != nil {
			w.logger.Error("step failed",
				"step", step.Name,
				"error", err,
			)

			// 检查是否可以重试
			if step.retryCount < step.MaxRetries {
				step.retryCount++
				step.status = StatusRetrying
				w.logger.Info("retrying step",
					"step", step.Name,
					"retry", step.retryCount,
					"max", step.MaxRetries,
				)

				// 等待后重试
				time.Sleep(step.RetryDelay)
				continue
			}

			// 重试次数耗尽，工作流失败
			step.status = StatusFailure
			step.err = err
			w.status = StatusFailure
			return fmt.Errorf("step %s failed after %d retries: %w", step.Name, step.MaxRetries, err)
		}

		// 步骤成功
		step.status = StatusSuccess
		step.completeTime = time.Now()
		w.logger.Info("step completed",
			"step", step.Name,
			"duration", step.completeTime.Sub(step.startTime),
		)

		// 移动到下一步
		w.current++
	}

	w.status = StatusSuccess
	w.logger.Info("workflow completed successfully")
	return nil
}

// executeStep 执行单个步骤
func (w *Workflow) executeStep(ctx context.Context, step *Step) error {
	step.startTime = time.Now()
	step.status = StatusRunning

	// 创建带超时的 context
	stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
	defer cancel()

	// 在 goroutine 中执行步骤
	errChan := make(chan error, 1)
	go func() {
		errChan <- step.Func(stepCtx, w.data)
	}()

	// 等待完成或超时
	select {
	case err := <-errChan:
		return err
	case <-stepCtx.Done():
		return fmt.Errorf("step timeout after %v", step.Timeout)
	}
}

// GetData 获取工作流数据
func (w *Workflow) GetData() *Data {
	return w.data
}

// GetStatus 获取工作流状态
func (w *Workflow) GetStatus() Status {
	return w.status
}

// GetCurrentStep 获取当前步骤索引
func (w *Workflow) GetCurrentStep() int {
	return w.current
}

// GetSteps 获取所有步骤
func (w *Workflow) GetSteps() []*Step {
	return w.steps
}

// GetStepStatus 获取步骤状态
func (s *Step) GetStatus() Status {
	return s.status
}

// GetError 获取步骤错误
func (s *Step) GetError() error {
	return s.err
}

// Reset 重置工作流
func (w *Workflow) Reset() {
	w.current = 0
	w.status = StatusPending
	w.data = NewData()
	for _, step := range w.steps {
		step.status = StatusPending
		step.retryCount = 0
		step.err = nil
	}
}
