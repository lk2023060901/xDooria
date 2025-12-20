package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
)

func main() {
	fmt.Println("=== etcd KV 键值操作示例 ===\n")

	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	kv := client.KV()
	ctx := context.Background()

	// 1. Put/Get 基础操作
	fmt.Println("【1. Put/Get 基础操作】")
	testPutGet(ctx, kv)

	// 2. 前缀操作
	fmt.Println("\n【2. 前缀操作】")
	testPrefix(ctx, kv)

	// 3. CAS 操作
	fmt.Println("\n【3. CAS 比较并交换】")
	testCAS(ctx, kv)

	// 4. PutIfNotExists
	fmt.Println("\n【4. PutIfNotExists】")
	testPutIfNotExists(ctx, kv)

	// 5. Delete 操作
	fmt.Println("\n【5. Delete 操作】")
	testDelete(ctx, kv)

	fmt.Println("\n✅ 示例完成")
}

func testPutGet(ctx context.Context, kv *etcd.KV) {
	// Put
	if err := kv.Put(ctx, "/app/config/host", "localhost"); err != nil {
		log.Printf("  ❌ Put 失败: %v", err)
		return
	}
	fmt.Println("  ✓ Put: /app/config/host = localhost")

	// Get
	value, err := kv.Get(ctx, "/app/config/host")
	if err != nil {
		log.Printf("  ❌ Get 失败: %v", err)
		return
	}
	fmt.Printf("  ✓ Get: %s = %s (version=%d)\n", value.Key, string(value.Value), value.Version)
}

func testPrefix(ctx context.Context, kv *etcd.KV) {
	// 写入多个键
	keys := map[string]string{
		"/app/config/host":    "localhost",
		"/app/config/port":    "8080",
		"/app/config/timeout": "30s",
		"/app/feature/cache":  "true",
		"/app/feature/log":    "debug",
	}

	for k, v := range keys {
		if err := kv.Put(ctx, k, v); err != nil {
			log.Printf("  ❌ Put %s 失败: %v", k, err)
			return
		}
	}
	fmt.Println("  ✓ 写入 5 个键值对")

	// 按前缀获取
	values, err := kv.GetWithPrefix(ctx, "/app/config/")
	if err != nil {
		log.Printf("  ❌ GetWithPrefix 失败: %v", err)
		return
	}

	fmt.Printf("  ✓ 获取前缀 /app/config/ 的键值:\n")
	for _, v := range values {
		fmt.Printf("    %s = %s\n", v.Key, string(v.Value))
	}
}

func testCAS(ctx context.Context, kv *etcd.KV) {
	key := "/app/counter"

	// 初始化
	kv.Put(ctx, key, "100")
	fmt.Println("  初始值: 100")

	// CAS 成功
	success, err := kv.CompareAndSwap(ctx, key, "100", "200")
	if err != nil {
		log.Printf("  ❌ CAS 失败: %v", err)
		return
	}

	if success {
		fmt.Println("  ✓ CAS 成功: 100 -> 200")
	} else {
		fmt.Println("  ✗ CAS 失败: 值不匹配")
	}

	// CAS 失败
	success, err = kv.CompareAndSwap(ctx, key, "100", "300")
	if err != nil {
		log.Printf("  ❌ CAS 失败: %v", err)
		return
	}

	if !success {
		fmt.Println("  ✓ CAS 正确失败: 值已变更为 200")
	}
}

func testPutIfNotExists(ctx context.Context, kv *etcd.KV) {
	key := "/app/init-flag"

	// 第一次创建
	success, err := kv.PutIfNotExists(ctx, key, "initialized")
	if err != nil {
		log.Printf("  ❌ PutIfNotExists 失败: %v", err)
		return
	}

	if success {
		fmt.Println("  ✓ 首次创建成功")
	}

	// 第二次尝试（应该失败）
	success, err = kv.PutIfNotExists(ctx, key, "re-initialized")
	if err != nil {
		log.Printf("  ❌ PutIfNotExists 失败: %v", err)
		return
	}

	if !success {
		fmt.Println("  ✓ 重复创建被拒绝（键已存在）")
	}
}

func testDelete(ctx context.Context, kv *etcd.KV) {
	// 单个删除
	kv.Put(ctx, "/app/temp", "value")
	deleted, err := kv.Delete(ctx, "/app/temp")
	if err != nil {
		log.Printf("  ❌ Delete 失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 删除单个键: 删除了 %d 个\n", deleted)

	// 前缀删除
	deleted, err = kv.DeleteWithPrefix(ctx, "/app/config/")
	if err != nil {
		log.Printf("  ❌ DeleteWithPrefix 失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 删除前缀 /app/config/: 删除了 %d 个\n", deleted)
}
