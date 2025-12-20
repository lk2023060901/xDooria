package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// applyQueryTimeout 应用查询超时到 context
func (c *Client) applyQueryTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.cfg.QueryTimeout > 0 {
		return context.WithTimeout(ctx, c.cfg.QueryTimeout)
	}
	return ctx, func() {}
}

// QueryOne 查询单条记录
func QueryOne[T any](c *Client, ctx context.Context, sql string, args ...any) (*T, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getSlave() // 使用从库查询

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanOne[T](rows)
}

// QueryAll 查询多条记录
func QueryAll[T any](c *Client, ctx context.Context, sql string, args ...any) ([]*T, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getSlave() // 使用从库查询

	rows, err := pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	return scanAll[T](rows)
}

// Exec 执行写操作（INSERT/UPDATE/DELETE）
func (c *Client) Exec(ctx context.Context, sql string, args ...any) (int64, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getMaster() // 写操作使用主库

	result, err := pool.Exec(ctx, sql, args...)
	if err != nil {
		return 0, fmt.Errorf("exec failed: %w", err)
	}

	return result.RowsAffected(), nil
}

// Exists 检查记录是否存在
func (c *Client) Exists(ctx context.Context, sql string, args ...any) (bool, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getSlave() // 使用从库查询

	var exists bool
	err := pool.QueryRow(ctx, sql, args...).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("exists query failed: %w", err)
	}

	return exists, nil
}

// Insert 插入单条记录
func (c *Client) Insert(ctx context.Context, sql string, args ...any) (int64, error) {
	return c.Exec(ctx, sql, args...)
}

// InsertBatch 批量插入记录（使用 Pipeline）
func (c *Client) InsertBatch(ctx context.Context, sql string, argsList [][]any) (int64, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getMaster() // 写操作使用主库

	batch := &pgx.Batch{}
	for _, args := range argsList {
		batch.Queue(sql, args...)
	}

	results := pool.SendBatch(ctx, batch)
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
func (c *Client) Update(ctx context.Context, sql string, args ...any) (int64, error) {
	return c.Exec(ctx, sql, args...)
}

// UpdateBatch 批量更新记录（使用 Pipeline）
func (c *Client) UpdateBatch(ctx context.Context, sql string, argsList [][]any) (int64, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getMaster() // 写操作使用主库

	batch := &pgx.Batch{}
	for _, args := range argsList {
		batch.Queue(sql, args...)
	}

	results := pool.SendBatch(ctx, batch)
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
func (c *Client) Delete(ctx context.Context, sql string, args ...any) (int64, error) {
	return c.Exec(ctx, sql, args...)
}

// DeleteBatch 批量删除记录（使用 Pipeline）
func (c *Client) DeleteBatch(ctx context.Context, sql string, argsList [][]any) (int64, error) {
	ctx, cancel := c.applyQueryTimeout(ctx)
	defer cancel()

	pool := c.getMaster() // 写操作使用主库

	batch := &pgx.Batch{}
	for _, args := range argsList {
		batch.Queue(sql, args...)
	}

	results := pool.SendBatch(ctx, batch)
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
