package dao

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/lk2023060901/xdooria/app/game/internal/metrics"
	"github.com/lk2023060901/xdooria/app/game/internal/model"
	"github.com/lk2023060901/xdooria/pkg/database/postgres"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// RoleDAO 角色数据访问对象
type RoleDAO struct {
	db      *postgres.Client
	logger  logger.Logger
	metrics *metrics.GameMetrics
}

// NewRoleDAO 创建角色 DAO
func NewRoleDAO(db *postgres.Client, l logger.Logger, m *metrics.GameMetrics) *RoleDAO {
	return &RoleDAO{
		db:      db,
		logger:  l.Named("dao.role"),
		metrics: m,
	}
}

// GetByID 根据角色 ID 获取角色
func (d *RoleDAO) GetByID(ctx context.Context, roleID int64) (*model.Role, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("select", true, duration)
	}()

	query, args, err := squirrel.
		Select("*").
		From("roles").
		Where(squirrel.Eq{"id": roleID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var role model.Role
	if err := d.db.QueryRow(ctx, query, args...).Scan(
		&role.ID,
		&role.UID,
		&role.Nickname,
		&role.Gender,
		&role.Signature,
		&role.AvatarURL,
		&role.Appearance,
		&role.Outfit,
		&role.Gold,
		&role.Diamond,
		&role.Level,
		&role.Exp,
		&role.VIPLevel,
		&role.VIPExp,
		&role.Status,
		&role.BanExpireAt,
		&role.LastLoginAt,
		&role.CreatedAt,
		&role.UpdatedAt,
	); err != nil {
		d.logger.Error("failed to get role by id",
			"role_id", roleID,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// GetByUID 根据用户 UID 获取角色
func (d *RoleDAO) GetByUID(ctx context.Context, uid int64) (*model.Role, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("select", true, duration)
	}()

	query, args, err := squirrel.
		Select("*").
		From("roles").
		Where(squirrel.Eq{"uid": uid}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	var role model.Role
	if err := d.db.QueryRow(ctx, query, args...).Scan(
		&role.ID,
		&role.UID,
		&role.Nickname,
		&role.Gender,
		&role.Signature,
		&role.AvatarURL,
		&role.Appearance,
		&role.Outfit,
		&role.Gold,
		&role.Diamond,
		&role.Level,
		&role.Exp,
		&role.VIPLevel,
		&role.VIPExp,
		&role.Status,
		&role.BanExpireAt,
		&role.LastLoginAt,
		&role.CreatedAt,
		&role.UpdatedAt,
	); err != nil {
		d.logger.Error("failed to get role by uid",
			"uid", uid,
			"error", err,
		)
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return &role, nil
}

// ListByUID 根据用户 UID 获取所有角色
func (d *RoleDAO) ListByUID(ctx context.Context, uid int64) ([]*model.Role, error) {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("select", true, duration)
	}()

	query, args, err := squirrel.
		Select("*").
		From("roles").
		Where(squirrel.Eq{"uid": uid}).
		OrderBy("created_at ASC").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	rows, err := d.db.Query(ctx, query, args...)
	if err != nil {
		d.logger.Error("failed to list roles by uid",
			"uid", uid,
			"error", err,
		)
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []*model.Role
	for rows.Next() {
		var role model.Role
		if err := rows.Scan(
			&role.ID,
			&role.UID,
			&role.Nickname,
			&role.Gender,
			&role.Signature,
			&role.AvatarURL,
			&role.Appearance,
			&role.Outfit,
			&role.Gold,
			&role.Diamond,
			&role.Level,
			&role.Exp,
			&role.VIPLevel,
			&role.VIPExp,
			&role.Status,
			&role.BanExpireAt,
			&role.LastLoginAt,
			&role.CreatedAt,
			&role.UpdatedAt,
		); err != nil {
			d.logger.Error("failed to scan role",
				"uid", uid,
				"error", err,
			)
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, &role)
	}

	if err := rows.Err(); err != nil {
		d.logger.Error("rows iteration error",
			"uid", uid,
			"error", err,
		)
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return roles, nil
}

// Create 创建角色
func (d *RoleDAO) Create(ctx context.Context, role *model.Role) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("insert", true, duration)
	}()

	query, args, err := squirrel.
		Insert("roles").
		Columns(
			"uid", "nickname", "gender", "signature", "avatar_url",
			"appearance", "outfit",
			"gold", "diamond",
			"level", "exp", "vip_level", "vip_exp",
			"status",
		).
		Values(
			role.UID, role.Nickname, role.Gender, role.Signature, role.AvatarURL,
			role.Appearance, role.Outfit,
			role.Gold, role.Diamond,
			role.Level, role.Exp, role.VIPLevel, role.VIPExp,
			role.Status,
		).
		Suffix("RETURNING id, created_at, updated_at").
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if err := d.db.QueryRow(ctx, query, args...).Scan(
		&role.ID,
		&role.CreatedAt,
		&role.UpdatedAt,
	); err != nil {
		d.logger.Error("failed to create role",
			"uid", role.UID,
			"nickname", role.Nickname,
			"error", err,
		)
		return fmt.Errorf("failed to create role: %w", err)
	}

	d.logger.Info("role created",
		"role_id", role.ID,
		"uid", role.UID,
		"nickname", role.Nickname,
	)

	return nil
}

// Update 更新角色
func (d *RoleDAO) Update(ctx context.Context, role *model.Role) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("update", true, duration)
	}()

	query, args, err := squirrel.
		Update("roles").
		Set("nickname", role.Nickname).
		Set("gender", role.Gender).
		Set("signature", role.Signature).
		Set("avatar_url", role.AvatarURL).
		Set("appearance", role.Appearance).
		Set("outfit", role.Outfit).
		Set("gold", role.Gold).
		Set("diamond", role.Diamond).
		Set("level", role.Level).
		Set("exp", role.Exp).
		Set("vip_level", role.VIPLevel).
		Set("vip_exp", role.VIPExp).
		Set("status", role.Status).
		Set("ban_expire_at", role.BanExpireAt).
		Set("last_login_at", role.LastLoginAt).
		Where(squirrel.Eq{"id": role.ID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to update role",
			"role_id", role.ID,
			"error", err,
		)
		return fmt.Errorf("failed to update role: %w", err)
	}

	d.logger.Debug("role updated",
		"role_id", role.ID,
	)

	return nil
}

// UpdateLastLogin 更新最后登录时间
func (d *RoleDAO) UpdateLastLogin(ctx context.Context, roleID int64) error {
	start := time.Now()
	defer func() {
		duration := time.Since(start).Seconds()
		d.metrics.RecordDBQuery("update", true, duration)
	}()

	query, args, err := squirrel.
		Update("roles").
		Set("last_login_at", time.Now()).
		Where(squirrel.Eq{"id": roleID}).
		PlaceholderFormat(squirrel.Dollar).
		ToSql()

	if err != nil {
		return fmt.Errorf("failed to build query: %w", err)
	}

	if _, err := d.db.Exec(ctx, query, args...); err != nil {
		d.logger.Error("failed to update last login",
			"role_id", roleID,
			"error", err,
		)
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}
