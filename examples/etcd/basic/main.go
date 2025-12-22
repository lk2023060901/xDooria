package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
)

func main() {
	fmt.Println("=== etcd 基础使用示例 ===")

	// 1. 创建客户端
	fmt.Println("【1. 创建客户端】")
	testCreateClient()

	// 2. 健康检查
	fmt.Println("\n【2. 健康检查】")
	testHealthCheck()

	// 3. 端点管理
	fmt.Println("\n【3. 端点管理】")
	testEndpoints()

	// 4. 使用默认配置
	fmt.Println("\n【4. 默认配置】")
	testDefaultConfig()

	fmt.Println("\n✅ 示例完成")
}

func testCreateClient() {
	cfg := &etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	}

	client, err := etcd.New(cfg)
	if err != nil {
		log.Printf("  ❌ 创建客户端失败: %v", err)
		return
	}
	defer client.Close()

	fmt.Println("  ✓ 客户端创建成功")
	fmt.Printf("  端点: %v\n", client.Endpoints())
}

func testHealthCheck() {
	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Printf("  ❌ 创建客户端失败: %v", err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		log.Printf("  ❌ 健康检查失败: %v", err)
		return
	}

	fmt.Println("  ✓ etcd 集群健康")
}

func testEndpoints() {
	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Printf("  ❌ 创建客户端失败: %v", err)
		return
	}
	defer client.Close()

	endpoints := client.Endpoints()
	fmt.Printf("  配置的端点: %v\n", endpoints)

	// 同步端点
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Sync(ctx); err != nil {
		log.Printf("  ⚠️ 同步端点失败: %v", err)
	} else {
		fmt.Printf("  同步后端点: %v\n", client.Endpoints())
	}
}

func testDefaultConfig() {
	cfg := etcd.DefaultConfig()

	fmt.Printf("  默认端点: %v\n", cfg.Endpoints)
	fmt.Printf("  连接超时: %v\n", cfg.DialTimeout)
	fmt.Printf("  启用重试: %v\n", cfg.EnableRetry)
	fmt.Printf("  最大重试次数: %d\n", cfg.MaxRetries)
}
