package router

import "context"

// roleIDContextKey Context 中存储 RoleID 的 key
type roleIDContextKey struct{}

// RoleIDKey 用于在 Context 中存取 RoleID（跨服务通用）
var RoleIDKey = roleIDContextKey{}

// WithRoleID 将 RoleID 放入 Context
func WithRoleID(ctx context.Context, roleID int64) context.Context {
	return context.WithValue(ctx, RoleIDKey, roleID)
}

// GetRoleIDFromContext 从 Context 中提取 RoleID
func GetRoleIDFromContext(ctx context.Context) (int64, bool) {
	roleID, ok := ctx.Value(RoleIDKey).(int64)
	return roleID, ok
}
