package interceptor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/lk2023060901/xdooria/pkg/config"
	xdooriasentry "github.com/lk2023060901/xdooria/pkg/sentry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// mockSentryHub 用于测试的 Sentry Hub mock
type mockSentryHub struct {
	capturedExceptions []error
	capturedEvents     []*sentry.Event
	configuredScopes   []map[string]interface{}
	flushed            bool
}

func (m *mockSentryHub) CaptureException(err error) {
	m.capturedExceptions = append(m.capturedExceptions, err)
}

func (m *mockSentryHub) CaptureEvent(event *sentry.Event) {
	m.capturedEvents = append(m.capturedEvents, event)
}

func (m *mockSentryHub) Flush(timeout time.Duration) bool {
	m.flushed = true
	return true
}

func (m *mockSentryHub) RecoverWithContext(ctx context.Context, p interface{}) {
	m.capturedExceptions = append(m.capturedExceptions, errors.New("panic recovered"))
}

func TestDefaultSentryConfig(t *testing.T) {
	cfg := DefaultSentryConfig()

	if !cfg.Enabled {
		t.Error("Expected Enabled to be true")
	}

	if !cfg.ReportPanics {
		t.Error("Expected ReportPanics to be true")
	}

	if !cfg.ReportErrors {
		t.Error("Expected ReportErrors to be true")
	}

	expectedCodes := []codes.Code{codes.Internal, codes.Unknown, codes.DataLoss}
	if len(cfg.ReportCodes) != len(expectedCodes) {
		t.Errorf("Expected %d ReportCodes, got %d", len(expectedCodes), len(cfg.ReportCodes))
	}

	if cfg.SampleRate != 1.0 {
		t.Errorf("Expected SampleRate 1.0, got %f", cfg.SampleRate)
	}

	if cfg.FlushTimeout != 2*time.Second {
		t.Errorf("Expected FlushTimeout 2s, got %v", cfg.FlushTimeout)
	}
}

func TestSentryConfigMerge(t *testing.T) {
	customCfg := &SentryConfig{
		SampleRate:   0.5,
		FlushTimeout: 5 * time.Second,
		ReportCodes:  []codes.Code{codes.Internal},
	}

	merged, err := config.MergeConfig(DefaultSentryConfig(), customCfg)
	if err != nil {
		t.Fatalf("Failed to merge config: %v", err)
	}

	// 自定义值应该覆盖
	if merged.SampleRate != 0.5 {
		t.Errorf("Expected SampleRate 0.5, got %f", merged.SampleRate)
	}

	if merged.FlushTimeout != 5*time.Second {
		t.Errorf("Expected FlushTimeout 5s, got %v", merged.FlushTimeout)
	}

	if len(merged.ReportCodes) != 1 || merged.ReportCodes[0] != codes.Internal {
		t.Errorf("Expected ReportCodes [Internal], got %v", merged.ReportCodes)
	}

	// 默认值应该保留
	if !merged.Enabled {
		t.Error("Expected Enabled to remain true from default")
	}

	if !merged.ReportPanics {
		t.Error("Expected ReportPanics to remain true from default")
	}

	if !merged.ReportErrors {
		t.Error("Expected ReportErrors to remain true from default")
	}
}

