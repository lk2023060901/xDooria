package model

import (
	"database/sql"
	"encoding/json"
	"time"
)

// Role 角色模型，对应 roles 表
type Role struct {
	// 基础信息
	ID        int64  `db:"id"`
	UID       int64  `db:"uid"`
	Nickname  string `db:"nickname"`
	Gender    int16  `db:"gender"`
	Signature string `db:"signature"`
	AvatarURL string `db:"avatar_url"`

	// 形象装扮（JSONB 字段）
	Appearance json.RawMessage `db:"appearance"`
	Outfit     json.RawMessage `db:"outfit"`

	// 经济系统
	Gold    int64 `db:"gold"`
	Diamond int64 `db:"diamond"`

	// 等级成长
	Level    int32 `db:"level"`
	Exp      int64 `db:"exp"`
	VIPLevel int16 `db:"vip_level"`
	VIPExp   int32 `db:"vip_exp"`

	// 封禁状态
	Status      int16          `db:"status"`
	BanExpireAt sql.NullTime   `db:"ban_expire_at"`

	// 时间戳
	LastLoginAt sql.NullTime  `db:"last_login_at"`
	CreatedAt   time.Time     `db:"created_at"`
	UpdatedAt   time.Time     `db:"updated_at"`
}

// RoleStatus 角色状态枚举
const (
	RoleStatusNormal = 0 // 正常
	RoleStatusBanned = 1 // 封禁
)

// NewRole 创建新角色实例
func NewRole(uid int64, nickname string, gender int16, appearance json.RawMessage) *Role {
	now := time.Now()
	return &Role{
		UID:        uid,
		Nickname:   nickname,
		Gender:     gender,
		Appearance: appearance,
		Outfit:     json.RawMessage("{}"),
		Gold:       0,
		Diamond:    0,
		Level:      1,
		Exp:        0,
		VIPLevel:   0,
		VIPExp:     0,
		Status:     RoleStatusNormal,
		Signature:  "",
		AvatarURL:  "",
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// IsBanned 判断角色是否被封禁
func (r *Role) IsBanned() bool {
	if r.Status != RoleStatusBanned {
		return false
	}

	// 如果没有设置到期时间，表示永久封禁
	if !r.BanExpireAt.Valid {
		return true
	}

	// 检查是否已过期
	return time.Now().Before(r.BanExpireAt.Time)
}

// UpdateLastLogin 更新最后登录时间
func (r *Role) UpdateLastLogin() {
	r.LastLoginAt = sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}
}
