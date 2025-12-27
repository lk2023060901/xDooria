package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
)

var (
	port           = flag.Int("port", 50051, "server port")
	serverName     = flag.String("name", "server-1", "server name")
	enableRegistry = flag.Bool("registry", false, "enable service registry")
	etcdAddr       = flag.String("etcd", "127.0.0.1:2379", "etcd address")
)

// greeterServer 实现 Greeter 服务
type greeterServer struct {
	pb.UnimplementedGreeterServer
	name string
}

func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("[%s] Received: %s", s.name, req.GetName())
	return &pb.HelloReply{
		Message: fmt.Sprintf("Hello %s from %s", req.GetName(), s.name),
	}, nil
}

func main() {
	flag.Parse()

	address := fmt.Sprintf(":%d", *port)

	// 创建 Server 配置
	cfg := &server.Config{
		Name:              *serverName,
		Address:           address,
		EnableHealthCheck: true,
		EnableReflection:  true, // 启用反射服务，用于 grpcurl 等工具
	}

	// 配置选项
	opts := make([]server.Option, 0)

	// 如果启用服务注册
	if *enableRegistry {
		// 创建 etcd 注册器
		registrar, err := etcd.NewRegistrar(&etcd.Config{
			Endpoints: []string{*etcdAddr},
		})
		if err != nil {
			log.Fatalf("Failed to create registrar: %v", err)
		}

		// 配置服务注册
		cfg.ServiceRegistry = &server.ServiceRegistryConfig{
			Enabled:     true,
			Endpoints:   []string{*etcdAddr},
			ServiceName: "helloworld.Greeter",
			Metadata: map[string]string{
				"server_name": *serverName,
				"version":     "1.0.0",
				"region":      "local",
			},
			TTL: 10 * time.Second,
		}

		opts = append(opts, server.WithRegistrar(registrar))
	}

	// 创建 Server
	srv, err := server.New(cfg, opts...)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 注册 gRPC 服务
	pb.RegisterGreeterServer(srv.GetGRPCServer(), &greeterServer{
		name: *serverName,
	})

	// 启动 Server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Printf("Server started successfully")
	log.Printf("  - Address: %s", address)
	log.Printf("  - Health Check: enabled")
	log.Printf("  - Reflection: enabled")
	log.Printf("  - Service Registry: %v", *enableRegistry)
	if *enableRegistry {
		log.Printf("  - Service Name: helloworld.Greeter")
	}
	log.Printf("\nTips:")
	log.Printf("  - Health Check: grpcurl -plaintext localhost%s grpc.health.v1.Health/Check", address)
	log.Printf("  - List Services: grpcurl -plaintext localhost%s list", address)
	log.Printf("  - Call Service: grpcurl -plaintext -d '{\"name\":\"World\"}' localhost%s helloworld.Greeter/SayHello", address)

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("\nShutting down server gracefully...")

	// 优雅停止
	if err := srv.GracefulStop(); err != nil {
		log.Fatalf("Failed to stop server: %v", err)
	}

	log.Println("Server stopped successfully")
}
