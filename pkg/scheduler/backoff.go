// pkg/scheduler/backoff.go
package scheduler

import (
	"math"
	"time"
)

// Backoff 退避计算器接口
type Backoff interface {
	// Next 计算下一次退避时间
	// attempt 从 1 开始
	Next(attempt int) time.Duration
	// Reset 重置退避状态
	Reset()
}

// NewBackoff 根据策略创建退避计算器
func NewBackoff(opts JobOptions) Backoff {
	switch opts.BackoffStrategy {
	case BackoffFixed:
		return &fixedBackoff{
			interval: opts.InitialBackoff,
		}
	case BackoffExponential:
		return &exponentialBackoff{
			initial:    opts.InitialBackoff,
			max:        opts.MaxBackoff,
			multiplier: opts.BackoffMultiplier,
		}
	default:
		return &noBackoff{}
	}
}

// noBackoff 不退避
type noBackoff struct{}

func (b *noBackoff) Next(attempt int) time.Duration {
	return 0
}

func (b *noBackoff) Reset() {}

// fixedBackoff 固定间隔退避
type fixedBackoff struct {
	interval time.Duration
}

func (b *fixedBackoff) Next(attempt int) time.Duration {
	return b.interval
}

func (b *fixedBackoff) Reset() {}

// exponentialBackoff 指数退避
type exponentialBackoff struct {
	initial    time.Duration
	max        time.Duration
	multiplier float64
}

func (b *exponentialBackoff) Next(attempt int) time.Duration {
	if attempt <= 0 {
		return b.initial
	}

	// 计算退避时间: initial * multiplier^(attempt-1)
	backoff := float64(b.initial) * math.Pow(b.multiplier, float64(attempt-1))

	// 限制最大值
	if backoff > float64(b.max) {
		return b.max
	}

	return time.Duration(backoff)
}

func (b *exponentialBackoff) Reset() {}

// RetryExecutor 带重试的执行器
type RetryExecutor struct {
	options JobOptions
	backoff Backoff
}

// NewRetryExecutor 创建重试执行器
func NewRetryExecutor(opts JobOptions) *RetryExecutor {
	return &RetryExecutor{
		options: opts,
		backoff: NewBackoff(opts),
	}
}

// Execute 执行函数，失败时按策略重试
func (r *RetryExecutor) Execute(fn func() error) error {
	if r.options.MaxRetries <= 0 || r.options.BackoffStrategy == BackoffNone {
		// 不重试，直接执行
		return fn()
	}

	var lastErr error
	for attempt := 0; attempt <= r.options.MaxRetries; attempt++ {
		if attempt > 0 {
			// 等待退避时间
			backoffDuration := r.backoff.Next(attempt)
			if backoffDuration > 0 {
				time.Sleep(backoffDuration)
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
	}

	return lastErr
}

// ExecuteWithCallback 执行函数，失败时按策略重试，每次重试前调用回调
func (r *RetryExecutor) ExecuteWithCallback(fn func() error, onRetry func(attempt int, err error, backoff time.Duration)) error {
	if r.options.MaxRetries <= 0 || r.options.BackoffStrategy == BackoffNone {
		return fn()
	}

	var lastErr error
	for attempt := 0; attempt <= r.options.MaxRetries; attempt++ {
		if attempt > 0 {
			backoffDuration := r.backoff.Next(attempt)
			if onRetry != nil {
				onRetry(attempt, lastErr, backoffDuration)
			}
			if backoffDuration > 0 {
				time.Sleep(backoffDuration)
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}
	}

	return lastErr
}
