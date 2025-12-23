// pkg/pool/bytebuff/pool.go
package bytebuff

import (
	"bytes"
	"sync"
	"sync/atomic"
)

const (
	// 分级配置: 64B, 512B, 4KB, 32KB, 256KB, 1MB
	minSize      = 64
	maxSize      = 1 << 20 // 1MB，超过此大小的 buffer 不放回池中
	numPools     = 6       // 分级数量
	calibrateCap = 10000   // 校准阈值
)

// 分级大小: 64, 512, 4096, 32768, 262144, 1048576
var poolSizes = [numPools]int{
	1 << 6,  // 64B
	1 << 9,  // 512B
	1 << 12, // 4KB
	1 << 15, // 32KB
	1 << 18, // 256KB
	1 << 20, // 1MB
}

// Pool 是分级的 bytes.Buffer 对象池
type Pool struct {
	pools [numPools]sync.Pool

	// 统计信息
	gets   uint64
	puts   uint64
	misses uint64
}

// defaultPool 是默认的全局池
var defaultPool = NewPool()

// NewPool 创建一个新的分级 buffer pool
func NewPool() *Pool {
	p := &Pool{}
	for i := 0; i < numPools; i++ {
		size := poolSizes[i]
		p.pools[i].New = func() interface{} {
			return &bytes.Buffer{}
		}
		// 预热: 避免启动时的分配抖动
		_ = size // 保留变量供后续可能的预热使用
	}
	return p
}

// Get 从池中获取一个 Buffer
// sizeHint 是期望的容量提示，用于选择合适的分级池
func (p *Pool) Get(sizeHint int) *bytes.Buffer {
	atomic.AddUint64(&p.gets, 1)

	idx := p.selectPool(sizeHint)
	v := p.pools[idx].Get()
	buf := v.(*bytes.Buffer)

	// 确保 buffer 有足够的容量
	if buf.Cap() < sizeHint {
		atomic.AddUint64(&p.misses, 1)
		buf.Grow(sizeHint - buf.Cap())
	}

	return buf
}

// Put 将 Buffer 归还到池中
func (p *Pool) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	cap := buf.Cap()

	// 超过最大限制的 buffer 不放回池中，让 GC 回收
	if cap > maxSize {
		return
	}

	atomic.AddUint64(&p.puts, 1)

	buf.Reset()
	idx := p.selectPoolByCap(cap)
	p.pools[idx].Put(buf)
}

// selectPool 根据 sizeHint 选择合适的分级池
func (p *Pool) selectPool(sizeHint int) int {
	if sizeHint <= 0 {
		return 0
	}
	for i := 0; i < numPools; i++ {
		if sizeHint <= poolSizes[i] {
			return i
		}
	}
	return numPools - 1
}

// selectPoolByCap 根据实际容量选择归还的池
func (p *Pool) selectPoolByCap(cap int) int {
	for i := 0; i < numPools; i++ {
		if cap <= poolSizes[i] {
			return i
		}
	}
	return numPools - 1
}

// Stats 返回池的统计信息
func (p *Pool) Stats() (gets, puts, misses uint64) {
	return atomic.LoadUint64(&p.gets),
		atomic.LoadUint64(&p.puts),
		atomic.LoadUint64(&p.misses)
}

// --- 全局便捷函数 ---

// Get 从默认池中获取一个 Buffer
func Get(sizeHint int) *bytes.Buffer {
	return defaultPool.Get(sizeHint)
}

// Put 将 Buffer 归还到默认池中
func Put(buf *bytes.Buffer) {
	defaultPool.Put(buf)
}

// Stats 返回默认池的统计信息
func Stats() (gets, puts, misses uint64) {
	return defaultPool.Stats()
}
