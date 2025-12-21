package interceptor

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

func TestInterceptorChain_Priority(t *testing.T) {
	// 记录拦截器执行顺序
	executionOrder := make([]int, 0)

	// 创建不同优先级的拦截器
	interceptor1 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		executionOrder = append(executionOrder, 1)
		return handler(ctx, req)
	}

	interceptor2 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		executionOrder = append(executionOrder, 2)
		return handler(ctx, req)
	}

	interceptor3 := func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		executionOrder = append(executionOrder, 3)
		return handler(ctx, req)
	}

	// 创建链并添加拦截器（不按优先级顺序添加）
	chain := NewChain().
		AddUnaryWithPriority(interceptor2, 20). // 优先级 20
		AddUnaryWithPriority(interceptor3, 30). // 优先级 30
		AddUnaryWithPriority(interceptor1, 10)  // 优先级 10（最高）

	// 获取排序后的拦截器
	sortedInterceptors := chain.GetUnaryInterceptors()

	if len(sortedInterceptors) != 3 {
		t.Fatalf("expected 3 interceptors, got %d", len(sortedInterceptors))
	}

	// 模拟执行
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, nil
	}

	for _, interceptor := range sortedInterceptors {
		interceptor(context.Background(), nil, &grpc.UnaryServerInfo{}, handler)
	}

	// 验证执行顺序：应该是 1, 2, 3（按优先级从小到大）
	if len(executionOrder) != 3 {
		t.Fatalf("expected 3 executions, got %d", len(executionOrder))
	}

	if executionOrder[0] != 1 {
		t.Errorf("first interceptor should be 1, got %d", executionOrder[0])
	}
	if executionOrder[1] != 2 {
		t.Errorf("second interceptor should be 2, got %d", executionOrder[1])
	}
	if executionOrder[2] != 3 {
		t.Errorf("third interceptor should be 3, got %d", executionOrder[2])
	}
}

func TestInterceptorChain_DefaultPriority(t *testing.T) {
	chain := NewChain().
		AddUnary(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		})

	interceptors := chain.GetUnaryInterceptors()
	if len(interceptors) != 1 {
		t.Fatalf("expected 1 interceptor, got %d", len(interceptors))
	}
}

func TestInterceptorChain_Build(t *testing.T) {
	chain := NewChain().
		AddUnaryWithPriority(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}, PriorityLogging).
		AddStreamWithPriority(func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}, PriorityMetrics)

	opts := chain.Build()

	// 应该有 2 个 ServerOption（一个 unary，一个 stream）
	if len(opts) != 2 {
		t.Fatalf("expected 2 ServerOptions, got %d", len(opts))
	}
}

func TestInterceptorChain_Empty(t *testing.T) {
	chain := NewChain()
	opts := chain.Build()

	// 空链应该返回空切片
	if len(opts) != 0 {
		t.Fatalf("expected 0 ServerOptions for empty chain, got %d", len(opts))
	}
}

func TestClientInterceptorChain_Priority(t *testing.T) {
	executionOrder := make([]int, 0)

	interceptor1 := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		executionOrder = append(executionOrder, 1)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	interceptor2 := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		executionOrder = append(executionOrder, 2)
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	chain := NewClientChain().
		AddUnaryWithPriority(interceptor2, 20).
		AddUnaryWithPriority(interceptor1, 10)

	sortedInterceptors := chain.GetUnaryInterceptors()

	if len(sortedInterceptors) != 2 {
		t.Fatalf("expected 2 interceptors, got %d", len(sortedInterceptors))
	}

	// 模拟执行
	invoker := func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, opts ...grpc.CallOption) error {
		return nil
	}

	for _, interceptor := range sortedInterceptors {
		interceptor(context.Background(), "", nil, nil, nil, invoker)
	}

	// 验证执行顺序
	if len(executionOrder) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(executionOrder))
	}

	if executionOrder[0] != 1 || executionOrder[1] != 2 {
		t.Errorf("execution order incorrect: got %v, expected [1, 2]", executionOrder)
	}
}

func TestInterceptorChain_ChainedCalls(t *testing.T) {
	// 测试链式调用
	chain := NewChain().
		AddUnaryWithPriority(nil, PriorityRecovery).
		AddUnaryWithPriority(nil, PriorityLogging).
		AddUnaryWithPriority(nil, PriorityMetrics)

	if chain == nil {
		t.Fatal("chain should not be nil after chained calls")
	}

	interceptors := chain.GetUnaryInterceptors()
	if len(interceptors) != 3 {
		t.Fatalf("expected 3 interceptors, got %d", len(interceptors))
	}
}

func TestInterceptorChain_PriorityConstants(t *testing.T) {
	// 验证优先级常量的顺序
	priorities := []int{
		PriorityRecovery,
		PriorityLogging,
		PriorityMetrics,
		PriorityTracing,
		PriorityRateLimit,
		PriorityValidation,
		PriorityAuth,
		PriorityRetry,
		PriorityTimeout,
		PriorityDefault,
	}

	// 验证优先级递增
	for i := 0; i < len(priorities)-1; i++ {
		if priorities[i] >= priorities[i+1] {
			t.Errorf("priority[%d]=%d should be less than priority[%d]=%d",
				i, priorities[i], i+1, priorities[i+1])
		}
	}
}

func BenchmarkInterceptorChain_GetUnaryInterceptors(b *testing.B) {
	chain := NewChain()
	for i := 0; i < 10; i++ {
		chain.AddUnaryWithPriority(func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}, i*10)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = chain.GetUnaryInterceptors()
	}
}
