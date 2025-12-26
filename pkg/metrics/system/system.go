package system

import (
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// Collector 系统指标收集器
type Collector struct {
	pid     int32
	proc    *process.Process
	mu      sync.RWMutex
	stats   Stats
	pool    *conc.Pool[struct{}]
	stopCh  chan struct{}
	running bool
}

// Stats 系统统计数据
type Stats struct {
	// CPU 使用率 (0-100)
	CPUPercent float64 `json:"cpu_percent"`
	// 内存使用率 (0-100)
	MemoryPercent float64 `json:"memory_percent"`
	// 内存使用字节数
	MemoryBytes uint64 `json:"memory_bytes"`
	// Goroutine 数量
	Goroutines int `json:"goroutines"`
	// 更新时间
	UpdatedAt time.Time `json:"updated_at"`
}

// New 创建系统指标收集器
func New() (*Collector, error) {
	pid := int32(os.Getpid())
	proc, err := process.NewProcess(pid)
	if err != nil {
		return nil, err
	}

	return &Collector{
		pid:    pid,
		proc:   proc,
		pool:   conc.NewPool[struct{}](1),
		stopCh: make(chan struct{}),
	}, nil
}

// Start 启动定期采集
func (c *Collector) Start(interval time.Duration) {
	if interval <= 0 {
		interval = 5 * time.Second
	}

	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return
	}
	c.running = true
	c.mu.Unlock()

	// 立即采集一次
	c.collect()

	c.pool.Submit(func() (struct{}, error) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				c.collect()
			case <-c.stopCh:
				return struct{}{}, nil
			}
		}
	})
}

// Stop 停止采集
func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		close(c.stopCh)
		c.running = false
		c.pool.Release()
	}
}

// collect 执行一次采集
func (c *Collector) collect() {
	var stats Stats

	// CPU 使用率（进程级别）
	if cpuPercent, err := c.proc.CPUPercent(); err == nil {
		stats.CPUPercent = cpuPercent
	}

	// 内存使用（进程级别）
	if memInfo, err := c.proc.MemoryInfo(); err == nil {
		stats.MemoryBytes = memInfo.RSS

		// 计算内存使用率
		if virtualMem, err := mem.VirtualMemory(); err == nil && virtualMem.Total > 0 {
			stats.MemoryPercent = float64(memInfo.RSS) / float64(virtualMem.Total) * 100
		}
	}

	// Goroutine 数量
	stats.Goroutines = runtime.NumGoroutine()
	stats.UpdatedAt = time.Now()

	c.mu.Lock()
	c.stats = stats
	c.mu.Unlock()
}

// GetStats 获取当前统计数据
func (c *Collector) GetStats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats
}

// GetCPUPercent 获取 CPU 使用率
func (c *Collector) GetCPUPercent() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats.CPUPercent
}

// GetMemoryPercent 获取内存使用率
func (c *Collector) GetMemoryPercent() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.stats.MemoryPercent
}

// GetSystemCPUPercent 获取系统整体 CPU 使用率
func GetSystemCPUPercent() (float64, error) {
	percentages, err := cpu.Percent(0, false)
	if err != nil {
		return 0, err
	}
	if len(percentages) > 0 {
		return percentages[0], nil
	}
	return 0, nil
}

// GetSystemMemoryPercent 获取系统整体内存使用率
func GetSystemMemoryPercent() (float64, error) {
	v, err := mem.VirtualMemory()
	if err != nil {
		return 0, err
	}
	return v.UsedPercent, nil
}
