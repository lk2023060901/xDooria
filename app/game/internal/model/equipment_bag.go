package model

import (
	"fmt"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
)

// EquipmentBag 装备栏实现（带槽位限制）
type EquipmentBag struct {
	*BaseBag
}

// NewEquipmentBag 创建装备栏
func NewEquipmentBag(id, roleID int64, capacity int32) *EquipmentBag {
	return &EquipmentBag{
		BaseBag: NewBaseBag(id, roleID, gameconfig.BagType_Equipment, capacity),
	}
}

// ===== 装备栏特殊逻辑：槽位限制 =====

// 装备栏槽位映射（槽位索引 -> 装备子类型）
var equipmentSlotMap = map[int32]int32{
	0: gameconfig.ItemSubType_EquipWeapon,  // 武器
	1: gameconfig.ItemSubType_EquipHelmet,  // 头盔
	2: gameconfig.ItemSubType_EquipGloves,  // 手套
	3: gameconfig.ItemSubType_EquipArmor,   // 上衣
	4: gameconfig.ItemSubType_EquipPants,   // 裤子
	5: gameconfig.ItemSubType_EquipBoots,   // 鞋子
}

// validateSlot 验证物品是否可以放入指定槽位
func (b *EquipmentBag) validateSlot(item *Item, slotIndex int32) error {
	// 获取该槽位允许的装备类型
	allowedSubType, exists := equipmentSlotMap[slotIndex]
	if !exists {
		return fmt.Errorf("invalid equipment slot: %d", slotIndex)
	}

	// 获取物品配置
	cfg := getItemConfig(item.ConfigID)
	if cfg == nil {
		return fmt.Errorf("item config not found: %d", item.ConfigID)
	}

	// 验证物品类型
	if cfg.Type != gameconfig.ItemType_Equipment {
		return fmt.Errorf("item is not equipment: %d", item.ConfigID)
	}

	// 验证物品子类型是否匹配槽位
	if cfg.SubType != allowedSubType {
		return fmt.Errorf("item subtype %d cannot be equipped in slot %d (requires %d)",
			cfg.SubType, slotIndex, allowedSubType)
	}

	return nil
}

// ===== 重写需要槽位验证的方法 =====

func (b *EquipmentBag) CanAdd(item *Item, slotIndex int32) error {
	// 先调用基类验证
	if err := b.BaseBag.CanAdd(item, slotIndex); err != nil {
		return err
	}

	// 再验证槽位限制
	return b.validateSlot(item, slotIndex)
}

func (b *EquipmentBag) Add(item *Item, slotIndex int32) error {
	// 验证槽位限制
	if err := b.validateSlot(item, slotIndex); err != nil {
		return err
	}

	// 调用基类方法
	return b.BaseBag.Add(item, slotIndex)
}

func (b *EquipmentBag) AutoAdd(item *Item) (int32, error) {
	// 获取物品配置
	cfg := getItemConfig(item.ConfigID)
	if cfg == nil {
		return -1, fmt.Errorf("item config not found: %d", item.ConfigID)
	}

	// 验证物品类型
	if cfg.Type != gameconfig.ItemType_Equipment {
		return -1, fmt.Errorf("item is not equipment: %d", item.ConfigID)
	}

	// 找到对应的槽位
	targetSlot := int32(-1)
	for slot, subType := range equipmentSlotMap {
		if subType == cfg.SubType {
			targetSlot = slot
			break
		}
	}

	if targetSlot == -1 {
		return -1, fmt.Errorf("no slot found for equipment subtype: %d", cfg.SubType)
	}

	// 检查槽位是否为空
	if b.HasItem(targetSlot) {
		return -1, fmt.Errorf("equipment slot %d is already occupied", targetSlot)
	}

	// 添加到对应槽位
	if err := b.Add(item, targetSlot); err != nil {
		return -1, err
	}

	return targetSlot, nil
}

// ===== 装备栏不支持的操作 =====

func (b *EquipmentBag) AddItemByConfigID(itemConfigID int32, count int32) error {
	return fmt.Errorf("equipment bag does not support AddItemByConfigID")
}

func (b *EquipmentBag) Sort(sortType int32, ascending bool) error {
	return fmt.Errorf("equipment bag does not support sorting")
}

func (b *EquipmentBag) Compact() ([]*Item, error) {
	return nil, fmt.Errorf("equipment bag does not support compacting")
}

func (b *EquipmentBag) CanStack(itemA, itemB *Item) bool {
	// 装备不可堆叠
	return false
}

func (b *EquipmentBag) Stack(fromSlot, toSlot int32) error {
	return fmt.Errorf("equipment cannot be stacked")
}
