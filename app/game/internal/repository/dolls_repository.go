package repository

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/app/game/internal/model"
)

// ============ 玩偶相关实现 ============

// GetDolls 获取玩家所有玩偶（优先从缓存）
func (r *playerRepositoryImpl) GetDolls(ctx context.Context, playerID int64) ([]*model.Doll, error) {
	// 1. 先尝试从缓存获取
	dolls, err := r.cacheDAO.GetDolls(ctx, playerID)
	if err != nil {
		r.logger.Warn("failed to get dolls from cache, fallback to db",
			"player_id", playerID,
			"error", err,
		)
	} else if dolls != nil {
		r.logger.Debug("dolls loaded from cache",
			"player_id", playerID,
			"count", len(dolls),
		)
		return dolls, nil
	}

	// 2. 缓存未命中，从数据库加载
	dolls, err = r.dollDAO.ListByPlayerID(ctx, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to load dolls from db: %w", err)
	}

	r.logger.Debug("dolls loaded from db",
		"player_id", playerID,
		"count", len(dolls),
	)

	// 3. 回写缓存
	if err := r.cacheDAO.SetDolls(ctx, playerID, dolls, 0); err != nil {
		r.logger.Warn("failed to set dolls cache",
			"player_id", playerID,
			"error", err,
		)
	}

	return dolls, nil
}

// GetDollByID 获取指定玩偶（从内存数据中查找）
func (r *playerRepositoryImpl) GetDollByID(ctx context.Context, playerID int64, dollID int64) (*model.Doll, error) {
	dolls, err := r.GetDolls(ctx, playerID)
	if err != nil {
		return nil, err
	}

	for _, doll := range dolls {
		if doll.ID == dollID {
			return doll, nil
		}
	}

	return nil, fmt.Errorf("doll not found: player_id=%d, doll_id=%d", playerID, dollID)
}

// AddDoll 添加玩偶（写数据库 + 刷新缓存）
func (r *playerRepositoryImpl) AddDoll(ctx context.Context, doll *model.Doll) error {
	// 1. 写入数据库
	if err := r.dollDAO.Create(ctx, doll); err != nil {
		return fmt.Errorf("failed to create doll: %w", err)
	}

	// 2. 删除缓存，下次查询时重新加载
	if err := r.cacheDAO.DeleteDolls(ctx, doll.PlayerID); err != nil {
		r.logger.Warn("failed to delete dolls cache after add",
			"player_id", doll.PlayerID,
			"error", err,
		)
	}

	r.logger.Info("doll added",
		"player_id", doll.PlayerID,
		"doll_id", doll.DollID,
		"id", doll.ID,
	)

	return nil
}

// AddDolls 批量添加玩偶（写数据库 + 刷新缓存）
func (r *playerRepositoryImpl) AddDolls(ctx context.Context, dolls []*model.Doll) error {
	if len(dolls) == 0 {
		return nil
	}

	// 1. 批量写入数据库
	if err := r.dollDAO.BatchCreate(ctx, dolls); err != nil {
		return fmt.Errorf("failed to batch create dolls: %w", err)
	}

	// 2. 删除所有相关玩家的缓存
	playerIDs := make(map[int64]bool)
	for _, doll := range dolls {
		playerIDs[doll.PlayerID] = true
	}

	for playerID := range playerIDs {
		if err := r.cacheDAO.DeleteDolls(ctx, playerID); err != nil {
			r.logger.Warn("failed to delete dolls cache after batch add",
				"player_id", playerID,
				"error", err,
			)
		}
	}

	r.logger.Info("dolls batch added",
		"count", len(dolls),
		"players", len(playerIDs),
	)

	return nil
}

// UpdateDollLock 更新玩偶锁定状态（写数据库 + 更新缓存）
func (r *playerRepositoryImpl) UpdateDollLock(ctx context.Context, playerID int64, dollID int64, isLocked bool) error {
	// 1. 更新数据库
	if err := r.dollDAO.UpdateLockStatus(ctx, dollID, isLocked); err != nil {
		return fmt.Errorf("failed to update lock status: %w", err)
	}

	// 2. 更新缓存中的数据
	dolls, err := r.cacheDAO.GetDolls(ctx, playerID)
	if err == nil && dolls != nil {
		for _, doll := range dolls {
			if doll.ID == dollID {
				doll.IsLocked = isLocked
				break
			}
		}
		if err := r.cacheDAO.SetDolls(ctx, playerID, dolls, 0); err != nil {
			r.logger.Warn("failed to update dolls cache after lock change",
				"player_id", playerID,
				"error", err,
			)
		}
	}

	r.logger.Debug("doll lock status updated",
		"player_id", playerID,
		"doll_id", dollID,
		"is_locked", isLocked,
	)

	return nil
}

