package postgres

import (
	"context"
	"errors"
	"testing"
)

// TestBeginTxCommit 测试事务提交
func TestBeginTxCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 开始事务
	tx, err := client.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	// 在事务中插入数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	rowsAffected, err := tx.Exec(ctx, insertSQL, "txuser1", "tx1@example.com", 25, 100.00)
	if err != nil {
		t.Errorf("Tx.Exec() error = %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Tx.Exec() rowsAffected = %d, want 1", rowsAffected)
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("Tx.Commit() error = %v", err)
	}

	// 验证数据已提交
	user, err := QueryOne[TestUser](client, ctx, "SELECT * FROM test_users WHERE username = $1", "txuser1")
	if err != nil {
		t.Errorf("Failed to verify committed data: %v", err)
	}
	if user == nil {
		t.Error("Committed data not found")
	}
}

// TestBeginTxRollback 测试事务回滚
func TestBeginTxRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 开始事务
	tx, err := client.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}

	// 在事务中插入数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	_, err = tx.Exec(ctx, insertSQL, "txuser2", "tx2@example.com", 30, 200.00)
	if err != nil {
		t.Errorf("Tx.Exec() error = %v", err)
	}

	// 回滚事务
	if err := tx.Rollback(ctx); err != nil {
		t.Errorf("Tx.Rollback() error = %v", err)
	}

	// 验证数据未提交
	user, err := QueryOne[TestUser](client, ctx, "SELECT * FROM test_users WHERE username = $1", "txuser2")
	if err != ErrNoRows {
		t.Errorf("Expected ErrNoRows after rollback, got: %v", err)
	}
	if user != nil {
		t.Error("Rolled back data should not exist")
	}
}

// TestTxQueryOne 测试事务中的查询
func TestTxQueryOne(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 先插入一条数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	_, err = client.Exec(ctx, insertSQL, "txquery1", "txquery1@example.com", 25, 150.00)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 开始事务
	tx, err := client.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer tx.Rollback(ctx)

	// 在事务中查询
	var user TestUser
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
	err = tx.QueryOne(ctx, &user, querySQL, "txquery1")
	if err != nil {
		t.Errorf("Tx.QueryOne() error = %v", err)
	}

	if user.Username != "txquery1" {
		t.Errorf("Tx.QueryOne() username = %v, want txquery1", user.Username)
	}

	// 在事务中更新
	updateSQL := "UPDATE test_users SET balance = $1 WHERE username = $2"
	_, err = tx.Exec(ctx, updateSQL, 200.00, "txquery1")
	if err != nil {
		t.Errorf("Tx.Exec() update error = %v", err)
	}

	// 在事务中再次查询，应该看到更新后的值
	err = tx.QueryOne(ctx, &user, querySQL, "txquery1")
	if err != nil {
		t.Errorf("Tx.QueryOne() after update error = %v", err)
	}

	if user.Balance != 200.00 {
		t.Errorf("Tx.QueryOne() balance = %v, want 200.00", user.Balance)
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("Tx.Commit() error = %v", err)
	}
}

// TestTxQueryAll 测试事务中的批量查询
func TestTxQueryAll(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 插入测试数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	argsList := [][]any{
		{"txall1", "txall1@example.com", 20, 100.00},
		{"txall2", "txall2@example.com", 25, 200.00},
		{"txall3", "txall3@example.com", 30, 300.00},
	}
	_, err = client.InsertBatch(ctx, insertSQL, argsList)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 开始事务
	tx, err := client.BeginTx(ctx)
	if err != nil {
		t.Fatalf("BeginTx() error = %v", err)
	}
	defer tx.Rollback(ctx)

	// 在事务中查询所有数据
	var users []*TestUser
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users ORDER BY age"
	err = tx.QueryAll(ctx, &users, querySQL)
	if err != nil {
		t.Errorf("Tx.QueryAll() error = %v", err)
	}

	if len(users) != 3 {
		t.Errorf("Tx.QueryAll() returned %d users, want 3", len(users))
	}

	// 提交事务
	if err := tx.Commit(ctx); err != nil {
		t.Errorf("Tx.Commit() error = %v", err)
	}
}

// TestWithTx 测试 WithTx 便捷方法
func TestWithTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 使用 WithTx 执行事务操作
	err = client.WithTx(ctx, func(tx Tx) error {
		// 插入数据
		insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
		_, err := tx.Exec(ctx, insertSQL, "withtx1", "withtx1@example.com", 28, 180.00)
		if err != nil {
			return err
		}

		// 查询验证
		var user TestUser
		querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
		err = tx.QueryOne(ctx, &user, querySQL, "withtx1")
		if err != nil {
			return err
		}

		if user.Username != "withtx1" {
			t.Errorf("WithTx query username = %v, want withtx1", user.Username)
		}

		return nil
	})

	if err != nil {
		t.Errorf("WithTx() error = %v", err)
	}

	// 验证数据已提交
	user, err := QueryOne[TestUser](client, ctx, "SELECT * FROM test_users WHERE username = $1", "withtx1")
	if err != nil {
		t.Errorf("Failed to verify committed data: %v", err)
	}
	if user == nil {
		t.Error("Committed data not found")
	}
}

