package main

import (
	"context"
	"log"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/client"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

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

	// 创建 gRPC Client 配置
	cfg := &client.Config{
		Target:         "localhost:50053",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 3 * time.Second,
	}

	// 创建 Client 并添加拦截器
	cli, err := client.New(cfg,
		client.WithLogger(appLogger),
		// 添加拦截器
		client.WithUnaryInterceptors(
			// 1. Recovery
			interceptor.ClientRecoveryInterceptor(appLogger, interceptor.DefaultRecoveryConfig()),
			// 2. Logging
			interceptor.ClientLoggingInterceptor(appLogger, interceptor.DefaultLoggingConfig()),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// 建立连接
	if err := cli.Dial(); err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer cli.Close()

	log.Println("Connected to interceptors server")

	// 获取连接
	conn, err := cli.GetConn()
	if err != nil {
		log.Fatalf("Failed to get connection: %v", err)
	}

	// 创建 Greeter 客户端
	greeterClient := pb.NewGreeterClient(conn)

	// 测试 1: 正常请求
	log.Println("\n=== Test 1: Normal Request ===")
	ctx := context.Background()
	resp, err := greeterClient.SayHello(ctx, &pb.HelloRequest{Name: "World"})
	if err != nil {
		log.Printf("Failed to call SayHello: %v", err)
	} else {
		log.Printf("Response: %s", resp.GetMessage())
	}

	time.Sleep(1 * time.Second)

	// 测试 2: 触发 panic 的请求
	log.Println("\n=== Test 2: Panic Request ===")
	resp, err = greeterClient.SayHello(ctx, &pb.HelloRequest{Name: "panic"})
	if err != nil {
		log.Printf("Expected error (from panic): %v", err)
	} else {
		log.Printf("Unexpected success: %s", resp.GetMessage())
	}

	time.Sleep(1 * time.Second)

	// 测试 3: 再次正常请求（验证服务没有崩溃）
	log.Println("\n=== Test 3: Normal Request After Panic ===")
	resp, err = greeterClient.SayHello(ctx, &pb.HelloRequest{Name: "Recovery Test"})
	if err != nil {
		log.Printf("Failed to call SayHello: %v", err)
	} else {
		log.Printf("Response: %s", resp.GetMessage())
	}

	log.Println("\n=== All tests completed ===")
	log.Println("Check server logs for detailed logging")
	log.Println("Check http://localhost:9090/metrics for Prometheus metrics")
}
