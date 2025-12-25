// framer/seqid/seqid.go
// SeqId 管理器，用于生成序列号和防重放攻击
package seqid

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// Config SeqId 管理器配置
type Config struct {
	// 初始 SeqId（0 表示随机生成）
	InitialSeqId uint32

	// LRU 缓存大小（用于检测重复）
	CacheSize int

	// 时间窗口（秒），超过此时间的 seqId 被视为过期
	TimeWindow int64
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		InitialSeqId: 1,     // 从 1 开始递增
		CacheSize:    10000, // 缓存最近 10000 个 seqId
		TimeWindow:   300,   // 5 分钟
	}
}

// Manager SeqId 管理器接口
type Manager interface {
	// Next 生成下一个 SeqId
	Next() uint32

	// Validate 验证 SeqId 是否有效（未重放）
	// timestamp: Unix 时间戳（秒）
	Validate(seqId uint32, timestamp uint64) bool
}

// seqEntry 序列号条目
type seqEntry struct {
	timestamp int64 // Unix 时间戳（秒）
}

// managerImpl SeqId 管理器实现
type managerImpl struct {
	config *Config

	// 当前序列号（原子操作）
	current atomic.Uint32

	// 已见过的 seqId（用于防重放）
	mu   sync.RWMutex
	seen map[uint32]*seqEntry

	// LRU 队列（用于淘汰过期条目）
	lruMu   sync.Mutex
	lruList []uint32
}

// New 创建新的 SeqId 管理器
func New(cfg *Config) (Manager, error) {
	// 使用 MergeConfig 确保配置完整
	mergedCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	m := &managerImpl{
		config:  mergedCfg,
		seen:    make(map[uint32]*seqEntry),
		lruList: make([]uint32, 0, mergedCfg.CacheSize),
	}

	// 初始化序列号（从配置的初始值开始，默认为 1）
	m.current.Store(mergedCfg.InitialSeqId)

	return m, nil
}

// Next 生成下一个 SeqId
func (m *managerImpl) Next() uint32 {
	// 原子递增并处理溢出
	next := m.current.Add(1)
	if next == 0 {
		// 跳过 0（保留值）
		next = m.current.Add(1)
	}
	return next
}

// Validate 验证 SeqId 是否有效
func (m *managerImpl) Validate(seqId uint32, timestamp uint64) bool {
	now := time.Now().Unix()
	msgTime := int64(timestamp)

	// 1. 检查时间窗口
	if now-msgTime > m.config.TimeWindow || msgTime-now > m.config.TimeWindow {
		return false
	}

	// 2. 检查是否重复
	m.mu.Lock()
	defer m.mu.Unlock()

	if entry, exists := m.seen[seqId]; exists {
		// 已存在，检查时间戳是否相同（可能是重放）
		if entry.timestamp == msgTime {
			return false // 完全相同的 seqId + timestamp，判定为重放
		}
		// 时间戳不同，允许通过（SeqId 可能复用，但时间戳应不同）
	}

	// 3. 记录到 seen
	m.seen[seqId] = &seqEntry{
		timestamp: msgTime,
	}

	// 4. 添加到 LRU 队列
	m.lruMu.Lock()
	m.lruList = append(m.lruList, seqId)

	// 5. 如果超过缓存大小，清理旧条目
	if len(m.lruList) > m.config.CacheSize {
		m.evictOldEntries()
	}
	m.lruMu.Unlock()

	return true
}

// evictOldEntries 清理过期条目（必须在持有 lruMu 锁时调用）
func (m *managerImpl) evictOldEntries() {
	now := time.Now().Unix()
	evictCount := len(m.lruList) - m.config.CacheSize

	if evictCount <= 0 {
		return
	}

	// 从队列头部移除旧条目
	toRemove := m.lruList[:evictCount]
	m.lruList = m.lruList[evictCount:]

	// 从 seen map 中删除（需要检查时间戳）
	for _, seqId := range toRemove {
		if entry, exists := m.seen[seqId]; exists {
			// 只删除超过时间窗口的条目
			if now-entry.timestamp > m.config.TimeWindow {
				delete(m.seen, seqId)
			}
		}
	}
}
