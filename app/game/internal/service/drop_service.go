package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// DropResult 掉落结果
type DropResult struct {
	ItemID int32
	Count  int32
}

// DropService 掉落服务，负责处理通用的随机掉落逻辑
type DropService struct {
	logger logger.Logger
	rand   *rand.Rand
}

// NewDropService 创建掉落服务
func NewDropService(l logger.Logger) *DropService {
	return &DropService{
		logger: l.Named("service.drop"),
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ExecuteDrop 执行掉落逻辑
// dropID: 掉落ID，对应 TbDropGroup 中的 DropId
func (s *DropService) ExecuteDrop(ctx context.Context, dropID int32) ([]*DropResult, error) {
	// 1. 查找所有属于该 dropID 的 DropGroup
	var groups []*gameconfig.DropGroup
	for _, group := range gameconfig.T.TbDropGroup.GetDataList() {
		if group.DropId == dropID {
			groups = append(groups, group)
		}
	}

	if len(groups) == 0 {
		return nil, fmt.Errorf("drop_id %d not found in drop groups", dropID)
	}

	var results []*DropResult
	for _, group := range groups {
		// 根据 RollCount 执行多次该组的掉落
		for i := int32(0); i < group.RollCount; i++ {
			items, err := s.executeGroupDrop(group)
			if err != nil {
				return nil, err
			}
			if items != nil {
				results = append(results, items...)
			}
		}
	}

	return results, nil
}

// executeGroupDrop 执行具体的组掉落逻辑
func (s *DropService) executeGroupDrop(group *gameconfig.DropGroup) ([]*DropResult, error) {
	// 1. 查找该组下的所有掉落项
	var items []*gameconfig.DropItem
	totalWeight := int32(0)
	for _, item := range gameconfig.T.TbDropItem.GetDataList() {
		if item.GroupId == group.Id {
			items = append(items, item)
			totalWeight += item.DropValue
		}
	}

	if len(items) == 0 {
		return nil, nil
	}

	// 2. 根据配置的 DropType 执行不同算法
	switch group.DropType {
	case gameconfig.DropType_WEIGHT: // 权重随机（从列表中按权重抽取一个）
		if totalWeight <= 0 {
			return nil, nil
		}
		r := s.rand.Int31n(totalWeight)
		current := int32(0)
		for _, item := range items {
			current += item.DropValue
			if r < current {
				return []*DropResult{s.createDropResult(item)}, nil
			}
		}

	case gameconfig.DropType_PROBABILITY: // 独立概率（每个项按万分比独立判定是否掉落）
		var results []*DropResult
		for _, item := range items {
			if s.rand.Int31n(10000) < item.DropValue {
				results = append(results, s.createDropResult(item))
			}
		}
		return results, nil

	case gameconfig.DropType_FIXED: // 固定掉落（组内所有项全部掉落）
		var results []*DropResult
		for _, item := range items {
			results = append(results, s.createDropResult(item))
		}
		return results, nil
	}

	return nil, nil
}

// createDropResult 根据配置项生成最终的掉落结果
func (s *DropService) createDropResult(item *gameconfig.DropItem) *DropResult {
	count := item.CountMin
	if item.CountMax > item.CountMin {
		// 在 [CountMin, CountMax] 范围内取随机数
		count = item.CountMin + s.rand.Int31n(item.CountMax-item.CountMin+1)
	}
	return &DropResult{
		ItemID: item.ItemId,
		Count:  count,
	}
}
