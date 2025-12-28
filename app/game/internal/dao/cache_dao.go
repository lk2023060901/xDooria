package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/database/redis"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

const (
	// Redis key 前缀
	roleKeyPrefix    = "cache:role:"
	sessionKeyPrefix = "session:player:"
	onlineKeyPrefix  = "online:player:"

	// TTL
	roleCacheTTL    = 30 * time.Minute
	sessionTTL      = 1 * time.Hour
	onlineStatusTTL = 60 * time.Second
)

// CacheDAO 缓存数据访问对象
type CacheDAO struct {
	redis   *redis.Client
	logger  logger.Logger
	metrics *metrics.GameMetrics
}

// NewCacheDAO 创建缓存 DAO
func NewCacheDAO(rdb *redis.Client, l logger.Logger, m *metrics.GameMetrics) *CacheDAO {
	return &CacheDAO{
		redis:   rdb,
		logger:  l.Named("dao.cache"),
		metrics: m,
	}
}

// GetRole 从缓存获取角色
func (d *CacheDAO) GetRole(ctx context.Context, roleID int64) (*model.Role, error) {
	key := fmt.Sprintf("%s%d", roleKeyPrefix, roleID)

	data, err := d.redis.Get(ctx, key)
	if err != nil {
		if err == redis.ErrNil {
			d.metrics.RecordCacheMiss("redis")
			return nil, nil
		}
		d.logger.Error("failed to get role from cache",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get role from cache: %w", err)
	}

	d.metrics.RecordCacheHit("redis")

	var role model.Role
	if err := json.Unmarshal([]byte(data), &role); err != nil {
		d.logger.Error("failed to unmarshal role",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to unmarshal role: %w", err)
	}

	return &role, nil
}

// SetRole 设置角色缓存
func (d *CacheDAO) SetRole(ctx context.Context, role *model.Role, ttl time.Duration) error {
	key := fmt.Sprintf("%s%d", roleKeyPrefix, role.ID)

	data, err := json.Marshal(role)
	if err != nil {
		d.logger.Error("failed to marshal role",
			"role_id", role.ID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal role: %w", err)
	}

	if ttl == 0 {
		ttl = roleCacheTTL
	}

	if err := d.redis.Set(ctx, key, string(data), ttl); err != nil {
		d.logger.Error("failed to set role cache",
			"role_id", role.ID,
			"error", err,
		)
		return fmt.Errorf("failed to set role cache: %w", err)
	}

	return nil
}

// DeleteRole 删除角色缓存
func (d *CacheDAO) DeleteRole(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", roleKeyPrefix, roleID)

	deleted, err := d.redis.Del(ctx, key)
	if err != nil {
		d.logger.Error("failed to delete role cache",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("failed to delete role cache: %w", err)
	}

	d.logger.Debug("deleted role cache",
		"role_id", roleID,
		"deleted_count", deleted,
	)

	return nil
}

// GetSession 获取会话信息
func (d *CacheDAO) GetSession(ctx context.Context, roleID int64) (*model.Session, error) {
	key := fmt.Sprintf("%s%d", sessionKeyPrefix, roleID)

	data, err := d.redis.Get(ctx, key)
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		d.logger.Error("failed to get session from cache",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get session from cache: %w", err)
	}

	var session model.Session
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		d.logger.Error("failed to unmarshal session",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to unmarshal session: %w", err)
	}

	return &session, nil
}

// SetSession 设置会话信息
func (d *CacheDAO) SetSession(ctx context.Context, session *model.Session, ttl time.Duration) error {
	key := fmt.Sprintf("%s%d", sessionKeyPrefix, session.RoleID)

	data, err := json.Marshal(session)
	if err != nil {
		d.logger.Error("failed to marshal session",
			"role_id", session.RoleID,
			"error", err,
		)
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if ttl == 0 {
		ttl = sessionTTL
	}

	if err := d.redis.Set(ctx, key, string(data), ttl); err != nil {
		d.logger.Error("failed to set session cache",
			"role_id", session.RoleID,
			"error", err,
		)
		return fmt.Errorf("failed to set session cache: %w", err)
	}

	return nil
}

// DeleteSession 删除会话信息
func (d *CacheDAO) DeleteSession(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", sessionKeyPrefix, roleID)

	deleted, err := d.redis.Del(ctx, key)
	if err != nil {
		d.logger.Error("failed to delete session cache",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("failed to delete session cache: %w", err)
	}

	d.logger.Debug("deleted session cache",
		"role_id", roleID,
		"deleted_count", deleted,
	)

	return nil
}

// SetOnlineStatus 设置在线状态
func (d *CacheDAO) SetOnlineStatus(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", onlineKeyPrefix, roleID)

	if err := d.redis.Set(ctx, key, "1", onlineStatusTTL); err != nil {
		d.logger.Error("failed to set online status",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("failed to set online status: %w", err)
	}

	return nil
}

// IsOnline 检查角色是否在线
func (d *CacheDAO) IsOnline(ctx context.Context, roleID int64) (bool, error) {
	key := fmt.Sprintf("%s%d", onlineKeyPrefix, roleID)

	exists, err := d.redis.Exists(ctx, key)
	if err != nil {
		d.logger.Error("failed to check online status",
			"role_id", roleID,
			"error", err,
		)
		return false, fmt.Errorf("failed to check online status: %w", err)
	}

	return exists > 0, nil
}

// DeleteOnlineStatus 删除在线状态
func (d *CacheDAO) DeleteOnlineStatus(ctx context.Context, roleID int64) error {
	key := fmt.Sprintf("%s%d", onlineKeyPrefix, roleID)

	deleted, err := d.redis.Del(ctx, key)
	if err != nil {
		d.logger.Error("failed to delete online status",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("failed to delete online status: %w", err)
	}

	d.logger.Debug("deleted online status",
		"role_id", roleID,
		"deleted_count", deleted,
	)

	return nil
}
