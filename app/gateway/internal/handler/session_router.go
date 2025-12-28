package handler

import (
	"context"
	"fmt"
	"sync"

	"google.golang.org/protobuf/proto"

	"github.com/lk2023060901/xdooria/pkg/network/session"
)

// SessionHandler 带 Session 的原始消息处理器
// 返回: 响应的操作码, 响应的 payload, 错误
type SessionHandler func(ctx context.Context, s session.Session, payload []byte) (respOp uint32, respPayload []byte, err error)

// SessionRouter 支持 Session 的消息路由器
type SessionRouter struct {
	mu       sync.RWMutex
	handlers map[uint32]SessionHandler
}

// NewSessionRouter 创建 Session Router
func NewSessionRouter() *SessionRouter {
	return &SessionRouter{
		handlers: make(map[uint32]SessionHandler),
	}
}

// Register 注册原始 Handler
func (r *SessionRouter) Register(op uint32, handler SessionHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[op] = handler
}

// RegisterHandler 使用泛型注册处理器，自动处理 Proto 反序列化和序列化
// TReq: 请求类型, TResp: 响应类型
// reqOp: 请求操作码, respOp: 响应操作码
func RegisterHandler[TReq any, TResp any, PReq interface {
	*TReq
	proto.Message
}, PResp interface {
	*TResp
	proto.Message
}](
	r *SessionRouter,
	reqOp uint32,
	respOp uint32,
	handler func(ctx context.Context, s session.Session, req PReq) (PResp, error),
) {
	// 封装为原始 SessionHandler
	r.Register(reqOp, func(ctx context.Context, s session.Session, payload []byte) (uint32, []byte, error) {
		// 1. 反序列化请求
		req := PReq(new(TReq))
		if err := proto.Unmarshal(payload, req); err != nil {
			return 0, nil, fmt.Errorf("unmarshal request failed: %w", err)
		}

		// 2. 调用业务 Handler
		resp, err := handler(ctx, s, req)
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

// Dispatch 调度消息处理
func (r *SessionRouter) Dispatch(ctx context.Context, s session.Session, op uint32, payload []byte) (uint32, []byte, error) {
	r.mu.RLock()
	handler, ok := r.handlers[op]
	r.mu.RUnlock()

	if !ok {
		return 0, nil, fmt.Errorf("handler not found for op: %d", op)
	}

	return handler(ctx, s, payload)
}
