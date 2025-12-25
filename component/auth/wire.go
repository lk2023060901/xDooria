package auth

import (
	"github.com/google/wire"
)

// ProviderSet 导出 auth 组件的 Provider
var ProviderSet = wire.NewSet(
	NewManager,
)
