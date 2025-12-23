package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// mockLogger 模拟日志记录器
type mockLogger struct {
	debugCalls int
	errorCalls int
	infoCalls  int
	warnCalls  int
}

func (m *mockLogger) Debug(msg string, keysAndValues ...interface{})                               { m.debugCalls++ }
func (m *mockLogger) Info(msg string, keysAndValues ...interface{})                                { m.infoCalls++ }
func (m *mockLogger) Warn(msg string, keysAndValues ...interface{})                                { m.warnCalls++ }
func (m *mockLogger) Error(msg string, keysAndValues ...interface{})                               { m.errorCalls++ }
func (m *mockLogger) DebugContext(ctx context.Context, msg string, keysAndValues ...interface{})   { m.debugCalls++ }
func (m *mockLogger) InfoContext(ctx context.Context, msg string, keysAndValues ...interface{})    { m.infoCalls++ }
func (m *mockLogger) WarnContext(ctx context.Context, msg string, keysAndValues ...interface{})    { m.warnCalls++ }
func (m *mockLogger) ErrorContext(ctx context.Context, msg string, keysAndValues ...interface{})   { m.errorCalls++ }
func (m *mockLogger) Named(name string) logger.Logger                                              { return m }
func (m *mockLogger) WithFields(keysAndValues ...interface{}) logger.Logger                        { return m }
func (m *mockLogger) Sync() error                                                                  { return nil }

// 确保 mockLogger 实现 logger.Logger 接口
var _ logger.Logger = (*mockLogger)(nil)

// ================================
// Consumer Middleware Tests
// ================================

func TestLoggingMiddleware(t *testing.T) {
	log := &mockLogger{}
	middleware := LoggingMiddleware(log)

	handler := func(ctx context.Context, msg *Message) error {
		return nil
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
		Key:       []byte("test-key"),
		Value:     []byte("test-value"),
	}

	err := wrappedHandler(context.Background(), msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 应该有 2 次 Debug 调用（消费前和消费后）
	if log.debugCalls != 2 {
		t.Errorf("expected 2 debug calls, got %d", log.debugCalls)
	}
}

func TestLoggingMiddleware_WithError(t *testing.T) {
	log := &mockLogger{}
	middleware := LoggingMiddleware(log)

	expectedErr := errors.New("handler error")
	handler := func(ctx context.Context, msg *Message) error {
		return expectedErr
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
	}

	err := wrappedHandler(context.Background(), msg)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// 应该有 1 次 Debug 调用（消费前）和 1 次 Error 调用（消费失败）
	if log.debugCalls != 1 {
		t.Errorf("expected 1 debug call, got %d", log.debugCalls)
	}
	if log.errorCalls != 1 {
		t.Errorf("expected 1 error call, got %d", log.errorCalls)
	}
}

func TestRecoveryMiddleware(t *testing.T) {
	log := &mockLogger{}
	middleware := RecoveryMiddleware(log)

	handler := func(ctx context.Context, msg *Message) error {
		panic("test panic")
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
	}

	err := wrappedHandler(context.Background(), msg)
	if err != ErrConsumerPanic {
		t.Errorf("expected ErrConsumerPanic, got %v", err)
	}

	if log.errorCalls != 1 {
		t.Errorf("expected 1 error call for panic recovery, got %d", log.errorCalls)
	}
}

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	log := &mockLogger{}
	middleware := RecoveryMiddleware(log)

	handler := func(ctx context.Context, msg *Message) error {
		return nil
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
	}

	err := wrappedHandler(context.Background(), msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if log.errorCalls != 0 {
		t.Errorf("expected 0 error calls, got %d", log.errorCalls)
	}
}

func TestRetryMiddleware_Success(t *testing.T) {
	callCount := 0
	middleware := RetryMiddleware(3, 10*time.Millisecond)

	handler := func(ctx context.Context, msg *Message) error {
		callCount++
		return nil
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic: "test-topic",
	}

	err := wrappedHandler(context.Background(), msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 call on success, got %d", callCount)
	}
}

func TestRetryMiddleware_RetryThenSuccess(t *testing.T) {
	callCount := 0
	middleware := RetryMiddleware(3, 10*time.Millisecond)

	handler := func(ctx context.Context, msg *Message) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error")
		}
		return nil
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic: "test-topic",
	}

	err := wrappedHandler(context.Background(), msg)
	if err != nil {
		t.Errorf("expected no error after retries, got %v", err)
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls (2 retries + 1 success), got %d", callCount)
	}
}

func TestRetryMiddleware_AllRetrysFail(t *testing.T) {
	callCount := 0
	expectedErr := errors.New("persistent error")
	middleware := RetryMiddleware(2, 10*time.Millisecond)

	handler := func(ctx context.Context, msg *Message) error {
		callCount++
		return expectedErr
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic: "test-topic",
	}

	err := wrappedHandler(context.Background(), msg)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	// 初始调用 + 2 次重试 = 3 次
	if callCount != 3 {
		t.Errorf("expected 3 calls (initial + 2 retries), got %d", callCount)
	}
}

