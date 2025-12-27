package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria-proto-api/login"
	"github.com/lk2023060901/xdooria/app/login/internal/metrics"
	"github.com/lk2023060901/xdooria/component/auth"
	"github.com/lk2023060901/xdooria/pkg/balancer"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/router"
	"github.com/lk2023060901/xdooria/pkg/security"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

type LoginService struct {
	authMgr  *auth.Manager
	jwtMgr   *security.JWTManager
	metrics  *metrics.LoginMetrics
	resolver registry.Resolver
	balancer balancer.Balancer

	// 网关缓存
	gatewayMu    sync.RWMutex
	gatewayNodes []*balancer.Node
	watchCancel  context.CancelFunc
}

func NewLoginService(
	authMgr *auth.Manager,
	jwtMgr *security.JWTManager,
	m *metrics.LoginMetrics,
	r registry.Resolver,
	b balancer.Balancer,
) *LoginService {
	if b == nil {
		b = balancer.New(balancer.RoundRobinName)
	}
	return &LoginService{
		authMgr:  authMgr,
		jwtMgr:   jwtMgr,
		metrics:  m,
		resolver: r,
		balancer: b,
	}
}

// Init 注册路由
func (s *LoginService) Init(r router.Router) {
	router.RegisterHandler(r, uint32(api.OpCode_OP_LOGIN_REQ), uint32(api.OpCode_OP_LOGIN_RES), s.Login)

	// 启动网关服务监听
	s.startGatewayWatch()
}

// startGatewayWatch 启动网关服务监听
func (s *LoginService) startGatewayWatch() {
	if s.resolver == nil {
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.watchCancel = cancel

	// 先同步获取一次
	if gateways, err := s.resolver.Resolve(ctx, "gateway"); err == nil {
		s.updateGatewayNodes(gateways)
	}

	// 启动 Watch
	ch, err := s.resolver.Watch(ctx, "gateway")
	if err != nil {
		return
	}

	conc.Go(func() (struct{}, error) {
		for {
			select {
			case <-ctx.Done():
				return struct{}{}, nil
			case gateways, ok := <-ch:
				if !ok {
					return struct{}{}, nil
				}
				s.updateGatewayNodes(gateways)
			}
		}
	})
}

// updateGatewayNodes 更新网关节点缓存
func (s *LoginService) updateGatewayNodes(gateways []*registry.ServiceInfo) {
	nodes := make([]*balancer.Node, len(gateways))
	for i, gw := range gateways {
		nodes[i] = &balancer.Node{
			Address:  gw.Address,
			Metadata: gw.Metadata,
		}
	}

	s.gatewayMu.Lock()
	s.gatewayNodes = nodes
	s.gatewayMu.Unlock()
}

// pickGateway 选择网关
func (s *LoginService) pickGateway() string {
	s.gatewayMu.RLock()
	nodes := s.gatewayNodes
	s.gatewayMu.RUnlock()

	if len(nodes) == 0 {
		return ""
	}

	if node := s.balancer.Pick(nodes, balancer.PickInfo{}); node != nil {
		return node.Address
	}
	return ""
}

// Close 关闭服务
func (s *LoginService) Close() {
	if s.watchCancel != nil {
		s.watchCancel()
	}
}

func (s *LoginService) Login(ctx context.Context, req *login.LoginRequest) (*login.LoginResponse, error) {
	start := time.Now()
	loginType := req.LoginType
	loginTypeStr := loginType.String()

	// 1. 调用通用认证组件
	identity, err := s.authMgr.Verify(ctx, loginType, req.Credentials)
	if err != nil {
		duration := time.Since(start).Seconds()
		s.metrics.RecordLogin(loginTypeStr, false, duration)
		s.metrics.RecordAuthFailure(loginTypeStr, "verify_failed")
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	// 2. 生成 JWT Token
	claims := &security.Claims{
		Payload: map[string]any{
			"uid": identity.UID,
		},
	}
	token, err := s.jwtMgr.GenerateToken(claims)
	if err != nil {
		duration := time.Since(start).Seconds()
		s.metrics.RecordLogin(loginTypeStr, false, duration)
		s.metrics.RecordAuthFailure(loginTypeStr, "token_generate_failed")
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// 3. 分配网关（从缓存中轮询选择）
	gatewayAddr := s.pickGateway()

	// 记录成功指标
	duration := time.Since(start).Seconds()
	s.metrics.RecordLogin(loginTypeStr, true, duration)

	// 4. 返回结果
	var uid uint64
	fmt.Sscanf(identity.UID, "%d", &uid)

	return &login.LoginResponse{
		Token:       token,
		Uid:         uid,
		Nickname:    identity.Nickname,
		GatewayAddr: gatewayAddr,
	}, nil
}
