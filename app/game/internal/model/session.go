package model

import "time"

// Session 会话模型，表示角色与网关的连接关系
type Session struct {
	RoleID      int64     `json:"role_id"`
	SessionID   string    `json:"session_id"`
	GatewayAddr string    `json:"gateway_addr"`
	ConnectedAt time.Time `json:"connected_at"`
}

// NewSession 创建新会话
func NewSession(roleID int64, sessionID, gatewayAddr string) *Session {
	return &Session{
		RoleID:      roleID,
		SessionID:   sessionID,
		GatewayAddr: gatewayAddr,
		ConnectedAt: time.Now(),
	}
}
