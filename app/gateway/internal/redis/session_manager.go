package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/database/redis"
	"github.com/lk2023060901/xdooria/pkg/logger"
)

// RoleSession 角色会话信息
type RoleSession struct {
	ZoneID     int32  `json:"zone_id"`
	RoleID     int64  `json:"role_id"`
	UID        int64  `json:"uid"`
	GatewayID  string `json:"gateway_id"`
	SessionID  string `json:"session_id"`
	Timestamp  int64  `json:"timestamp"`
}

// SessionManager Redis 会话管理器
type SessionManager struct {
	logger logger.Logger
	client redis.Client
	zoneID int32 // 当前区服 ID
}

// NewSessionManager 创建 Redis 会话管理器
func NewSessionManager(l logger.Logger, client redis.Client, zoneID int32) *SessionManager {
	return &SessionManager{
		logger: l.Named("redis.session"),
		client: client,
		zoneID: zoneID,
	}
}

// RegisterRoleSession 注册角色会话到 Redis
// 返回值：是否存在旧会话（需要踢人）
func (m *SessionManager) RegisterRoleSession(ctx context.Context, session *RoleSession) (bool, error) {
	// 使用 Lua 脚本保证原子性
	script := `
		local zone_id = ARGV[1]
		local role_id = ARGV[2]
		local uid = ARGV[3]
		local gateway_id = ARGV[4]
		local session_id = ARGV[5]
		local timestamp = ARGV[6]

		-- 1. 检查角色是否已在线
		local role_key = "role:" .. zone_id .. ":" .. role_id .. ":session"
		local old_session = redis.call('GET', role_key)

		-- 2. 检查该账号在此区服是否有其他角色在线
		local account_key = "account:" .. zone_id .. ":" .. uid .. ":role"
		local old_role = redis.call('GET', account_key)

		if old_role and old_role ~= role_id then
			-- 该账号有其他角色在线，不允许登录
			return {-1, old_role}
		end

		-- 3. 设置新会话
		local session_data = '{"zone_id":' .. zone_id .. ',"role_id":' .. role_id .. ',"uid":' .. uid .. ',"gateway_id":"' .. gateway_id .. '","session_id":"' .. session_id .. '","timestamp":' .. timestamp .. '}'
		redis.call('SETEX', role_key, 3600, session_data)

		-- 4. 记录账号在该区服的角色
		redis.call('SET', account_key, role_id)
		redis.call('SADD', 'account:' .. uid .. ':zones', zone_id .. ':' .. role_id)

		-- 返回是否存在旧会话（0=不存在，1=存在）
		if old_session then
			return {1, old_session}
		else
			return {0, ""}
		end
	`

	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return false, fmt.Errorf("marshal session failed: %w", err)
	}

	result, err := m.client.Eval(ctx, script, nil,
		session.ZoneID,
		session.RoleID,
		session.UID,
		session.GatewayID,
		session.SessionID,
		session.Timestamp,
	)
	if err != nil {
		return false, fmt.Errorf("redis eval failed: %w", err)
	}

	resultSlice, ok := result.([]interface{})
	if !ok || len(resultSlice) < 2 {
		return false, fmt.Errorf("invalid lua script result")
	}

	code, ok := resultSlice[0].(int64)
	if !ok {
		return false, fmt.Errorf("invalid result code type")
	}

	if code == -1 {
		// 该账号有其他角色在线
		oldRole := resultSlice[1]
		return false, fmt.Errorf("account already has role online: %v", oldRole)
	}

	hasOldSession := code == 1
	if hasOldSession {
		m.logger.Info("role already online, will kick old session",
			"zone_id", session.ZoneID,
			"role_id", session.RoleID,
			"old_session", resultSlice[1],
		)
	}

	m.logger.Debug("role session registered",
		"zone_id", session.ZoneID,
		"role_id", session.RoleID,
		"uid", session.UID,
		"gateway_id", session.GatewayID,
		"session_id", session.SessionID,
		"has_old_session", hasOldSession,
		"session_json", string(sessionJSON),
	)

	return hasOldSession, nil
}

// UnregisterRoleSession 注销角色会话
func (m *SessionManager) UnregisterRoleSession(ctx context.Context, zoneID int32, roleID int64, uid int64) error {
	script := `
		local zone_id = ARGV[1]
		local role_id = ARGV[2]
		local uid = ARGV[3]

		-- 删除角色会话
		local role_key = "role:" .. zone_id .. ":" .. role_id .. ":session"
		redis.call('DEL', role_key)

		-- 删除账号区服映射
		local account_key = "account:" .. zone_id .. ":" .. uid .. ":role"
		redis.call('DEL', account_key)

		-- 从账号区服集合中移除
		redis.call('SREM', 'account:' .. uid .. ':zones', zone_id .. ':' .. role_id)

		return 1
	`

	_, err := m.client.Eval(ctx, script, nil, zoneID, roleID, uid)
	if err != nil {
		return fmt.Errorf("redis eval failed: %w", err)
	}

	m.logger.Debug("role session unregistered",
		"zone_id", zoneID,
		"role_id", roleID,
		"uid", uid,
	)

	return nil
}

// GetRoleSession 获取角色会话信息
func (m *SessionManager) GetRoleSession(ctx context.Context, zoneID int32, roleID int64) (*RoleSession, error) {
	key := fmt.Sprintf("role:%d:%d:session", zoneID, roleID)
	val, err := m.client.Get(ctx, key)
	if err != nil {
		if err == redis.ErrNil {
			return nil, nil
		}
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	var session RoleSession
	if err := json.Unmarshal([]byte(val), &session); err != nil {
		return nil, fmt.Errorf("unmarshal session failed: %w", err)
	}

	return &session, nil
}

// RefreshSession 刷新会话 TTL
func (m *SessionManager) RefreshSession(ctx context.Context, zoneID int32, roleID int64) error {
	key := fmt.Sprintf("role:%d:%d:session", zoneID, roleID)
	return m.client.Expire(ctx, key, 3600*time.Second)
}

// PublishKickCommand 发布踢人命令
func (m *SessionManager) PublishKickCommand(ctx context.Context, roleID int64, reason uint32, message string) error {
	channel := fmt.Sprintf("kick:role:%d", roleID)
	payload := map[string]interface{}{
		"role_id": roleID,
		"reason":  reason,
		"message": message,
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal kick payload failed: %w", err)
	}

	if err := m.client.Publish(ctx, channel, string(payloadJSON)); err != nil {
		return fmt.Errorf("redis publish failed: %w", err)
	}

	m.logger.Info("kick command published",
		"role_id", roleID,
		"reason", reason,
		"message", message,
		"channel", channel,
	)

	return nil
}
