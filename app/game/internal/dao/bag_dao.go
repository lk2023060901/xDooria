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

// BagDAO 背包数据访问对象
type BagDAO struct {
	db      *postgres.Client
	logger  logger.Logger
	metrics *metrics.GameMetrics
}

func NewBagDAO(db *postgres.Client, l logger.Logger, m *metrics.GameMetrics) *BagDAO {
	return &BagDAO{
		db:      db,
		logger:  l.Named("dao.bag"),
		metrics: m,
	}
}

// GetBag 获取玩家指定类型的背包
func (d *BagDAO) GetBag(ctx context.Context, roleID int64, bagType int32) (*model.PlayerBag, error) {
	start := time.Now()
	defer func() {
		d.metrics.RecordDBQuery("select", true, time.Since(start).Seconds())
	}()

	query, args, err := squirrel.
		Select("items").
		From("player_bags").
		Where(squirrel.Eq{"role_id": roleID, "bag_type": bagType}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	var itemsJSON []byte
	err = d.db.QueryRow(ctx, query, args...).Scan(&itemsJSON)
	if err != nil {
		if err == postgres.ErrNoRows {
			return model.NewPlayerBag(roleID, bagType), nil
		}
		return nil, fmt.Errorf("failed to get player bag: %w", err)
	}

	bag := model.NewPlayerBag(roleID, bagType)
	if err := json.Unmarshal(itemsJSON, &bag.Items); err != nil {
		return nil, fmt.Errorf("failed to unmarshal bag items: %w", err)
	}

	return bag, nil
}

// SaveBag 保存玩家背包 (Upsert)
func (d *BagDAO) SaveBag(ctx context.Context, bag *model.PlayerBag) error {
	start := time.Now()
	defer func() {
		d.metrics.RecordDBQuery("upsert", true, time.Since(start).Seconds())
	}()

	itemsJSON, err := json.Marshal(bag.Items)
	if err != nil {
		return err
	}

	query, args, err := squirrel.
		Insert("player_bags").
		Columns("role_id", "bag_type", "items", "updated_at").
		Values(bag.RoleID, bag.BagType, itemsJSON, time.Now()).
		Suffix("ON CONFLICT (role_id, bag_type) DO UPDATE SET items = EXCLUDED.items, updated_at = EXCLUDED.updated_at").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return err
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to save player bag: %w", err)
	}

	return nil
}
