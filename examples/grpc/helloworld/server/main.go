package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/grpc/server"
)

// greeterServer 实现 Greeter 服务
type greeterServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello 实现问候方法
func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %s", req.GetName())
	return &pb.HelloReply{
		Message: "Hello " + req.GetName(),
	}, nil
}

func main() {
	// 创建 gRPC Server
	cfg := &server.Config{
		Name:    "helloworld-server",
		Address: ":50051",
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 注册服务
	pb.RegisterGreeterServer(srv.GetGRPCServer(), &greeterServer{})

	// 启动 Server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server started on :50051")

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
