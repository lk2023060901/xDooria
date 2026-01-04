package service

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
	"github.com/lk2023060901/xdooria/app/game/internal/repository"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// BagService 背包服务
type BagService struct {
	logger     logger.Logger
	playerRepo repository.PlayerRepository
}

func NewBagService(l logger.Logger, playerRepo repository.PlayerRepository) *BagService {
	return &BagService{
		logger:     l.Named("service.bag"),
		playerRepo: playerRepo,
	}
}

// AddItem 添加道具
func (s *BagService) AddItem(ctx context.Context, roleID int64, itemID int32, count int32) error {
	if count <= 0 {
		return nil
	}

	// 1. 获取道具配置以确定背包类型
	cfg := gameconfig.T.TbItem.Get(itemID)
	if cfg == nil {
		return fmt.Errorf("item config %d not found", itemID)
	}

	// 2. 加载对应背包
	bag, err := s.playerRepo.GetBag(ctx, roleID, cfg.Type)
	if err != nil {
		return err
	}

	// 3. 更新数量
	bag.Items[itemID] += count

	// 4. 保存
	return s.playerRepo.SaveBag(ctx, bag)
}

// ConsumeItem 消耗道具
func (s *BagService) ConsumeItem(ctx context.Context, roleID int64, itemID int32, count int32) error {
	if count <= 0 {
		return nil
	}

	cfg := gameconfig.T.TbItem.Get(itemID)
	if cfg == nil {
		return fmt.Errorf("item config %d not found", itemID)
	}

	bag, err := s.playerRepo.GetBag(ctx, roleID, cfg.Type)
	if err != nil {
		return err
	}

	// 检查余额
	current := bag.Items[itemID]
	if current < count {
		return fmt.Errorf("insufficient item %d: have %d, need %d", itemID, current, count)
	}

	// 更新并保存
	bag.Items[itemID] -= count
	if bag.Items[itemID] == 0 {
		delete(bag.Items, itemID)
	}

	return s.playerRepo.SaveBag(ctx, bag)
}
