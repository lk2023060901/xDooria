package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// greeterServer 实现 Greeter 服务
type greeterServer struct {
	pb.UnimplementedGreeterServer
}

// SayHello 正常响应
func (s *greeterServer) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("Received: %s", req.GetName())

	// 模拟 panic（当名字是 "panic" 时）
	if req.GetName() == "panic" {
		panic("intentional panic for testing recovery interceptor")
	}

	return &pb.HelloReply{
		Message: "Hello " + req.GetName(),
	}, nil
}

func main() {
	// 创建 logger
	logCfg := &logger.Config{
		Level:      "info",
		Format:     "json",
		OutputPath: "stdout",
	}
	appLogger, err := logger.New(logCfg)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	// 创建 Prometheus registry
	registry := prometheus.NewRegistry()

	// 创建 Metrics
	serverMetrics := interceptor.NewServerMetrics(interceptor.DefaultServerMetricsConfig())
	if err := serverMetrics.Register(registry); err != nil {
		log.Fatalf("Failed to register metrics: %v", err)
	}

	// 创建 gRPC Server 配置
	cfg := &server.Config{
		Name:    "interceptors-server",
		Address: ":50053",
	}

	// 创建 Server 并添加拦截器
	srv, err := server.New(cfg,
		server.WithLogger(appLogger),
		// 添加拦截器（按顺序执行）
		server.WithUnaryInterceptors(
			// 1. Recovery（最外层，捕获所有 panic）
			interceptor.ServerRecoveryInterceptor(appLogger, interceptor.DefaultRecoveryConfig()),
			// 2. Logging（记录请求日志）
			interceptor.ServerLoggingInterceptor(appLogger, interceptor.DefaultLoggingConfig()),
			// 3. Metrics（收集指标）
			interceptor.ServerMetricsInterceptor(serverMetrics, interceptor.DefaultServerMetricsConfig()),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 注册服务
	pb.RegisterGreeterServer(srv.GetGRPCServer(), &greeterServer{})

	// 启动 Prometheus HTTP server
	go func() {
		http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
		log.Println("Prometheus metrics server started on :9090")
		if err := http.ListenAndServe(":9090", nil); err != nil {
			log.Printf("Prometheus server error: %v", err)
		}
	}()

	// 启动 gRPC Server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Interceptors server started on :50053")
	log.Println("Prometheus metrics available at http://localhost:9090/metrics")
	log.Println("Try: grpcurl -plaintext -d '{\"name\":\"World\"}' localhost:50053 helloworld.Greeter/SayHello")
	log.Println("Try: grpcurl -plaintext -d '{\"name\":\"panic\"}' localhost:50053 helloworld.Greeter/SayHello")

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
