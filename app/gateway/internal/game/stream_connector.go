package game

import (
	"context"
	"fmt"
	"sync"
	"time"

	common "github.com/lk2023060901/xdooria-proto-common"
	internal "github.com/lk2023060901/xdooria-proto-internal"
	gwsession "github.com/lk2023060901/xdooria/app/gateway/internal/session"
	grpcpkg "github.com/lk2023060901/xdooria/pkg/network/grpc"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"google.golang.org/protobuf/proto"
)

// StreamConnector Gateway 到 Game 的流连接器
type StreamConnector struct {
	logger     logger.Logger
	connector  *grpcpkg.Connector
	sessMgr    *gwsession.Manager
	gatewayID  string
	zoneID     int32

	mu          sync.RWMutex
	gameSession session.Session
	connected   bool
	stopCh      chan struct{}
}

// NewStreamConnector 创建流连接器
func NewStreamConnector(
	l logger.Logger,
	grpcClient common.CommonServiceClient,
	sessionConfig *session.Config,
	sessMgr *gwsession.Manager,
	gatewayID string,
	zoneID int32,
) *StreamConnector {
	sc := &StreamConnector{
		logger:    l.Named("game.stream_connector"),
		sessMgr:   sessMgr,
		gatewayID: gatewayID,
		zoneID:    zoneID,
		stopCh:    make(chan struct{}),
	}

	// 创建 Connector，使用自定义 handler
	sc.connector = grpcpkg.NewConnector(grpcClient, sessionConfig, sc)

	return sc
}

// Connect 连接到 Game 服务
func (sc *StreamConnector) Connect(ctx context.Context, gameAddr string) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.connected {
		return fmt.Errorf("already connected")
	}

	// 建立流连接
	sess, err := sc.connector.Connect(ctx, gameAddr)
	if err != nil {
		return fmt.Errorf("connect to game failed: %w", err)
	}

	sc.gameSession = sess
	sc.connected = true

	sc.logger.Info("connected to game service", "addr", gameAddr)

	// 启动心跳
	conc.Go(func() (struct{}, error) {
		return struct{}{}, sc.heartbeatLoop()
	})

	return nil
}

// Close 关闭连接
func (sc *StreamConnector) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.connected {
		return nil
	}

	close(sc.stopCh)

	if sc.gameSession != nil {
		if err := sc.gameSession.Close(); err != nil {
			sc.logger.Warn("close game session failed", "error", err)
		}
	}

	sc.connected = false
	sc.logger.Info("stream connector closed")

	return nil
}

// ForwardMessage 转发客户端消息到 Game
func (sc *StreamConnector) ForwardMessage(ctx context.Context, roleID int64, sessionID string, clientOp uint32, clientPayload []byte) error {
	sc.mu.RLock()
	sess := sc.gameSession
	connected := sc.connected
	sc.mu.RUnlock()

	if !connected || sess == nil {
		return fmt.Errorf("not connected to game")
	}

	// 构造转发消息
	req := &internal.ForwardMessageRequest{
		RoleId:        roleID,
		SessionId:     sessionID,
		ClientOp:      clientOp,
		ClientPayload: clientPayload,
		GatewayId:     sc.gatewayID,
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal forward message request failed: %w", err)
	}

	env := &common.Envelope{
		Header: &common.MessageHeader{
			Op: uint32(internal.OpCode_OP_GATEWAY_FORWARD_MESSAGE),
		},
		Payload: payload,
	}

	if err := sess.Send(ctx, env); err != nil {
		return fmt.Errorf("send forward message failed: %w", err)
	}

	sc.logger.Debug("forwarded message to game",
		"role_id", roleID,
		"client_op", clientOp,
		"payload_len", len(clientPayload),
	)

	return nil
}

