package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
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

func NewDollDAO(db *postgres.Client, l logger.Logger, m *metrics.GameMetrics) *DollDAO {
	return &DollDAO{
		db:      db,
		logger:  l.Named("dao.doll"),
		metrics: m,
	}
}

// dollBagData 玩偶背包存储结构
type dollBagData struct {
	Dolls []*model.Doll `json:"dolls"`
}

// ListByPlayerID 获取玩家所有玩偶
func (d *DollDAO) ListByPlayerID(ctx context.Context, playerID int64) ([]*model.Doll, error) {
	start := time.Now()
	defer func() {
		d.metrics.RecordDBQuery("select", true, time.Since(start).Seconds())
	}()

	query, args, err := squirrel.
		Select("data").
		From("player_bags").
		Where(squirrel.Eq{"role_id": playerID, "bag_type": gameconfig.BagType_Costume}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, err
	}

	var data []byte
	err = d.db.QueryRow(ctx, query, args...).Scan(&data)
	if err != nil {
		if err == postgres.ErrNoRows {
			return []*model.Doll{}, nil
		}
		return nil, fmt.Errorf("failed to get dolls: %w", err)
	}

	var bagData dollBagData
	if err := json.Unmarshal(data, &bagData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal doll data: %w", err)
	}

	return bagData.Dolls, nil
}

// saveDolls 保存玩偶列表
func (d *DollDAO) saveDolls(ctx context.Context, playerID int64, dolls []*model.Doll) error {
	start := time.Now()
	defer func() {
		d.metrics.RecordDBQuery("upsert", true, time.Since(start).Seconds())
	}()

	bagData := dollBagData{Dolls: dolls}
	data, err := json.Marshal(bagData)
	if err != nil {
		return err
	}

	query, args, err := squirrel.
		Insert("player_bags").
		Columns("role_id", "bag_type", "data", "updated_at").
		Values(playerID, gameconfig.BagType_Costume, data, time.Now()).
		Suffix("ON CONFLICT (role_id, bag_type) DO UPDATE SET data = EXCLUDED.data, updated_at = EXCLUDED.updated_at").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return err
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		return fmt.Errorf("failed to save dolls: %w", err)
	}

	return nil
}

// Create 添加玩偶
func (d *DollDAO) Create(ctx context.Context, doll *model.Doll) error {
	dolls, err := d.ListByPlayerID(ctx, doll.PlayerID)
	if err != nil {
		return err
	}

	dolls = append(dolls, doll)
	return d.saveDolls(ctx, doll.PlayerID, dolls)
}

// BatchCreate 批量添加玩偶
func (d *DollDAO) BatchCreate(ctx context.Context, newDolls []*model.Doll) error {
	if len(newDolls) == 0 {
		return nil
	}

	// 按 playerID 分组
	byPlayer := make(map[int64][]*model.Doll)
	for _, doll := range newDolls {
		byPlayer[doll.PlayerID] = append(byPlayer[doll.PlayerID], doll)
	}

	for playerID, playerDolls := range byPlayer {
		existing, err := d.ListByPlayerID(ctx, playerID)
		if err != nil {
			return err
		}
		existing = append(existing, playerDolls...)
		if err := d.saveDolls(ctx, playerID, existing); err != nil {
			return err
		}
	}

	return nil
}

// UpdateLockStatus 更新锁定状态
func (d *DollDAO) UpdateLockStatus(ctx context.Context, dollID int64, isLocked bool) error {
	// 需要先找到玩偶所属的玩家，遍历所有背包（效率较低，但符合当前存储模型）
	// 实际使用时，repository 层应该有缓存或传递 playerID
	return fmt.Errorf("UpdateLockStatus requires playerID, use UpdateLockStatusByPlayer instead")
}

// UpdateLockStatusByPlayer 更新锁定状态（带 playerID）
func (d *DollDAO) UpdateLockStatusByPlayer(ctx context.Context, playerID int64, dollID int64, isLocked bool) error {
	dolls, err := d.ListByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}

	for _, doll := range dolls {
		if doll.ID == dollID {
			doll.IsLocked = isLocked
			return d.saveDolls(ctx, playerID, dolls)
		}
	}

	return fmt.Errorf("doll not found")
}

// UpdateRedeemStatus 更新兑换状态
func (d *DollDAO) UpdateRedeemStatus(ctx context.Context, dollID int64, isRedeemed bool) error {
	return fmt.Errorf("UpdateRedeemStatus requires playerID, use UpdateRedeemStatusByPlayer instead")
}

// UpdateRedeemStatusByPlayer 更新兑换状态（带 playerID）
func (d *DollDAO) UpdateRedeemStatusByPlayer(ctx context.Context, playerID int64, dollID int64, isRedeemed bool) error {
	dolls, err := d.ListByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}

	for _, doll := range dolls {
		if doll.ID == dollID {
			doll.IsRedeemed = isRedeemed
			return d.saveDolls(ctx, playerID, dolls)
		}
	}

	return fmt.Errorf("doll not found")
}

// UpdateQuality 更新品质
func (d *DollDAO) UpdateQuality(ctx context.Context, dollID int64, quality int16) error {
	return fmt.Errorf("UpdateQuality requires playerID, use UpdateQualityByPlayer instead")
}

// UpdateQualityByPlayer 更新品质（带 playerID）
func (d *DollDAO) UpdateQualityByPlayer(ctx context.Context, playerID int64, dollID int64, quality int16) error {
	dolls, err := d.ListByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}

	for _, doll := range dolls {
		if doll.ID == dollID {
			doll.Quality = quality
			return d.saveDolls(ctx, playerID, dolls)
		}
	}

	return fmt.Errorf("doll not found")
}

// Delete 删除玩偶
func (d *DollDAO) Delete(ctx context.Context, dollID int64) error {
	return fmt.Errorf("Delete requires playerID, use DeleteByPlayer instead")
}

// DeleteByPlayer 删除玩偶（带 playerID）
func (d *DollDAO) DeleteByPlayer(ctx context.Context, playerID int64, dollID int64) error {
	dolls, err := d.ListByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}

	for i, doll := range dolls {
		if doll.ID == dollID {
			dolls = append(dolls[:i], dolls[i+1:]...)
			return d.saveDolls(ctx, playerID, dolls)
		}
	}

	return fmt.Errorf("doll not found")
}

// BatchDelete 批量删除玩偶
func (d *DollDAO) BatchDelete(ctx context.Context, dollIDs []int64) error {
	return fmt.Errorf("BatchDelete requires playerID, use BatchDeleteByPlayer instead")
}

// BatchDeleteByPlayer 批量删除玩偶（带 playerID）
func (d *DollDAO) BatchDeleteByPlayer(ctx context.Context, playerID int64, dollIDs []int64) error {
	dolls, err := d.ListByPlayerID(ctx, playerID)
	if err != nil {
		return err
	}

	idSet := make(map[int64]bool)
	for _, id := range dollIDs {
		idSet[id] = true
	}

	filtered := make([]*model.Doll, 0, len(dolls))
	for _, doll := range dolls {
		if !idSet[doll.ID] {
			filtered = append(filtered, doll)
		}
	}

	return d.saveDolls(ctx, playerID, filtered)
}
