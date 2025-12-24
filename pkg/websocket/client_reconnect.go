// pkg/websocket/client_reconnect.go
package websocket

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Reconnector 重连器
type Reconnector struct {
	config *ReconnectConfig
	logger logger.Logger
	client *Client

	// 状态
	retryCount   atomic.Int32
	currentDelay time.Duration

	// 控制
	mu        sync.Mutex
	running   atomic.Bool
	stopCh    chan struct{}
	stopOnce  sync.Once
	resetOnce sync.Once
}

// NewReconnector 创建重连器
func NewReconnector(cfg *ReconnectConfig, client *Client, log logger.Logger) *Reconnector {
	return &Reconnector{
		config:       cfg,
		client:       client,
		logger:       log,
		currentDelay: cfg.InitialDelay,
		stopCh:       make(chan struct{}),
	}
}

// Start 开始重连
func (r *Reconnector) Start(ctx context.Context) error {
	if !r.config.Enable {
		return nil
	}

	// 防止重复启动
	if !r.running.CompareAndSwap(false, true) {
		return nil
	}
	defer r.running.Store(false)

	// 重置 stopOnce 以便下次可以再次停止
	r.stopOnce = sync.Once{}
	r.stopCh = make(chan struct{})

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.stopCh:
			return nil
		case <-r.client.closeCh:
			return ErrConnectionClosed
		default:
		}

		attempt := int(r.retryCount.Add(1))

		// 检查最大重试次数
		if r.config.MaxRetries > 0 && attempt > r.config.MaxRetries {
			if r.logger != nil {
				r.logger.Warn("websocket reconnect max retries exceeded",
					"max_retries", r.config.MaxRetries,
				)
			}
			if r.client.onReconnectFailed != nil {
				r.client.onReconnectFailed(ErrMaxRetriesExceeded)
			}
			return ErrMaxRetriesExceeded
		}

		// 重连中回调
		if r.client.onReconnecting != nil {
			r.client.onReconnecting(attempt)
		}

		if r.logger != nil {
			r.logger.Info("websocket reconnecting",
				"attempt", attempt,
				"delay", r.currentDelay,
			)
		}

		// 等待延迟
		select {
		case <-time.After(r.currentDelay):
		case <-ctx.Done():
			return ctx.Err()
		case <-r.stopCh:
			return nil
		case <-r.client.closeCh:
			return ErrConnectionClosed
		}

		// 尝试连接
		if err := r.client.connect(ctx); err != nil {
			if r.logger != nil {
				r.logger.Warn("websocket reconnect failed",
					"attempt", attempt,
					"error", err,
				)
			}

			// 计算下次延迟
			r.currentDelay = r.calculateDelay()

			// 更新指标
			if r.client.metrics != nil {
				r.client.metrics.OnReconnectAttempt()
			}

			continue
		}

		// 重连成功
		if r.logger != nil {
			r.logger.Info("websocket reconnected",
				"attempt", attempt,
			)
		}

		// 重连成功回调
		if r.client.onReconnected != nil {
			r.client.onReconnected()
		}

		// 更新指标
		if r.client.metrics != nil {
			r.client.metrics.OnReconnected()
		}

		// 重置状态
		r.Reset()

		return nil
	}
}

// Stop 停止重连
func (r *Reconnector) Stop() {
	r.stopOnce.Do(func() {
		close(r.stopCh)
	})
}

// Reset 重置重连状态
func (r *Reconnector) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.retryCount.Store(0)
	r.currentDelay = r.config.InitialDelay
}

// calculateDelay 计算下次重连延迟（指数退避 + 随机抖动）
func (r *Reconnector) calculateDelay() time.Duration {
	r.mu.Lock()
	defer r.mu.Unlock()

	// 指数增长
	delay := float64(r.currentDelay) * r.config.Multiplier

	// 添加随机抖动
	if r.config.RandomFactor > 0 {
		jitter := delay * r.config.RandomFactor
		delay = delay - jitter + (rand.Float64() * 2 * jitter)
	}

	// 限制最大延迟
	if delay > float64(r.config.MaxDelay) {
		delay = float64(r.config.MaxDelay)
	}

	return time.Duration(delay)
}

// RetryCount 获取重试次数
func (r *Reconnector) RetryCount() int {
	return int(r.retryCount.Load())
}

// IsRunning 检查是否正在重连
func (r *Reconnector) IsRunning() bool {
	return r.running.Load()
}
