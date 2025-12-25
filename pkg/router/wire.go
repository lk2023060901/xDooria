package router

import (
	"github.com/google/wire"
)

// ProviderSet 导出 router 相关的 Provider
var ProviderSet = wire.NewSet(
	New,          // 返回 Router
	NewProcessor, // 返回 Processor
	NewBridge,    // 返回 *Bridge
)