// NotifyPlayerOnline 通知 Game 玩家上线
func (sc *StreamConnector) NotifyPlayerOnline(ctx context.Context, roleID, uid int64, sessionID string) error {
	sc.mu.RLock()
	sess := sc.gameSession
	connected := sc.connected
	sc.mu.RUnlock()

	if !connected || sess == nil {
		return fmt.Errorf("not connected to game")
	}

	// 构造上线通知
	notify := &internal.PlayerOnlineNotify{
		RoleId:      roleID,
		Uid:         uid,
		SessionId:   sessionID,
		GatewayId:   sc.gatewayID,
		GatewayAddr: sc.gatewayID, // TODO: 使用实际的 Gateway 地址
		ZoneId:      sc.zoneID,
	}

	payload, err := proto.Marshal(notify)
	if err != nil {
		return fmt.Errorf("marshal player online notify failed: %w", err)
	}

	env := &common.Envelope{
		Header: &common.MessageHeader{
			Op: uint32(internal.OpCode_OP_GATEWAY_PLAYER_ONLINE),
		},
		Payload: payload,
	}

	if err := sess.Send(ctx, env); err != nil {
		return fmt.Errorf("send player online notify failed: %w", err)
	}

	sc.logger.Info("notified player online",
		"role_id", roleID,
		"uid", uid,
		"session_id", sessionID,
	)

	return nil
}

// NotifyPlayerOffline 通知 Game 玩家下线
func (sc *StreamConnector) NotifyPlayerOffline(ctx context.Context, roleID int64, sessionID string, reason int32) error {
	sc.mu.RLock()
	sess := sc.gameSession
	connected := sc.connected
	sc.mu.RUnlock()

	if !connected || sess == nil {
		return fmt.Errorf("not connected to game")
	}

	// 构造下线通知
	notify := &internal.PlayerOfflineNotify{
		RoleId:    roleID,
		SessionId: sessionID,
		GatewayId: sc.gatewayID,
		Reason:    reason,
	}

	payload, err := proto.Marshal(notify)
	if err != nil {
		return fmt.Errorf("marshal player offline notify failed: %w", err)
	}

	env := &common.Envelope{
		Header: &common.MessageHeader{
			Op: uint32(internal.OpCode_OP_GATEWAY_PLAYER_OFFLINE),
		},
		Payload: payload,
	}

	if err := sess.Send(ctx, env); err != nil {
		return fmt.Errorf("send player offline notify failed: %w", err)
	}

	sc.logger.Info("notified player offline",
		"role_id", roleID,
		"session_id", sessionID,
		"reason", reason,
	)

	return nil
}

// heartbeatLoop 心跳循环
func (sc *StreamConnector) heartbeatLoop() error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sc.stopCh:
			return nil
		case <-ticker.C:
			sc.sendHeartbeat()
		}
	}
}