// TestWithTxRollback 测试 WithTx 自动回滚
func TestWithTxRollback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 使用 WithTx 执行事务操作，返回错误会自动回滚
	testErr := errors.New("test error for rollback")
	err = client.WithTx(ctx, func(tx Tx) error {
		// 插入数据
		insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
		_, err := tx.Exec(ctx, insertSQL, "withtx2", "withtx2@example.com", 32, 220.00)
		if err != nil {
			return err
		}

		// 返回错误触发回滚
		return testErr
	})

	if err != testErr {
		t.Errorf("WithTx() error = %v, want %v", err, testErr)
	}

	// 验证数据未提交
	user, err := QueryOne[TestUser](client, ctx, "SELECT * FROM test_users WHERE username = $1", "withtx2")
	if err != ErrNoRows {
		t.Errorf("Expected ErrNoRows after rollback, got: %v", err)
	}
	if user != nil {
		t.Error("Rolled back data should not exist")
	}
}

// TestTxIsolationLevel 测试事务隔离级别
func TestTxIsolationLevel(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 插入初始数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	_, err = client.Exec(ctx, insertSQL, "isolation1", "iso1@example.com", 25, 100.00)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 测试不同的隔离级别
	tests := []struct {
		name     string
		isoLevel TxIsolationLevel
	}{
		{"read uncommitted", TxIsolationLevelReadUncommitted},
		{"read committed", TxIsolationLevelReadCommitted},
		{"repeatable read", TxIsolationLevelRepeatableRead},
		{"serializable", TxIsolationLevelSerializable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 使用指定的隔离级别开始事务
			tx, err := client.BeginTxWithOptions(ctx, TxOptions{
				IsoLevel:   tt.isoLevel,
				AccessMode: TxAccessModeReadWrite,
			})
			if err != nil {
				t.Errorf("BeginTxWithOptions(%v) error = %v", tt.isoLevel, err)
				return
			}

			// 在事务中查询
			var user TestUser
			querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
			err = tx.QueryOne(ctx, &user, querySQL, "isolation1")
			if err != nil {
				t.Errorf("Tx.QueryOne() error = %v", err)
			}

			// 提交事务
			if err := tx.Commit(ctx); err != nil {
				t.Errorf("Tx.Commit() error = %v", err)
			}
		})
	}
}

// TestTxAccessMode 测试事务访问模式
func TestTxAccessMode(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 插入测试数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	_, err = client.Exec(ctx, insertSQL, "access1", "access1@example.com", 25, 100.00)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 测试只读事务
	tx, err := client.BeginTxWithOptions(ctx, TxOptions{
		IsoLevel:   TxIsolationLevelReadCommitted,
		AccessMode: TxAccessModeReadOnly,
	})
	if err != nil {
		t.Fatalf("BeginTxWithOptions(read-only) error = %v", err)
	}

	// 只读事务可以查询
	var user TestUser
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
	err = tx.QueryOne(ctx, &user, querySQL, "access1")
	if err != nil {
		t.Errorf("Read-only tx QueryOne() error = %v", err)
	}

	// 只读事务不能写入（应该失败）
	updateSQL := "UPDATE test_users SET balance = $1 WHERE username = $2"
	_, err = tx.Exec(ctx, updateSQL, 200.00, "access1")
	if err == nil {
		t.Error("Read-only tx should not allow write operations")
	}

	// 回滚（或提交都可以，因为没有修改）
	_ = tx.Rollback(ctx)
}

// TestConcurrentTransactions 测试并发事务
func TestConcurrentTransactions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	client, err := New(standaloneConfig)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 插入初始数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	_, err = client.Exec(ctx, insertSQL, "concurrent1", "con1@example.com", 25, 1000.00)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 并发执行多个事务
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()

			err := client.WithTx(ctx, func(tx Tx) error {
				// 读取当前余额
				var user TestUser
				querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
				err := tx.QueryOne(ctx, &user, querySQL, "concurrent1")
				if err != nil {
					return err
				}

				// 更新余额（减少 10）
				updateSQL := "UPDATE test_users SET balance = balance - 10 WHERE username = $1"
				_, err = tx.Exec(ctx, updateSQL, "concurrent1")
				return err
			})

			if err != nil {
				t.Errorf("Concurrent tx %d error: %v", id, err)
			}
		}(i)
	}

	// 等待所有事务完成
	for i := 0; i < 5; i++ {
		<-done
	}

	// 验证最终余额（应该是 1000 - 5*10 = 950）
	user, err := QueryOne[TestUser](client, ctx, "SELECT * FROM test_users WHERE username = $1", "concurrent1")
	if err != nil {
		t.Fatalf("Failed to verify concurrent transactions: %v", err)
	}

	if user.Balance != 950.00 {
		t.Errorf("Concurrent transactions result: balance = %v, want 950.00", user.Balance)
	}
}

// BenchmarkWithTx Benchmark 事务操作
func BenchmarkWithTx(b *testing.B) {
	client, err := New(standaloneConfig)
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	t := &testing.T{}
	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.WithTx(ctx, func(tx Tx) error {
			insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
			_, err := tx.Exec(ctx, insertSQL, "benchtx", "benchtx@example.com", 25, 100.00)
			if err != nil {
				return err
			}

			deleteSQL := "DELETE FROM test_users WHERE username = $1"
			_, err = tx.Exec(ctx, deleteSQL, "benchtx")
			return err
		})
	}
}