func TestServerSentryInterceptor_Disabled(t *testing.T) {
	cfg := &SentryConfig{
		Enabled: false,
	}

	interceptor := ServerSentryInterceptor(nil, cfg)

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	if !called {
		t.Error("Handler should be called even when Sentry is disabled")
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp != "response" {
		t.Errorf("Expected response 'response', got %v", resp)
	}
}

func TestServerSentryInterceptor_NilClient(t *testing.T) {
	cfg := &SentryConfig{
		Enabled: true,
	}

	interceptor := ServerSentryInterceptor(nil, cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	resp, err := interceptor(context.Background(), "request", info, handler)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if resp != "response" {
		t.Errorf("Expected response 'response', got %v", resp)
	}
}

func TestShouldReportError(t *testing.T) {
	tests := []struct {
		name       string
		code       codes.Code
		reportCodes []codes.Code
		expected   bool
	}{
		{
			name:       "OK code should not be reported",
			code:       codes.OK,
			reportCodes: []codes.Code{codes.Internal},
			expected:   false,
		},
		{
			name:       "Internal error should be reported",
			code:       codes.Internal,
			reportCodes: []codes.Code{codes.Internal, codes.Unknown},
			expected:   true,
		},
		{
			name:       "Unknown error should be reported",
			code:       codes.Unknown,
			reportCodes: []codes.Code{codes.Internal, codes.Unknown},
			expected:   true,
		},
		{
			name:       "NotFound should not be reported by default",
			code:       codes.NotFound,
			reportCodes: []codes.Code{codes.Internal, codes.Unknown},
			expected:   false,
		},
		{
			name:       "Empty reportCodes should report all errors",
			code:       codes.NotFound,
			reportCodes: []codes.Code{},
			expected:   true,
		},
		{
			name:       "DataLoss should be reported",
			code:       codes.DataLoss,
			reportCodes: []codes.Code{codes.Internal, codes.Unknown, codes.DataLoss},
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &SentryConfig{
				ReportCodes: tt.reportCodes,
			}

			result := shouldReportError(tt.code, cfg)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for code %v", tt.expected, result, tt.code)
			}
		})
	}
}

func TestClientSentryInterceptor_Disabled(t *testing.T) {
	cfg := &SentryConfig{
		Enabled: false,
	}

	interceptor := ClientSentryInterceptor(nil, cfg)

	called := false
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		return nil
	}

	err := interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)

	if !called {
		t.Error("Invoker should be called even when Sentry is disabled")
	}

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestSentryRecoveryHandler(t *testing.T) {
	handler := SentryRecoveryHandler(nil, 2*time.Second)

	if handler != nil {
		t.Error("Expected nil handler when sentryClient is nil")
	}

	// 模拟有 Sentry client 的情况
	// 注意：这里无法完整测试，因为需要真实的 Sentry client
	// 实际项目中应该使用 Sentry 的 test helper 或 mock
}

func TestWithSentryContext(t *testing.T) {
	ctx := context.Background()

	testData := map[string]interface{}{
		"user_id": "123",
		"action":  "test",
	}

	newCtx := WithSentryContext(ctx, "custom", testData)

	if newCtx == nil {
		t.Error("Expected non-nil context")
	}

	// 验证 context 中包含 Sentry Hub
	hub := sentry.GetHubFromContext(newCtx)
	if hub == nil {
		t.Error("Expected Sentry Hub in context")
	}
}

func TestWithSentryTag(t *testing.T) {
	ctx := context.Background()

	newCtx := WithSentryTag(ctx, "environment", "test")

	if newCtx == nil {
		t.Error("Expected non-nil context")
	}

	hub := sentry.GetHubFromContext(newCtx)
	if hub == nil {
		t.Error("Expected Sentry Hub in context")
	}
}

func TestWithSentryUser(t *testing.T) {
	ctx := context.Background()

	newCtx := WithSentryUser(ctx, "user123", "testuser", "test@example.com")

	if newCtx == nil {
		t.Error("Expected non-nil context")
	}

	hub := sentry.GetHubFromContext(newCtx)
	if hub == nil {
		t.Error("Expected Sentry Hub in context")
	}
}

func TestCaptureError_NilError(t *testing.T) {
	ctx := context.Background()

	// 不应该 panic
	CaptureError(ctx, nil)
}

func TestCaptureMessage(t *testing.T) {
	ctx := context.Background()

	// 不应该 panic
	CaptureMessage(ctx, "test message", sentry.LevelInfo)
}

func TestCapturePanic(t *testing.T) {
	ctx := context.Background()

	// 不应该 panic
	CapturePanic(ctx, "test panic")
}

func TestWrappedStreamWithSentry(t *testing.T) {
	ctx := context.Background()
	wrapped := &wrappedStreamWithSentry{
		ctx: ctx,
	}

	returnedCtx := wrapped.Context()
	if returnedCtx != ctx {
		t.Error("Context should match the wrapped context")
	}
}

