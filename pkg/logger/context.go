package logger

import (
	"context"

	"go.uber.org/zap"
)

// ContextFieldExtractor 从 context 提取字段的函数类型
// 用户可以自定义此函数来从 context 中提取需要的字段
type ContextFieldExtractor func(ctx context.Context) []zap.Field

// DefaultContextExtractor 默认的 context 提取器（不提取任何字段）
// 通过提供默认实现，避免在每次调用时进行 nil 检查
func DefaultContextExtractor(ctx context.Context) []zap.Field {
	return nil
}
