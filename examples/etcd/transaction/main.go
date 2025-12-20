package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/xdooria/pkg/etcd"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	fmt.Println("=== etcd Transaction 事务示例 ===\n")

	client, err := etcd.New(&etcd.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatalf("创建客户端失败: %v", err)
	}
	defer client.Close()

	txn := client.Transaction()
	kv := client.KV()
	ctx := context.Background()

	// 1. 比较并交换 (CAS)
	fmt.Println("【1. 比较并交换 (CAS)】")
	testCompareAndSwap(ctx, txn, kv)

	// 2. 比较版本并更新
	fmt.Println("\n【2. 比较版本并更新】")
	testCompareVersion(ctx, txn, kv)

	// 3. 比较并删除
	fmt.Println("\n【3. 比较并删除】")
	testCompareAndDelete(ctx, txn, kv)

	// 4. 原子递增
	fmt.Println("\n【4. 原子递增】")
	testAtomicIncrement(ctx, txn, kv)

	// 5. 复杂事务
	fmt.Println("\n【5. 复杂事务】")
	testComplexTransaction(ctx, txn, kv)

	fmt.Println("\n✅ 示例完成")
}

func testCompareAndSwap(ctx context.Context, txn *etcd.Transaction, kv *etcd.KV) {
	key := "/txn/cas"

	// 初始化
	if err := kv.Put(ctx, key, "100"); err != nil {
		log.Printf("  ❌ 初始化失败: %v", err)
		return
	}
	fmt.Printf("  初始值: %s = 100\n", key)

	// CAS: 100 -> 200
	success, err := txn.CompareAndSwapTxn(ctx, key, "100", "200")
	if err != nil {
		log.Printf("  ❌ CAS 失败: %v", err)
		return
	}

	if success {
		fmt.Println("  ✓ CAS 成功: 100 -> 200")
		value, _ := kv.Get(ctx, key)
		fmt.Printf("  当前值: %s\n", string(value.Value))
	} else {
		fmt.Println("  ✗ CAS 失败（值不匹配）")
	}

	// 再次 CAS，应该失败
	success, err = txn.CompareAndSwapTxn(ctx, key, "100", "300")
	if err != nil {
		log.Printf("  ❌ CAS 失败: %v", err)
		return
	}

	if !success {
		fmt.Println("  ✓ CAS 失败（预期）: 旧值不匹配")
	} else {
		fmt.Println("  ✗ CAS 意外成功")
	}

	// 清理
	kv.Delete(ctx, key)
}

func testCompareVersion(ctx context.Context, txn *etcd.Transaction, kv *etcd.KV) {
	key := "/txn/version"

	// 初始化
	if err := kv.Put(ctx, key, "v1"); err != nil {
		log.Printf("  ❌ 初始化失败: %v", err)
		return
	}

	// 获取当前版本
	value, err := kv.Get(ctx, key)
	if err != nil {
		log.Printf("  ❌ 获取键值失败: %v", err)
		return
	}
	fmt.Printf("  当前值: %s (version=%d)\n", string(value.Value), value.Version)

	// 比较版本并更新
	success, err := txn.CompareVersionAndPut(ctx, key, value.Version, "v2")
	if err != nil {
		log.Printf("  ❌ 更新失败: %v", err)
		return
	}

	if success {
		fmt.Println("  ✓ 版本匹配，更新成功")
		value, _ = kv.Get(ctx, key)
		fmt.Printf("  新值: %s (version=%d)\n", string(value.Value), value.Version)
	} else {
		fmt.Println("  ✗ 版本不匹配，更新失败")
	}

	// 使用旧版本更新，应该失败
	success, err = txn.CompareVersionAndPut(ctx, key, 1, "v3")
	if err != nil {
		log.Printf("  ❌ 更新失败: %v", err)
		return
	}

	if !success {
		fmt.Println("  ✓ 版本不匹配，更新失败（预期）")
	} else {
		fmt.Println("  ✗ 更新意外成功")
	}

	// 清理
	kv.Delete(ctx, key)
}

