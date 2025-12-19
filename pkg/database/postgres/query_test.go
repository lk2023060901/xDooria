package postgres

import (
	"context"
	"testing"
	"time"
)

// 测试用的结构体
type TestUser struct {
	ID        int64     `db:"id"`
	Username  string    `db:"username"`
	Email     string    `db:"email"`
	Age       int       `db:"age"`
	Balance   float64   `db:"balance"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
}

// setupTestTable 创建测试表
func setupTestTable(t *testing.T, client *Client) {
	ctx := context.Background()

	// 删除旧表（如果存在）
	_, _ = client.Exec(ctx, "DROP TABLE IF EXISTS test_users")

	// 创建测试表
	createTableSQL := `
		CREATE TABLE test_users (
			id SERIAL PRIMARY KEY,
			username VARCHAR(50) NOT NULL UNIQUE,
			email VARCHAR(100) NOT NULL,
			age INTEGER NOT NULL,
			balance NUMERIC(10, 2) DEFAULT 0.00,
			is_active BOOLEAN DEFAULT true,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)
	`

	_, err := client.Exec(ctx, createTableSQL)
	if err != nil {
		t.Fatalf("setupTestTable() failed to create table: %v", err)
	}
}

// cleanupTestTable 清理测试表
func cleanupTestTable(t *testing.T, client *Client) {
	ctx := context.Background()
	_, _ = client.Exec(ctx, "DROP TABLE IF EXISTS test_users")
}

// TestExec 测试 Exec 执行
func TestExec(t *testing.T) {
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

	// 测试插入
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	rowsAffected, err := client.Exec(ctx, insertSQL, "testuser1", "test1@example.com", 25, 100.50)
	if err != nil {
		t.Errorf("Exec() insert error = %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Exec() insert rowsAffected = %d, want 1", rowsAffected)
	}

	// 测试更新
	updateSQL := "UPDATE test_users SET age = $1 WHERE username = $2"
	rowsAffected, err = client.Exec(ctx, updateSQL, 26, "testuser1")
	if err != nil {
		t.Errorf("Exec() update error = %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Exec() update rowsAffected = %d, want 1", rowsAffected)
	}

	// 测试删除
	deleteSQL := "DELETE FROM test_users WHERE username = $1"
	rowsAffected, err = client.Exec(ctx, deleteSQL, "testuser1")
	if err != nil {
		t.Errorf("Exec() delete error = %v", err)
	}
	if rowsAffected != 1 {
		t.Errorf("Exec() delete rowsAffected = %d, want 1", rowsAffected)
	}
}

// TestQueryOne 测试 QueryOne 查询单条记录
func TestQueryOne(t *testing.T) {
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
	insertSQL := "INSERT INTO test_users (username, email, age, balance, is_active) VALUES ($1, $2, $3, $4, $5)"
	_, err = client.Exec(ctx, insertSQL, "queryuser1", "query1@example.com", 30, 250.75, true)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 测试查询
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
	user, err := QueryOne[TestUser](client, ctx, querySQL, "queryuser1")
	if err != nil {
		t.Errorf("QueryOne() error = %v", err)
	}

	if user == nil {
		t.Fatal("QueryOne() returned nil user")
	}

	if user.Username != "queryuser1" {
		t.Errorf("QueryOne() username = %v, want queryuser1", user.Username)
	}
	if user.Email != "query1@example.com" {
		t.Errorf("QueryOne() email = %v, want query1@example.com", user.Email)
	}
	if user.Age != 30 {
		t.Errorf("QueryOne() age = %v, want 30", user.Age)
	}
	if user.Balance != 250.75 {
		t.Errorf("QueryOne() balance = %v, want 250.75", user.Balance)
	}
	if !user.IsActive {
		t.Errorf("QueryOne() is_active = %v, want true", user.IsActive)
	}
}

// TestQueryOneNotFound 测试 QueryOne 查询不存在的记录
func TestQueryOneNotFound(t *testing.T) {
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

	// 查询不存在的用户
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"
	user, err := QueryOne[TestUser](client, ctx, querySQL, "nonexistent")
	if err != ErrNoRows {
		t.Errorf("QueryOne() error = %v, want ErrNoRows", err)
	}
	if user != nil {
		t.Errorf("QueryOne() should return nil for non-existent user")
	}
}

// TestQueryAll 测试 QueryAll 查询多条记录
func TestQueryAll(t *testing.T) {
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

	// 插入多条测试数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	testData := []struct {
		username string
		email    string
		age      int
		balance  float64
	}{
		{"user1", "user1@example.com", 20, 100.00},
		{"user2", "user2@example.com", 25, 200.00},
		{"user3", "user3@example.com", 30, 300.00},
		{"user4", "user4@example.com", 35, 400.00},
	}

	for _, data := range testData {
		_, err := client.Exec(ctx, insertSQL, data.username, data.email, data.age, data.balance)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// 测试查询所有记录
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users ORDER BY age"
	users, err := QueryAll[TestUser](client, ctx, querySQL)
	if err != nil {
		t.Errorf("QueryAll() error = %v", err)
	}

	if len(users) != 4 {
		t.Fatalf("QueryAll() returned %d users, want 4", len(users))
	}

	// 验证排序和数据
	if users[0].Username != "user1" || users[0].Age != 20 {
		t.Errorf("QueryAll() first user incorrect")
	}
	if users[3].Username != "user4" || users[3].Age != 35 {
		t.Errorf("QueryAll() last user incorrect")
	}

	// 测试带条件查询
	querySQL = "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE age >= $1 ORDER BY age"
	users, err = QueryAll[TestUser](client, ctx, querySQL, 25)
	if err != nil {
		t.Errorf("QueryAll() with condition error = %v", err)
	}

	if len(users) != 3 {
		t.Errorf("QueryAll() with condition returned %d users, want 3", len(users))
	}
}

// TestQueryAllEmpty 测试 QueryAll 查询空结果
func TestQueryAllEmpty(t *testing.T) {
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

	// 查询空表
	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users"
	users, err := QueryAll[TestUser](client, ctx, querySQL)
	if err != nil {
		t.Errorf("QueryAll() on empty table error = %v", err)
	}

	if len(users) != 0 {
		t.Errorf("QueryAll() on empty table returned %d users, want 0", len(users))
	}
}

// TestInsertBatch 测试批量插入
func TestInsertBatch(t *testing.T) {
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

	// 准备批量插入数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	argsList := [][]any{
		{"batch1", "batch1@example.com", 20, 100.00},
		{"batch2", "batch2@example.com", 25, 200.00},
		{"batch3", "batch3@example.com", 30, 300.00},
		{"batch4", "batch4@example.com", 35, 400.00},
		{"batch5", "batch5@example.com", 40, 500.00},
	}

	// 执行批量插入
	rowsAffected, err := client.InsertBatch(ctx, insertSQL, argsList)
	if err != nil {
		t.Errorf("InsertBatch() error = %v", err)
	}

	if rowsAffected != 5 {
		t.Errorf("InsertBatch() rowsAffected = %d, want 5", rowsAffected)
	}

	// 验证插入结果
	querySQL := "SELECT COUNT(*) FROM test_users"
	var count int64
	err = client.master.QueryRow(ctx, querySQL).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify batch insert: %v", err)
	}

	if count != 5 {
		t.Errorf("InsertBatch() inserted %d rows, want 5", count)
	}
}

// TestUpdateBatch 测试批量更新
func TestUpdateBatch(t *testing.T) {
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

	// 先插入一些数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	argsList := [][]any{
		{"update1", "update1@example.com", 20, 100.00},
		{"update2", "update2@example.com", 25, 200.00},
		{"update3", "update3@example.com", 30, 300.00},
	}
	_, err = client.InsertBatch(ctx, insertSQL, argsList)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 批量更新
	updateSQL := "UPDATE test_users SET balance = $1 WHERE username = $2"
	updateArgsList := [][]any{
		{150.00, "update1"},
		{250.00, "update2"},
		{350.00, "update3"},
	}

	rowsAffected, err := client.UpdateBatch(ctx, updateSQL, updateArgsList)
	if err != nil {
		t.Errorf("UpdateBatch() error = %v", err)
	}

	if rowsAffected != 3 {
		t.Errorf("UpdateBatch() rowsAffected = %d, want 3", rowsAffected)
	}

	// 验证更新结果
	user, err := QueryOne[TestUser](client, ctx, "SELECT * FROM test_users WHERE username = $1", "update1")
	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}
	if user.Balance != 150.00 {
		t.Errorf("UpdateBatch() balance = %v, want 150.00", user.Balance)
	}
}

// TestDeleteBatch 测试批量删除
func TestDeleteBatch(t *testing.T) {
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

	// 先插入一些数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	argsList := [][]any{
		{"delete1", "delete1@example.com", 20, 100.00},
		{"delete2", "delete2@example.com", 25, 200.00},
		{"delete3", "delete3@example.com", 30, 300.00},
		{"delete4", "delete4@example.com", 35, 400.00},
	}
	_, err = client.InsertBatch(ctx, insertSQL, argsList)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// 批量删除
	deleteSQL := "DELETE FROM test_users WHERE username = $1"
	deleteArgsList := [][]any{
		{"delete1"},
		{"delete2"},
		{"delete3"},
	}

	rowsAffected, err := client.DeleteBatch(ctx, deleteSQL, deleteArgsList)
	if err != nil {
		t.Errorf("DeleteBatch() error = %v", err)
	}

	if rowsAffected != 3 {
		t.Errorf("DeleteBatch() rowsAffected = %d, want 3", rowsAffected)
	}

	// 验证删除结果
	users, err := QueryAll[TestUser](client, ctx, "SELECT * FROM test_users")
	if err != nil {
		t.Fatalf("Failed to verify delete: %v", err)
	}
	if len(users) != 1 {
		t.Errorf("DeleteBatch() left %d users, want 1", len(users))
	}
	if users[0].Username != "delete4" {
		t.Errorf("DeleteBatch() wrong user remaining: %s", users[0].Username)
	}
}

// BenchmarkQueryOne Benchmark 单条查询
func BenchmarkQueryOne(b *testing.B) {
	client, err := New(standaloneConfig)
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	// 使用 testing.T wrapper 来调用 setup
	t := &testing.T{}
	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()

	// 插入测试数据
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"
	_, _ = client.Exec(ctx, insertSQL, "benchuser", "bench@example.com", 25, 100.00)

	querySQL := "SELECT id, username, email, age, balance, is_active, created_at FROM test_users WHERE username = $1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = QueryOne[TestUser](client, ctx, querySQL, "benchuser")
	}
}

// BenchmarkInsertBatch Benchmark 批量插入
func BenchmarkInsertBatch(b *testing.B) {
	client, err := New(standaloneConfig)
	if err != nil {
		b.Fatalf("New() error = %v", err)
	}
	defer client.Close()

	t := &testing.T{}
	setupTestTable(t, client)
	defer cleanupTestTable(t, client)

	ctx := context.Background()
	insertSQL := "INSERT INTO test_users (username, email, age, balance) VALUES ($1, $2, $3, $4)"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		argsList := [][]any{
			{b.Name() + "_user1", "bench1@example.com", 20, 100.00},
			{b.Name() + "_user2", "bench2@example.com", 25, 200.00},
			{b.Name() + "_user3", "bench3@example.com", 30, 300.00},
		}
		_, _ = client.InsertBatch(ctx, insertSQL, argsList)

		// 清理以便下次迭代
		_, _ = client.Exec(ctx, "DELETE FROM test_users")
	}
}
