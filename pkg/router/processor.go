package router

import (
	"context"
	"fmt"
)

// Processor 消息处理器接口，封装 Router 调度
type Processor interface {
	// Process 处理已解码的消息，返回响应操作码和载荷
	Process(ctx context.Context, op uint32, payload []byte) (respOp uint32, respPayload []byte, err error)
}

type processor struct {
	router Router
}

// NewProcessor 创建一个新的处理器实例
func NewProcessor(r Router) Processor {
	return &processor{
		router: r,
	}
}

// Process 调度到对应的 Handler 处理消息
func (p *processor) Process(ctx context.Context, op uint32, payload []byte) (uint32, []byte, error) {
	respOp, respPayload, err := p.router.Dispatch(ctx, op, payload)
	if err != nil {
		return 0, nil, fmt.Errorf("router dispatch failed (op=%d): %w", op, err)
	}
	return respOp, respPayload, nil
}
