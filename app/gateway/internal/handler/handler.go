package handler

import (
	"context"
	"strconv"

	api "github.com/lk2023060901/xdooria-proto-api"
	common "github.com/lk2023060901/xdooria-proto-common"
	gamepb "github.com/lk2023060901/xdooria-proto-internal/game"
	gwsession "github.com/lk2023060901/xdooria/app/gateway/internal/session"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/session"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
)

// RoleProvider 提供角色查询和创建接口（由 Game 服务实现）
type RoleProvider interface {
	// ListRolesByUID 根据 UID 查询所有角色
	ListRolesByUID(ctx context.Context, uid int64) ([]*api.RoleInfo, error)

	// CreateRole 创建新角色
	CreateRole(ctx context.Context, uid int64, nickname string, gender int32, appearance string) (*api.RoleInfo, error)

	// CheckNicknameExists 检查昵称是否已存在
	CheckNicknameExists(ctx context.Context, nickname string) (bool, error)
}

// GatewayHandler 处理客户端连接和消息。
type GatewayHandler struct {
	session.NopSessionHandler
	logger        logger.Logger
	jwtMgr        *security.JWTManager
	sessionRouter *SessionRouter // Gateway 专用的 Session Router
	processor     router.Processor
	sessMgr       *gwsession.Manager
	roleProvider  RoleProvider
	gameClient    gamepb.GameServiceClient
}

func NewGatewayHandler(
	l logger.Logger,
	jwtMgr *security.JWTManager,
	p router.Processor,
	sessMgr *gwsession.Manager,
	roleProvider RoleProvider,
) *GatewayHandler {
	h := &GatewayHandler{
		logger:        l.Named("gateway.handler"),
		jwtMgr:        jwtMgr,
		sessionRouter: NewSessionRouter(),
		processor:     p,
		sessMgr:       sessMgr,
		roleProvider:  roleProvider,
	}

	// 注册所有 Gateway 特定的消息处理器
	h.registerHandlers()

	return h
}

func NewGatewayHandlerWithGame(
	l logger.Logger,
	jwtMgr *security.JWTManager,
	p router.Processor,
	sessMgr *gwsession.Manager,
	roleProvider RoleProvider,
	gameClient gamepb.GameServiceClient,
) *GatewayHandler {
	h := &GatewayHandler{
		logger:        l.Named("gateway.handler"),
		jwtMgr:        jwtMgr,
		sessionRouter: NewSessionRouter(),
		processor:     p,
		sessMgr:       sessMgr,
		roleProvider:  roleProvider,
		gameClient:    gameClient,
	}

	// 注册所有 Gateway 特定的消息处理器
	h.registerHandlers()

	return h
}

// registerHandlers 注册所有消息处理器到 SessionRouter
func (h *GatewayHandler) registerHandlers() {
	// 认证相关
	RegisterHandler(h.sessionRouter, uint32(api.OpCode_OP_AUTH_REQ), uint32(api.OpCode_OP_AUTH_RES), h.handleAuth)
	RegisterHandler(h.sessionRouter, uint32(api.OpCode_OP_RECONNECT_REQ), uint32(api.OpCode_OP_RECONNECT_RES), h.handleReconnect)

	// 角色相关
	RegisterHandler(h.sessionRouter, uint32(api.OpCode_OP_GET_ROLES_REQ), uint32(api.OpCode_OP_GET_ROLES_RES), h.handleGetRoles)
	RegisterHandler(h.sessionRouter, uint32(api.OpCode_OP_CREATE_ROLE_REQ), uint32(api.OpCode_OP_CREATE_ROLE_RES), h.handleCreateRole)
	RegisterHandler(h.sessionRouter, uint32(api.OpCode_OP_SELECT_ROLE_REQ), uint32(api.OpCode_OP_SELECT_ROLE_RES), h.handleSelectRole)
}

func (h *GatewayHandler) OnOpened(s session.Session) {
	// 注册到 Session 管理器
	h.sessMgr.Register(s)
	h.logger.Info("client connected", "id", s.ID(), "addr", s.RemoteAddr())
}

func (h *GatewayHandler) OnClosed(s session.Session, err error) {
	// 从 Session 管理器注销
	h.sessMgr.Unregister(s.ID())
	h.logger.Info("client disconnected", "id", s.ID(), "error", err)
}

