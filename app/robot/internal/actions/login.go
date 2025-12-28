package actions

import (
	"context"
	"fmt"

	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria/app/robot/internal/bt"
	"github.com/lk2023060901/xdooria/app/robot/internal/client"
	"google.golang.org/protobuf/proto"
)

// ConnectToGateway 连接到 Gateway
type ConnectToGateway struct {
	bt.BaseNode
	robot *client.Robot
}

// NewConnectToGateway 创建连接到 Gateway 节点
func NewConnectToGateway(robot *client.Robot) *ConnectToGateway {
	return &ConnectToGateway{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (a *ConnectToGateway) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 检查是否已连接
	if a.robot.IsConnected() {
		return bt.StatusSuccess
	}

	// 连接到 Gateway
	if err := a.robot.Connect(); err != nil {
		bb.Set("error", err.Error())
		return bt.StatusFailure
	}

	return bt.StatusSuccess
}

func (a *ConnectToGateway) Name() string {
	return "ConnectToGateway"
}

// AuthWithToken 使用 Token 认证
type AuthWithToken struct {
	bt.BaseNode
	robot *client.Robot
}

// NewAuthWithToken 创建认证节点
func NewAuthWithToken(robot *client.Robot) *AuthWithToken {
	return &AuthWithToken{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (a *AuthWithToken) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 从黑板获取 login_token
	loginToken, ok := bb.GetString("login_token")
	if !ok {
		bb.Set("error", "login_token not found")
		return bt.StatusFailure
	}

	// 构造认证请求
	req := &api.AuthRequest{
		LoginToken: loginToken,
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		bb.Set("error", fmt.Sprintf("marshal auth request failed: %v", err))
		return bt.StatusFailure
	}

	// 发送认证请求
	if err := a.robot.Send(uint32(api.OpCode_OP_AUTH_REQ), payload); err != nil {
		bb.Set("error", fmt.Sprintf("send auth request failed: %v", err))
		return bt.StatusFailure
	}

	// 等待响应 (简化处理，实际应该异步等待)
	// TODO: 实现异步响应处理
	bb.Set("waiting_for", "auth_response")

	return bt.StatusRunning
}

func (a *AuthWithToken) Name() string {
	return "AuthWithToken"
}

// SelectRole 选择角色
type SelectRole struct {
	bt.BaseNode
	robot *client.Robot
}

// NewSelectRole 创建选择角色节点
func NewSelectRole(robot *client.Robot) *SelectRole {
	return &SelectRole{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (s *SelectRole) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 从黑板获取角色ID
	roleID, ok := bb.GetInt64("role_id")
	if !ok {
		// 如果没有角色ID，使用第一个角色
		roles, hasRoles := bb.Get("roles")
		if !hasRoles {
			bb.Set("error", "no roles available")
			return bt.StatusFailure
		}

		roleList := roles.([]*api.RoleInfo)
		if len(roleList) == 0 {
			bb.Set("error", "role list is empty")
			return bt.StatusFailure
		}

		roleID = roleList[0].RoleId
		bb.Set("role_id", roleID)
	}

	// 构造选择角色请求
	req := &api.SelectRoleRequest{
		RoleId: roleID,
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		bb.Set("error", fmt.Sprintf("marshal select role request failed: %v", err))
		return bt.StatusFailure
	}

	// 发送请求
	if err := s.robot.Send(uint32(api.OpCode_OP_SELECT_ROLE_REQ), payload); err != nil {
		bb.Set("error", fmt.Sprintf("send select role request failed: %v", err))
		return bt.StatusFailure
	}

	bb.Set("waiting_for", "select_role_response")
	return bt.StatusRunning
}

func (s *SelectRole) Name() string {
	return "SelectRole"
}

// CreateRole 创建角色
type CreateRole struct {
	bt.BaseNode
	robot    *client.Robot
	nickname string
	gender   int32
}

// NewCreateRole 创建角色节点
func NewCreateRole(robot *client.Robot, nickname string, gender int32) *CreateRole {
	return &CreateRole{
		BaseNode: bt.BaseNode{},
		robot:    robot,
		nickname: nickname,
		gender:   gender,
	}
}

func (c *CreateRole) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 构造创建角色请求
	req := &api.CreateRoleRequest{
		Nickname:   c.nickname,
		Gender:     c.gender,
		Appearance: "default", // 默认外观
	}

	payload, err := proto.Marshal(req)
	if err != nil {
		bb.Set("error", fmt.Sprintf("marshal create role request failed: %v", err))
		return bt.StatusFailure
	}

	// 发送请求
	if err := c.robot.Send(uint32(api.OpCode_OP_CREATE_ROLE_REQ), payload); err != nil {
		bb.Set("error", fmt.Sprintf("send create role request failed: %v", err))
		return bt.StatusFailure
	}

	bb.Set("waiting_for", "create_role_response")
	return bt.StatusRunning
}

func (c *CreateRole) Name() string {
	return "CreateRole"
}

// GetRoles 获取角色列表
type GetRoles struct {
	bt.BaseNode
	robot *client.Robot
}

// NewGetRoles 创建获取角色列表节点
func NewGetRoles(robot *client.Robot) *GetRoles {
	return &GetRoles{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (g *GetRoles) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 构造获取角色列表请求
	req := &api.GetRolesRequest{}

	payload, err := proto.Marshal(req)
	if err != nil {
		bb.Set("error", fmt.Sprintf("marshal get roles request failed: %v", err))
		return bt.StatusFailure
	}

	// 发送请求
	if err := g.robot.Send(uint32(api.OpCode_OP_GET_ROLES_REQ), payload); err != nil {
		bb.Set("error", fmt.Sprintf("send get roles request failed: %v", err))
		return bt.StatusFailure
	}

	bb.Set("waiting_for", "get_roles_response")
	return bt.StatusRunning
}

func (g *GetRoles) Name() string {
	return "GetRoles"
}

// EnterScene 进入场景
type EnterScene struct {
	bt.BaseNode
	robot *client.Robot
}

// NewEnterScene 创建进入场景节点
func NewEnterScene(robot *client.Robot) *EnterScene {
	return &EnterScene{
		BaseNode: bt.BaseNode{},
		robot:    robot,
	}
}

func (e *EnterScene) Tick(ctx context.Context, bb *bt.Blackboard) bt.Status {
	// 构造进入场景请求
	req := &api.EnterSceneRequest{}

	payload, err := proto.Marshal(req)
	if err != nil {
		bb.Set("error", fmt.Sprintf("marshal enter scene request failed: %v", err))
		return bt.StatusFailure
	}

	// 发送请求
	if err := e.robot.Send(uint32(api.OpCode_OP_ENTER_SCENE_REQ), payload); err != nil {
		bb.Set("error", fmt.Sprintf("send enter scene request failed: %v", err))
		return bt.StatusFailure
	}

	bb.Set("waiting_for", "enter_scene_response")
	return bt.StatusRunning
}

func (e *EnterScene) Name() string {
	return "EnterScene"
}
