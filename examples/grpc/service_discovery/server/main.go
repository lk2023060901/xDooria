package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/etcd"
	"github.com/lk2023060901/xdooria/pkg/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/registry"
	etcdRegistry "github.com/lk2023060901/xdooria/pkg/registry/etcd"
)

// greeterServer 实现 Greeter 服务
type greeterServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello 实现问候方法
func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %s", req.GetName())
	return &pb.HelloReply{
		Message: "Hello " + req.GetName() + " from service discovery",
	}, nil
}

func main() {
	// 创建 etcd 客户端
	etcdCfg := &etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	etcdClient, err := etcd.New(etcdCfg)
	if err != nil {
		log.Fatalf("Failed to create etcd client: %v", err)
	}
	defer etcdClient.Close()

	// 创建服务注册器
	registrarCfg := &etcdRegistry.Config{
		TTL: 10 * time.Second,
	}

	registrar, err := etcdRegistry.NewRegistrar(etcdClient, registrarCfg)
	if err != nil {
		log.Fatalf("Failed to create registrar: %v", err)
	}

	// 创建 gRPC Server 配置
	cfg := &server.Config{
		Name:    "greeter-service",
		Address: ":50051",
		ServiceRegistry: &server.ServiceRegistryConfig{
			ServiceName: "xdooria.greeter",
			Metadata: map[string]string{
				"version": "1.0.0",
				"region":  "us-west",
			},
		},
	}

	// 创建 Server，注入 registrar
	srv, err := server.New(cfg, server.WithRegistrar(registrar))
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 注册服务
	pb.RegisterGreeterServer(srv.GetGRPCServer(), &greeterServer{})

	// 启动 Server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server started with service discovery on :50051")

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
