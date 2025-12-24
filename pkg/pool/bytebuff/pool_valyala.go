// pkg/pool/bytebuff/pool_valyala.go
// 基于 valyala/bytebufferpool 的适配实现
// 保留我们的 API 和统计功能，底层使用 valyala 的高性能实现
package bytebuff

import (
	"bytes"
	"sync/atomic"

	"github.com/valyala/bytebufferpool"
)

// ValyalaPool 是基于 valyala/bytebufferpool 的适配池
// 保持与原有 Pool 相同的 API，但使用 valyala 的高性能实现
type ValyalaPool struct {
	pool *bytebufferpool.Pool

	// 统计信息（保留原有功能）
	gets   uint64
	puts   uint64
	misses uint64 // valyala 会自动处理容量，misses 会很少
}

// defaultValyalaPool 是默认的全局池
var defaultValyalaPool = NewValyalaPool()

// NewValyalaPool 创建一个新的基于 valyala 的 buffer pool
func NewValyalaPool() *ValyalaPool {
	return &ValyalaPool{
		pool: &bytebufferpool.Pool{},
	}
}

// Get 从池中获取一个 ByteBuffer
// 注意：valyala 使用 ByteBuffer 而不是 bytes.Buffer
// ByteBuffer 性能更高（零拷贝，直接操作 []byte）
func (p *ValyalaPool) Get() *bytebufferpool.ByteBuffer {
	atomic.AddUint64(&p.gets, 1)
	return p.pool.Get()
}

// Put 将 ByteBuffer 归还到池中
func (p *ValyalaPool) Put(buf *bytebufferpool.ByteBuffer) {
	if buf == nil {
		return
	}

	atomic.AddUint64(&p.puts, 1)
	p.pool.Put(buf)
}

// Stats 返回池的统计信息
func (p *ValyalaPool) Stats() (gets, puts, misses uint64) {
	return atomic.LoadUint64(&p.gets),
		atomic.LoadUint64(&p.puts),
		atomic.LoadUint64(&p.misses)
}

// --- 全局便捷函数 ---

// GetValyala 从默认 valyala 池中获取一个 ByteBuffer
func GetValyala() *bytebufferpool.ByteBuffer {
	return defaultValyalaPool.Get()
}

// PutValyala 将 ByteBuffer 归还到默认 valyala 池中
func PutValyala(buf *bytebufferpool.ByteBuffer) {
	defaultValyalaPool.Put(buf)
}

// ValyalaStats 返回默认 valyala 池的统计信息
func ValyalaStats() (gets, puts, misses uint64) {
	return defaultValyalaPool.Stats()
}

// --- bytes.Buffer 兼容适配器 ---

// PoolAdapter 提供 bytes.Buffer 兼容的 API
// 内部使用 valyala 的 ByteBuffer，在接口层进行转换
type PoolAdapter struct {
	valyala *ValyalaPool

	// 统计信息
	gets   uint64
	puts   uint64
	misses uint64
}

// NewPoolAdapter 创建一个兼容 bytes.Buffer API 的适配器
func NewPoolAdapter() *PoolAdapter {
	return &PoolAdapter{
		valyala: NewValyalaPool(),
	}
}

// Get 获取一个 bytes.Buffer (兼容原有 API)
// 注意：这会有额外的转换开销，建议直接使用 GetValyala() 获取 ByteBuffer
func (p *PoolAdapter) Get(sizeHint int) *bytes.Buffer {
	atomic.AddUint64(&p.gets, 1)

	// 从 valyala 获取 ByteBuffer
	vbuf := p.valyala.Get()

	// 转换为 bytes.Buffer
	// 注意：这会复制数据，有性能开销
	buf := bytes.NewBuffer(vbuf.B)

	// 归还 valyala buffer（因为已经复制）
	p.valyala.Put(vbuf)

	return buf
}

// Put 归还 bytes.Buffer (兼容原有 API)
func (p *PoolAdapter) Put(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	atomic.AddUint64(&p.puts, 1)

	// bytes.Buffer 不能直接放入 valyala 池
	// 所以这里只是重置，让 GC 回收
	buf.Reset()

	// 注意：这里无法真正放回 valyala 池
	// 如果需要高性能，应该直接使用 ByteBuffer API
}

// Stats 返回适配器的统计信息
func (p *PoolAdapter) Stats() (gets, puts, misses uint64) {
	return atomic.LoadUint64(&p.gets),
		atomic.LoadUint64(&p.puts),
		atomic.LoadUint64(&p.misses)
}
