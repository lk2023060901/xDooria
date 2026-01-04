package model

import (
	"fmt"
	"math"
	"sort"
	"sync"

	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
)

// BaseBag 通用背包实现
type BaseBag struct {
	mu       sync.RWMutex
	id       int64
	roleID   int64
	bagType  int32 // 背包类型：1装扮 2道具 3装备栏
	capacity int32
	items    map[int32]*Item // key: slotIndex, value: Item
}

// NewBaseBag 创建通用背包
func NewBaseBag(id, roleID int64, bagType, capacity int32) *BaseBag {
	return &BaseBag{
		id:       id,
		roleID:   roleID,
		bagType:  bagType,
		capacity: capacity,
		items:    make(map[int32]*Item),
	}
}

// ===== 基础信息 =====

func (b *BaseBag) GetID() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.id
}

func (b *BaseBag) GetRoleID() int64 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.roleID
}

func (b *BaseBag) GetType() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.bagType
}

func (b *BaseBag) GetCapacity() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.capacity
}

func (b *BaseBag) GetItemCount() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return int32(len(b.items))
}

func (b *BaseBag) IsFull() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return int32(len(b.items)) >= b.capacity
}

// ===== 槽位查询 =====

func (b *BaseBag) GetItem(slotIndex int32) *Item {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.items[slotIndex]
}

func (b *BaseBag) HasItem(slotIndex int32) bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	_, exists := b.items[slotIndex]
	return exists
}

func (b *BaseBag) FindEmptySlot() int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for i := int32(0); i < b.capacity; i++ {
		if _, exists := b.items[i]; !exists {
			return i
		}
	}
	return -1
}

func (b *BaseBag) FindItems(configID int32) []*Item {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*Item, 0)
	for _, item := range b.items {
		if item.ConfigID == configID {
			result = append(result, item)
		}
	}
	return result
}

func (b *BaseBag) GetAllItems() []*Item {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make([]*Item, 0, len(b.items))
	for _, item := range b.items {
		result = append(result, item)
	}
	return result
}

// ===== 添加物品（完整物品对象） =====

func (b *BaseBag) CanAdd(item *Item, slotIndex int32) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if slotIndex < 0 || slotIndex >= b.capacity {
		return fmt.Errorf("invalid slot index: %d", slotIndex)
	}

	if _, exists := b.items[slotIndex]; exists {
		return fmt.Errorf("slot %d is already occupied", slotIndex)
	}

	return nil
}

func (b *BaseBag) Add(item *Item, slotIndex int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if slotIndex < 0 || slotIndex >= b.capacity {
		return fmt.Errorf("invalid slot index: %d", slotIndex)
	}

	if _, exists := b.items[slotIndex]; exists {
		return fmt.Errorf("slot %d is already occupied", slotIndex)
	}

	item.SlotIndex = slotIndex
	item.IsNew = true // 标记为新获得
	b.items[slotIndex] = item
	return nil
}

func (b *BaseBag) AutoAdd(item *Item) (int32, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i := int32(0); i < b.capacity; i++ {
		if _, exists := b.items[i]; !exists {
			item.SlotIndex = i
			item.IsNew = true // 标记为新获得
			b.items[i] = item
			return i, nil
		}
	}

	return -1, fmt.Errorf("bag is full")
}

// ===== 添加物品（仅配置ID和数量） =====

func (b *BaseBag) AddItemByConfigID(itemConfigID int32, count int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	// 获取物品最大堆叠数量
	maxStack, err := getMaxStack(itemConfigID)
	if err != nil {
		return err
	}

	// 根据背包类型判断是否可堆叠
	canStack := b.canStackByBagType()

	remaining := count

	if canStack {
		// 1. 先尝试堆叠到已有物品（考虑最大堆叠数量）
		for _, item := range b.items {
			if item.ConfigID == itemConfigID && remaining > 0 {
				// 计算当前格子还能堆叠多少
				canAdd := maxStack - item.Count
				if canAdd > 0 {
					if canAdd >= remaining {
						item.Count += remaining
						item.IsNew = true // 数量变化，标记为新获得
						return nil
					} else {
						item.Count += canAdd
						item.IsNew = true // 数量变化，标记为新获得
						remaining -= canAdd
					}
				}
			}
		}
	}

	// 2. 创建新物品（可能需要多个格子）
	for i := int32(0); i < b.capacity && remaining > 0; i++ {
		if _, exists := b.items[i]; !exists {
			stackCount := min(remaining, maxStack)

			newItem := &Item{
				RoleID:     b.roleID,
				ConfigID:   itemConfigID,
				SlotIndex:  i,
				Count:      stackCount,
				IsNew:      true, // 新物品，标记为新获得
				CreateTime: 0,    // TODO: 使用实际时间
				UpdateTime: 0,
			}
			b.items[i] = newItem
			remaining -= stackCount
		}
	}

	if remaining > 0 {
		return fmt.Errorf("bag is full, %d items not added", remaining)
	}

	return nil
}

