package main

import (
	"context"
	"log"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/user"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/client"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc/codes"
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

	// 创建重试配置
	retryCfg := interceptor.DefaultRetryConfig()
	retryCfg.MaxAttempts = 5
	retryCfg.RetryableCodes = []codes.Code{codes.Unavailable, codes.DeadlineExceeded}

	// 创建 gRPC Client 配置
	cfg := &client.Config{
		Target:         "localhost:50054",
		DialTimeout:    5 * time.Second,
		RequestTimeout: 10 * time.Second,
	}

	// 创建 Client 并添加拦截器
	cli, err := client.New(cfg,
		client.WithLogger(appLogger),
		client.WithUnaryInterceptors(
			interceptor.ClientRecoveryInterceptor(appLogger, interceptor.DefaultRecoveryConfig()),
			interceptor.ClientRetryInterceptor(appLogger, retryCfg),
			interceptor.ClientValidationInterceptor(appLogger, interceptor.DefaultValidationConfig()),
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

	log.Println("Connected to advanced interceptors server")

	// 获取连接
	conn, err := cli.GetConn()
	if err != nil {
		log.Fatalf("Failed to get connection: %v", err)
	}

	// 创建 UserService 客户端
	userClient := pb.NewUserServiceClient(conn)

	// 测试 1: 有效的创建用户请求（会触发重试）
	log.Println("\n=== Test 1: Valid CreateUser (with retry) ===")
	ctx := context.Background()
	resp, err := userClient.CreateUser(ctx, &pb.CreateUserRequest{
		Username: "alice",
		Email:    "alice@example.com",
		Age:      25,
	})
	if err != nil {
		log.Printf("CreateUser failed: %v", err)
	} else {
		log.Printf("CreateUser succeeded: user_id=%d, message=%s", resp.GetUserId(), resp.GetMessage())
	}

	time.Sleep(500 * time.Millisecond)

	// 测试 2: 无效的请求（校验失败）
	log.Println("\n=== Test 2: Invalid CreateUser (validation error) ===")
	resp, err = userClient.CreateUser(ctx, &pb.CreateUserRequest{
		Username: "ab", // 太短
		Email:    "",   // 空邮箱
		Age:      200,  // 超出范围
	})
	if err != nil {
		log.Printf("Expected validation error: %v", err)
	} else {
		log.Printf("Unexpected success: %+v", resp)
	}

	time.Sleep(500 * time.Millisecond)

	// 测试 3: 限流测试（快速发送多个请求）
	log.Println("\n=== Test 3: Rate Limiting (sending 10 requests rapidly) ===")
	for i := 0; i < 10; i++ {
		_, err := userClient.GetUser(ctx, &pb.GetUserRequest{UserId: int64(i + 1)})
		if err != nil {
			log.Printf("Request %d failed (rate limited): %v", i+1, err)
		} else {
			log.Printf("Request %d succeeded", i+1)
		}
	}

	log.Println("\n=== All tests completed ===")
	log.Println("Check server logs for detailed information")
}
