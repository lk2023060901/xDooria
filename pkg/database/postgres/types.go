package postgres

import (
	"time"

	"github.com/Masterminds/squirrel"
)

// PoolStats 连接池统计信息
type PoolStats struct {
	AcquireCount            int64         // 获取连接的总次数
	AcquireDuration         time.Duration // 获取连接的总时长
	AcquiredConns           int32         // 当前已获取的连接数
	CanceledAcquireCount    int64         // 取消获取连接的次数
	ConstructingConns       int32         // 正在创建的连接数
	EmptyAcquireCount       int64         // 空闲获取的次数
	IdleConns               int32         // 空闲连接数
	MaxConns                int32         // 最大连接数
	TotalConns              int32         // 总连接数
	NewConnsCount           int64         // 新建连接的次数
	MaxLifetimeDestroyCount int64         // 因超过最大生命周期而销毁的连接数
	MaxIdleDestroyCount     int64         // 因超过最大空闲时间而销毁的连接数
}

// QueryBuilder SQL 查询构建器（基于 squirrel）
var QueryBuilder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