// ===== 移除物品 =====

func (b *BaseBag) Remove(slotIndex int32) (*Item, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	item, exists := b.items[slotIndex]
	if !exists {
		return nil, fmt.Errorf("slot %d is empty", slotIndex)
	}

	delete(b.items, slotIndex)
	return item, nil
}

func (b *BaseBag) ReduceCount(slotIndex int32, count int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	item, exists := b.items[slotIndex]
	if !exists {
		return fmt.Errorf("slot %d is empty", slotIndex)
	}

	if item.Count < count {
		return fmt.Errorf("insufficient item count: have %d, need %d", item.Count, count)
	}

	item.Count -= count
	if item.Count <= 0 {
		delete(b.items, slotIndex)
	}

	return nil
}

func (b *BaseBag) RemoveItemByConfigID(itemConfigID int32, count int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if count <= 0 {
		return fmt.Errorf("invalid count: %d", count)
	}

	// 先计算总数量
	totalCount := int32(0)
	for _, item := range b.items {
		if item.ConfigID == itemConfigID {
			totalCount += item.Count
		}
	}

	if totalCount < count {
		return fmt.Errorf("insufficient item count: have %d, need %d", totalCount, count)
	}

	// 开始扣除
	remaining := count
	for slotIndex, item := range b.items {
		if item.ConfigID == itemConfigID {
			if item.Count >= remaining {
				item.Count -= remaining
				if item.Count <= 0 {
					delete(b.items, slotIndex)
				}
				return nil
			} else {
				remaining -= item.Count
				delete(b.items, slotIndex)
			}
		}
	}

	return nil
}

// ===== 查询物品数量 =====

func (b *BaseBag) GetItemCountByConfigID(itemConfigID int32) int32 {
	b.mu.RLock()
	defer b.mu.RUnlock()

	totalCount := int32(0)
	for _, item := range b.items {
		if item.ConfigID == itemConfigID {
			totalCount += item.Count
		}
	}
	return totalCount
}

func (b *BaseBag) HasItemByConfigID(itemConfigID int32, count int32) bool {
	return b.GetItemCountByConfigID(itemConfigID) >= count
}

// ===== 移动物品 =====

func (b *BaseBag) Move(fromSlot, toSlot int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if fromSlot < 0 || fromSlot >= b.capacity || toSlot < 0 || toSlot >= b.capacity {
		return fmt.Errorf("invalid slot index")
	}

	item, exists := b.items[fromSlot]
	if !exists {
		return fmt.Errorf("source slot %d is empty", fromSlot)
	}

	if _, exists := b.items[toSlot]; exists {
		return fmt.Errorf("target slot %d is occupied", toSlot)
	}

	delete(b.items, fromSlot)
	item.SlotIndex = toSlot
	b.items[toSlot] = item

	return nil
}

func (b *BaseBag) Swap(slotA, slotB int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if slotA < 0 || slotA >= b.capacity || slotB < 0 || slotB >= b.capacity {
		return fmt.Errorf("invalid slot index")
	}

	itemA, existsA := b.items[slotA]
	itemB, existsB := b.items[slotB]

	if !existsA && !existsB {
		return fmt.Errorf("both slots are empty")
	}

	if existsA {
		itemA.SlotIndex = slotB
		b.items[slotB] = itemA
	} else {
		delete(b.items, slotB)
	}

	if existsB {
		itemB.SlotIndex = slotA
		b.items[slotA] = itemB
	} else {
		delete(b.items, slotA)
	}

	return nil
}

// ===== 堆叠整理 =====

func (b *BaseBag) CanStack(itemA, itemB *Item) bool {
	if itemA == nil || itemB == nil {
		return false
	}

	// 装扮背包不可堆叠
	if b.bagType == gameconfig.BagType_Costume {
		return false
	}

	// 相同配置ID且绑定类型相同才可堆叠
	return itemA.ConfigID == itemB.ConfigID && itemA.BindType == itemB.BindType
}

func (b *BaseBag) Stack(fromSlot, toSlot int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	itemFrom, existsFrom := b.items[fromSlot]
	itemTo, existsTo := b.items[toSlot]

	if !existsFrom || !existsTo {
		return fmt.Errorf("slot is empty")
	}

	if !b.CanStack(itemFrom, itemTo) {
		return fmt.Errorf("items cannot be stacked")
	}

	// 获取物品最大堆叠数量
	maxStack, err := getMaxStack(itemTo.ConfigID)
	if err != nil {
		return err
	}

	// 计算目标格子还能堆叠多少
	canAdd := maxStack - itemTo.Count
	if canAdd <= 0 {
		return fmt.Errorf("target slot is already full (max: %d)", maxStack)
	}

	// 堆叠物品
	if itemFrom.Count <= canAdd {
		// 全部堆叠过去
		itemTo.Count += itemFrom.Count
		delete(b.items, fromSlot)
	} else {
		// 部分堆叠
		itemTo.Count += canAdd
		itemFrom.Count -= canAdd
	}

	return nil
}

