package sentry

import "github.com/getsentry/sentry-go"

// ContextEnricher 上下文增强函数（由使用者定义如何收集数据）
type ContextEnricher func(scope *sentry.Scope)

// ErrorHandler 错误处理器（框架无关）
type ErrorHandler struct {
	client   *Client
	enricher ContextEnricher // 上下文增强器（可选）
}

// ErrorHandlerOption 错误处理器选项
type ErrorHandlerOption func(*ErrorHandler)

// WithContextEnricher 设置上下文增强器
func WithContextEnricher(enricher ContextEnricher) ErrorHandlerOption {
	return func(h *ErrorHandler) {
		h.enricher = enricher
	}
}

// NewErrorHandler 创建错误处理器
func (c *Client) NewErrorHandler(opts ...ErrorHandlerOption) *ErrorHandler {
	handler := &ErrorHandler{
		client: c,
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler
}

// CaptureError 捕获错误
func (h *ErrorHandler) CaptureError(err error) *sentry.EventID {
	if h.enricher != nil {
		h.client.Hub().WithScope(func(scope *sentry.Scope) {
			h.enricher(scope) // 调用使用者提供的增强函数
		})
	}

	return h.client.CaptureException(err)
}

// RecoverPanic 恢复 panic
func (h *ErrorHandler) RecoverPanic() {
	if r := recover(); r != nil {
		if h.enricher != nil {
			h.client.Hub().WithScope(func(scope *sentry.Scope) {
				h.enricher(scope)
			})
		}

		h.client.RecoverWithContext(r)
		panic(r) // 重新抛出
	}
}

// WrapGoroutine 包装 Goroutine（自动 recover）
func (c *Client) WrapGoroutine(f func()) func() {
	return func() {
		defer c.Recover()
		f()
	}
}
