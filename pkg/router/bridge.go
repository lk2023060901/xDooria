package router

import (
	"context"
	"io"

	pb "github.com/xDooria/xDooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// Bridge 实现了 pb.CommonServiceServer 接口
type Bridge struct {
	pb.UnimplementedCommonServiceServer
	processor Processor
	opts      Options
}

// NewBridge 创建一个新的桥接器
func NewBridge(p Processor, opts ...Option) *Bridge {
	options := DefaultOptions()
	for _, opt := range opts {
		opt(&options)
	}
	return &Bridge{
		processor: p,
		opts:      options,
	}
}

// Stream 处理双向流（实现 读-逻辑-写 三隔离模型）
func (b *Bridge) Stream(stream pb.CommonService_StreamServer) error {
	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	// 初始化队列
	sendCh := make(chan *pb.Envelope, b.opts.SendQueueSize)
	recvCh := make(chan *pb.Envelope, b.opts.RecvQueueSize)

	// A. 写协程 (WriteLoop) - 保证发送有序
	writeFuture := conc.Go(func() (struct{}, error) {
		for {
			select {
			case <-ctx.Done():
				return struct{}{}, ctx.Err()
			case env, ok := <-sendCh:
				if !ok {
					return struct{}{}, nil
				}
				if err := stream.Send(env); err != nil {
					return struct{}{}, err
				}
			}
		}
	})

	// B. 逻辑协程 (LogicLoop) - 保证单个连接内的业务串行化
	logicFuture := conc.Go(func() (struct{}, error) {
		defer close(sendCh)
		for {
			select {
			case <-ctx.Done():
				return struct{}{}, ctx.Err()
			case req, ok := <-recvCh:
				if !ok {
					return struct{}{}, nil
				}
				resp, err := b.processor.Process(ctx, req)
				if err != nil {
					continue // 业务报错不中断逻辑循环
				}
				if resp != nil {
					select {
					case sendCh <- resp:
					case <-ctx.Done():
						return struct{}{}, ctx.Err()
					}
				}
			}
		}
	})

	// C. 读主循环 (ReadLoop) - 仅负责快速接收并投递
	defer close(recvCh)
	for {
		req, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err // 网络错误中断流
		}

		select {
		case recvCh <- req:
		case <-ctx.Done():
			return ctx.Err()
		default:
			// 逻辑处理太慢导致接收队列满，属于过载，可选择断开或报警
		}
	}

	// 等待逻辑和写入协程退出并返回错误
	return conc.AwaitAll(logicFuture, writeFuture)
}