package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/grpc/client"
	"github.com/lk2023060901/xdooria/pkg/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.Println("启动 gRPC 客户端...")

	// 获取配置
	serverAddr := getEnv("SERVER_ADDR", "localhost:50051")
	metricsPort := getEnv("METRICS_PORT", "9101")
	requestRate := getEnvInt("REQUEST_RATE", 10)

	// 初始化 logger
	appLogger := logger.Default()

	// 创建 Prometheus 注册器
	reg := prometheus.NewRegistry()

	// 创建客户端指标配置
	metricsCfg := interceptor.DefaultClientMetricsConfig()
	clientMetrics := interceptor.NewClientMetrics(metricsCfg)

	// 注册指标到 Prometheus
	if err := clientMetrics.Register(reg); err != nil {
		log.Fatalf("注册指标失败: %v", err)
	}

	// 创建拦截器链
	chain := interceptor.NewClientChain()

	// 添加 Metrics 拦截器
	chain.AddUnaryWithPriority(
		interceptor.ClientMetricsInterceptor(clientMetrics, metricsCfg),
		interceptor.PriorityMetrics,
	)

	// 添加 Logging 拦截器
	chain.AddUnaryWithPriority(
		interceptor.ClientLoggingInterceptor(appLogger, interceptor.DefaultLoggingConfig()),
		interceptor.PriorityLogging,
	)

	// 创建连接池配置
	poolCfg := &client.PoolConfig{
		Size:                5,
		HealthCheckInterval: 30 * time.Second,
		GetTimeout:          5 * time.Second,
	}

	// 创建客户端配置
	clientCfg := &client.Config{
		Target:      serverAddr,
		DialTimeout: 5 * time.Second,
	}

	// 创建连接池
	pool, err := client.NewPool(
		clientCfg,
		poolCfg,
		client.WithUnaryInterceptors(chain.GetUnaryInterceptors()...),
	)
	if err != nil {
		log.Fatalf("创建连接池失败: %v", err)
	}
	defer pool.Close()

	// 启动 Prometheus HTTP 服务器
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
			Registry: reg,
		}))

		addr := fmt.Sprintf(":%s", metricsPort)
		log.Printf("Prometheus metrics 服务启动在 http://0.0.0.0:%s/metrics\n", metricsPort)

		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动请求生成器
	ticker := time.NewTicker(time.Second / time.Duration(requestRate))
	defer ticker.Stop()

	log.Printf("开始发送请求，速率: %d req/s\n", requestRate)

	// 等待服务器启动
	time.Sleep(2 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				// 随机选择一个方法调用
				switch rand.Intn(3) {
				case 0:
					callSayHello(ctx, pool)
				case 1:
					callGetUser(ctx, pool)
				case 2:
					callCreateUser(ctx, pool)
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("关闭客户端...")
}

func callSayHello(ctx context.Context, pool *client.Pool) {
	conn, err := pool.Get(ctx)
	if err != nil {
		log.Printf("获取连接失败: %v", err)
		return
	}
	defer pool.Put(conn)

	// 这里应该调用实际的 gRPC 方法
	// client := pb.NewDemoServiceClient(conn)
	// resp, err := client.SayHello(ctx, &pb.HelloRequest{Name: "世界"})

	// 模拟调用
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
	log.Println("调用 SayHello 成功")
}

func callGetUser(ctx context.Context, pool *client.Pool) {
	conn, err := pool.Get(ctx)
	if err != nil {
		log.Printf("获取连接失败: %v", err)
		return
	}
	defer pool.Put(conn)

	// 这里应该调用实际的 gRPC 方法
	// client := pb.NewDemoServiceClient(conn)
	// user, err := client.GetUser(ctx, &pb.GetUserRequest{UserId: rand.Int63n(1000)})

	// 模拟调用
	time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
	log.Println("调用 GetUser 成功")
}

func callCreateUser(ctx context.Context, pool *client.Pool) {
	conn, err := pool.Get(ctx)
	if err != nil {
		log.Printf("获取连接失败: %v", err)
		return
	}
	defer pool.Put(conn)

	// 这里应该调用实际的 gRPC 方法
	// client := pb.NewDemoServiceClient(conn)
	// user, err := client.CreateUser(ctx, &pb.CreateUserRequest{...})

	// 模拟调用
	time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)
	log.Println("调用 CreateUser 成功")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		var result int
		fmt.Sscanf(value, "%d", &result)
		return result
	}
	return defaultValue
}
