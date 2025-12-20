package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
)

func main() {
	fmt.Println("=== etcd Lock 分布式锁示例 ===\n")

	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	locker := client.Locker()
	ctx := context.Background()

	// 1. 基本锁操作
	fmt.Println("【1. 基本锁操作】")
	testBasicLock(ctx, locker)

	// 2. 超时锁
	fmt.Println("\n【2. 超时锁】")
	testLockWithTimeout(ctx, locker)

	// 3. TryLock 非阻塞
	fmt.Println("\n【3. TryLock 非阻塞】")
	testTryLock(ctx, locker)

	// 4. WithLockDo 辅助函数
	fmt.Println("\n【4. WithLockDo 辅助函数】")
	testWithLockDo(ctx, locker)

	// 5. 并发锁竞争
	fmt.Println("\n【5. 并发锁竞争】")
	testConcurrentLock(ctx, locker)

	fmt.Println("\n✅ 示例完成")
}

func testBasicLock(ctx context.Context, locker *etcd.Locker) {
	lockKey := "/locks/basic"

	// 创建锁
	lock, err := locker.NewLock(lockKey, etcd.WithLockTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建锁失败: %v", err)
		return
	}
	defer lock.Close()

	fmt.Printf("  ✓ 创建锁: %s (ttl=10s)\n", lockKey)

	// 获取锁
	if err := lock.Lock(ctx); err != nil {
		log.Printf("  ❌ 获取锁失败: %v", err)
		return
	}
	fmt.Println("  ✓ 已获取锁")

	// 模拟临界区操作
	fmt.Println("  执行临界区操作...")
	time.Sleep(2 * time.Second)

	// 释放锁
	if err := lock.Unlock(ctx); err != nil {
		log.Printf("  ❌ 释放锁失败: %v", err)
		return
	}
	fmt.Println("  ✓ 已释放锁")
}

func testLockWithTimeout(ctx context.Context, locker *etcd.Locker) {
	lockKey := "/locks/timeout"

	lock, err := locker.NewLock(lockKey, etcd.WithLockTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建锁失败: %v", err)
		return
	}
	defer lock.Close()

	// 使用超时获取锁
	timeout := 3 * time.Second
	fmt.Printf("  尝试在 %v 内获取锁...\n", timeout)

	if err := lock.LockWithTimeout(ctx, timeout); err != nil {
		log.Printf("  ❌ 获取锁超时: %v", err)
		return
	}
	fmt.Println("  ✓ 已获取锁")

	time.Sleep(1 * time.Second)

	if err := lock.Unlock(ctx); err != nil {
		log.Printf("  ❌ 释放锁失败: %v", err)
		return
	}
	fmt.Println("  ✓ 已释放锁")
}

func testTryLock(ctx context.Context, locker *etcd.Locker) {
	lockKey := "/locks/trylock"

	// 第一个锁持有者
	lock1, err := locker.NewLock(lockKey, etcd.WithLockTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建锁1失败: %v", err)
		return
	}
	defer lock1.Close()

	// 获取锁
	if err := lock1.Lock(ctx); err != nil {
		log.Printf("  ❌ 锁1获取失败: %v", err)
		return
	}
	fmt.Println("  ✓ 锁1已获取")

	// 第二个锁尝试者
	lock2, err := locker.NewLock(lockKey, etcd.WithLockTTL(10))
	if err != nil {
		log.Printf("  ❌ 创建锁2失败: %v", err)
		return
	}
	defer lock2.Close()

	// 尝试获取锁（非阻塞）
	fmt.Println("  锁2尝试获取（TryLock）...")
	if err := lock2.TryLock(ctx); err != nil {
		fmt.Printf("  ✓ 锁2获取失败（预期）: %v\n", err)
	} else {
		fmt.Println("  ✗ 锁2意外获取成功")
		lock2.Unlock(ctx)
	}

	// 释放锁1
	if err := lock1.Unlock(ctx); err != nil {
		log.Printf("  ❌ 释放锁1失败: %v", err)
		return
	}
	fmt.Println("  ✓ 锁1已释放")

	// 再次尝试获取锁2
	time.Sleep(500 * time.Millisecond)
	fmt.Println("  锁2再次尝试获取...")
	if err := lock2.TryLock(ctx); err != nil {
		log.Printf("  ❌ 锁2获取失败: %v", err)
	} else {
		fmt.Println("  ✓ 锁2获取成功")
		lock2.Unlock(ctx)
	}
}

func testWithLockDo(ctx context.Context, locker *etcd.Locker) {
	lockKey := "/locks/withdo"

	// 使用 WithLockDo 自动管理锁
	fmt.Println("  使用 WithLockDo 执行临界区操作...")
	err := locker.WithLockDo(ctx, lockKey, func() error {
		fmt.Println("  ✓ 已获取锁，执行操作")
		time.Sleep(1 * time.Second)
		fmt.Println("  ✓ 操作完成")
		return nil
	}, etcd.WithLockTTL(10))

	if err != nil {
		log.Printf("  ❌ WithLockDo 失败: %v", err)
		return
	}
	fmt.Println("  ✓ 锁已自动释放")
}

func testConcurrentLock(ctx context.Context, locker *etcd.Locker) {
	lockKey := "/locks/concurrent"
	goroutines := 5

	var wg sync.WaitGroup
	var counter int
	var mu sync.Mutex // 仅用于打印，不保护 counter

	fmt.Printf("  启动 %d 个协程竞争锁...\n", goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			lock, err := locker.NewLock(lockKey, etcd.WithLockTTL(10))
			if err != nil {
				log.Printf("  ❌ [协程%d] 创建锁失败: %v", id, err)
				return
			}
			defer lock.Close()

			// 尝试获取锁（3秒超时）
			if err := lock.LockWithTimeout(ctx, 3*time.Second); err != nil {
				mu.Lock()
				fmt.Printf("  ✗ [协程%d] 获取锁超时\n", id)
				mu.Unlock()
				return
			}

			// 临界区
			mu.Lock()
			fmt.Printf("  ✓ [协程%d] 获取锁成功\n", id)
			mu.Unlock()

			counter++
			time.Sleep(300 * time.Millisecond)

			mu.Lock()
			fmt.Printf("  ✓ [协程%d] 释放锁 (counter=%d)\n", id, counter)
			mu.Unlock()

			lock.Unlock(ctx)
		}(i)
	}

	wg.Wait()
	fmt.Printf("  ✓ 所有协程完成，最终 counter=%d\n", counter)
}
