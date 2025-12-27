package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/user"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/network/grpc/server"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var requestCount int32

// userServer 实现 UserService
type userServer struct {
	pb.UnimplementedUserServiceServer
}

// CreateUser 创建用户（带参数校验）
func (s *userServer) CreateUser(ctx context.Context, req *pb.CreateUserRequest) (*pb.CreateUserResponse, error) {
	log.Printf("CreateUser called: %+v", req)

	// 模拟偶尔失败，用于测试重试
	count := atomic.AddInt32(&requestCount, 1)
	if count%3 == 1 {
		log.Println("Simulating Unavailable error for retry test")
		return nil, status.Error(codes.Unavailable, "service temporarily unavailable")
	}

	return &pb.CreateUserResponse{
		UserId:  12345,
		Message: "User created successfully",
	}, nil
}

// GetUser 获取用户
func (s *userServer) GetUser(ctx context.Context, req *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	log.Printf("GetUser called: user_id=%d", req.GetUserId())

	if req.GetUserId() == 0 {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	return &pb.GetUserResponse{
		UserId:   req.GetUserId(),
		Username: "testuser",
		Email:    "test@example.com",
		Age:      25,
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

	// 创建限流器（每秒 5 个请求，用于测试）
	rateLimiter := interceptor.NewRateLimiter(appLogger, &interceptor.RateLimitConfig{
		Enabled:           true,
		RequestsPerSecond: 5,
		Burst:             10,
		PerMethod:         false,
		LogRateLimits:     true,
	})

	// 创建 gRPC Server 配置
	cfg := &server.Config{
		Name:    "advanced-interceptors-server",
		Address: ":50054",
	}

	// 创建 Server 并添加拦截器
	srv, err := server.New(cfg,
		server.WithLogger(appLogger),
		server.WithUnaryInterceptors(
			// 拦截器执行顺序
			interceptor.ServerRecoveryInterceptor(appLogger, interceptor.DefaultRecoveryConfig()),
			interceptor.ServerRateLimitInterceptor(rateLimiter),
			interceptor.ServerValidationInterceptor(appLogger, interceptor.DefaultValidationConfig()),
			interceptor.ServerLoggingInterceptor(appLogger, interceptor.DefaultLoggingConfig()),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// 注册服务
	pb.RegisterUserServiceServer(srv.GetGRPCServer(), &userServer{})

	// 启动 Server
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Advanced interceptors server started on :50054")
	log.Println("Server demonstrates:")
	log.Println("  - Rate Limiting: 5 req/s")
	log.Println("  - Validation: CreateUserRequest validation")
	log.Println("  - Retry simulation: CreateUser fails 1 out of 3 times")

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
