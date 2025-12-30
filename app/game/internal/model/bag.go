package model

// Item 物品实例（数据库实体 + 运行时对象）
type Item struct {
	ID         int64 // 物品唯一ID
	RoleID     int64 // 所属角色ID
	ConfigID   int32 // 配置ID（关联item配置表）
	SlotIndex  int32 // 槽位索引
	Count      int32 // 数量（堆叠用）
	BindType   int32 // 绑定类型
	ExpireTime int64 // 过期时间（时间戳，0表示永久）
	IsFavorite bool  // 是否收藏（所有背包类型）
	IsNew      bool  // 是否新获得/数量变化（所有背包类型）
	IsEquipped bool  // 是否已装备（仅装备栏、装扮背包）
	CreateTime int64 // 创建时间
	UpdateTime int64 // 更新时间
}

// Bag 背包接口（所有类型背包的统一接口）
type Bag interface {
	// ===== 基础信息 =====
	GetID() int64      // 获取背包ID
	GetRoleID() int64  // 获取所属角色ID
	GetType() int32    // 获取背包类型：1装扮 2道具 3装备栏
	GetCapacity() int32 // 获取背包容量
	GetItemCount() int32 // 获取当前物品数量
	IsFull() bool      // 是否已满

	// ===== 槽位查询 =====
	GetItem(slotIndex int32) *Item        // 获取指定槽位的物品
	HasItem(slotIndex int32) bool         // 检查槽位是否有物品
	FindEmptySlot() int32                 // 查找第一个空槽位，返回-1表示无空位
	FindItems(configID int32) []*Item     // 根据配置ID查找所有该类物品
	GetAllItems() []*Item                 // 获取所有物品列表

	// ===== 添加物品（完整物品对象） =====
	CanAdd(item *Item, slotIndex int32) error // 检查是否可以添加到指定槽位
	Add(item *Item, slotIndex int32) error    // 添加物品到指定槽位
	AutoAdd(item *Item) (int32, error)        // 自动寻找空位添加，返回槽位索引

	// ===== 添加物品（仅配置ID和数量） =====
	AddItemByConfigID(itemConfigID int32, count int32) error // 根据配置ID添加物品（自动堆叠）

	// ===== 移除物品 =====
	Remove(slotIndex int32) (*Item, error)              // 移除整个槽位的物品
	ReduceCount(slotIndex int32, count int32) error     // 减少指定槽位的数量（堆叠物品用）
	RemoveItemByConfigID(itemConfigID int32, count int32) error // 根据配置ID移除指定数量的物品

	// ===== 查询物品数量 =====
	GetItemCountByConfigID(itemConfigID int32) int32           // 获取指定配置ID物品的总数量
	HasItemByConfigID(itemConfigID int32, count int32) bool    // 检查是否拥有足够数量

	// ===== 移动物品 =====
	Move(fromSlot, toSlot int32) error // 移动物品到新槽位
	Swap(slotA, slotB int32) error     // 交换两个槽位的物品

	// ===== 堆叠整理 =====
	CanStack(itemA, itemB *Item) bool      // 判断两个物品是否可堆叠
	Stack(fromSlot, toSlot int32) error    // 堆叠物品（fromSlot堆叠到toSlot）
	Sort(sortType int32, ascending bool) error // 排序背包：sortType=排序类型，ascending=true升序/false降序
	Compact() ([]*Item, error)             // 压缩背包（合并可堆叠物品，消除空隙），返回需要删除的物品

	// ===== 容量管理 =====
	Expand(addCapacity int32) error // 扩展背包容量

	// ===== 标记管理 =====
	SetFavorite(slotIndex int32, favorite bool) error // 设置收藏状态
	ClearNew(slotIndex int32) error                   // 清除新获得标记

	// ===== 过期物品处理 =====
	GetExpiredItems(currentTime int64) []*Item // 获取所有过期物品（ExpireTime > 0 且 <= currentTime）
}
