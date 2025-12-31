package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/database/postgres"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// DollDAO 玩偶数据访问对象
type DollDAO struct {
	db      *postgres.Client
	logger  logger.Logger
	metrics *metrics.GameMetrics
}

// NewDollDAO 创建玩偶 DAO
func NewDollDAO(db *postgres.Client, l logger.Logger, m *metrics.GameMetrics) *DollDAO {
	return &DollDAO{
		db:      db,
		logger:  l.Named("dao.doll"),
		metrics: m,
	}
}

// ListByPlayerID 查询玩家所有玩偶
func (d *DollDAO) ListByPlayerID(ctx context.Context, playerID int64) ([]*model.Doll, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("select", true, duration)
	}()

	query, args, err := squirrel.
		Select("id", "player_id", "doll_id", "quality", "is_locked", "is_redeemed", "created_at").
		From("player_doll").
		Where(squirrel.Eq{"player_id": playerID}).
		OrderBy("created_at DESC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := d.db.Query(ctx, query, args...)
	if err != nil {
		d.logger.Error("failed to list dolls by player_id",
			"player_id", playerID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list dolls: %w", err)
	}
	defer rows.Close()

	var dolls []*model.Doll
	for rows.Next() {
		var doll model.Doll
		if err := rows.Scan(
			&doll.ID,
			&doll.PlayerID,
			&doll.DollID,
			&doll.Quality,
			&doll.IsLocked,
			&doll.IsRedeemed,
			&doll.CreatedAt,
		); err != nil {
			d.logger.Error("failed to scan doll",
				"player_id", playerID,
				"error", err,
			)
			return nil, fmt.Errorf("failed to scan doll: %w", err)
		}
		dolls = append(dolls, &doll)
	}

	if err := rows.Err(); err != nil {
		d.logger.Error("rows iteration error",
			"player_id", playerID,
			"error", err,
		)
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return dolls, nil
}

// GetByID 根据ID查询玩偶
func (d *DollDAO) GetByID(ctx context.Context, id int64) (*model.Doll, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("select", true, duration)
	}()

	query, args, err := squirrel.
		Select("id", "player_id", "doll_id", "quality", "is_locked", "is_redeemed", "created_at").
		From("player_doll").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var doll model.Doll
	if err := d.db.QueryRow(ctx, query, args...).Scan(
		&doll.ID,
		&doll.PlayerID,
		&doll.DollID,
		&doll.Quality,
		&doll.IsLocked,
		&doll.IsRedeemed,
		&doll.CreatedAt,
	); err != nil {
		d.logger.Error("failed to get doll by id",
			"id", id,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get doll: %w", err)
	}

	return &doll, nil
}

// Create 创建玩偶实例
func (d *DollDAO) Create(ctx context.Context, doll *model.Doll) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("insert", true, duration)
	}()

	query, args, err := squirrel.
		Insert("player_doll").
		Columns("id", "player_id", "doll_id", "quality", "is_locked", "is_redeemed", "created_at").
		Values(doll.ID, doll.PlayerID, doll.DollID, doll.Quality, doll.IsLocked, doll.IsRedeemed, doll.CreatedAt).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to create doll",
			"player_id", doll.PlayerID,
			"doll_id", doll.DollID,
			"error", err,
		)
		return fmt.Errorf("failed to create doll: %w", err)
	}

	d.logger.Info("doll created",
		"id", doll.ID,
		"player_id", doll.PlayerID,
		"doll_id", doll.DollID,
		"quality", doll.Quality,
	)

	return nil
}

// BatchCreate 批量创建玩偶实例
func (d *DollDAO) BatchCreate(ctx context.Context, dolls []*model.Doll) error {
	if len(dolls) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("insert", true, duration)
	}()

	builder := squirrel.
		Insert("player_doll").
		Columns("id", "player_id", "doll_id", "quality", "is_locked", "is_redeemed", "created_at")

	for _, doll := range dolls {
		builder = builder.Values(doll.ID, doll.PlayerID, doll.DollID, doll.Quality, doll.IsLocked, doll.IsRedeemed, doll.CreatedAt)
	}

	query, args, err := builder.PlaceholderFormat(squirrel.Dollar).ToSql()
	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to batch create dolls",
			"count", len(dolls),
			"error", err,
		)
		return fmt.Errorf("failed to batch create dolls: %w", err)
	}

	d.logger.Info("dolls batch created",
		"count", len(dolls),
	)

	return nil
}

// UpdateLockStatus 更新锁定状态
func (d *DollDAO) UpdateLockStatus(ctx context.Context, id int64, isLocked bool) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("update", true, duration)
	}()

	query, args, err := squirrel.
		Update("player_doll").
		Set("is_locked", isLocked).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to update lock status",
			"id", id,
			"is_locked", isLocked,
			"error", err,
		)
		return fmt.Errorf("failed to update lock status: %w", err)
	}

	d.logger.Debug("doll lock status updated",
		"id", id,
		"is_locked", isLocked,
	)

	return nil
}

// UpdateRedeemStatus 更新兑换状态
func (d *DollDAO) UpdateRedeemStatus(ctx context.Context, id int64, isRedeemed bool) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("update", true, duration)
	}()

	query, args, err := squirrel.
		Update("player_doll").
		Set("is_redeemed", isRedeemed).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to update redeem status",
			"id", id,
			"is_redeemed", isRedeemed,
			"error", err,
		)
		return fmt.Errorf("failed to update redeem status: %w", err)
	}

	d.logger.Info("doll redeem status updated",
		"id", id,
		"is_redeemed", isRedeemed,
	)

	return nil
}

// UpdateQuality 更新品质（熔炼用）
func (d *DollDAO) UpdateQuality(ctx context.Context, id int64, quality int16) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("update", true, duration)
	}()

	query, args, err := squirrel.
		Update("player_doll").
		Set("quality", quality).
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to update quality",
			"id", id,
			"quality", quality,
			"error", err,
		)
		return fmt.Errorf("failed to update quality: %w", err)
	}

	d.logger.Info("doll quality updated",
		"id", id,
		"quality", quality,
	)

	return nil
}

// Delete 删除玩偶（单个）
func (d *DollDAO) Delete(ctx context.Context, id int64) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("delete", true, duration)
	}()

	query, args, err := squirrel.
		Delete("player_doll").
		Where(squirrel.Eq{"id": id}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to delete doll",
			"id", id,
			"error", err,
		)
		return fmt.Errorf("failed to delete doll: %w", err)
	}

	d.logger.Info("doll deleted",
		"id", id,
	)

	return nil
}

// BatchDelete 批量删除玩偶（熔炼用）
func (d *DollDAO) BatchDelete(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("delete", true, duration)
	}()

	query, args, err := squirrel.
		Delete("player_doll").
		Where(squirrel.Eq{"id": ids}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to batch delete dolls",
			"count", len(ids),
			"error", err,
		)
		return fmt.Errorf("failed to batch delete dolls: %w", err)
	}

	d.logger.Info("dolls batch deleted",
		"count", len(ids),
		"ids", ids,
	)

	return nil
}
