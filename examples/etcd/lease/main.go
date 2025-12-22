package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
)

func main() {
	fmt.Println("=== etcd Lease 租约管理示例 ===")

	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	lease := client.Lease()
	kv := client.KV()
	ctx := context.Background()

	// 1. 创建租约
	fmt.Println("【1. 创建租约】")
	testGrant(ctx, lease, kv)

	// 2. 自动续约
	fmt.Println("\n【2. 自动续约】")
	testKeepAlive(ctx, lease, kv)

	// 3. 租约 TTL
	fmt.Println("\n【3. 租约 TTL】")
	testTTL(ctx, lease)

	// 4. 撤销租约
	fmt.Println("\n【4. 撤销租约】")
	testRevoke(ctx, lease, kv)

	fmt.Println("\n✅ 示例完成")
}

func testGrant(ctx context.Context, lease *etcd.Lease, kv *etcd.KV) {
	// 创建 10 秒租约
	leaseID, err := lease.Grant(ctx, 10)
	if err != nil {
		log.Printf("  ❌ 创建租约失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 创建租约成功: ID=%d, TTL=10s\n", leaseID)

	// 绑定键值到租约
	if err := kv.PutWithLease(ctx, "/temp/session", "active", leaseID); err != nil {
		log.Printf("  ❌ 绑定键值失败: %v", err)
		return
	}
	fmt.Println("  ✓ 键值已绑定到租约（10秒后自动删除）")

	// 查看键值
	value, err := kv.Get(ctx, "/temp/session")
	if err != nil {
		log.Printf("  ❌ 获取键值失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 当前值: %s = %s (lease=%d)\n", value.Key, string(value.Value), value.Lease)
}

func testKeepAlive(ctx context.Context, lease *etcd.Lease, kv *etcd.KV) {
	// 使用自动续约
	leaseID, stopFunc, err := lease.GrantWithKeepAlive(ctx, 5)
	if err != nil {
		log.Printf("  ❌ 创建自动续约租约失败: %v", err)
		return
	}
	defer stopFunc() // 停止续约并撤销租约

	fmt.Printf("  ✓ 创建自动续约租约: ID=%d, TTL=5s\n", leaseID)

	// 绑定键值
	if err := kv.PutWithLease(ctx, "/temp/heartbeat", "alive", leaseID); err != nil {
		log.Printf("  ❌ 绑定键值失败: %v", err)
		return
	}
	fmt.Println("  ✓ 键值已绑定到租约（自动续约中...）")

	// 等待一段时间，观察续约
	fmt.Println("  等待 8 秒，观察自动续约...")
	time.Sleep(8 * time.Second)

	// 检查键是否仍然存在（说明续约成功）
	if _, err := kv.Get(ctx, "/temp/heartbeat"); err != nil {
		log.Printf("  ❌ 键值已过期")
	} else {
		fmt.Println("  ✓ 键值仍然存在（续约成功）")
	}
}

func testTTL(ctx context.Context, lease *etcd.Lease) {
	// 创建租约
	leaseID, err := lease.Grant(ctx, 30)
	if err != nil {
		log.Printf("  ❌ 创建租约失败: %v", err)
		return
	}
	defer lease.Revoke(context.Background(), leaseID)

	// 获取 TTL
	ttl, err := lease.TTL(ctx, leaseID)
	if err != nil {
		log.Printf("  ❌ 获取 TTL 失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 租约 %d 剩余时间: %d 秒\n", leaseID, ttl)

	// 等待一段时间
	time.Sleep(3 * time.Second)

	// 再次获取 TTL
	ttl, err = lease.TTL(ctx, leaseID)
	if err != nil {
		log.Printf("  ❌ 获取 TTL 失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 3秒后剩余时间: %d 秒\n", ttl)
}

func testRevoke(ctx context.Context, lease *etcd.Lease, kv *etcd.KV) {
	// 创建租约
	leaseID, err := lease.Grant(ctx, 60)
	if err != nil {
		log.Printf("  ❌ 创建租约失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 创建租约: ID=%d\n", leaseID)

	// 绑定键值
	if err := kv.PutWithLease(ctx, "/temp/revoke-test", "data", leaseID); err != nil {
		log.Printf("  ❌ 绑定键值失败: %v", err)
		return
	}
	fmt.Println("  ✓ 键值已绑定")

	// 撤销租约
	if err := lease.Revoke(ctx, leaseID); err != nil {
		log.Printf("  ❌ 撤销租约失败: %v", err)
		return
	}
	fmt.Println("  ✓ 租约已撤销")

	// 检查键是否被删除
	if _, err := kv.Get(ctx, "/temp/revoke-test"); err != nil {
		if err == etcd.ErrKeyNotFound {
			fmt.Println("  ✓ 键值已被删除（租约撤销生效）")
		} else {
			log.Printf("  ❌ 检查键值失败: %v", err)
		}
	} else {
		fmt.Println("  ✗ 键值仍然存在（异常）")
	}
}
