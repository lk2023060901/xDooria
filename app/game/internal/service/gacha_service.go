package service

import (
	"context"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/app/game/internal/repository"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// GachaService 盲盒服务
type GachaService struct {
	logger     logger.Logger
	playerRepo repository.PlayerRepository
	dropSvc    *DropService
	dollSvc    *DollService
	bagSvc     *BagService
	// maxPityMap 缓存每个池子的最大保底次数 {PoolID: MaxCount}
	maxPityMap map[int32]int32
}

func NewGachaService(
	l logger.Logger,
	playerRepo repository.PlayerRepository,
	dropSvc *DropService,
	dollSvc *DollService,
	bagSvc *BagService,
) *GachaService {
	s := &GachaService{
		logger:     l.Named("service.gacha"),
		playerRepo: playerRepo,
		dropSvc:    dropSvc,
		dollSvc:    dollSvc,
		bagSvc:     bagSvc,
		maxPityMap: make(map[int32]int32),
	}
	s.initMaxPityMap()
	return s
}

// initMaxPityMap 在启动时预计算每个池子的最大保底次数
func (s *GachaService) initMaxPityMap() {
	for _, pity := range gameconfig.T.TbGachaPity.GetDataList() {
		if pity.TriggerCount > s.maxPityMap[pity.GachaId] {
			s.maxPityMap[pity.GachaId] = pity.TriggerCount
		}
	}
	s.logger.Info("gacha max pity map initialized", "count", len(s.maxPityMap))
}

// GetMaxTriggerCount 获取指定池子的重置阈值
func (s *GachaService) GetMaxTriggerCount(poolID int32) int32 {
	return s.maxPityMap[poolID]
}

// Draw 盲盒抽取主逻辑
func (s *GachaService) Draw(ctx context.Context, roleID int64, poolID int32, count int32) ([]*DropResult, int32, error) {
	// 1. 获取配置
	poolCfg := gameconfig.T.TbGacha.Get(poolID)
	if poolCfg == nil {
		return nil, 0, fmt.Errorf("gacha pool %d not found", poolID)
	}

	// 2. 扣除消耗
	if poolCfg.CostItem > 0 && poolCfg.CostCount > 0 {
		if err := s.bagSvc.ConsumeItem(ctx, roleID, poolCfg.CostItem, poolCfg.CostCount*count); err != nil {
			return nil, 0, err
		}
	}

	// 3. 读取玩家抽卡记录
	gachaData, err := s.playerRepo.GetGachaRecords(ctx, roleID)
	if err != nil {
		return nil, 0, err
	}
	record := s.getOrCreateRecord(gachaData, poolID, poolCfg.Type)
	maxTrigger := s.GetMaxTriggerCount(poolID)

	// 4. 执行循环抽取
	var finalResults []*DropResult
	for i := int32(0); i < count; i++ {
		record.TotalCount++
		record.LastTime = time.Now().Unix()

		// 判定当前次数是否命中保底
		dropID := poolCfg.DropId
		if pityDropID := s.checkPity(poolID, record.TotalCount); pityDropID != 0 {
			dropID = pityDropID
		}

		// 执行掉落库随机
		results, err := s.dropSvc.ExecuteDrop(ctx, dropID)
		if err != nil {
			return nil, 0, err
		}
		finalResults = append(finalResults, results...)

		// 只有达到最大保底次数时，才清零
		if maxTrigger > 0 && record.TotalCount >= maxTrigger {
			record.TotalCount = 0
		}
	}

	// 5. 产出处理
	for _, res := range finalResults {
		// 检查该物品是否为玩偶
		if dollCfg := gameconfig.T.TbDoll.Get(res.ItemID); dollCfg != nil {
			// 盲盒产出通常是一个个的玩偶实例
			for c := int32(0); c < res.Count; c++ {
				if _, err := s.dollSvc.AddDoll(ctx, roleID, res.ItemID); err != nil {
					s.logger.Error("failed to add doll from gacha", "role_id", roleID, "doll_id", res.ItemID, "error", err)
				}
			}
		} else {
			// 普通道具入背包
			if err := s.bagSvc.AddItem(ctx, roleID, res.ItemID, res.Count); err != nil {
				s.logger.Error("failed to add item from gacha", "role_id", roleID, "item_id", res.ItemID, "error", err)
			}
		}
	}

	// 6. 持久化记录
	if err := s.playerRepo.SaveGachaRecords(ctx, gachaData); err != nil {
		return nil, 0, err
	}

	return finalResults, record.TotalCount, nil
}

func (s *GachaService) checkPity(poolID int32, currentCount int32) int32 {
	for _, pity := range gameconfig.T.TbGachaPity.GetDataList() {
		if pity.GachaId == poolID && pity.TriggerCount == currentCount {
			return pity.PityDropId
		}
	}
	return 0
}

func (s *GachaService) getOrCreateRecord(data *model.PlayerGacha, poolID int32, gameplayType int32) *model.GachaRecord {
	for _, r := range data.Records {
		if r.ID == poolID {
			return r
		}
	}
	r := &model.GachaRecord{ID: poolID, Type: gameplayType}
	data.Records = append(data.Records, r)
	return r
}
