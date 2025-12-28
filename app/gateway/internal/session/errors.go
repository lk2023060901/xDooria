package session

import "errors"

var (
	// ErrSessionNotFound 会话不存在
	ErrSessionNotFound = errors.New("session not found")

	// ErrNotAuthenticated 未认证
	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrAlreadyAuthenticated 已经认证
	ErrAlreadyAuthenticated = errors.New("already authenticated")

	// ErrRoleNotSelected 未选择角色
	ErrRoleNotSelected = errors.New("role not selected")

	// ErrInvalidRole 无效的角色
	ErrInvalidRole = errors.New("invalid role")
)
