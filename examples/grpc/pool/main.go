package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	pb "github.com/lk2023060901/xdooria/examples/grpc/proto/helloworld"
	"github.com/lk2023060901/xdooria/pkg/grpc/client"
)

var (
	target      = flag.String("target", "localhost:50051", "server address")
	poolSize    = flag.Int("pool-size", 5, "connection pool size")
	concurrency = flag.Int("concurrency", 20, "concurrent requests")
	requests    = flag.Int("requests", 100, "total requests")
)

func main() {
	flag.Parse()

	// 创建客户端配置（使用默认配置并覆盖部分值）
	cfg := client.DefaultConfig()
	cfg.Target = *target

	// 创建连接池配置
	poolCfg := &client.PoolConfig{
		Size:                *poolSize,
		MaxIdleTime:         30 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		GetTimeout:          5 * time.Second,
		WaitForConn:         true,
	}

	// 创建连接池
	pool, err := client.NewPool(cfg, poolCfg)
	if err != nil {
		log.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Close()

	log.Printf("Connection pool created:")
	log.Printf("  - Target: %s", *target)
	log.Printf("  - Pool Size: %d", *poolSize)
	log.Printf("  - Concurrency: %d", *concurrency)
	log.Printf("  - Total Requests: %d", *requests)

	// 打印初始统计信息
	printStats(pool)

	// 启动并发请求
	start := time.Now()
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, *concurrency)
	successCount := 0
	errorCount := 0
	var mu sync.Mutex

	for i := 0; i < *requests; i++ {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 从池中获取连接
			ctx := context.Background()
			conn, err := pool.Get(ctx)
			if err != nil {
				log.Printf("[Request %d] Failed to get connection: %v", reqNum, err)
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			// 使用完毕后归还连接
			defer pool.Put(conn)

			// 创建客户端并发起请求
			grpcClient := pb.NewGreeterClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			resp, err := grpcClient.SayHello(ctx, &pb.HelloRequest{
				Name: fmt.Sprintf("User-%d", reqNum),
			})

			if err != nil {
				log.Printf("[Request %d] RPC failed: %v", reqNum, err)
				mu.Lock()
				errorCount++
				mu.Unlock()
				return
			}

			log.Printf("[Request %d] Success: %s", reqNum, resp.Message)
			mu.Lock()
			successCount++
			mu.Unlock()
		}(i + 1)

		// 每 10 个请求打印一次统计信息
		if (i+1)%10 == 0 {
			time.Sleep(100 * time.Millisecond)
			printStats(pool)
		}
	}

	wg.Wait()
	elapsed := time.Since(start)

	// 打印最终统计信息
	log.Printf("\n=== Final Results ===")
	log.Printf("Total Time: %v", elapsed)
	log.Printf("Success: %d", successCount)
	log.Printf("Errors: %d", errorCount)
	log.Printf("QPS: %.2f", float64(*requests)/elapsed.Seconds())
	printStats(pool)
}

func printStats(pool *client.Pool) {
	stats := pool.Stats()
	log.Printf("[Pool Stats] Total: %d, InUse: %d, Available: %d, Ready: %d, Idle: %d",
		stats.Total, stats.InUse, stats.Available, stats.Ready, stats.Idle)
}
