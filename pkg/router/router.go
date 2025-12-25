package router

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"
)

// Handler 原始消息处理函数定义
// 返回: 响应的操作码, 响应的 payload, 错误
type Handler func(ctx context.Context, payload []byte) (respOp uint32, respPayload []byte, err error)

// Router 消息路由接口
type Router interface {
	// Register 注册原始 Handler
	Register(op uint32, handler Handler)
	// Dispatch 调度并执行处理器
	Dispatch(ctx context.Context, op uint32, payload []byte) (uint32, []byte, error)
}

// router 路由实现
type router struct {
	mu       sync.RWMutex
	handlers map[uint32]Handler
}

func New() Router {
	return &router{
		handlers: make(map[uint32]Handler),
	}
}

func (r *router) Register(op uint32, handler Handler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[op] = handler
}

func (r *router) Dispatch(ctx context.Context, op uint32, payload []byte) (uint32, []byte, error) {
	r.mu.RLock()
	handler, ok := r.handlers[op]
	r.mu.RUnlock()

	if !ok {
		return 0, nil, fmt.Errorf("handler not found for op: %d", op)
	}

	return handler(ctx, payload)
}

// --- 泛型支持部分 ---

// RegisterHandler 使用泛型注册处理器，自动处理 Proto 反序列化和序列化
// respOp: 该请求对应的响应操作码
func RegisterHandler[T1 any, T2 any, PT1 interface {
	*T1
	proto.Message
}, PT2 interface {
	*T2
	proto.Message
}](r Router, reqOp uint32, respOp uint32, handler func(context.Context, PT1) (PT2, error)) {
	
	// 封装为原始 Handler
	r.Register(reqOp, func(ctx context.Context, payload []byte) (uint32, []byte, error) {
		// 1. 反序列化请求
		req := PT1(new(T1))
		if err := proto.Unmarshal(payload, req); err != nil {
			return 0, nil, fmt.Errorf("unmarshal request failed: %w", err)
		}

		// 2. 调用业务 Handler
		resp, err := handler(ctx, req)
		if err != nil {
			return 0, nil, err
		}

		// 3. 序列化响应
		respBytes, err := proto.Marshal(resp)
		if err != nil {
			return 0, nil, fmt.Errorf("marshal response failed: %w", err)
		}

		return respOp, respBytes, nil
	})
}
