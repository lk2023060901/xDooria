package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// 为 Redlock 测试创建多个独立的客户端
func getRedlockClients(t *testing.T) []*Client {
	// 使用 Redlock 专用环境的 3 个独立实例: 18001, 18002, 18003
	configs := []*Config{
		{
			Standalone: &NodeConfig{Host: "localhost", Port: 18001, Password: "", DB: 0},
			Pool:       getTestPoolConfig(),
		},
		{
			Standalone: &NodeConfig{Host: "localhost", Port: 18002, Password: "", DB: 0},
			Pool:       getTestPoolConfig(),
		},
		{
			Standalone: &NodeConfig{Host: "localhost", Port: 18003, Password: "", DB: 0},
			Pool:       getTestPoolConfig(),
		},
	}

	clients := make([]*Client, len(configs))
	for i, config := range configs {
		client, err := NewClient(config)
		if err != nil {
			t.Fatalf("NewClient(%d) error = %v", i, err)
		}
		clients[i] = client
	}

	return clients
}

// closeRedlockClients 关闭所有 Redlock 客户端
func closeRedlockClients(clients []*Client) {
	for _, client := range clients {
		client.Close()
	}
}

// TestNewRedlock 测试创建 Redlock
func TestNewRedlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	tests := []struct {
		name    string
		clients []*Client
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "valid clients",
			clients: clients,
			ttl:     5 * time.Second,
			wantErr: false,
		},
		{
			name:    "empty clients",
			clients: []*Client{},
			ttl:     5 * time.Second,
			wantErr: true,
		},
		{
			name:    "nil clients",
			clients: nil,
			ttl:     5 * time.Second,
			wantErr: true,
		},
		{
			name:    "default ttl",
			clients: clients,
			ttl:     0, // 使用默认值
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redlock, err := NewRedlock(tt.clients, tt.ttl)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRedlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if redlock == nil {
					t.Error("NewRedlock() returned nil")
					return
				}

				// 验证 quorum 计算
				expectedQuorum := len(tt.clients)/2 + 1
				if redlock.quorum != expectedQuorum {
					t.Errorf("quorum = %v, want %v", redlock.quorum, expectedQuorum)
				}

				// 验证 TTL
				if tt.ttl > 0 && redlock.ttl != tt.ttl {
					t.Errorf("ttl = %v, want %v", redlock.ttl, tt.ttl)
				}
				if tt.ttl == 0 && redlock.ttl != defaultRedlockTTL {
					t.Errorf("ttl = %v, want default %v", redlock.ttl, defaultRedlockTTL)
				}
			}
		})
	}
}

// TestRedlockLockUnlock 测试 Redlock 锁的获取和释放
func TestRedlockLockUnlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "test:redlock:basic"

	// 清理测试数据
	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	instance := redlock.NewInstance(key)

	// Test Lock
	if err := instance.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}

	// 验证锁在多个节点上存在
	lockedCount := 0
	for _, client := range clients {
		val, err := client.Get(ctx, key)
		if err == nil && val == instance.value {
			lockedCount++
		}
	}

	if lockedCount < redlock.quorum {
		t.Errorf("Lock() locked on %v nodes, want >= %v (quorum)", lockedCount, redlock.quorum)
	}

	// Test Unlock
	if err := instance.Unlock(ctx); err != nil {
		t.Errorf("Unlock() error = %v", err)
	}

	// 验证锁已在所有节点上释放
	for i, client := range clients {
		_, err := client.Get(ctx, key)
		if err != ErrNil {
			t.Errorf("node %d: key still exists after Unlock(), error = %v", i, err)
		}
	}
}

// TestRedlockTryLock 测试 Redlock 非阻塞获取锁
func TestRedlockTryLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "test:redlock:trylock"

	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	instance1 := redlock.NewInstance(key)
	instance2 := redlock.NewInstance(key)

	// instance1 获取锁
	ok, err := instance1.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if !ok {
		t.Error("TryLock() = false, want true for first lock")
	}

	// instance2 尝试获取锁（应该失败）
	ok, err = instance2.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if ok {
		t.Error("TryLock() = true, want false for concurrent lock")
	}

	// 释放 instance1
	if err := instance1.Unlock(ctx); err != nil {
		t.Errorf("Unlock() error = %v", err)
	}

	// instance2 再次尝试获取锁（应该成功）
	ok, err = instance2.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if !ok {
		t.Error("TryLock() = false, want true after instance1 released")
	}

	instance2.Unlock(ctx)
}

