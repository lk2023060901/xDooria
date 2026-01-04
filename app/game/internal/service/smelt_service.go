package service

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/app/game/internal/repository"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// SmeltService 熔炼服务
type SmeltService struct {
	logger     logger.Logger
	playerRepo repository.PlayerRepository
	dropSvc    *DropService
	dollSvc    *DollService
	bagSvc     *BagService
}

func NewSmeltService(
	l logger.Logger,
	playerRepo repository.PlayerRepository,
	dropSvc *DropService,
	dollSvc *DollService,
	bagSvc *BagService,
) *SmeltService {
	return &SmeltService{
		logger:     l.Named("service.smelt"),
		playerRepo: playerRepo,
		dropSvc:    dropSvc,
		dollSvc:    dollSvc,
		bagSvc:     bagSvc,
	}
}

// Smelt 熔炼核心逻辑
// 返回产出的新玩偶、是否升品成功、以及错误
func (s *SmeltService) Smelt(ctx context.Context, roleID int64, itemUIDs []int64) (*model.Doll, bool, error) {
	if len(itemUIDs) == 0 {
		return nil, false, fmt.Errorf("no dolls provided for smelting")
	}

	// 1. 获取当前有效的熔炼活动（简单起见取第一个，实际应按时间判定）
	activities := gameconfig.T.TbSmeltActivity.GetDataList()
	if len(activities) == 0 {
		return nil, false, fmt.Errorf("no active smelt activity")
	}
	activity := activities[0]

	// 2. 验证玩偶状态并执行物理删除
	for _, uid := range itemUIDs {
		doll, err := s.playerRepo.GetDollByID(ctx, roleID, uid)
		if err != nil {
			return nil, false, fmt.Errorf("doll %d not found: %w", uid, err)
		}

		if doll.IsLocked {
			return nil, false, fmt.Errorf("doll %d is locked and cannot be smelted", uid)
		}
		if doll.IsRedeemed {
			return nil, false, fmt.Errorf("doll %d is already redeemed and cannot be smelted", uid)
		}

		// 销毁旧玩偶
		if err := s.playerRepo.DeleteDoll(ctx, roleID, uid); err != nil {
			return nil, false, fmt.Errorf("failed to destroy doll %d: %w", uid, err)
		}
	}

	// 3. 执行熔炼掉落
	results, err := s.dropSvc.ExecuteDrop(ctx, activity.DropId)
	if err != nil {
		return nil, false, err
	}

	// 4. 发放奖励并返回第一个新玩偶 (符合 SmeltResult proto)
	var newDoll *model.Doll
	for _, res := range results {
		if dollCfg := gameconfig.T.TbDoll.Get(res.ItemID); dollCfg != nil {
			// 熔炼产出新玩偶
			for c := int32(0); c < res.Count; c++ {
				d, err := s.dollSvc.AddDoll(ctx, roleID, res.ItemID)
				if err != nil {
					s.logger.Error("failed to add reward doll from smelt", "role_id", roleID, "doll_id", res.ItemID, "error", err)
					continue
				}
				if newDoll == nil {
					newDoll = d
				}
			}
		} else {
			// 其他道具进背包
			if err := s.bagSvc.AddItem(ctx, roleID, res.ItemID, res.Count); err != nil {
				s.logger.Error("failed to add reward item from smelt", "role_id", roleID, "item_id", res.ItemID, "error", err)
			}
		}
	}

	s.logger.Info("smelt success", "role_id", roleID, "consumed_count", len(itemUIDs))

	// 暂时固定返回 upgraded=false，后续可根据掉落逻辑判定
	return newDoll, false, nil
}
