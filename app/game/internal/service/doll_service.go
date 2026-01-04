package service

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/app/game/internal/repository"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// DollService 玩偶服务
type DollService struct {
	logger     logger.Logger
	playerRepo repository.PlayerRepository
	metrics    *metrics.GameMetrics
}

// NewDollService 创建玩偶服务
func NewDollService(
	l logger.Logger,
	playerRepo repository.PlayerRepository,
	m *metrics.GameMetrics,
) *DollService {
	return &DollService{
		logger:     l.Named("service.doll"),
		playerRepo: playerRepo,
		metrics:    m,
	}
}

// AddDoll 为玩家添加一个玩偶
func (s *DollService) AddDoll(ctx context.Context, roleID int64, dollConfigID int32) (*model.Doll, error) {
	// 1. 获取配置
	cfg := gameconfig.T.TbDoll.Get(dollConfigID)
	if cfg == nil {
		return nil, fmt.Errorf("doll config %d not found", dollConfigID)
	}

	// 2. 创建实例
	doll := &model.Doll{
		ID:         0, // 由数据库生成
		PlayerID:   roleID,
		DollID:     dollConfigID,
		Quality:    int16(cfg.MaxQuality),
		IsLocked:   false,
		IsRedeemed: false,
		CreatedAt:  time.Now(),
	}

	// 3. 存储
	if err := s.playerRepo.AddDoll(ctx, doll); err != nil {
		return nil, err
	}

	s.logger.Info("doll added to player", "role_id", roleID, "doll_id", dollConfigID)
	return doll, nil
}