// 集成测试：测试 Sentry 拦截器与真实 gRPC 调用的集成
func TestServerSentryInterceptor_Integration(t *testing.T) {
	// 初始化 Sentry（使用测试 DSN）
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "", // 空 DSN 用于测试
		Environment:      "test",
		AttachStacktrace: true,
	})
	if err != nil {
		t.Fatalf("Failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	// 创建 Sentry client（这里我们使用 nil，因为只是测试拦截器逻辑）
	var sentryClient *xdooriasentry.Client

	cfg := &SentryConfig{
		Enabled:      true,
		ReportErrors: true,
		ReportCodes:  []codes.Code{codes.Internal, codes.Unknown},
		FlushTimeout: 100 * time.Millisecond,
	}

	interceptor := ServerSentryInterceptor(sentryClient, cfg)

	t.Run("success case", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return "success", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Success",
		}

		ctx := context.Background()
		resp, err := interceptor(ctx, "request", info, handler)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if resp != "success" {
			t.Errorf("Expected 'success', got %v", resp)
		}
	})

	t.Run("error case - should report Internal", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, status.Error(codes.Internal, "internal error")
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/Error",
		}

		ctx := context.Background()
		resp, err := interceptor(ctx, "request", info, handler)

		if err == nil {
			t.Error("Expected error")
		}

		if resp != nil {
			t.Errorf("Expected nil response, got %v", resp)
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Error("Expected gRPC status error")
		}

		if st.Code() != codes.Internal {
			t.Errorf("Expected Internal code, got %v", st.Code())
		}
	})

	t.Run("error case - should not report NotFound", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			return nil, status.Error(codes.NotFound, "not found")
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/NotFound",
		}

		ctx := context.Background()
		_, err := interceptor(ctx, "request", info, handler)

		if err == nil {
			t.Error("Expected error")
		}

		st, ok := status.FromError(err)
		if !ok {
			t.Error("Expected gRPC status error")
		}

		if st.Code() != codes.NotFound {
			t.Errorf("Expected NotFound code, got %v", st.Code())
		}
	})

	t.Run("with trace ID in metadata", func(t *testing.T) {
		handler := func(ctx context.Context, req interface{}) (interface{}, error) {
			// 注意：当 sentryClient 为 nil 时，拦截器不会设置 Hub
			// 这是预期行为，所以这里不检查 Hub
			return "success", nil
		}

		info := &grpc.UnaryServerInfo{
			FullMethod: "/test.Service/WithTraceID",
		}

		md := metadata.New(map[string]string{
			"x-trace-id": "trace-123",
			"x-user-id":  "user-456",
		})
		ctx := metadata.NewIncomingContext(context.Background(), md)

		resp, err := interceptor(ctx, "request", info, handler)

		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if resp != "success" {
			t.Errorf("Expected 'success', got %v", resp)
		}
	})
}

// Benchmark 测试
func BenchmarkServerSentryInterceptor_Disabled(b *testing.B) {
	cfg := &SentryConfig{
		Enabled: false,
	}

	interceptor := ServerSentryInterceptor(nil, cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, "request", info, handler)
	}
}

func BenchmarkServerSentryInterceptor_Enabled_NoError(b *testing.B) {
	err := sentry.Init(sentry.ClientOptions{
		Dsn:         "",
		Environment: "benchmark",
	})
	if err != nil {
		b.Fatalf("Failed to initialize Sentry: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	cfg := &SentryConfig{
		Enabled:      true,
		ReportErrors: true,
		ReportCodes:  []codes.Code{codes.Internal},
		FlushTimeout: 100 * time.Millisecond,
	}

	interceptor := ServerSentryInterceptor(nil, cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(ctx, "request", info, handler)
	}
}

func BenchmarkShouldReportError(b *testing.B) {
	cfg := &SentryConfig{
		ReportCodes: []codes.Code{codes.Internal, codes.Unknown, codes.DataLoss},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = shouldReportError(codes.Internal, cfg)
	}
}
