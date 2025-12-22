package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
)

func main() {
	fmt.Println("=== etcd Watch 监听功能示例 ===")

	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	watcher := client.Watcher()
	kv := client.KV()
	ctx := context.Background()

	// 1. 监听单个键
	fmt.Println("【1. 监听单个键】")
	testWatchKey(ctx, watcher, kv)

	// 2. 监听前缀
	fmt.Println("\n【2. 监听前缀】")
	testWatchPrefix(ctx, watcher, kv)

	// 3. 停止监听
	fmt.Println("\n【3. 停止监听】")
	testStopWatch(ctx, watcher, kv)

	fmt.Println("\n✅ 示例完成")
}

func testWatchKey(ctx context.Context, watcher *etcd.Watcher, kv *etcd.KV) {
	key := "/app/config/debug"

	// 启动监听
	err := watcher.Watch(ctx, key, func(event *etcd.WatchEvent) {
		switch event.Type {
		case etcd.EventTypePut:
			fmt.Printf("  [事件] PUT: %s = %s (rev=%d)\n", event.Key, string(event.Value), event.Revision)
		case etcd.EventTypeDelete:
			fmt.Printf("  [事件] DELETE: %s (rev=%d)\n", event.Key, event.Revision)
		}
	})

	if err != nil {
		log.Printf("  ❌ 启动监听失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 开始监听键: %s\n", key)

	// 等待监听启动
	time.Sleep(500 * time.Millisecond)

	// 触发事件
	kv.Put(ctx, key, "true")
	time.Sleep(500 * time.Millisecond)

	kv.Put(ctx, key, "false")
	time.Sleep(500 * time.Millisecond)

	kv.Delete(ctx, key)
	time.Sleep(500 * time.Millisecond)

	// 停止监听
	watcher.StopWatch(key)
	fmt.Println("  ✓ 已停止监听")
}

func testWatchPrefix(ctx context.Context, watcher *etcd.Watcher, kv *etcd.KV) {
	prefix := "/app/feature/"

	// 启动前缀监听
	err := watcher.WatchPrefix(ctx, prefix, func(event *etcd.WatchEvent) {
		switch event.Type {
		case etcd.EventTypePut:
			fmt.Printf("  [事件] PUT: %s = %s\n", event.Key, string(event.Value))
		case etcd.EventTypeDelete:
			fmt.Printf("  [事件] DELETE: %s\n", event.Key)
		}
	})

	if err != nil {
		log.Printf("  ❌ 启动前缀监听失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 开始监听前缀: %s\n", prefix)

	// 等待监听启动
	time.Sleep(500 * time.Millisecond)

	// 触发多个事件
	kv.Put(ctx, "/app/feature/cache", "enabled")
	time.Sleep(300 * time.Millisecond)

	kv.Put(ctx, "/app/feature/log", "debug")
	time.Sleep(300 * time.Millisecond)

	kv.Put(ctx, "/app/feature/metrics", "true")
	time.Sleep(300 * time.Millisecond)

	kv.Delete(ctx, "/app/feature/cache")
	time.Sleep(300 * time.Millisecond)

	// 停止监听
	watcher.StopWatch(prefix)
	fmt.Println("  ✓ 已停止监听")
}

func testStopWatch(ctx context.Context, watcher *etcd.Watcher, kv *etcd.KV) {
	keys := []string{"/test/watch1", "/test/watch2", "/test/watch3"}

	// 启动多个监听
	for _, key := range keys {
		err := watcher.Watch(ctx, key, func(event *etcd.WatchEvent) {
			fmt.Printf("  [事件] %s: %s\n", event.Key, string(event.Value))
		})
		if err != nil {
			log.Printf("  ❌ 启动监听 %s 失败: %v", key, err)
			continue
		}
	}
	fmt.Printf("  ✓ 启动了 %d 个监听\n", len(keys))

	time.Sleep(500 * time.Millisecond)

	// 触发一个事件
	kv.Put(ctx, "/test/watch1", "value1")
	time.Sleep(300 * time.Millisecond)

	// 停止所有监听
	watcher.StopAll()
	fmt.Println("  ✓ 已停止所有监听")

	// 再次写入，不应该收到事件
	kv.Put(ctx, "/test/watch2", "value2")
	time.Sleep(300 * time.Millisecond)
	fmt.Println("  ✓ 停止后不再收到事件")
}