// TestRedlockWithRetry 测试 Redlock 带重试的锁获取
func TestRedlockWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 2*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "test:redlock:retry"

	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	instance1 := redlock.NewInstance(key)
	instance2 := redlock.NewInstance(key)

	// instance1 获取锁
	if err := instance1.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}

	// 启动 goroutine 在 1 秒后释放 instance1
	go func() {
		time.Sleep(1 * time.Second)
		instance1.Unlock(ctx)
	}()

	// instance2 尝试获取锁（最多重试 5 次，每次间隔 500ms）
	start := time.Now()
	err = instance2.LockWithRetry(ctx, 5, 500*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("LockWithRetry() error = %v", err)
	}

	// 应该在 1-3 秒内获取到锁
	if elapsed < 1*time.Second || elapsed > 3*time.Second {
		t.Errorf("LockWithRetry() took %v, expected 1-3s", elapsed)
	}

	instance2.Unlock(ctx)
}

// TestRedlockRefresh 测试 Redlock 锁续期
func TestRedlockRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 2*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "test:redlock:refresh"

	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	instance := redlock.NewInstance(key)

	// 获取锁
	if err := instance.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}
	defer instance.Unlock(ctx)

	// 等待 1 秒
	time.Sleep(1 * time.Second)

	// 刷新锁
	if err := instance.Refresh(ctx); err != nil {
		t.Errorf("Refresh() error = %v", err)
	}

	// 检查 TTL（应该接近 2 秒，至少在 quorum 数量的节点上）
	refreshedCount := 0
	for _, client := range clients {
		ttl, err := client.TTL(ctx, key)
		if err != nil {
			continue
		}
		if ttl >= 1500*time.Millisecond && ttl <= 2*time.Second {
			refreshedCount++
		}
	}

	if refreshedCount < redlock.quorum {
		t.Errorf("Refresh() succeeded on %v nodes, want >= %v (quorum)", refreshedCount, redlock.quorum)
	}
}

// TestRedlockWithRedlock 测试 WithRedlock 便捷方法
func TestRedlockWithRedlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "test:redlock:with"

	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	var counter int
	err = redlock.WithRedlock(ctx, key, func() error {
		counter++
		return nil
	})

	if err != nil {
		t.Errorf("WithRedlock() error = %v", err)
	}

	if counter != 1 {
		t.Errorf("counter = %v, want 1", counter)
	}

	// 验证锁已释放
	for i, client := range clients {
		_, err := client.Get(ctx, key)
		if err != ErrNil {
			t.Errorf("node %d: lock should be released after WithRedlock()", i)
		}
	}
}

// TestRedlockConcurrent 测试并发场景下的 Redlock
func TestRedlockConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "test:redlock:concurrent"

	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	var (
		counter int64
		wg      sync.WaitGroup
	)

	// 启动 10 个 goroutine 并发执行
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := redlock.WithRedlockRetry(ctx, key, 50, 100*time.Millisecond, func() error {
				// 读取-修改-写入（应该是原子的）
				val := atomic.LoadInt64(&counter)
				time.Sleep(10 * time.Millisecond) // 模拟耗时操作
				atomic.StoreInt64(&counter, val+1)
				return nil
			})

			if err != nil {
				t.Errorf("WithRedlockRetry() error = %v", err)
			}
		}()
	}

	wg.Wait()

	// 验证计数器（应该是 10）
	if counter != 10 {
		t.Errorf("counter = %v, want 10 (lock should ensure atomicity)", counter)
	}
}

