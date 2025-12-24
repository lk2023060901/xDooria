// pkg/websocket/client_heartbeat.go
package websocket

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// HeartbeatManager 心跳管理器
type HeartbeatManager struct {
	config *HeartbeatConfig
	logger logger.Logger
	conn   *Connection

	// 工作池
	workerPool *conc.Pool[struct{}]

	// 状态
	lastPong  atomic.Value // time.Time
	missCount atomic.Int32

	// 回调
	onTimeout func()

	// 控制
	mu       sync.Mutex
	running  atomic.Bool
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewHeartbeatManager 创建心跳管理器
func NewHeartbeatManager(cfg *HeartbeatConfig, conn *Connection, log logger.Logger, pool *conc.Pool[struct{}]) *HeartbeatManager {
	h := &HeartbeatManager{
		config:     cfg,
		conn:       conn,
		logger:     log,
		workerPool: pool,
		stopCh:     make(chan struct{}),
	}
	h.lastPong.Store(time.Now())
	return h
}

// SetOnTimeout 设置超时回调
func (h *HeartbeatManager) SetOnTimeout(fn func()) {
	h.onTimeout = fn
}

// Start 启动心跳
func (h *HeartbeatManager) Start() {
	if !h.config.Enable {
		return
	}

	// 防止重复启动
	if !h.running.CompareAndSwap(false, true) {
		return
	}

	// 重置 stopOnce
	h.stopOnce = sync.Once{}
	h.stopCh = make(chan struct{})

	h.workerPool.Submit(func() (struct{}, error) {
		h.loop()
		return struct{}{}, nil
	})
}

// loop 心跳循环
func (h *HeartbeatManager) loop() {
	ticker := time.NewTicker(h.config.Interval)
	defer ticker.Stop()
	defer h.running.Store(false)

	for {
		select {
		case <-ticker.C:
			if err := h.sendPing(); err != nil {
				if h.logger != nil {
					h.logger.Debug("websocket heartbeat ping error",
						"error", err,
						"conn_id", h.conn.ID(),
					)
				}
				h.handleMiss()
				continue
			}

			// 检查是否超时
			if h.checkTimeout() {
				if h.logger != nil {
					h.logger.Warn("websocket heartbeat timeout",
						"miss_count", h.missCount.Load(),
						"conn_id", h.conn.ID(),
					)
				}
				if h.onTimeout != nil {
					h.onTimeout()
				}
				return
			}

		case <-h.stopCh:
			return
		}
	}
}

// sendPing 发送 Ping
func (h *HeartbeatManager) sendPing() error {
	return h.conn.Ping()
}

// OnPong 收到 Pong 回复
func (h *HeartbeatManager) OnPong() {
	h.lastPong.Store(time.Now())
	h.missCount.Store(0)
}

// handleMiss 处理丢失
func (h *HeartbeatManager) handleMiss() {
	h.missCount.Add(1)
}

// checkTimeout 检查是否超时
func (h *HeartbeatManager) checkTimeout() bool {
	// 检查丢失次数
	if h.config.MaxMissCount > 0 && int(h.missCount.Load()) >= h.config.MaxMissCount {
		return true
	}

	// 检查超时时间
	if h.config.Timeout > 0 {
		lastPong := h.lastPong.Load().(time.Time)
		if time.Since(lastPong) > h.config.Timeout {
			return true
		}
	}

	return false
}

// Stop 停止心跳
func (h *HeartbeatManager) Stop() {
	h.stopOnce.Do(func() {
		close(h.stopCh)
	})
}

// IsRunning 检查是否正在运行
func (h *HeartbeatManager) IsRunning() bool {
	return h.running.Load()
}

// LastPong 获取最后一次 Pong 时间
func (h *HeartbeatManager) LastPong() time.Time {
	return h.lastPong.Load().(time.Time)
}

// MissCount 获取丢失次数
func (h *HeartbeatManager) MissCount() int {
	return int(h.missCount.Load())
}
