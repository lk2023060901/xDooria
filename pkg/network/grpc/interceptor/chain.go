package interceptor

import (
	"sort"

	"google.golang.org/grpc"
)

// InterceptorChain 拦截器链（服务端）
type InterceptorChain struct {
	unary  []unaryEntry
	stream []streamEntry
}

type unaryEntry struct {
	interceptor grpc.UnaryServerInterceptor
	priority    int
}

type streamEntry struct {
	interceptor grpc.StreamServerInterceptor
	priority    int
}

// NewChain 创建新的拦截器链
func NewChain() *InterceptorChain {
	return &InterceptorChain{
		unary:  make([]unaryEntry, 0),
		stream: make([]streamEntry, 0),
	}
}

// AddUnary 添加一元拦截器（默认优先级 100）
func (c *InterceptorChain) AddUnary(interceptor grpc.UnaryServerInterceptor) *InterceptorChain {
	return c.AddUnaryWithPriority(interceptor, PriorityDefault)
}

// AddUnaryWithPriority 添加一元拦截器（指定优先级）
func (c *InterceptorChain) AddUnaryWithPriority(interceptor grpc.UnaryServerInterceptor, priority int) *InterceptorChain {
	c.unary = append(c.unary, unaryEntry{
		interceptor: interceptor,
		priority:    priority,
	})
	return c
}

// AddStream 添加流式拦截器（默认优先级 100）
func (c *InterceptorChain) AddStream(interceptor grpc.StreamServerInterceptor) *InterceptorChain {
	return c.AddStreamWithPriority(interceptor, PriorityDefault)
}

// AddStreamWithPriority 添加流式拦截器（指定优先级）
func (c *InterceptorChain) AddStreamWithPriority(interceptor grpc.StreamServerInterceptor, priority int) *InterceptorChain {
	c.stream = append(c.stream, streamEntry{
		interceptor: interceptor,
		priority:    priority,
	})
	return c
}

// Build 构建 gRPC ServerOption
func (c *InterceptorChain) Build() []grpc.ServerOption {
	opts := make([]grpc.ServerOption, 0, 2)

	// 获取排序后的拦截器
	sortedUnary := c.GetUnaryInterceptors()
	if len(sortedUnary) > 0 {
		opts = append(opts, grpc.ChainUnaryInterceptor(sortedUnary...))
	}

	sortedStream := c.GetStreamInterceptors()
	if len(sortedStream) > 0 {
		opts = append(opts, grpc.ChainStreamInterceptor(sortedStream...))
	}

	return opts
}

// GetUnaryInterceptors 获取所有一元拦截器（按优先级排序）
func (c *InterceptorChain) GetUnaryInterceptors() []grpc.UnaryServerInterceptor {
	// 复制切片
	entries := make([]unaryEntry, len(c.unary))
	copy(entries, c.unary)

	// 按优先级排序（数值越小越先执行）
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority < entries[j].priority
	})

	// 提取拦截器
	result := make([]grpc.UnaryServerInterceptor, len(entries))
	for i, entry := range entries {
		result[i] = entry.interceptor
	}

	return result
}

// GetStreamInterceptors 获取所有流式拦截器（按优先级排序）
func (c *InterceptorChain) GetStreamInterceptors() []grpc.StreamServerInterceptor {
	// 复制切片
	entries := make([]streamEntry, len(c.stream))
	copy(entries, c.stream)

	// 按优先级排序
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority < entries[j].priority
	})

	// 提取拦截器
	result := make([]grpc.StreamServerInterceptor, len(entries))
	for i, entry := range entries {
		result[i] = entry.interceptor
	}

	return result
}

// ClientInterceptorChain 拦截器链（客户端）
type ClientInterceptorChain struct {
	unary  []clientUnaryEntry
	stream []clientStreamEntry
}

type clientUnaryEntry struct {
	interceptor grpc.UnaryClientInterceptor
	priority    int
}