// GetDolls 获取玩偶列表（支持排序）
// sortType: 使用 gameconfig.DollSortType_* 常量
func (s *DollService) GetDolls(ctx context.Context, playerID int64, sortType int32) ([]*model.Doll, error) {
	// 1. 从 repository 获取玩偶列表
	dolls, err := s.playerRepo.GetDolls(ctx, playerID)
	if err != nil {
		s.logger.Error("failed to get dolls",
			"player_id", playerID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get dolls: %w", err)
	}

	// 2. 排序
	s.sortDolls(dolls, sortType)

	s.logger.Debug("got dolls with sorting",
		"player_id", playerID,
		"count", len(dolls),
		"sort_type", sortType,
	)

	return dolls, nil
}

// GetDollByID 获取指定玩偶
func (s *DollService) GetDollByID(ctx context.Context, playerID int64, dollID int64) (*model.Doll, error) {
	doll, err := s.playerRepo.GetDollByID(ctx, playerID, dollID)
	if err != nil {
		s.logger.Error("failed to get doll by id",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get doll: %w", err)
	}

	return doll, nil
}

// LockDoll 锁定玩偶
func (s *DollService) LockDoll(ctx context.Context, playerID int64, dollID int64) error {
	// 1. 检查玩偶是否存在
	doll, err := s.playerRepo.GetDollByID(ctx, playerID, dollID)
	if err != nil {
		s.logger.Error("failed to get doll for locking",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return fmt.Errorf("doll not found: %w", err)
	}

	// 2. 检查是否已锁定
	if doll.IsLocked {
		s.logger.Warn("doll already locked",
			"player_id", playerID,
			"doll_id", dollID,
		)
		return fmt.Errorf("doll already locked")
	}

	// 3. 更新锁定状态
	if err := s.playerRepo.UpdateDollLock(ctx, playerID, dollID, true); err != nil {
		s.logger.Error("failed to lock doll",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return fmt.Errorf("failed to lock doll: %w", err)
	}

	s.logger.Info("doll locked",
		"player_id", playerID,
		"doll_id", dollID,
	)

	return nil
}

// UnlockDoll 解锁玩偶
func (s *DollService) UnlockDoll(ctx context.Context, playerID int64, dollID int64) error {
	// 1. 检查玩偶是否存在
	doll, err := s.playerRepo.GetDollByID(ctx, playerID, dollID)
	if err != nil {
		s.logger.Error("failed to get doll for unlocking",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return fmt.Errorf("doll not found: %w", err)
	}

	// 2. 检查是否已解锁
	if !doll.IsLocked {
		s.logger.Warn("doll already unlocked",
			"player_id", playerID,
			"doll_id", dollID,
		)
		return fmt.Errorf("doll already unlocked")
	}

	// 3. 更新锁定状态
	if err := s.playerRepo.UpdateDollLock(ctx, playerID, dollID, false); err != nil {
		s.logger.Error("failed to unlock doll",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return fmt.Errorf("failed to unlock doll: %w", err)
	}

	s.logger.Info("doll unlocked",
		"player_id", playerID,
		"doll_id", dollID,
	)

	return nil
}

// RedeemRealDoll 兑换实体玩偶（检查 can_exchange_real 和扣除成本）
func (s *DollService) RedeemRealDoll(ctx context.Context, playerID int64, dollID int64) error {
	// 1. 检查玩偶是否存在
	doll, err := s.playerRepo.GetDollByID(ctx, playerID, dollID)
	if err != nil {
		s.logger.Error("failed to get doll for redeeming",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return fmt.Errorf("doll not found: %w", err)
	}

	// 2. 检查是否已兑换
	if doll.IsRedeemed {
		s.logger.Warn("doll already redeemed",
			"player_id", playerID,
			"doll_id", dollID,
		)
		return fmt.Errorf("doll already redeemed")
	}

	// 3. 从配置表检查该玩偶是否可兑换实体
	dollConfig := gameconfig.T.TbDoll.Get(doll.DollID)
	if dollConfig == nil {
		s.logger.Error("doll config not found",
			"player_id", playerID,
			"doll_id", dollID,
			"doll_config_id", doll.DollID,
		)
		return fmt.Errorf("doll config not found")
	}

	if !dollConfig.CanExchangeReal {
		s.logger.Warn("doll cannot be redeemed",
			"player_id", playerID,
			"doll_id", dollID,
			"doll_config_id", doll.DollID,
		)
		return fmt.Errorf("doll cannot be redeemed")
	}

	// TODO: 4. 检查并扣除兑换成本
	// redeemCost := dollConfig.RealExchangeCost
	// 需要解析 RealExchangeCost 字符串（可能是 "币种:数量" 格式）
	// if err := s.currencyService.DeductCurrency(ctx, playerID, redeemCost); err != nil {
	//     return fmt.Errorf("insufficient currency: %w", err)
	// }

	// 5. 更新兑换状态
	if err := s.playerRepo.UpdateDollRedeem(ctx, playerID, dollID, true); err != nil {
		// TODO: 回滚货币扣除
		s.logger.Error("failed to redeem doll",
			"player_id", playerID,
			"doll_id", dollID,
			"error", err,
		)
		return fmt.Errorf("failed to redeem doll: %w", err)
	}

	s.logger.Info("doll redeemed",
		"player_id", playerID,
		"doll_id", dollID,
		"doll_config_id", doll.DollID,
	)

	return nil
}

// DollCollectionStats 玩偶图鉴统计数据
type DollCollectionStats struct {
	TotalDollTypes   int32           // 总玩偶种类数
	OwnedDollTypes   int32           // 已拥有的玩偶种类数
	TotalDollCount   int32           // 玩偶实例总数
	SeriesCompletion map[int32]int32 // 系列完成度：系列ID -> 已拥有数量
	SeriesTotal      map[int32]int32 // 系列总数：系列ID -> 总种类数
}

// GetCollectionStats 获取玩偶图鉴统计
func (s *DollService) GetCollectionStats(ctx context.Context, playerID int64) (*DollCollectionStats, error) {
	// 1. 获取玩家拥有的所有玩偶
	dolls, err := s.playerRepo.GetDolls(ctx, playerID)
	if err != nil {
		s.logger.Error("failed to get dolls for stats",
			"player_id", playerID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get dolls: %w", err)
	}

	// 2. 统计已拥有的玩偶种类（去重）
	ownedDollTypes := make(map[int32]bool)
	seriesOwned := make(map[int32]map[int32]bool) // 系列ID -> 玩偶配置ID集合

	for _, doll := range dolls {
		ownedDollTypes[doll.DollID] = true

		// 从配置表获取玩偶所属系列
		dollConfig := gameconfig.T.TbDoll.Get(doll.DollID)
		if dollConfig != nil {
			seriesID := dollConfig.Series
			if seriesOwned[seriesID] == nil {
				seriesOwned[seriesID] = make(map[int32]bool)
			}
			seriesOwned[seriesID][doll.DollID] = true
		}
	}

	// 3. 从配置表获取总玩偶种类数和系列信息
	allDollConfigs := gameconfig.T.TbDoll.GetDataList()
	totalDollTypes := len(allDollConfigs)
	seriesTotal := make(map[int32]int32)
	for _, dollConfig := range allDollConfigs {
		seriesTotal[dollConfig.Series]++
	}

	// 4. 构建统计数据
	stats := &DollCollectionStats{
		TotalDollTypes:   int32(totalDollTypes),
		OwnedDollTypes:   int32(len(ownedDollTypes)),
		TotalDollCount:   int32(len(dolls)),
		SeriesCompletion: make(map[int32]int32),
		SeriesTotal:      seriesTotal,
	}

	// 5. 统计每个系列的完成度
	for seriesID, ownedSet := range seriesOwned {
		stats.SeriesCompletion[seriesID] = int32(len(ownedSet))
	}

	s.logger.Debug("got collection stats",
		"player_id", playerID,
		"owned_types", stats.OwnedDollTypes,
		"total_types", stats.TotalDollTypes,
		"total_count", stats.TotalDollCount,
	)

	return stats, nil
}

// sortDolls 玩偶排序
// sortType: 使用 gameconfig.DollSortType_* 常量
func (s *DollService) sortDolls(dolls []*model.Doll, sortType int32) {
	switch sortType {
	case gameconfig.DollSortType_Quality:
		// 品质降序 -> 获得时间降序
		sort.SliceStable(dolls, func(i, j int) bool {
			if dolls[i].Quality != dolls[j].Quality {
				return dolls[i].Quality > dolls[j].Quality
			}
			return dolls[i].CreatedAt.After(dolls[j].CreatedAt)
		})

	case gameconfig.DollSortType_CreateTime:
		// 获得时间降序
		sort.SliceStable(dolls, func(i, j int) bool {
			return dolls[i].CreatedAt.After(dolls[j].CreatedAt)
		})

	case gameconfig.DollSortType_Favorite:
		// 收藏优先 -> 品质降序 -> 获得时间降序
		sort.SliceStable(dolls, func(i, j int) bool {
			// TODO: 需要在 model.Doll 中添加 IsFavorite 字段
			// if dolls[i].IsFavorite != dolls[j].IsFavorite {
			//     return dolls[i].IsFavorite
			// }

			if dolls[i].Quality != dolls[j].Quality {
				return dolls[i].Quality > dolls[j].Quality
			}

			return dolls[i].CreatedAt.After(dolls[j].CreatedAt)
		})

	default:
		// 默认排序：品质降序 -> 获得时间降序
		sort.SliceStable(dolls, func(i, j int) bool {
			if dolls[i].Quality != dolls[j].Quality {
				return dolls[i].Quality > dolls[j].Quality
			}
			return dolls[i].CreatedAt.After(dolls[j].CreatedAt)
		})
	}
}
