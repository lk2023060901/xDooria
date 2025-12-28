package role

import (
	"context"
	"fmt"

	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria/pkg/database/postgres"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// Provider 角色数据提供者
type Provider struct {
	logger logger.Logger
	pg     *postgres.Client
}

// NewProvider 创建角色提供者
func NewProvider(l logger.Logger, pg *postgres.Client) *Provider {
	return &Provider{
		logger: l.Named("role.provider"),
		pg:     pg,
	}
}

// ListRolesByUID 根据 UID 查询所有角色
func (p *Provider) ListRolesByUID(ctx context.Context, uid int64) ([]*api.RoleInfo, error) {
	query := `
		SELECT id, nickname, level, status
		FROM roles
		WHERE uid = $1
		ORDER BY created_at DESC
	`

	rows, err := p.pg.Query(ctx, query, uid)
	if err != nil {
		p.logger.Error("failed to query roles", "uid", uid, "error", err)
		return nil, fmt.Errorf("query roles failed: %w", err)
	}
	defer rows.Close()

	var roles []*api.RoleInfo
	for rows.Next() {
		var role api.RoleInfo
		if err := rows.Scan(&role.RoleId, &role.Nickname, &role.Level, &role.Status); err != nil {
			p.logger.Error("failed to scan role", "error", err)
			continue
		}
		roles = append(roles, &role)
	}

	p.logger.Debug("listed roles", "uid", uid, "count", len(roles))
	return roles, nil
}

// CreateRole 创建新角色
func (p *Provider) CreateRole(ctx context.Context, uid int64, nickname string, gender int32, appearance string) (*api.RoleInfo, error) {
	// 检查昵称是否已存在
	exists, err := p.CheckNicknameExists(ctx, nickname)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, fmt.Errorf("nickname already exists")
	}

	// 插入新角色
	query := `
		INSERT INTO roles (uid, nickname, level, gender, appearance, status, created_at, updated_at)
		VALUES ($1, $2, 1, $3, $4, 0, NOW(), NOW())
		RETURNING id, nickname, level, status
	`

	var role api.RoleInfo
	err = p.pg.QueryRow(ctx, query, uid, nickname, gender, appearance).Scan(
		&role.RoleId, &role.Nickname, &role.Level, &role.Status,
	)
	if err != nil {
		p.logger.Error("failed to create role", "uid", uid, "nickname", nickname, "error", err)
		return nil, fmt.Errorf("create role failed: %w", err)
	}

	p.logger.Info("created role", "uid", uid, "role_id", role.RoleId, "nickname", nickname)
	return &role, nil
}

// CheckNicknameExists 检查昵称是否已存在
func (p *Provider) CheckNicknameExists(ctx context.Context, nickname string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM roles WHERE nickname = $1)`

	var exists bool
	err := p.pg.QueryRow(ctx, query, nickname).Scan(&exists)
	if err != nil {
		p.logger.Error("failed to check nickname exists", "nickname", nickname, "error", err)
		return false, fmt.Errorf("check nickname failed: %w", err)
	}

	return exists, nil
}
