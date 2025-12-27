package interceptor

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestServerTracingInterceptor_Enabled(t *testing.T) {
	// 设置测试用的 TracerProvider
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := &TracingConfig{
		Enabled:       true,
		TracerName:    "test-tracer",
		RecordPayload: false,
	}

	interceptor := ServerTracingInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	resp, err := interceptor(
		context.Background(),
		"test-request",
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
		handler,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp != "response" {
		t.Errorf("expected response 'response', got %v", resp)
	}
}

func TestServerTracingInterceptor_Disabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled: false,
	}

	interceptor := ServerTracingInterceptor(cfg)

	called := false
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		called = true
		return "response", nil
	}

	_, err := interceptor(
		context.Background(),
		"test-request",
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
		handler,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler should be called even when tracing is disabled")
	}
}

func TestServerTracingInterceptor_WithError(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := DefaultTracingConfig()
	interceptor := ServerTracingInterceptor(cfg)

	expectedErr := status.Error(codes.Internal, "test error")
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	}

	_, err := interceptor(
		context.Background(),
		"test-request",
		&grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"},
		handler,
	)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestStreamServerTracingInterceptor_Enabled(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := &TracingConfig{
		Enabled:    true,
		TracerName: "test-stream-tracer",
	}

	interceptor := StreamServerTracingInterceptor(cfg)

	mockStream := &mockServerStream{ctx: context.Background()}
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		return nil
	}

	err := interceptor(
		nil,
		mockStream,
		&grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"},
		handler,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStreamServerTracingInterceptor_Disabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled: false,
	}

	interceptor := StreamServerTracingInterceptor(cfg)

	called := false
	mockStream := &mockServerStream{ctx: context.Background()}
	handler := func(srv interface{}, ss grpc.ServerStream) error {
		called = true
		return nil
	}

	err := interceptor(
		nil,
		mockStream,
		&grpc.StreamServerInfo{FullMethod: "/test.Service/StreamMethod"},
		handler,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler should be called even when tracing is disabled")
	}
}

func TestClientTracingInterceptor_Enabled(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := &TracingConfig{
		Enabled:    true,
		TracerName: "test-client-tracer",
	}

	interceptor := ClientTracingInterceptor(cfg)

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}

	err := interceptor(
		context.Background(),
		"/test.Service/Method",
		"request",
		"reply",
		nil,
		invoker,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientTracingInterceptor_Disabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled: false,
	}

	interceptor := ClientTracingInterceptor(cfg)

	called := false
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		called = true
		return nil
	}

	err := interceptor(
		context.Background(),
		"/test.Service/Method",
		"request",
		"reply",
		nil,
		invoker,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("invoker should be called even when tracing is disabled")
	}
}

func TestClientTracingInterceptor_WithError(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := DefaultTracingConfig()
	interceptor := ClientTracingInterceptor(cfg)

	expectedErr := status.Error(codes.Unavailable, "service unavailable")
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return expectedErr
	}

	err := interceptor(
		context.Background(),
		"/test.Service/Method",
		"request",
		"reply",
		nil,
		invoker,
	)

	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestStreamClientTracingInterceptor_Enabled(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := &TracingConfig{
		Enabled:    true,
		TracerName: "test-client-stream-tracer",
	}

	interceptor := StreamClientTracingInterceptor(cfg)

	mockClientStream := &mockClientStream{}
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		return mockClientStream, nil
	}

	stream, err := interceptor(
		context.Background(),
		&grpc.StreamDesc{},
		nil,
		"/test.Service/StreamMethod",
		streamer,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stream == nil {
		t.Fatal("expected stream, got nil")
	}
}

func TestStreamClientTracingInterceptor_Disabled(t *testing.T) {
	cfg := &TracingConfig{
		Enabled: false,
	}

	interceptor := StreamClientTracingInterceptor(cfg)

	called := false
	mockClientStream := &mockClientStream{}
	streamer := func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		called = true
		return mockClientStream, nil
	}

	_, err := interceptor(
		context.Background(),
		&grpc.StreamDesc{},
		nil,
		"/test.Service/StreamMethod",
		streamer,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !called {
		t.Error("streamer should be called even when tracing is disabled")
	}
}

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		fullMethod  string
		serviceName string
	}{
		{"/test.Service/Method", "test.Service"},
		{"/package.SubPackage.Service/Method", "package.SubPackage.Service"},
		{"NoSlash", "NoSlash"},
		{"", ""},
		{"/Service", "Service"}, // 无第二个斜杠时，返回整个字符串（去掉前导斜杠）
	}

	for _, tt := range tests {
		result := extractServiceName(tt.fullMethod)
		if result != tt.serviceName {
			t.Errorf("extractServiceName(%q) = %q, want %q", tt.fullMethod, result, tt.serviceName)
		}
	}
}

func TestExtractMethodName(t *testing.T) {
	tests := []struct {
		fullMethod string
		methodName string
	}{
		{"/test.Service/Method", "Method"},
		{"/package.SubPackage.Service/MethodName", "MethodName"},
		{"NoSlash", "NoSlash"},
		{"", ""},
		{"/Service/", ""},
	}

	for _, tt := range tests {
		result := extractMethodName(tt.fullMethod)
		if result != tt.methodName {
			t.Errorf("extractMethodName(%q) = %q, want %q", tt.fullMethod, result, tt.methodName)
		}
	}
}

func TestDefaultTracingConfig(t *testing.T) {
	cfg := DefaultTracingConfig()

	if !cfg.Enabled {
		t.Error("default config should have Enabled=true")
	}

	if cfg.TracerName != "grpc" {
		t.Errorf("default TracerName should be 'grpc', got %q", cfg.TracerName)
	}

	if cfg.RecordPayload {
		t.Error("default config should have RecordPayload=false")
	}
}

// Mock implementations for testing

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

type mockClientStream struct {
	grpc.ClientStream
}

func (m *mockClientStream) RecvMsg(msg interface{}) error {
	return nil
}

func (m *mockClientStream) CloseSend() error {
	return nil
}

func BenchmarkServerTracingInterceptor(b *testing.B) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := DefaultTracingConfig()
	interceptor := ServerTracingInterceptor(cfg)

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "response", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = interceptor(context.Background(), "request", info, handler)
	}
}

func BenchmarkClientTracingInterceptor(b *testing.B) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	cfg := DefaultTracingConfig()
	interceptor := ClientTracingInterceptor(cfg)

	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = interceptor(context.Background(), "/test.Service/Method", "request", "reply", nil, invoker)
	}
}
