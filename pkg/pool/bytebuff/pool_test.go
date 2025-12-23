package bytebuff

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPool_GetPut(t *testing.T) {
	p := NewPool()

	t.Run("get returns buffer", func(t *testing.T) {
		buf := p.Get(100)
		assert.NotNil(t, buf)
		assert.GreaterOrEqual(t, buf.Cap(), 100)
		p.Put(buf)
	})

	t.Run("get with zero hint", func(t *testing.T) {
		buf := p.Get(0)
		assert.NotNil(t, buf)
		p.Put(buf)
	})

	t.Run("put nil is safe", func(t *testing.T) {
		p.Put(nil) // should not panic
	})

	t.Run("large buffer not returned to pool", func(t *testing.T) {
		// Use a fresh pool to avoid interference from other tests
		pp := NewPool()

		buf := pp.Get(maxSize + 1)
		assert.NotNil(t, buf)

		_, puts1, _ := pp.Stats()
		pp.Put(buf) // should not be returned to pool
		_, puts2, _ := pp.Stats()

		assert.Equal(t, puts1, puts2) // puts should not increase
	})
}

func TestPool_SelectPool(t *testing.T) {
	p := NewPool()

	tests := []struct {
		sizeHint    int
		expectedIdx int
	}{
		{0, 0},
		{64, 0},
		{65, 1},
		{512, 1},
		{513, 2},
		{4096, 2},
		{4097, 3},
		{32768, 3},
		{32769, 4},
		{262144, 4},
		{262145, 5},
		{1048576, 5},
		{1048577, 5}, // exceeds max, still returns last pool
	}

	for _, tt := range tests {
		idx := p.selectPool(tt.sizeHint)
		assert.Equal(t, tt.expectedIdx, idx, "sizeHint=%d", tt.sizeHint)
	}
}

func TestPool_Stats(t *testing.T) {
	p := NewPool()

	// Initial stats should be zero
	gets, puts, misses := p.Stats()
	assert.Equal(t, uint64(0), gets)
	assert.Equal(t, uint64(0), puts)
	assert.Equal(t, uint64(0), misses)

	// Get and put
	buf := p.Get(100)
	p.Put(buf)

	gets, puts, _ = p.Stats()
	assert.Equal(t, uint64(1), gets)
	assert.Equal(t, uint64(1), puts)
}

func TestPool_Concurrent(t *testing.T) {
	p := NewPool()
	const goroutines = 100
	const iterations = 1000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				sizeHint := (id*iterations + j) % (maxSize + 100)
				buf := p.Get(sizeHint)
				buf.WriteString("test data")
				p.Put(buf)
			}
		}(i)
	}

	wg.Wait()

	gets, _, _ := p.Stats()
	assert.Equal(t, uint64(goroutines*iterations), gets)
}

func TestGlobalFunctions(t *testing.T) {
	buf := Get(256)
	assert.NotNil(t, buf)
	assert.GreaterOrEqual(t, buf.Cap(), 256)

	buf.WriteString("hello world")
	Put(buf)

	gets, _, _ := Stats()
	assert.Greater(t, gets, uint64(0))
}

func TestPool_BufferReuse(t *testing.T) {
	p := NewPool()

	// Get a buffer and write to it
	buf1 := p.Get(100)
	buf1.WriteString("first")
	p.Put(buf1)

	// Get another buffer - should be reused and reset
	buf2 := p.Get(100)
	assert.Equal(t, 0, buf2.Len()) // should be reset
	p.Put(buf2)
}

func BenchmarkPool_Get(b *testing.B) {
	p := NewPool()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := p.Get(1024)
		p.Put(buf)
	}
}

func BenchmarkPool_GetParallel(b *testing.B) {
	p := NewPool()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := p.Get(1024)
			buf.WriteString("benchmark data")
			p.Put(buf)
		}
	})
}

func BenchmarkPool_VariousSizes(b *testing.B) {
	p := NewPool()
	sizes := []int{64, 512, 4096, 32768, 262144}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		size := sizes[i%len(sizes)]
		buf := p.Get(size)
		p.Put(buf)
	}
}