func (h *GatewayHandler) OnMessage(s session.Session, env *common.Envelope) {
	op := env.Header.Op
	payload := env.Payload

	h.logger.Debug("received message", "id", s.ID(), "op", op, "len", len(payload))

	// 优先使用 SessionRouter 处理 Gateway 特定消息（认证、角色相关）
	respOp, respPayload, err := h.sessionRouter.Dispatch(s.Context(), s, op, payload)
	if err == nil {
		// SessionRouter 成功处理，发送响应
		respEnv := &common.Envelope{
			Header:  &common.MessageHeader{Op: respOp},
			Payload: respPayload,
		}
		if err := s.Send(s.Context(), respEnv); err != nil {
			h.logger.Error("send response failed", "id", s.ID(), "error", err)
		}
		return
	}

	// SessionRouter 未找到 handler，转发到业务服务（Game）
	// 1. 检查是否已选择角色
	gwSess, ok := h.sessMgr.Get(s.ID())
	if !ok || !gwSess.IsRoleSelected() {
		h.logger.Warn("message before role selection", "id", s.ID(), "op", op)
		return
	}

	// 2. 启动并投递到串行处理器
	// 确保处理器已启动（仅首次进入时启动）
	gwSess.StartProcessor(s.Context(), func(e *common.Envelope) {
		h.forwardToGame(gwSess, e)
	})

	// 3. 将任务推入队列（非阻塞，瞬间完成）
	if !gwSess.PushTask(env) {
		h.logger.Error("user task queue full, dropping message", "id", s.ID(), "op", op)
	}
}

// forwardToGame 真正的阻塞转发逻辑，运行在玩家私有的协程中
func (h *GatewayHandler) forwardToGame(s *gwsession.GatewaySession, env *common.Envelope) {
	op := env.Header.Op
	roleID := s.GetRoleID()

	if h.gameClient != nil {
		resp, err := h.gameClient.ForwardMessage(s.Context(), &gamepb.ForwardMessageRequest{
			RoleId:  roleID,
			OpCode:  op,
			Payload: env.Payload,
		})
		if err != nil {
			h.logger.Error("forward to game failed", "id", s.ID(), "op", op, "error", err)
			return
		}
		if !resp.Success {
			h.logger.Error("game returned error", "id", s.ID(), "op", op, "error", resp.Error)
			return
		}

		// 发送响应
		respEnv := &common.Envelope{
			Header:  &common.MessageHeader{Op: op + 1}, // 响应 op = 请求 op + 1
			Payload: resp.Payload,
		}
		if err := s.Send(s.Context(), respEnv); err != nil {
			h.logger.Error("send response failed", "id", s.ID(), "error", err)
		}
		return
	}

	// 没有 gameClient，使用 Processor（兼容旧代码）
	ctx := router.WithRoleID(s.Context(), roleID)
	respOp, respPayload, err := h.processor.Process(ctx, op, env.Payload)
	if err != nil {
		h.logger.Error("process message failed", "id", s.ID(), "op", op, "error", err)
		return
	}

	// 发送响应
	respEnv := &common.Envelope{
		Header:  &common.MessageHeader{Op: respOp},
		Payload: respPayload,
	}
	if err := s.Send(s.Context(), respEnv); err != nil {
		h.logger.Error("send response failed", "id", s.ID(), "error", err)
	}
}

