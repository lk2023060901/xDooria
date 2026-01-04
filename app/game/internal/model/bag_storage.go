package model

// PlayerBag 玩家单个类型的背包数据
type PlayerBag struct {
	RoleID  int64           `json:"role_id"`
	BagType int32           `json:"bag_type"`
	Items   map[int32]int32 `json:"items"` // Key: ItemID (配置ID), Value: Count
}

// NewPlayerBag 创建一个新的背包实例
func NewPlayerBag(roleID int64, bagType int32) *PlayerBag {
	return &PlayerBag{
		RoleID:  roleID,
		BagType: bagType,
		Items:   make(map[int32]int32),
	}
}