func testCompareAndDelete(ctx context.Context, txn *etcd.Transaction, kv *etcd.KV) {
	key := "/txn/delete"

	// 初始化
	if err := kv.Put(ctx, key, "delete-me"); err != nil {
		log.Printf("  ❌ 初始化失败: %v", err)
		return
	}
	fmt.Printf("  初始值: %s = delete-me\n", key)

	// 比较并删除（值匹配）
	success, err := txn.CompareAndDelete(ctx, key, "delete-me")
	if err != nil {
		log.Printf("  ❌ 删除失败: %v", err)
		return
	}

	if success {
		fmt.Println("  ✓ 值匹配，删除成功")

		// 验证已删除
		if _, err := kv.Get(ctx, key); err == etcd.ErrKeyNotFound {
			fmt.Println("  ✓ 键已删除")
		}
	} else {
		fmt.Println("  ✗ 值不匹配，删除失败")
	}

	// 尝试删除不存在的键
	success, err = txn.CompareAndDelete(ctx, key, "delete-me")
	if err != nil {
		log.Printf("  ❌ 删除失败: %v", err)
		return
	}

	if !success {
		fmt.Println("  ✓ 键不存在，删除失败（预期）")
	}
}

func testAtomicIncrement(ctx context.Context, txn *etcd.Transaction, kv *etcd.KV) {
	key := "/txn/counter"

	// 初始化计数器
	fmt.Println("  初始化计数器...")
	newValue, err := txn.AtomicIncrement(ctx, key, 1)
	if err != nil {
		log.Printf("  ❌ 递增失败: %v", err)
		return
	}
	fmt.Printf("  ✓ 初始值: %d\n", newValue)

	// 递增
	newValue, err = txn.AtomicIncrement(ctx, key, 5)
	if err != nil {
		log.Printf("  ❌ 递增失败: %v", err)
		return
	}
	fmt.Printf("  ✓ +5: %d\n", newValue)

	newValue, err = txn.AtomicIncrement(ctx, key, 10)
	if err != nil {
		log.Printf("  ❌ 递增失败: %v", err)
		return
	}
	fmt.Printf("  ✓ +10: %d\n", newValue)

	// 递减
	newValue, err = txn.AtomicIncrement(ctx, key, -3)
	if err != nil {
		log.Printf("  ❌ 递减失败: %v", err)
		return
	}
	fmt.Printf("  ✓ -3: %d\n", newValue)

	// 清理
	kv.Delete(ctx, key)
}

func testComplexTransaction(ctx context.Context, txn *etcd.Transaction, kv *etcd.KV) {
	key1 := "/txn/account1"
	key2 := "/txn/account2"

	// 初始化两个账户
	kv.Put(ctx, key1, "100")
	kv.Put(ctx, key2, "50")
	fmt.Println("  初始状态:")
	fmt.Println("    account1 = 100")
	fmt.Println("    account2 = 50")

	// 场景：从 account1 转账 30 到 account2
	// 条件：account1 >= 30
	fmt.Println("\n  执行转账: account1 -> account2 (30)...")

	resp, err := txn.Txn(ctx).
		If(
			clientv3.Compare(clientv3.Value(key1), "=", "100"),
		).
		Then(
			clientv3.OpPut(key1, "70"),  // 100 - 30
			clientv3.OpPut(key2, "80"),  // 50 + 30
		).
		Else(
			clientv3.OpGet(key1),
			clientv3.OpGet(key2),
		).
		Commit()

	if err != nil {
		log.Printf("  ❌ 事务失败: %v", err)
		return
	}

	if resp.Succeeded {
		fmt.Println("  ✓ 转账成功")

		// 验证结果
		v1, _ := kv.Get(ctx, key1)
		v2, _ := kv.Get(ctx, key2)
		fmt.Println("\n  转账后状态:")
		fmt.Printf("    account1 = %s\n", string(v1.Value))
		fmt.Printf("    account2 = %s\n", string(v2.Value))
	} else {
		fmt.Println("  ✗ 转账失败（条件不满足）")
		fmt.Println("  当前账户状态:")
		if len(resp.Responses) >= 2 {
			if kvs, ok := resp.Responses[0].([]*etcd.KeyValue); ok && len(kvs) > 0 {
				fmt.Printf("    account1 = %s\n", string(kvs[0].Value))
			}
			if kvs, ok := resp.Responses[1].([]*etcd.KeyValue); ok && len(kvs) > 0 {
				fmt.Printf("    account2 = %s\n", string(kvs[0].Value))
			}
		}
	}

	// 清理
	kv.Delete(ctx, key1)
	kv.Delete(ctx, key2)
}