type clientStreamEntry struct {
	interceptor grpc.StreamClientInterceptor
	priority    int
}

// NewClientChain 创建新的客户端拦截器链
func NewClientChain() *ClientInterceptorChain {
	return &ClientInterceptorChain{
		unary:  make([]clientUnaryEntry, 0),
		stream: make([]clientStreamEntry, 0),
	}
}

// AddUnary 添加一元拦截器（默认优先级 100）
func (c *ClientInterceptorChain) AddUnary(interceptor grpc.UnaryClientInterceptor) *ClientInterceptorChain {
	return c.AddUnaryWithPriority(interceptor, PriorityDefault)
}

// AddUnaryWithPriority 添加一元拦截器（指定优先级）
func (c *ClientInterceptorChain) AddUnaryWithPriority(interceptor grpc.UnaryClientInterceptor, priority int) *ClientInterceptorChain {
	c.unary = append(c.unary, clientUnaryEntry{
		interceptor: interceptor,
		priority:    priority,
	})
	return c
}

// AddStream 添加流式拦截器（默认优先级 100）
func (c *ClientInterceptorChain) AddStream(interceptor grpc.StreamClientInterceptor) *ClientInterceptorChain {
	return c.AddStreamWithPriority(interceptor, PriorityDefault)
}

// AddStreamWithPriority 添加流式拦截器（指定优先级）
func (c *ClientInterceptorChain) AddStreamWithPriority(interceptor grpc.StreamClientInterceptor, priority int) *ClientInterceptorChain {
	c.stream = append(c.stream, clientStreamEntry{
		interceptor: interceptor,
		priority:    priority,
	})
	return c
}

// Build 构建 gRPC DialOption
func (c *ClientInterceptorChain) Build() []grpc.DialOption {
	opts := make([]grpc.DialOption, 0, 2)

	sortedUnary := c.GetUnaryInterceptors()
	if len(sortedUnary) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(sortedUnary...))
	}

	sortedStream := c.GetStreamInterceptors()
	if len(sortedStream) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(sortedStream...))
	}

	return opts
}

// GetUnaryInterceptors 获取所有一元拦截器（按优先级排序）
func (c *ClientInterceptorChain) GetUnaryInterceptors() []grpc.UnaryClientInterceptor {
	entries := make([]clientUnaryEntry, len(c.unary))
	copy(entries, c.unary)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority < entries[j].priority
	})

	result := make([]grpc.UnaryClientInterceptor, len(entries))
	for i, entry := range entries {
		result[i] = entry.interceptor
	}

	return result
}

// GetStreamInterceptors 获取所有流式拦截器（按优先级排序）
func (c *ClientInterceptorChain) GetStreamInterceptors() []grpc.StreamClientInterceptor {
	entries := make([]clientStreamEntry, len(c.stream))
	copy(entries, c.stream)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].priority < entries[j].priority
	})

	result := make([]grpc.StreamClientInterceptor, len(entries))
	for i, entry := range entries {
		result[i] = entry.interceptor
	}

	return result
}

// 常见拦截器的推荐优先级
const (
	// PriorityRecovery panic 恢复应该最先执行（优先级最高）
	PriorityRecovery = 10

	// PriorityLogging 日志记录应该很早执行
	PriorityLogging = 20

	// PriorityMetrics 指标采集
	PriorityMetrics = 30

	// PriorityTracing 链路追踪
	PriorityTracing = 35

	// PriorityRateLimit 限流
	PriorityRateLimit = 40

	// PriorityValidation 参数验证
	PriorityValidation = 50

	// PriorityAuth 认证授权
	PriorityAuth = 60

	// PriorityRetry 重试（客户端）
	PriorityRetry = 70

	// PriorityTimeout 超时控制
	PriorityTimeout = 80

	// PriorityDefault 默认优先级
	PriorityDefault = 100
)
