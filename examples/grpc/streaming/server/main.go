package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/streaming"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/server"
)

// streamServer 实现 StreamService
type streamServer struct {
	pb.UnimplementedStreamServiceServer
}

// ListNumbers 实现 Server Streaming - 服务端流式返回数字
func (s *streamServer) ListNumbers(req *pb.ListRequest, stream pb.StreamService_ListNumbersServer) error {
	log.Printf("ListNumbers called with count: %d", req.GetCount())

	for i := int32(1); i <= req.GetCount(); i++ {
		// 发送数字
		if err := stream.Send(&pb.NumberResponse{Number: i}); err != nil {
			return err
		}

		log.Printf("Sent number: %d", i)
		time.Sleep(500 * time.Millisecond) // 模拟处理延迟
	}

	return nil
}

// Sum 实现 Client Streaming - 客户端流式发送数字，服务端返回总和
func (s *streamServer) Sum(stream pb.StreamService_SumServer) error {
	log.Println("Sum called")

	var sum int32
	var count int32

	for {
		req, err := stream.Recv()
		if err == io.EOF {
			// 客户端发送完毕
			log.Printf("Client finished sending. Sum: %d, Count: %d", sum, count)
			return stream.SendAndClose(&pb.SumResponse{
				Sum:   sum,
				Count: count,
			})
		}
		if err != nil {
			return err
		}

		sum += req.GetNumber()
		count++
		log.Printf("Received number: %d, current sum: %d", req.GetNumber(), sum)
	}
}

// Chat 实现 Bidirectional Streaming - 双向流式聊天
func (s *streamServer) Chat(stream pb.StreamService_ChatServer) error {
	log.Println("Chat started")

	for {
		msg, err := stream.Recv()
		if err == io.EOF {
			log.Println("Client closed chat")
			return nil
		}
		if err != nil {
			return err
		}

		log.Printf("Received from %s: %s", msg.GetUser(), msg.GetMessage())

		// 回显消息（服务端应答）
		response := &pb.ChatMessage{
			User:      "Server",
			Message:   fmt.Sprintf("Echo: %s", msg.GetMessage()),
			Timestamp: time.Now().Unix(),
		}

		if err := stream.Send(response); err != nil {
			return err
		}

		log.Printf("Sent to client: %s", response.GetMessage())
	}
}

func main() {
	// 创建 gRPC Server
	cfg := &server.Config{
		Name:    "streaming-server",
		Address: ":50052",
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 注册服务
	pb.RegisterStreamServiceServer(srv.GetGRPCServer(), &streamServer{})

	// 启动 Server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Streaming server started on :50052")

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// 优雅停止
	if err := srv.GracefulStop(); err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}

	log.Println("Server stopped")
}
