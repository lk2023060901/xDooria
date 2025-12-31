package model

import "time"

// Doll 玩偶实例（数据库实体）
// 对应表：player_doll
type Doll struct {
	ID         int64     // 实例ID（雪花ID）
	PlayerID   int64     // 玩家ID
	DollID     int32     // 配置ID（关联Doll配置表）
	Quality    int16     // 当前品质（可通过熔炼提升）
	IsLocked   bool      // 锁定状态（防误熔炼）
	IsRedeemed bool      // 是否已兑换实物
	CreatedAt  time.Time // 创建时间
}
