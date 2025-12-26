package service

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/lk2023060901/xdooria/app/login/internal/metrics"
	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria-proto-api/login"
)

type LoginService struct {
	authMgr  *auth.Manager
	jwtMgr   *security.JWTManager
	metrics  *metrics.LoginMetrics
	resolver registry.Resolver
}

func NewLoginService(
	authMgr *auth.Manager,
	jwtMgr *security.JWTManager,
	m *metrics.LoginMetrics,
	r registry.Resolver,
) *LoginService {
	return &LoginService{
		authMgr:  authMgr,
		jwtMgr:   jwtMgr,
		metrics:  m,
		resolver: r,
	}
}

// Init 注册路由
func (s *LoginService) Init(r router.Router) {
	router.RegisterHandler(r, uint32(api.OpCode_OP_LOGIN_REQ), uint32(api.OpCode_OP_LOGIN_RES), s.Login)
}

func (s *LoginService) Login(ctx context.Context, req *login.LoginRequest) (*login.LoginResponse, error) {
	start := time.Now()
	loginType := req.LoginType
	loginTypeStr := loginType.String()

	// 1. 调用通用认证组件
	identity, err := s.authMgr.Verify(ctx, loginType, req.Credentials)
	if err != nil {
		// 记录认证失败指标
		duration := time.Since(start).Seconds()
		s.metrics.RecordLogin(loginTypeStr, false, duration)
		s.metrics.RecordAuthFailure(loginTypeStr, "verify_failed")
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
		duration := time.Since(start).Seconds()
		s.metrics.RecordLogin(loginTypeStr, false, duration)
		s.metrics.RecordAuthFailure(loginTypeStr, "token_generate_failed")
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 3. 分配网关
	gatewayAddr := ""
	if s.resolver != nil {
		gateways, err := s.resolver.Resolve(ctx, "gateway")
		if err == nil && len(gateways) > 0 {
			// 随机分配一个网关
			idx := rand.Intn(len(gateways))
			gatewayAddr = gateways[idx].Address
		}
	}

	// 记录成功指标
	duration := time.Since(start).Seconds()
	s.metrics.RecordLogin(loginTypeStr, true, duration)

	// 4. 返回结果
	var uid uint64
	fmt.Sscanf(identity.UID, "%d", &uid)

	return &login.LoginResponse{
		Token:        token,
		Uid:          uid,
		Nickname:     identity.Nickname,
		GatewayAddr:  gatewayAddr,
	}, nil
}
