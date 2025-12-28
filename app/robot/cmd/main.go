package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	api "github.com/lk2023060901/xdooria-proto-api"
	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/app/robot/internal/client"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"github.com/lk2023060901/xdooria/pkg/security"
	"google.golang.org/protobuf/proto"
)

var (
	gatewayAddr = flag.String("gateway", "localhost:9000", "Gateway 地址")
	uid         = flag.Uint64("uid", 10001, "用户 UID（用于生成 JWT Token）")
	createRole  = flag.Bool("create", false, "是否创建新角色")
	nickname    = flag.String("nickname", "TestRobot", "角色昵称")
	jwtSecret   = flag.String("jwt-secret", "xdooria-secret-key-123456", "JWT 密钥（需与 Gateway 配置一致）")
)

func main() {
	flag.Parse()

	// 初始化日志
	l, err := logger.New(&logger.Config{
		Level:         "debug",
		Format:        "console",
		EnableConsole: true,
	})
	if err != nil {
		panic(err)
	}

	// 创建 Robot 客户端
	framerCfg := &framer.Config{
		EnableEncrypt:    false,
		EnableCompress:   false,
		CompressMinBytes: 1024,
	}

	robot, err := client.NewRobot(*gatewayAddr, framerCfg)
	if err != nil {
		l.Error("创建 Robot 失败", "error", err)
		os.Exit(1)
	}

	// 处理中断信号
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		l.Info("收到中断信号，正在退出...")
		cancel()
		robot.Close()
		os.Exit(0)
	}()

	// 生成测试用 JWT Token
	token, err := generateTestToken(*uid, *jwtSecret)
	if err != nil {
		l.Error("生成 JWT Token 失败", "error", err)
		os.Exit(1)
	}
	l.Info("生成测试 Token", "uid", *uid, "token_length", len(token))

	// 执行完整流程
	if err := runFullWorkflow(ctx, robot, l, token, *createRole, *nickname); err != nil {
		l.Error("工作流执行失败", "error", err)
		robot.Close()
		os.Exit(1)
	}

	l.Info("所有测试步骤完成，机器人将保持连接...")
	l.Info("按 Ctrl+C 退出")

	// 保持连接
	<-sigCh
}

// generateTestToken 生成测试用的 JWT Token
func generateTestToken(uid uint64, secretKey string) (string, error) {
	// 创建 JWT 管理器
	jwtMgr, err := security.NewJWTManager(&security.JWTConfig{
		SecretKey: secretKey,
		Algorithm: "HS256",
		ExpiresIn: 24 * time.Hour,
	})
	if err != nil {
		return "", fmt.Errorf("创建 JWT 管理器失败: %w", err)
	}

	// 生成 Token（与 Login 服务的格式一致）
	claims := &security.Claims{
		Payload: map[string]any{
			"uid": fmt.Sprintf("%d", uid),
		},
	}

	token, err := jwtMgr.GenerateToken(claims)
	if err != nil {
		return "", fmt.Errorf("生成 Token 失败: %w", err)
	}

	return token, nil
}