func (b *BaseBag) Sort(sortType int32, ascending bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 收集所有物品
	items := make([]*Item, 0, len(b.items))
	for _, item := range b.items {
		items = append(items, item)
	}

	// 获取排序策略
	comparator, err := getSortComparator(sortType, ascending)
	if err != nil {
		return err
	}

	// 执行排序
	sort.Slice(items, func(i, j int) bool {
		return comparator(items[i], items[j])
	})

	// 重新放置物品
	b.items = make(map[int32]*Item)
	for i, item := range items {
		item.SlotIndex = int32(i)
		b.items[int32(i)] = item
	}

	return nil
}

func (b *BaseBag) Compact() ([]*Item, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// 根据背包类型判断是否可堆叠
	canStack := b.canStackByBagType()
	toDelete := make([]*Item, 0)

	if canStack {
		// 1. 按 ConfigID + BindType 分组合并可堆叠物品
		type stackKey struct {
			ConfigID int32
			BindType int32
		}

		// 用于合并的临时数据结构
		stackGroups := make(map[stackKey][]*Item)

		// 收集所有物品并分组
		for _, item := range b.items {
			key := stackKey{
				ConfigID: item.ConfigID,
				BindType: item.BindType,
			}
			stackGroups[key] = append(stackGroups[key], item)
		}

		// 合并后的物品列表
		mergedItems := make([]*Item, 0)

		// 对每组进行堆叠合并
		for _, group := range stackGroups {
			if len(group) == 0 {
				continue
			}

			// 获取最大堆叠数量
			maxStack, err := getMaxStack(group[0].ConfigID)
			if err != nil {
				return nil, err
			}

			// 计算总数量
			totalCount := int32(0)
			for _, item := range group {
				totalCount += item.Count
			}

			// 按最大堆叠数量分割成多个格子
			for totalCount > 0 {
				stackCount := min(totalCount, maxStack)

				// 复用第一个物品对象（保留ID等信息）
				if len(group) > 0 {
					item := group[0]
					item.Count = stackCount
					mergedItems = append(mergedItems, item)
					group = group[1:]
				} else {
					// 不应该到这里，但保护一下
					break
				}

				totalCount -= stackCount
			}

			// 剩余的物品需要删除
			for _, item := range group {
				toDelete = append(toDelete, item)
			}
		}

		// 2. 重新排列，消除空隙
		b.items = make(map[int32]*Item)
		for i, item := range mergedItems {
			item.SlotIndex = int32(i)
			b.items[int32(i)] = item
		}
	} else {
		// 装扮类物品不可堆叠，只消除空隙
		items := make([]*Item, 0, len(b.items))
		for _, item := range b.items {
			items = append(items, item)
		}

		b.items = make(map[int32]*Item)
		for i, item := range items {
			item.SlotIndex = int32(i)
			b.items[int32(i)] = item
		}
	}

	return toDelete, nil
}

// ===== 容量管理 =====

func (b *BaseBag) Expand(addCapacity int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if addCapacity <= 0 {
		return fmt.Errorf("invalid capacity: %d", addCapacity)
	}

	b.capacity += addCapacity
	return nil
}

// ===== 辅助函数 =====

// canStackByBagType 根据背包类型判断是否可堆叠
func (b *BaseBag) canStackByBagType() bool {
	// 装扮背包不可堆叠
	return b.bagType != gameconfig.BagType_Costume
}

// getItemConfig 从配置表获取物品配置
// TODO: 需要注入全局配置表实例
func getItemConfig(configID int32) *gameconfig.Item {
	return gameconfig.T.TbItem.Get(configID)
}

// getMaxStack 获取物品的最大堆叠数量
// 如果配置的 MaxStack <= 0，表示无堆叠上限，返回 math.MaxInt32
func getMaxStack(itemConfigID int32) (int32, error) {
	cfg := getItemConfig(itemConfigID)
	if cfg == nil {
		return 0, fmt.Errorf("item config not found: %d", itemConfigID)
	}

	if cfg.MaxStack <= 0 {
		return math.MaxInt32, nil // 无堆叠上限
	}

	return cfg.MaxStack, nil
}

// ===== 排序策略 =====

// ItemComparator 物品比较函数类型
type ItemComparator func(itemA, itemB *Item) bool