// sendHeartbeat 发送心跳
func (sc *StreamConnector) sendHeartbeat() {
	sc.mu.RLock()
	sess := sc.gameSession
	connected := sc.connected
	sc.mu.RUnlock()

	if !connected || sess == nil {
		return
	}

	hb := &internal.GatewayHeartbeat{
		GatewayId:   sc.gatewayID,
		Timestamp:   time.Now().Unix(),
		OnlineCount: int32(sc.sessMgr.OnlineRoleCount()),
	}

	payload, err := proto.Marshal(hb)
	if err != nil {
		sc.logger.Warn("marshal heartbeat failed", "error", err)
		return
	}

	env := &common.Envelope{
		Header: &common.MessageHeader{
			Op: uint32(internal.OpCode_OP_GATEWAY_HEARTBEAT),
		},
		Payload: payload,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sess.Send(ctx, env); err != nil {
		sc.logger.Warn("send heartbeat failed", "error", err)
	}
}

// ============================================================================
// session.SessionHandler 接口实现（处理来自 Game 的消息）
// ============================================================================

func (sc *StreamConnector) OnOpened(s session.Session) {
	sc.logger.Info("game session opened", "session_id", s.ID())
}

func (sc *StreamConnector) OnClosed(s session.Session, err error) {
	sc.logger.Warn("game session closed", "session_id", s.ID(), "error", err)

	sc.mu.Lock()
	sc.connected = false
	sc.gameSession = nil
	sc.mu.Unlock()

	// TODO: 实现重连逻辑
}

func (sc *StreamConnector) OnMessage(s session.Session, env *common.Envelope) {
	op := env.Header.Op
	payload := env.Payload

	sc.logger.Debug("received message from game", "op", op, "payload_len", len(payload))

	// 处理来自 Game 的消息
	switch internal.OpCode(op) {
	case internal.OpCode_OP_GAME_SEND_TO_CLIENT:
		sc.handleSendToClient(payload)
	case internal.OpCode_OP_GAME_BROADCAST:
		sc.handleBroadcast(payload)
	case internal.OpCode_OP_GAME_KICK_CLIENT:
		sc.handleKickClient(payload)
	case internal.OpCode_OP_GAME_HEARTBEAT_ACK:
		// 心跳响应，无需处理
	default:
		sc.logger.Warn("unknown opcode from game", "op", op)
	}
}

func (sc *StreamConnector) OnError(s session.Session, err error) {
	sc.logger.Error("game session error", "session_id", s.ID(), "error", err)
}

// handleSendToClient 处理推送消息给客户端
func (sc *StreamConnector) handleSendToClient(payload []byte) {
	var req internal.SendToClientRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		sc.logger.Error("unmarshal send to client request failed", "error", err)
		return
	}

	// 查找角色对应的会话
	gwSess, ok := sc.sessMgr.GetByRoleID(req.RoleId)
	if !ok {
		sc.logger.Debug("role session not found", "role_id", req.RoleId)
		return
	}

	// 发送给客户端
	env := &common.Envelope{
		Header: &common.MessageHeader{
			Op: req.Op,
		},
		Payload: req.Payload,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := gwSess.Send(ctx, env); err != nil {
		sc.logger.Warn("send to client failed",
			"role_id", req.RoleId,
			"session_id", gwSess.ID(),
			"op", req.Op,
			"error", err,
		)
	}
}

// handleBroadcast 处理广播消息
func (sc *StreamConnector) handleBroadcast(payload []byte) {
	var req internal.BroadcastRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		sc.logger.Error("unmarshal broadcast request failed", "error", err)
		return
	}

	env := &common.Envelope{
		Header: &common.MessageHeader{
			Op: req.Op,
		},
		Payload: req.Payload,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 如果 role_ids 为空，则广播给所有在线角色
	if len(req.RoleIds) == 0 {
		// TODO: 实现全服广播
		sc.logger.Warn("broadcast to all not implemented yet")
		return
	}

	// 广播给指定角色
	for _, roleID := range req.RoleIds {
		gwSess, ok := sc.sessMgr.GetByRoleID(roleID)
		if !ok {
			continue
		}

		if err := gwSess.Send(ctx, env); err != nil {
			sc.logger.Warn("broadcast to client failed",
				"role_id", roleID,
				"session_id", gwSess.ID(),
				"op", req.Op,
				"error", err,
			)
		}
	}

	sc.logger.Debug("broadcast sent", "role_count", len(req.RoleIds), "op", req.Op)
}

// handleKickClient 处理踢人命令
func (sc *StreamConnector) handleKickClient(payload []byte) {
	var req internal.KickClientRequest
	if err := proto.Unmarshal(payload, &req); err != nil {
		sc.logger.Error("unmarshal kick client request failed", "error", err)
		return
	}

	sc.logger.Info("received kick client command",
		"role_id", req.RoleId,
		"reason", req.Reason,
		"message", req.Message,
	)

	// 查找角色对应的会话
	gwSess, ok := sc.sessMgr.GetByRoleID(req.RoleId)
	if !ok {
		sc.logger.Debug("role session not found for kick", "role_id", req.RoleId)
		return
	}

	// 发送踢人通知
	kickEnv := &common.Envelope{
		Header: &common.MessageHeader{
			Op: uint32(common.OpCode_OP_KICK_NOTICE),
		},
		Payload: []byte(req.Message),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := gwSess.Send(ctx, kickEnv); err != nil {
		sc.logger.Warn("send kick notice failed",
			"role_id", req.RoleId,
			"session_id", gwSess.ID(),
			"error", err,
		)
	}

	// 关闭会话
	if err := gwSess.Close(); err != nil {
		sc.logger.Warn("close session failed",
			"role_id", req.RoleId,
			"session_id", gwSess.ID(),
			"error", err,
		)
	}

	sc.logger.Info("kicked client",
		"role_id", req.RoleId,
		"session_id", gwSess.ID(),
		"reason", req.Reason,
	)
}
