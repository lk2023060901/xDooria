package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// Tx 事务接口
type Tx interface {
	// QueryOne 查询单条记录（通过 dest 参数接收结果）
	QueryOne(ctx context.Context, dest any, sql string, args ...any) error
	// QueryAll 查询多条记录（通过 dest 参数接收结果）
	QueryAll(ctx context.Context, dest any, sql string, args ...any) error
	// Exec 执行写操作
	Exec(ctx context.Context, sql string, args ...any) (int64, error)
	// Exists 检查记录是否存在
	Exists(ctx context.Context, sql string, args ...any) (bool, error)
	// Insert 插入单条记录
	Insert(ctx context.Context, sql string, args ...any) (int64, error)
	// InsertBatch 批量插入记录
	InsertBatch(ctx context.Context, sql string, argsList [][]any) (int64, error)
	// Update 更新记录
	Update(ctx context.Context, sql string, args ...any) (int64, error)
	// UpdateBatch 批量更新记录
	UpdateBatch(ctx context.Context, sql string, argsList [][]any) (int64, error)
	// Delete 删除记录
	Delete(ctx context.Context, sql string, args ...any) (int64, error)
	// DeleteBatch 批量删除记录
	DeleteBatch(ctx context.Context, sql string, argsList [][]any) (int64, error)
	// Commit 提交事务
	Commit(ctx context.Context) error
	// Rollback 回滚事务
	Rollback(ctx context.Context) error
}

// txWrapper 事务包装器
type txWrapper struct {
	tx pgx.Tx
}

// QueryOne 查询单条记录
func (t *txWrapper) QueryOne(ctx context.Context, dest any, sql string, args ...any) error {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return ErrNoRows
	}

	return scanRowToStruct(rows, dest)
}

// QueryAll 查询多条记录
func (t *txWrapper) QueryAll(ctx context.Context, dest any, sql string, args ...any) error {
	rows, err := t.tx.Query(ctx, sql, args...)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanRowsToSlice(rows, dest)
}

// Exec 执行写操作
func (t *txWrapper) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	result, err := t.tx.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("exec failed: %w", err)
	}

	return result.RowsAffected(), nil
}

// Exists 检查记录是否存在
func (t *txWrapper) Exists(ctx context.Context, sql string, args ...any) (bool, error) {
	var exists bool
	err := t.tx.QueryRow(ctx, sql, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists query failed: %w", err)
	}

	return exists, nil
}

// Commit 提交事务
func (t *txWrapper) Commit(ctx context.Context) error {
	if err := t.tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}

// Rollback 回滚事务
func (t *txWrapper) Rollback(ctx context.Context) error {
	if err := t.tx.Rollback(ctx); err != nil {
		return fmt.Errorf("failed to rollback transaction: %w", err)
	}
	return nil
}

// Insert 插入单条记录
func (t *txWrapper) Insert(ctx context.Context, sql string, args ...any) (int64, error) {
	return t.Exec(ctx, sql, args...)
}

// InsertBatch 批量插入记录（使用 Pipeline）
func (t *txWrapper) InsertBatch(ctx context.Context, sql string, argsList [][]any) (int64, error) {
	batch := &pgx.Batch{}
	for _, args := range argsList {
		batch.Queue(sql, args...)
	}

	results := t.tx.SendBatch(ctx, batch)
	defer results.Close()

	var totalAffected int64
	for i := 0; i < len(argsList); i++ {
		ct, err := results.Exec()
		if err != nil {
			return totalAffected, fmt.Errorf("batch insert failed at index %d: %w", i, err)
		}
		totalAffected += ct.RowsAffected()
	}

	return totalAffected, nil
}

// Update 更新记录
func (t *txWrapper) Update(ctx context.Context, sql string, args ...any) (int64, error) {
	return t.Exec(ctx, sql, args...)
}

// UpdateBatch 批量更新记录（使用 Pipeline）
func (t *txWrapper) UpdateBatch(ctx context.Context, sql string, argsList [][]any) (int64, error) {
	batch := &pgx.Batch{}
	for _, args := range argsList {
		batch.Queue(sql, args...)
	}

	results := t.tx.SendBatch(ctx, batch)
	defer results.Close()

	var totalAffected int64
	for i := 0; i < len(argsList); i++ {
		ct, err := results.Exec()
		if err != nil {
			return totalAffected, fmt.Errorf("batch update failed at index %d: %w", i, err)
		}
		totalAffected += ct.RowsAffected()
	}

	return totalAffected, nil
}

// Delete 删除记录
func (t *txWrapper) Delete(ctx context.Context, sql string, args ...any) (int64, error) {
	return t.Exec(ctx, sql, args...)
}

// DeleteBatch 批量删除记录（使用 Pipeline）
func (t *txWrapper) DeleteBatch(ctx context.Context, sql string, argsList [][]any) (int64, error) {
	batch := &pgx.Batch{}
	for _, args := range argsList {
		batch.Queue(sql, args...)
	}

	results := t.tx.SendBatch(ctx, batch)
	defer results.Close()

	var totalAffected int64
	for i := 0; i < len(argsList); i++ {
		ct, err := results.Exec()
		if err != nil {
			return totalAffected, fmt.Errorf("batch delete failed at index %d: %w", i, err)
		}
		totalAffected += ct.RowsAffected()
	}

	return totalAffected, nil
}

// BeginTx 开启事务
func (c *Client) BeginTx(ctx context.Context) (Tx, error) {
	tx, err := c.getMaster().Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return &txWrapper{tx: tx}, nil
}

// TxIsolationLevel 事务隔离级别
type TxIsolationLevel string

const (
	TxIsolationLevelDefault         TxIsolationLevel = ""
	TxIsolationLevelReadUncommitted TxIsolationLevel = "read uncommitted"
	TxIsolationLevelReadCommitted   TxIsolationLevel = "read committed"
	TxIsolationLevelRepeatableRead  TxIsolationLevel = "repeatable read"
	TxIsolationLevelSerializable    TxIsolationLevel = "serializable"
)

// TxAccessMode 事务访问模式
type TxAccessMode string

const (
	TxAccessModeDefault   TxAccessMode = ""
	TxAccessModeReadWrite TxAccessMode = "read write"
	TxAccessModeReadOnly  TxAccessMode = "read only"
)

// TxOptions 事务选项
type TxOptions struct {
	IsoLevel   TxIsolationLevel // 隔离级别
	AccessMode TxAccessMode     // 访问模式
}

// BeginTxWithOptions 使用选项开启事务
func (c *Client) BeginTxWithOptions(ctx context.Context, opts TxOptions) (Tx, error) {
	pgxOpts := pgx.TxOptions{
		IsoLevel:   pgx.TxIsoLevel(opts.IsoLevel),
		AccessMode: pgx.TxAccessMode(opts.AccessMode),
	}

	tx, err := c.getMaster().BeginTx(ctx, pgxOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction with options: %w", err)
	}
	return &txWrapper{tx: tx}, nil
}

// WithTx 在事务中执行函数
func (c *Client) WithTx(ctx context.Context, fn func(Tx) error) error {
	return c.WithTxOptions(ctx, TxOptions{}, fn)
}

// WithTxOptions 使用选项在事务中执行函数
func (c *Client) WithTxOptions(ctx context.Context, opts TxOptions, fn func(Tx) error) error {
	tx, err := c.BeginTxWithOptions(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("transaction error: %w, rollback error: %v", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
