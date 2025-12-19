package redis

import "time"

// PoolStats 连接池统计信息（隐藏 go-redis 类型）
type PoolStats struct {
	Hits       uint32 // 连接池命中次数
	Misses     uint32 // 连接池未命中次数
	Timeouts   uint32 // 超时次数
	TotalConns uint32 // 总连接数
	IdleConns  uint32 // 空闲连接数
	StaleConns uint32 // 过期连接数
}

// ZItem 有序集合元素（隐藏 go-redis 类型）
type ZItem struct {
	Member string  // 成员
	Score  float64 // 分数
}

// ScanResult 扫描结果
type ScanResult struct {
	Keys   []string // 键列表
	Cursor uint64   // 游标
}

// Message Pub/Sub 消息（隐藏 go-redis 类型）
type Message struct {
	Channel string // 频道
	Pattern string // 模式（模式订阅时使用）
	Payload string // 消息内容
}

// PipelineResult Pipeline 执行结果
type PipelineResult struct {
	Err error // 错误信息
}

// TTLResult TTL 查询结果
type TTLResult struct {
	TTL    time.Duration // 过期时间
	Exists bool          // 键是否存在
}