// TestRedlockQuorum 测试 Redlock quorum 机制
func TestRedlockQuorum(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	// 验证 quorum 计算
	// 3 个节点，quorum 应该是 2
	if redlock.quorum != 2 {
		t.Errorf("quorum = %v, want 2 for 3 nodes", redlock.quorum)
	}

	ctx := context.Background()
	key := "test:redlock:quorum"

	defer func() {
		for _, client := range clients {
			client.Del(ctx, key)
		}
	}()

	instance := redlock.NewInstance(key)

	// 获取锁
	ok, err := instance.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if !ok {
		t.Error("TryLock() = false, want true")
	}

	// 验证至少在 quorum 数量的节点上获取了锁
	lockedCount := 0
	for _, client := range clients {
		val, err := client.Get(ctx, key)
		if err == nil && val == instance.value {
			lockedCount++
		}
	}

	if lockedCount < redlock.quorum {
		t.Errorf("locked on %v nodes, want >= %v (quorum)", lockedCount, redlock.quorum)
	}

	instance.Unlock(ctx)
}

// TestRedlockNodeFailure 测试节点故障场景
func TestRedlockNodeFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	// 创建包含 1 个失效节点的客户端列表
	clients := getRedlockClients(t)
	defer closeRedlockClients(clients)

	// 添加一个连接失败的客户端（模拟节点故障）
	failedClient, _ := NewClient(&Config{
		Standalone: &NodeConfig{Host: "localhost", Port: 19999, Password: "", DB: 0}, // 不存在的端口
		Pool:       getTestPoolConfig(),
	})
	if failedClient != nil {
		clients = append(clients, failedClient)
	}

	// 4 个节点（3 个正常 + 1 个失效），quorum 应该是 3
	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		t.Fatalf("NewRedlock() error = %v", err)
	}

	if redlock.quorum != 3 {
		t.Errorf("quorum = %v, want 3 for 4 nodes", redlock.quorum)
	}

	ctx := context.Background()
	key := "test:redlock:failure"

	defer func() {
		for i, client := range clients[:3] { // 只清理正常的节点
			client.Del(ctx, key)
			_ = i
		}
	}()

	instance := redlock.NewInstance(key)

	// 尝试获取锁（应该成功，因为 3 个正常节点 >= quorum 3）
	ok, err := instance.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if !ok {
		t.Error("TryLock() = false, want true even with 1 node failure")
	}

	// 验证锁在正常节点上存在
	lockedCount := 0
	for _, client := range clients[:3] { // 只检查正常的节点
		val, err := client.Get(ctx, key)
		if err == nil && val == instance.value {
			lockedCount++
		}
	}

	if lockedCount < 3 {
		t.Errorf("locked on %v nodes, want >= 3", lockedCount)
	}

	instance.Unlock(ctx)
}

// BenchmarkRedlockLock Benchmark Redlock 锁获取
func BenchmarkRedlockLock(b *testing.B) {
	clients := make([]*Client, 3)
	ports := []int{18001, 18002, 18003} // 使用 Redlock 专用节点
	for i := range clients {
		client, err := NewClient(&Config{
			Standalone: &NodeConfig{Host: "localhost", Port: ports[i], Password: "", DB: 0},
			Pool:       getTestPoolConfig(),
		})
		if err != nil {
			b.Fatalf("NewClient() error = %v", err)
		}
		clients[i] = client
		defer client.Close()
	}

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		b.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "bench:redlock"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		instance := redlock.NewInstance(key)
		if err := instance.Lock(ctx); err != nil {
			b.Errorf("Lock() error = %v", err)
		}
		instance.Unlock(ctx)
	}
}

// BenchmarkRedlockWithRedlock Benchmark WithRedlock
func BenchmarkRedlockWithRedlock(b *testing.B) {
	clients := make([]*Client, 3)
	ports := []int{18001, 18002, 18003} // 使用 Redlock 专用节点
	for i := range clients {
		client, err := NewClient(&Config{
			Standalone: &NodeConfig{Host: "localhost", Port: ports[i], Password: "", DB: 0},
			Pool:       getTestPoolConfig(),
		})
		if err != nil {
			b.Fatalf("NewClient() error = %v", err)
		}
		clients[i] = client
		defer client.Close()
	}

	redlock, err := NewRedlock(clients, 5*time.Second)
	if err != nil {
		b.Fatalf("NewRedlock() error = %v", err)
	}

	ctx := context.Background()
	key := "bench:redlock:with"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := redlock.WithRedlock(ctx, key, func() error {
			return nil
		})
		if err != nil {
			b.Errorf("WithRedlock() error = %v", err)
		}
	}
}
