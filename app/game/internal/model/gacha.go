package model

// GachaRecord 抽卡玩法记录
type GachaRecord struct {
	ID         int32 `json:"id"`          // 玩法ID (池子ID)
	Type       int32 `json:"type"`        // 玩法类型 (如：1=普通, 2=稀有)
	TotalCount int32 `json:"total_count"` // 该记录下的累计抽取次数
	LastTime   int64 `json:"last_time"`   // 最后操作时间 (Unix)
}

// PlayerGacha 玩家抽卡全记录
type PlayerGacha struct {
	RoleID  int64          `json:"role_id"`
	Records []*GachaRecord `json:"records"`
}
