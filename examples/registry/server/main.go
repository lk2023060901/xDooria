package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/logger"
	registryetcd "github.com/lk2023060901/xdooria/pkg/registry/etcd"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
)

var (
	port = flag.Int("port", 50051, "server port")
	name = flag.String("name", "server-1", "server name")
)

// server 实现 helloworld.GreeterServer
type server struct {
	helloworld.UnimplementedGreeterServer
	name string
}

func (s *server) SayHello(ctx context.Context, req *helloworld.HelloRequest) (*helloworld.HelloReply, error) {
	logger.Default().Info("received request",
		"name", req.Name,
		"server", s.name,
	)
	return &helloworld.HelloReply{
		Message: fmt.Sprintf("Hello %s from %s", req.Name, s.name),
	}, nil
}

func main() {
	flag.Parse()

	// 创建监听器
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Default().Error("failed to listen", "error", err)
		os.Exit(1)
	}

	address := fmt.Sprintf("localhost:%d", *port)

	// 创建 etcd 注册器
	registrar, err := registryetcd.NewRegistrar(&registryetcd.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	})
	if err != nil {
		logger.Default().Error("failed to create registrar", "error", err)
		os.Exit(1)
	}

	// 注册 etcd resolver (只需要注册一次，所有 client 都会使用)
	if err := registryetcd.RegisterBuilder(&registryetcd.Config{
		Endpoints: []string{"127.0.0.1:2379"},
	}); err != nil {
		logger.Default().Error("failed to register resolver builder", "error", err)
		os.Exit(1)
	}

	// 创建健康检查器
	healthChecker := registryetcd.NewHealthChecker()

	// 创建 gRPC Server
	grpcServer := grpc.NewServer()

	// 注册服务
	helloworld.RegisterGreeterServer(grpcServer, &server{name: *name})

	// 注册健康检查
	grpc_health_v1.RegisterHealthServer(grpcServer, healthChecker.Server())
	healthChecker.SetServing("helloworld.Greeter")

	logger.Default().Info("starting server",
		"address", address,
		"name", *name,
	)

	// 创建 context 用于优雅关闭
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 监听系统信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// 启动服务器并自动注册
	serveFuture := conc.Go(func() (error, error) {
		err := registryetcd.ServeWithRegistry(
			ctx,
			grpcServer,
			lis,
			registryetcd.WithServiceName("helloworld.Greeter"),
			registryetcd.WithAddress(address),
			registryetcd.WithMetadata(map[string]string{
				"server_name": *name,
				"version":     "1.0.0",
			}),
			registryetcd.WithRegistrar(registrar),
		)
		if err != nil {
			logger.Default().Error("server error", "error", err)
		}
		return err, err
	})

	// 等待退出信号或服务器错误
	select {
	case <-sigCh:
		logger.Default().Info("shutting down...")
		cancel()
		serveFuture.Await()
	case <-serveFuture.Inner():
		if err := serveFuture.Err(); err != nil {
			logger.Default().Error("server stopped with error", "error", err)
		}
	}
}
