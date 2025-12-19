package redis

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestNewLock 测试创建锁
func TestNewLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	lock := NewLock(client, "test:lock", 5*time.Second)
	if lock == nil {
		t.Error("NewLock() returned nil")
	}

	if lock.key != "test:lock" {
		t.Errorf("lock.key = %v, want test:lock", lock.key)
	}

	if lock.ttl != 5*time.Second {
		t.Errorf("lock.ttl = %v, want 5s", lock.ttl)
	}

	if lock.value == "" {
		t.Error("lock.value should not be empty")
	}
}

// TestLockUnlock 测试锁的获取和释放
func TestLockUnlock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:basic"

	defer client.Del(ctx, key)

	lock := NewLock(client, key, 5*time.Second)

	// Test Lock
	if err := lock.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}

	// 验证锁存在
	isLocked, err := client.IsLocked(ctx, key)
	if err != nil {
		t.Errorf("IsLocked() error = %v", err)
	}
	if !isLocked {
		t.Error("IsLocked() = false, want true")
	}

	// Test Unlock
	if err := lock.Unlock(ctx); err != nil {
		t.Errorf("Unlock() error = %v", err)
	}

	// 验证锁已释放
	isLocked, err = client.IsLocked(ctx, key)
	if err != nil {
		t.Errorf("IsLocked() error = %v", err)
	}
	if isLocked {
		t.Error("IsLocked() = true, want false after unlock")
	}
}

// TestTryLock 测试非阻塞获取锁
func TestTryLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:trylock"

	defer client.Del(ctx, key)

	lock1 := NewLock(client, key, 5*time.Second)
	lock2 := NewLock(client, key, 5*time.Second)

	// lock1 获取锁
	ok, err := lock1.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if !ok {
		t.Error("TryLock() = false, want true for first lock")
	}

	// lock2 尝试获取锁（应该失败）
	ok, err = lock2.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if ok {
		t.Error("TryLock() = true, want false for concurrent lock")
	}

	// 释放 lock1
	if err := lock1.Unlock(ctx); err != nil {
		t.Errorf("Unlock() error = %v", err)
	}

	// lock2 再次尝试获取锁（应该成功）
	ok, err = lock2.TryLock(ctx)
	if err != nil {
		t.Errorf("TryLock() error = %v", err)
	}
	if !ok {
		t.Error("TryLock() = false, want true after lock1 released")
	}

	lock2.Unlock(ctx)
}

// TestLockWithRetry 测试带重试的锁获取
func TestLockWithRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:retry"

	defer client.Del(ctx, key)

	lock1 := NewLock(client, key, 2*time.Second)
	lock2 := NewLock(client, key, 5*time.Second)

	// lock1 获取锁
	if err := lock1.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}

	// 启动 goroutine 在 1 秒后释放 lock1
	go func() {
		time.Sleep(1 * time.Second)
		lock1.Unlock(ctx)
	}()

	// lock2 尝试获取锁（最多重试 5 次，每次间隔 500ms）
	start := time.Now()
	err = lock2.LockWithRetry(ctx, 500*time.Millisecond, 5)
	elapsed := time.Since(start)

	if err != nil {
		t.Errorf("LockWithRetry() error = %v", err)
	}

	// 应该在 1-2 秒内获取到锁
	if elapsed < 1*time.Second || elapsed > 3*time.Second {
		t.Errorf("LockWithRetry() took %v, expected 1-3s", elapsed)
	}

	lock2.Unlock(ctx)
}

// TestLockRefresh 测试锁续期
func TestLockRefresh(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:refresh"

	defer client.Del(ctx, key)

	lock := NewLock(client, key, 2*time.Second)

	// 获取锁
	if err := lock.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}
	defer lock.Unlock(ctx)

	// 等待 1 秒
	time.Sleep(1 * time.Second)

	// 刷新锁
	if err := lock.Refresh(ctx); err != nil {
		t.Errorf("Refresh() error = %v", err)
	}

	// 检查 TTL（应该接近 2 秒）
	ttl, err := client.TTL(ctx, key)
	if err != nil {
		t.Errorf("TTL() error = %v", err)
	}

	if ttl < 1500*time.Millisecond || ttl > 2*time.Second {
		t.Errorf("TTL after Refresh() = %v, expected ~2s", ttl)
	}
}