func runFullWorkflow(ctx context.Context, robot *client.Robot, l logger.Logger, loginToken string, shouldCreateRole bool, nickname string) error {
	// 1. 连接到 Gateway
	l.Info("正在连接到 Gateway", "addr", *gatewayAddr)
	if err := robot.Connect(); err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	l.Info("已成功连接到 Gateway")

	// 2. 发送认证请求
	l.Info("发送认证请求", "token", loginToken)
	authReq := &api.AuthRequest{LoginToken: loginToken}
	payload, _ := proto.Marshal(authReq)
	if err := robot.Send(uint32(api.OpCode_OP_AUTH_REQ), payload); err != nil {
		return fmt.Errorf("发送认证请求失败: %w", err)
	}

	// 3. 等待认证响应
	env, err := robot.Recv(10 * time.Second)
	if err != nil {
		return fmt.Errorf("接收认证响应失败: %w", err)
	}
	if env.Header.Op != uint32(api.OpCode_OP_AUTH_RES) {
		return fmt.Errorf("收到意外的响应: op=%d", env.Header.Op)
	}
	authResp := &api.AuthResponse{}
	proto.Unmarshal(env.Payload, authResp)
	if authResp.Code != uint32(common.ErrCode_ERR_CODE_OK) {
		return fmt.Errorf("认证失败: code=%d", authResp.Code)
	}
	l.Info("认证成功", "uid", authResp.Uid)

	// 4. 获取角色列表
	l.Info("发送获取角色列表请求")
	getRolesReq := &api.GetRolesRequest{}
	payload, _ = proto.Marshal(getRolesReq)
	if err := robot.Send(uint32(api.OpCode_OP_GET_ROLES_REQ), payload); err != nil {
		return fmt.Errorf("发送获取角色列表请求失败: %w", err)
	}

	env, err = robot.Recv(10 * time.Second)
	if err != nil {
		return fmt.Errorf("接收角色列表响应失败: %w", err)
	}
	if env.Header.Op != uint32(api.OpCode_OP_GET_ROLES_RES) {
		return fmt.Errorf("收到意外的响应: op=%d", env.Header.Op)
	}
	getRolesResp := &api.GetRolesResponse{}
	proto.Unmarshal(env.Payload, getRolesResp)
	if getRolesResp.Code != uint32(common.ErrCode_ERR_CODE_OK) {
		return fmt.Errorf("获取角色列表失败: code=%d", getRolesResp.Code)
	}
	l.Info("获取角色列表成功", "count", len(getRolesResp.Roles))

	var roleID int64

	// 5. 创建或选择角色
	if shouldCreateRole || len(getRolesResp.Roles) == 0 {
		// 创建新角色
		l.Info("创建新角色", "nickname", nickname)
		createReq := &api.CreateRoleRequest{
			Nickname:   nickname,
			Gender:     1,
			Appearance: "{}",
		}
		payload, _ = proto.Marshal(createReq)
		if err := robot.Send(uint32(api.OpCode_OP_CREATE_ROLE_REQ), payload); err != nil {
			return fmt.Errorf("发送创建角色请求失败: %w", err)
		}

		env, err = robot.Recv(10 * time.Second)
		if err != nil {
			return fmt.Errorf("接收创建角色响应失败: %w", err)
		}
		if env.Header.Op != uint32(api.OpCode_OP_CREATE_ROLE_RES) {
			return fmt.Errorf("收到意外的响应: op=%d", env.Header.Op)
		}
		createResp := &api.CreateRoleResponse{}
		proto.Unmarshal(env.Payload, createResp)
		if createResp.Code != uint32(common.ErrCode_ERR_CODE_OK) {
			return fmt.Errorf("创建角色失败: code=%d", createResp.Code)
		}
		roleID = createResp.Role.RoleId
		l.Info("创建角色成功", "role_id", roleID, "nickname", createResp.Role.Nickname)
	} else {
		// 选择第一个角色
		roleID = getRolesResp.Roles[0].RoleId
		l.Info("选择角色", "role_id", roleID, "nickname", getRolesResp.Roles[0].Nickname)
	}

	// 发送选择角色请求
	selectReq := &api.SelectRoleRequest{RoleId: roleID}
	payload, _ = proto.Marshal(selectReq)
	if err := robot.Send(uint32(api.OpCode_OP_SELECT_ROLE_REQ), payload); err != nil {
		return fmt.Errorf("发送选择角色请求失败: %w", err)
	}

	env, err = robot.Recv(10 * time.Second)
	if err != nil {
		return fmt.Errorf("接收选择角色响应失败: %w", err)
	}
	if env.Header.Op != uint32(api.OpCode_OP_SELECT_ROLE_RES) {
		return fmt.Errorf("收到意外的响应: op=%d", env.Header.Op)
	}
	selectResp := &api.SelectRoleResponse{}
	proto.Unmarshal(env.Payload, selectResp)
	if selectResp.Code != uint32(common.ErrCode_ERR_CODE_OK) {
		return fmt.Errorf("选择角色失败: code=%d", selectResp.Code)
	}
	l.Info("选择角色成功", "role_id", roleID)

	// 6. 进入场景
	l.Info("发送进入场景请求")
	enterReq := &api.EnterSceneRequest{}
	payload, _ = proto.Marshal(enterReq)
	if err := robot.Send(uint32(api.OpCode_OP_ENTER_SCENE_REQ), payload); err != nil {
		return fmt.Errorf("发送进入场景请求失败: %w", err)
	}

	env, err = robot.Recv(10 * time.Second)
	if err != nil {
		return fmt.Errorf("接收进入场景响应失败: %w", err)
	}
	if env.Header.Op != uint32(api.OpCode_OP_ENTER_SCENE_RES) {
		return fmt.Errorf("收到意外的响应: op=%d", env.Header.Op)
	}
	enterResp := &api.EnterSceneResponse{}
	proto.Unmarshal(env.Payload, enterResp)
	if enterResp.Code != common.ErrCode_ERR_CODE_OK {
		return fmt.Errorf("进入场景失败: code=%d", enterResp.Code)
	}
	l.Info("进入场景成功",
		"map_id", enterResp.Scene.MapId,
		"position", fmt.Sprintf("(%.2f, %.2f, %.2f)",
			enterResp.Scene.Pos.X, enterResp.Scene.Pos.Y, enterResp.Scene.Pos.Z),
	)

	return nil
}