// handleAuth 处理首次认证请求
func (h *GatewayHandler) handleAuth(ctx context.Context, s session.Session, req *api.AuthRequest) (*api.AuthResponse, error) {
	// 验证 Login 签发的 token
	claims, err := h.jwtMgr.ValidateToken(req.LoginToken)
	if err != nil {
		h.logger.Warn("token validation failed", "id", s.ID(), "error", err)
		code := uint32(api.ErrorCode_ERR_TOKEN_INVALID)
		if err == security.ErrTokenExpired {
			code = uint32(api.ErrorCode_ERR_TOKEN_EXPIRED)
		}
		return &api.AuthResponse{Code: code}, nil
	}

	// 从 JWT 中提取 uid
	uidStr, ok := claims.Get("uid").(string)
	if !ok {
		h.logger.Error("uid not found in token", "id", s.ID())
		return &api.AuthResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	uid, err := strconv.ParseInt(uidStr, 10, 64)
	if err != nil {
		h.logger.Error("failed to parse uid", "id", s.ID(), "uid_str", uidStr, "error", err)
		return &api.AuthResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	// 查询该用户的所有角色
	var roles []*api.RoleInfo
	if h.roleProvider != nil {
		roles, err = h.roleProvider.ListRolesByUID(ctx, uid)
		if err != nil {
			h.logger.Error("failed to list roles", "id", s.ID(), "uid", uid, "error", err)
			return &api.AuthResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
		}
	}

	// 生成 Gateway 的 SessionToken（使用 Gateway 配置的过期时间）
	sessionToken, err := h.jwtMgr.GenerateToken(&security.Claims{
		Payload: claims.Payload, // 继承 uid 等信息
	})
	if err != nil {
		h.logger.Error("failed to generate session token", "id", s.ID(), "error", err)
		return &api.AuthResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	// 更新 Session 认证状态
	h.sessMgr.UpdateAuthState(s.ID(), uid)

	h.logger.Info("client authenticated",
		"id", s.ID(),
		"uid", uid,
		"role_count", len(roles),
	)

	return &api.AuthResponse{
		Code:  uint32(api.ErrorCode_ERR_SUCCESS),
		Token: sessionToken,
		Uid:   uint64(uid),
	}, nil
}

// handleReconnect 处理重连请求
func (h *GatewayHandler) handleReconnect(ctx context.Context, s session.Session, req *api.ReconnectRequest) (*api.ReconnectResponse, error) {
	// 验证 SessionToken
	claims, err := h.jwtMgr.ValidateToken(req.Token)
	if err != nil {
		h.logger.Warn("reconnect token validation failed", "id", s.ID(), "error", err)
		code := uint32(api.ErrorCode_ERR_TOKEN_INVALID)
		if err == security.ErrTokenExpired {
			code = uint32(api.ErrorCode_ERR_TOKEN_EXPIRED)
		}
		return &api.ReconnectResponse{Code: code}, nil
	}

	// 续期：生成新的 SessionToken
	newToken, err := h.jwtMgr.GenerateToken(&security.Claims{
		Payload: claims.Payload,
	})
	if err != nil {
		h.logger.Error("failed to generate new session token", "id", s.ID(), "error", err)
		return &api.ReconnectResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	h.logger.Info("client reconnected", "id", s.ID(), "uid", claims.Get("uid"))
	return &api.ReconnectResponse{
		Code:  uint32(api.ErrorCode_ERR_SUCCESS),
		Token: newToken,
	}, nil
}

// handleGetRoles 处理获取角色列表请求
func (h *GatewayHandler) handleGetRoles(ctx context.Context, s session.Session, req *api.GetRolesRequest) (*api.GetRolesResponse, error) {
	// 获取 Gateway Session
	gwSess, ok := h.sessMgr.Get(s.ID())
	if !ok {
		h.logger.Error("session not found in manager", "id", s.ID())
		return &api.GetRolesResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	// 检查是否已认证
	if !gwSess.IsAuthenticated() {
		h.logger.Warn("get roles before authentication", "id", s.ID())
		return &api.GetRolesResponse{Code: uint32(api.ErrorCode_ERR_NOT_AUTHENTICATED)}, nil
	}

	uid := gwSess.GetUID()

	// 查询角色列表
	if h.roleProvider == nil {
		h.logger.Error("role provider not available", "id", s.ID())
		return &api.GetRolesResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	roles, err := h.roleProvider.ListRolesByUID(ctx, uid)
	if err != nil {
		h.logger.Error("failed to list roles", "id", s.ID(), "uid", uid, "error", err)
		return &api.GetRolesResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	h.logger.Info("roles listed",
		"id", s.ID(),
		"uid", uid,
		"count", len(roles),
	)

	return &api.GetRolesResponse{
		Code:  uint32(api.ErrorCode_ERR_SUCCESS),
		Roles: roles,
	}, nil
}

// handleCreateRole 处理创建角色请求
func (h *GatewayHandler) handleCreateRole(ctx context.Context, s session.Session, req *api.CreateRoleRequest) (*api.CreateRoleResponse, error) {
	// 获取 Gateway Session
	gwSess, ok := h.sessMgr.Get(s.ID())
	if !ok {
		h.logger.Error("session not found in manager", "id", s.ID())
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	// 检查是否已认证
	if !gwSess.IsAuthenticated() {
		h.logger.Warn("create role before authentication", "id", s.ID())
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_NOT_AUTHENTICATED)}, nil
	}

	uid := gwSess.GetUID()

	// 验证昵称
	if len(req.Nickname) == 0 || len(req.Nickname) > 32 {
		h.logger.Warn("invalid nickname length", "id", s.ID(), "nickname", req.Nickname)
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_NICKNAME_INVALID)}, nil
	}

	// 检查昵称是否已存在
	if h.roleProvider == nil {
		h.logger.Error("role provider not available", "id", s.ID())
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	exists, err := h.roleProvider.CheckNicknameExists(ctx, req.Nickname)
	if err != nil {
		h.logger.Error("failed to check nickname", "id", s.ID(), "nickname", req.Nickname, "error", err)
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}
	if exists {
		h.logger.Warn("nickname already exists", "id", s.ID(), "nickname", req.Nickname)
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_NICKNAME_EXISTS)}, nil
	}

	// 检查该用户的角色数量是否达到上限（例如最多3个角色）
	roles, err := h.roleProvider.ListRolesByUID(ctx, uid)
	if err != nil {
		h.logger.Error("failed to list roles", "id", s.ID(), "uid", uid, "error", err)
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}
	if len(roles) >= 3 { // 最多3个角色
		h.logger.Warn("role limit exceeded", "id", s.ID(), "uid", uid, "count", len(roles))
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_ROLE_LIMIT_EXCEEDED)}, nil
	}

	// 创建角色
	roleInfo, err := h.roleProvider.CreateRole(ctx, uid, req.Nickname, req.Gender, req.Appearance)
	if err != nil {
		h.logger.Error("failed to create role", "id", s.ID(), "uid", uid, "nickname", req.Nickname, "error", err)
		return &api.CreateRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	h.logger.Info("role created",
		"id", s.ID(),
		"uid", uid,
		"role_id", roleInfo.RoleId,
		"nickname", roleInfo.Nickname,
	)

	return &api.CreateRoleResponse{
		Code: uint32(api.ErrorCode_ERR_SUCCESS),
		Role: roleInfo,
	}, nil
}

// handleSelectRole 处理选择角色请求
func (h *GatewayHandler) handleSelectRole(ctx context.Context, s session.Session, req *api.SelectRoleRequest) (*api.SelectRoleResponse, error) {
	// 获取 Gateway Session
	gwSess, ok := h.sessMgr.Get(s.ID())
	if !ok {
		h.logger.Error("session not found in manager", "id", s.ID())
		return &api.SelectRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	// 检查是否已认证
	if !gwSess.IsAuthenticated() {
		h.logger.Warn("select role before authentication", "id", s.ID())
		return &api.SelectRoleResponse{Code: uint32(api.ErrorCode_ERR_NOT_AUTHENTICATED)}, nil
	}

	uid := gwSess.GetUID()

	// 验证角色是否属于该用户
	// 这里可以选择查询数据库验证，或者相信客户端在认证时收到的角色列表
	// 为了安全起见，我们应该验证角色归属
	if h.roleProvider != nil {
		roles, err := h.roleProvider.ListRolesByUID(ctx, uid)
		if err != nil {
			h.logger.Error("failed to verify role ownership", "id", s.ID(), "uid", uid, "role_id", req.RoleId, "error", err)
			return &api.SelectRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
		}

		// 检查角色是否在列表中
		roleFound := false
		for _, role := range roles {
			if role.RoleId == req.RoleId {
				roleFound = true
				break
			}
		}

		if !roleFound {
			h.logger.Warn("role not owned by user", "id", s.ID(), "uid", uid, "role_id", req.RoleId)
			return &api.SelectRoleResponse{Code: uint32(api.ErrorCode_ERR_INVALID_ROLE)}, nil
		}
	}

	// 更新 Session 中的角色状态
	if err := h.sessMgr.UpdateRoleState(s.ID(), req.RoleId); err != nil {
		h.logger.Error("failed to update role state", "id", s.ID(), "role_id", req.RoleId, "error", err)
		return &api.SelectRoleResponse{Code: uint32(api.ErrorCode_ERR_INTERNAL)}, nil
	}

	h.logger.Info("role selected",
		"id", s.ID(),
		"uid", uid,
		"role_id", req.RoleId,
	)

	return &api.SelectRoleResponse{
		Code:   uint32(api.ErrorCode_ERR_SUCCESS),
		RoleId: req.RoleId,
	}, nil
}

func (h *GatewayHandler) OnError(s session.Session, err error) {
	h.logger.Warn("session error", "id", s.ID(), "error", err)
}