// getSortComparator 根据排序类型获取对应的比较器
func getSortComparator(sortType int32, ascending bool) (ItemComparator, error) {
	// 先应用收藏优先装饰器
	var baseComparator ItemComparator

	switch sortType {
	case gameconfig.BagSortType_Type:
		baseComparator = makeSortByType(ascending)
	case gameconfig.BagSortType_Quality:
		baseComparator = makeSortByQuality(ascending)
	case gameconfig.BagSortType_Id:
		baseComparator = makeSortById(ascending)
	case gameconfig.BagSortType_ExpireTime:
		baseComparator = makeSortByExpireTime(ascending)
	default:
		return nil, fmt.Errorf("invalid sort type: %d", sortType)
	}

	// 应用收藏优先装饰器
	return withFavoritePriority(baseComparator), nil
}

// withFavoritePriority 收藏优先装饰器
func withFavoritePriority(comparator ItemComparator) ItemComparator {
	return func(itemA, itemB *Item) bool {
		// 收藏优先
		if itemA.IsFavorite != itemB.IsFavorite {
			return itemA.IsFavorite
		}
		return comparator(itemA, itemB)
	}
}

// makeSortByType 按类型排序
func makeSortByType(ascending bool) ItemComparator {
	return func(itemA, itemB *Item) bool {
		cfgA := getItemConfig(itemA.ConfigID)
		cfgB := getItemConfig(itemB.ConfigID)
		if cfgA == nil || cfgB == nil {
			return itemA.ConfigID < itemB.ConfigID
		}

		// 先按大类排序
		if cfgA.Type != cfgB.Type {
			if ascending {
				return cfgA.Type < cfgB.Type
			}
			return cfgA.Type > cfgB.Type
		}
		// 大类相同，按子类排序
		if cfgA.SubType != cfgB.SubType {
			if ascending {
				return cfgA.SubType < cfgB.SubType
			}
			return cfgA.SubType > cfgB.SubType
		}
		// 子类相同，按配置的排序字段
		return cfgA.Id < cfgB.Id
	}
}

// makeSortByQuality 按品质排序
func makeSortByQuality(ascending bool) ItemComparator {
	return func(itemA, itemB *Item) bool {
		cfgA := getItemConfig(itemA.ConfigID)
		cfgB := getItemConfig(itemB.ConfigID)
		if cfgA == nil || cfgB == nil {
			return itemA.ConfigID < itemB.ConfigID
		}

		// 按品质排序
		if cfgA.Quality != cfgB.Quality {
			if ascending {
				return cfgA.Quality < cfgB.Quality
			}
			return cfgA.Quality > cfgB.Quality
		}
		// 品质相同，按配置的排序字段
		return cfgA.Id < cfgB.Id
	}
}

// makeSortById 按道具ID排序
func makeSortById(ascending bool) ItemComparator {
	return func(itemA, itemB *Item) bool {
		if ascending {
			return itemA.ConfigID < itemB.ConfigID
		}
		return itemA.ConfigID > itemB.ConfigID
	}
}

// makeSortByExpireTime 按限时排序
func makeSortByExpireTime(ascending bool) ItemComparator {
	return func(itemA, itemB *Item) bool {
		expireA := itemA.ExpireTime
		expireB := itemB.ExpireTime

		// 0表示永久，永久的排后面
		if expireA == 0 && expireB == 0 {
			return itemA.ConfigID < itemB.ConfigID
		}
		if expireA == 0 {
			return false // 永久的排后面
		}
		if expireB == 0 {
			return true // 永久的排后面
		}

		// 都是限时的，按过期时间排序
		if ascending {
			return expireA < expireB // 升序：快过期的在前
		}
		return expireA > expireB // 降序：慢过期的在前
	}
}

// ===== 标记管理 =====

// SetFavorite 设置收藏状态
func (b *BaseBag) SetFavorite(slotIndex int32, favorite bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	item, exists := b.items[slotIndex]
	if !exists {
		return fmt.Errorf("slot %d is empty", slotIndex)
	}

	item.IsFavorite = favorite
	return nil
}

// ClearNew 清除新获得标记
func (b *BaseBag) ClearNew(slotIndex int32) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	item, exists := b.items[slotIndex]
	if !exists {
		return fmt.Errorf("slot %d is empty", slotIndex)
	}

	item.IsNew = false
	return nil
}

// ===== 过期物品处理 =====

// GetExpiredItems 获取所有过期物品
// currentTime: 当前时间戳
// 返回所有 ExpireTime > 0 且 ExpireTime <= currentTime 的物品
func (b *BaseBag) GetExpiredItems(currentTime int64) []*Item {
	b.mu.RLock()
	defer b.mu.RUnlock()

	expiredItems := make([]*Item, 0)
	for _, item := range b.items {
		if item.ExpireTime > 0 && item.ExpireTime <= currentTime {
			expiredItems = append(expiredItems, item)
		}
	}

	return expiredItems
}
