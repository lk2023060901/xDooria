package service

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
	"github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria-proto-api/login"
)

type LoginService struct {
	authMgr *auth.Manager
	jwtMgr  *security.JWTManager
}

func NewLoginService(authMgr *auth.Manager, jwtMgr *security.JWTManager) *LoginService {
	return &LoginService{
		authMgr: authMgr,
		jwtMgr:  jwtMgr,
	}
}

// Init 注册路由
func (s *LoginService) Init(r router.Router) {
	router.RegisterHandler(r, uint32(api.OpCode_OP_LOGIN_REQ), uint32(api.OpCode_OP_LOGIN_RES), s.Login)
}

func (s *LoginService) Login(ctx context.Context, req *login.LoginRequest) (*login.LoginResponse, error) {
	// 1. 调用通用认证组件
	identity, err := s.authMgr.Verify(ctx, req.LoginType, req.Credentials)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// 2. 生成 JWT Token
	claims := &security.Claims{
		Payload: map[string]any{
			"uid":      identity.UID,
			"nickname": identity.Nickname,
		},
	}
	token, err := s.jwtMgr.GenerateToken(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 3. 返回结果
	// 注意：这里需要把 UID 转回 uint64
	var uid uint64
	fmt.Sscanf(identity.UID, "%d", &uid)

	return &login.LoginResponse{
		Token:    token,
		Uid:      uid,
		Nickname: identity.Nickname,
	}, nil
}
