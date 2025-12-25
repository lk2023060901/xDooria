package router

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/framer"
	pb "github.com/xDooria/xDooria-proto-common"
)

// Processor 消息处理器接口，连接 Framer 和 Router
type Processor interface {
	Process(ctx context.Context, reqEnvelope *pb.Envelope) (*pb.Envelope, error)
}

type processor struct {
	framer framer.Framer
	router Router
}

// NewProcessor 创建一个新的处理器实例
func NewProcessor(f framer.Framer, r Router) Processor {
	return &processor{
		framer: f,
		router: r,
	}
}

// Process 处理单个 Envelope 消息流
func (p *processor) Process(ctx context.Context, reqEnvelope *pb.Envelope) (*pb.Envelope, error) {
	// 1. 使用 Framer 解码 Envelope
	op, payload, err := p.framer.Decode(reqEnvelope)
	if err != nil {
		return nil, fmt.Errorf("framer decode failed: %w", err)
	}

	// 2. 使用 Router 调度到对应的 Handler，获取响应操作码和载荷
	respOp, respPayload, err := p.router.Dispatch(ctx, op, payload)
	if err != nil {
		return nil, fmt.Errorf("router dispatch failed (op=%d): %w", op, err)
	}

	// 3. 使用 Framer 编码响应 Payload
	respEnvelope, err := p.framer.Encode(respOp, respPayload)
	if err != nil {
		return nil, fmt.Errorf("framer encode failed: %w", err)
	}

	// 设置响应的 SeqId 为请求的 SeqId，方便客户端匹配
	if respEnvelope.Header != nil && reqEnvelope.Header != nil {
		respEnvelope.Header.SeqId = reqEnvelope.Header.SeqId
	}

	return respEnvelope, nil
}