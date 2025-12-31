package idgen

import (
	"sync"

	"github.com/cockroachdb/errors"
)

var (
	global Generator
	mu     sync.RWMutex
)

// Init 初始化全局ID生成器
func Init(g Generator) {
	mu.Lock()
	defer mu.Unlock()
	global = g
}

// NextID 使用全局生成器生成ID
func NextID() (int64, error) {
	mu.RLock()
	g := global
	mu.RUnlock()

	if g == nil {
		return 0, errors.New("id generator not initialized")
	}
	return g.NextID()
}