func TestRetryMiddleware_ContextCanceled(t *testing.T) {
	callCount := 0
	middleware := RetryMiddleware(5, 100*time.Millisecond)

	handler := func(ctx context.Context, msg *Message) error {
		callCount++
		return errors.New("error")
	}

	wrappedHandler := middleware(handler)

	ctx, cancel := context.WithCancel(context.Background())
	// 在第一次重试后取消
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	msg := &Message{
		Topic: "test-topic",
	}

	err := wrappedHandler(ctx, msg)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestTracingMiddleware(t *testing.T) {
	middleware := TracingMiddleware("test-tracer")

	callCount := 0
	handler := func(ctx context.Context, msg *Message) error {
		callCount++
		return nil
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
		Key:       []byte("test-key"),
		Headers:   map[string]string{},
	}

	err := wrappedHandler(context.Background(), msg)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected handler to be called once, got %d", callCount)
	}
}

func TestTracingMiddleware_WithError(t *testing.T) {
	middleware := TracingMiddleware("test-tracer")

	expectedErr := errors.New("handler error")
	handler := func(ctx context.Context, msg *Message) error {
		return expectedErr
	}

	wrappedHandler := middleware(handler)

	msg := &Message{
		Topic:     "test-topic",
		Partition: 0,
		Offset:    100,
	}

	err := wrappedHandler(context.Background(), msg)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

// ================================
// Producer Middleware Tests
// ================================

func TestProducerLoggingMiddleware(t *testing.T) {
	log := &mockLogger{}
	middleware := ProducerLoggingMiddleware(log)

	msg := &Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}

	next := func(ctx context.Context, m *Message) error {
		return nil
	}

	err := middleware(context.Background(), msg, next)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	// 应该有 2 次 Debug 调用（发布前和发布后）
	if log.debugCalls != 2 {
		t.Errorf("expected 2 debug calls, got %d", log.debugCalls)
	}
}

func TestProducerLoggingMiddleware_WithError(t *testing.T) {
	log := &mockLogger{}
	middleware := ProducerLoggingMiddleware(log)

	msg := &Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
	}

	expectedErr := errors.New("publish error")
	next := func(ctx context.Context, m *Message) error {
		return expectedErr
	}

	err := middleware(context.Background(), msg, next)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if log.debugCalls != 1 {
		t.Errorf("expected 1 debug call, got %d", log.debugCalls)
	}
	if log.errorCalls != 1 {
		t.Errorf("expected 1 error call, got %d", log.errorCalls)
	}
}

func TestProducerTracingMiddleware(t *testing.T) {
	middleware := ProducerTracingMiddleware("test-tracer")

	msg := &Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
		Value: []byte("test-value"),
	}

	callCount := 0
	next := func(ctx context.Context, m *Message) error {
		callCount++
		// 验证 headers 被注入
		if m.Headers == nil {
			t.Error("expected headers to be initialized")
		}
		return nil
	}

	err := middleware(context.Background(), msg, next)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected next to be called once, got %d", callCount)
	}
}

func TestProducerTracingMiddleware_WithExistingHeaders(t *testing.T) {
	middleware := ProducerTracingMiddleware("test-tracer")

	msg := &Message{
		Topic:   "test-topic",
		Headers: map[string]string{"existing": "header"},
	}

	next := func(ctx context.Context, m *Message) error {
		// 验证现有 header 保留
		if m.Headers["existing"] != "header" {
			t.Error("expected existing header to be preserved")
		}
		return nil
	}

	err := middleware(context.Background(), msg, next)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestProducerRecoveryMiddleware(t *testing.T) {
	log := &mockLogger{}
	middleware := ProducerRecoveryMiddleware(log)

	msg := &Message{
		Topic: "test-topic",
		Key:   []byte("test-key"),
	}

	next := func(ctx context.Context, m *Message) error {
		panic("test panic")
	}

	err := middleware(context.Background(), msg, next)
	if err != ErrProducerPanic {
		t.Errorf("expected ErrProducerPanic, got %v", err)
	}

	if log.errorCalls != 1 {
		t.Errorf("expected 1 error call for panic recovery, got %d", log.errorCalls)
	}
}

func TestProducerRecoveryMiddleware_NoPanic(t *testing.T) {
	log := &mockLogger{}
	middleware := ProducerRecoveryMiddleware(log)

	msg := &Message{
		Topic: "test-topic",
	}

	next := func(ctx context.Context, m *Message) error {
		return nil
	}

	err := middleware(context.Background(), msg, next)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if log.errorCalls != 0 {
		t.Errorf("expected 0 error calls, got %d", log.errorCalls)
	}
}

// ================================
// Middleware Chain Tests
// ================================

func TestMiddlewareChain(t *testing.T) {
	var order []string

	m1 := func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			order = append(order, "m1-before")
			err := next(ctx, msg)
			order = append(order, "m1-after")
			return err
		}
	}

	m2 := func(next Handler) Handler {
		return func(ctx context.Context, msg *Message) error {
			order = append(order, "m2-before")
			err := next(ctx, msg)
			order = append(order, "m2-after")
			return err
		}
	}

	handler := func(ctx context.Context, msg *Message) error {
		order = append(order, "handler")
		return nil
	}

	// 链式调用 m1 -> m2 -> handler
	wrapped := m1(m2(handler))

	msg := &Message{Topic: "test"}
	_ = wrapped(context.Background(), msg)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Errorf("expected %d calls, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected order[%d] to be %s, got %s", i, v, order[i])
		}
	}
}