// TestWithLock 测试 WithLock 便捷方法
func TestWithLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:withlock"

	defer client.Del(ctx, key)

	var counter int
	err = client.WithLock(ctx, key, 5*time.Second, func() error {
		counter++
		return nil
	})

	if err != nil {
		t.Errorf("WithLock() error = %v", err)
	}

	if counter != 1 {
		t.Errorf("counter = %v, want 1", counter)
	}

	// 验证锁已释放
	isLocked, err := client.IsLocked(ctx, key)
	if err != nil {
		t.Errorf("IsLocked() error = %v", err)
	}
	if isLocked {
		t.Error("lock should be released after WithLock()")
	}
}

// TestWithLockConcurrent 测试并发场景下的 WithLock
func TestWithLockConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:concurrent"

	defer client.Del(ctx, key)

	var (
		counter int64
		wg      sync.WaitGroup
	)

	// 启动 10 个 goroutine 并发执行
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			err := client.WithLockRetry(ctx, key, 5*time.Second, 100*time.Millisecond, 50, func() error {
				// 读取-修改-写入（应该是原子的）
				val := atomic.LoadInt64(&counter)
				time.Sleep(10 * time.Millisecond) // 模拟耗时操作
				atomic.StoreInt64(&counter, val+1)
				return nil
			})

			if err != nil {
				t.Errorf("WithLockRetry() error = %v", err)
			}
		}()
	}

	wg.Wait()

	// 验证计数器（应该是 10）
	if counter != 10 {
		t.Errorf("counter = %v, want 10 (lock should ensure atomicity)", counter)
	}
}

// TestUnlockWithoutHoldingLock 测试释放未持有的锁
func TestUnlockWithoutHoldingLock(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:notheld"

	defer client.Del(ctx, key)

	lock1 := NewLock(client, key, 5*time.Second)
	lock2 := NewLock(client, key, 5*time.Second)

	// lock1 获取锁
	if err := lock1.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}

	// lock2 尝试释放锁（应该失败，因为不是持有者）
	err = lock2.Unlock(ctx)
	if err != ErrLockNotHeld {
		t.Errorf("Unlock() error = %v, want ErrLockNotHeld", err)
	}

	// lock1 释放锁（应该成功）
	if err := lock1.Unlock(ctx); err != nil {
		t.Errorf("Unlock() error = %v", err)
	}
}

// TestLockExpiration 测试锁自动过期
func TestLockExpiration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := NewClient(standaloneConfig)
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "test:lock:expiration"

	defer client.Del(ctx, key)

	lock := NewLock(client, key, 1*time.Second)

	// 获取锁
	if err := lock.Lock(ctx); err != nil {
		t.Errorf("Lock() error = %v", err)
	}

	// 验证锁存在
	isLocked, err := client.IsLocked(ctx, key)
	if err != nil {
		t.Errorf("IsLocked() error = %v", err)
	}
	if !isLocked {
		t.Error("IsLocked() = false, want true")
	}

	// 等待锁过期（1.5 秒）
	time.Sleep(1500 * time.Millisecond)

	// 验证锁已过期
	isLocked, err = client.IsLocked(ctx, key)
	if err != nil {
		t.Errorf("IsLocked() error = %v", err)
	}
	if isLocked {
		t.Error("IsLocked() = true, want false after expiration")
	}
}

// BenchmarkLock Benchmark 锁获取
func BenchmarkLock(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "bench:lock"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		lock := NewLock(client, key, 5*time.Second)
		if err := lock.Lock(ctx); err != nil {
			b.Errorf("Lock() error = %v", err)
		}
		lock.Unlock(ctx)
	}
}

// BenchmarkWithLock Benchmark WithLock
func BenchmarkWithLock(b *testing.B) {
	client, err := NewClient(standaloneConfig)
	if err != nil {
		b.Fatalf("NewClient() error = %v", err)
	}
	defer client.Close()

	ctx := context.Background()
	key := "bench:withlock"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := client.WithLock(ctx, key, 5*time.Second, func() error {
			return nil
		})
		if err != nil {
			b.Errorf("WithLock() error = %v", err)
		}
	}
}