// UpdateDollRedeem 更新玩偶兑换状态（写数据库 + 更新缓存）
func (r *playerRepositoryImpl) UpdateDollRedeem(ctx context.Context, playerID int64, dollID int64, isRedeemed bool) error {
	// 1. 更新数据库
	if err := r.dollDAO.UpdateRedeemStatus(ctx, dollID, isRedeemed); err != nil {
		return fmt.Errorf("failed to update redeem status: %w", err)
	}

	// 2. 更新缓存中的数据
	dolls, err := r.cacheDAO.GetDolls(ctx, playerID)
	if err == nil && dolls != nil {
		for _, doll := range dolls {
			if doll.ID == dollID {
				doll.IsRedeemed = isRedeemed
				break
			}
		}
		if err := r.cacheDAO.SetDolls(ctx, playerID, dolls, 0); err != nil {
			r.logger.Warn("failed to update dolls cache after redeem change",
				"player_id", playerID,
				"error", err,
			)
		}
	}

	r.logger.Info("doll redeem status updated",
		"player_id", playerID,
		"doll_id", dollID,
		"is_redeemed", isRedeemed,
	)

	return nil
}

// UpdateDollQuality 更新玩偶品质（写数据库 + 更新缓存）
func (r *playerRepositoryImpl) UpdateDollQuality(ctx context.Context, playerID int64, dollID int64, quality int16) error {
	// 1. 更新数据库
	if err := r.dollDAO.UpdateQuality(ctx, dollID, quality); err != nil {
		return fmt.Errorf("failed to update quality: %w", err)
	}

	// 2. 更新缓存中的数据
	dolls, err := r.cacheDAO.GetDolls(ctx, playerID)
	if err == nil && dolls != nil {
		for _, doll := range dolls {
			if doll.ID == dollID {
				doll.Quality = quality
				break
			}
		}
		if err := r.cacheDAO.SetDolls(ctx, playerID, dolls, 0); err != nil {
			r.logger.Warn("failed to update dolls cache after quality change",
				"player_id", playerID,
				"error", err,
			)
		}
	}

	r.logger.Info("doll quality updated",
		"player_id", playerID,
		"doll_id", dollID,
		"quality", quality,
	)

	return nil
}

// DeleteDoll 删除玩偶（写数据库 + 更新缓存）
func (r *playerRepositoryImpl) DeleteDoll(ctx context.Context, playerID int64, dollID int64) error {
	// 1. 删除数据库记录
	if err := r.dollDAO.Delete(ctx, dollID); err != nil {
		return fmt.Errorf("failed to delete doll: %w", err)
	}

	// 2. 从缓存中移除
	dolls, err := r.cacheDAO.GetDolls(ctx, playerID)
	if err == nil && dolls != nil {
		newDolls := make([]*model.Doll, 0, len(dolls))
		for _, doll := range dolls {
			if doll.ID != dollID {
				newDolls = append(newDolls, doll)
			}
		}
		if err := r.cacheDAO.SetDolls(ctx, playerID, newDolls, 0); err != nil {
			r.logger.Warn("failed to update dolls cache after delete",
				"player_id", playerID,
				"error", err,
			)
		}
	}

	r.logger.Info("doll deleted",
		"player_id", playerID,
		"doll_id", dollID,
	)

	return nil
}

// DeleteDolls 批量删除玩偶（写数据库 + 更新缓存）
func (r *playerRepositoryImpl) DeleteDolls(ctx context.Context, playerID int64, dollIDs []int64) error {
	if len(dollIDs) == 0 {
		return nil
	}

	// 1. 批量删除数据库记录
	if err := r.dollDAO.BatchDelete(ctx, dollIDs); err != nil {
		return fmt.Errorf("failed to batch delete dolls: %w", err)
	}

	// 2. 从缓存中移除
	dolls, err := r.cacheDAO.GetDolls(ctx, playerID)
	if err == nil && dolls != nil {
		idSet := make(map[int64]bool)
		for _, id := range dollIDs {
			idSet[id] = true
		}

		newDolls := make([]*model.Doll, 0, len(dolls))
		for _, doll := range dolls {
			if !idSet[doll.ID] {
				newDolls = append(newDolls, doll)
			}
		}

		if err := r.cacheDAO.SetDolls(ctx, playerID, newDolls, 0); err != nil {
			r.logger.Warn("failed to update dolls cache after batch delete",
				"player_id", playerID,
				"error", err,
			)
		}
	}

	r.logger.Info("dolls batch deleted",
		"player_id", playerID,
		"count", len(dollIDs),
	)

	return nil
}
