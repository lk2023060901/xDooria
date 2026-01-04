package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/database/postgres"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// GachaDAO 抽卡数据访问对象
type GachaDAO struct {
	db      *postgres.Client
	logger  logger.Logger
	metrics *metrics.GameMetrics
}

// NewGachaDAO 创建抽卡 DAO
func NewGachaDAO(db *postgres.Client, l logger.Logger, m *metrics.GameMetrics) *GachaDAO {
	return &GachaDAO{
		db:      db,
		logger:  l.Named("dao.gacha"),
		metrics: m,
	}
}

// GetByRoleID 查询玩家抽卡记录
func (d *GachaDAO) GetByRoleID(ctx context.Context, roleID int64) (*model.PlayerGacha, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("select", true, duration)
	}()

	query, args, err := squirrel.
		Select("records").
		From("player_gacha").
		Where(squirrel.Eq{"role_id": roleID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var recordsJSON []byte
	err = d.db.QueryRow(ctx, query, args...).Scan(&recordsJSON)
	if err != nil {
		if err == postgres.ErrNoRows {
			return &model.PlayerGacha{RoleID: roleID, Records: []*model.GachaRecord{}}, nil
		}
		return nil, fmt.Errorf("failed to get player gacha: %w", err)
	}

	var records []*model.GachaRecord
	if err := json.Unmarshal(recordsJSON, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal gacha records: %w", err)
	}

	return &model.PlayerGacha{
		RoleID:  roleID,
		Records: records,
	}, nil
}

// Save 保存玩家抽卡记录 (Upsert)
func (d *GachaDAO) Save(ctx context.Context, gacha *model.PlayerGacha) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("upsert", true, duration)
	}()

	recordsJSON, err := json.Marshal(gacha.Records)
	if err != nil {
		return fmt.Errorf("failed to marshal gacha records: %w", err)
	}

	query, args, err := squirrel.
		Insert("player_gacha").
		Columns("role_id", "records", "updated_at").
		Values(gacha.RoleID, recordsJSON, time.Now()).
		Suffix("ON CONFLICT (role_id) DO UPDATE SET records = EXCLUDED.records, updated_at = EXCLUDED.updated_at").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to save player gacha: %w", err)
	}

	return nil
}
