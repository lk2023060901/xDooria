package client

import (
	"context"
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	grpcclient "github.com/lk2023060901/xdooria/pkg/network/grpc/client"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/registry/etcd"
	api "github.com/lk2023060901/xdooria-proto-api"
	"github.com/lk2023060901/xdooria-proto-api/login"
	pb "github.com/lk2023060901/xdooria-proto-common"
	"google.golang.org/grpc/resolver"
	"google.golang.org/protobuf/proto"
)

// LoginClientConfig Login 客户端配置
type LoginClientConfig struct {
	// gRPC 客户端配置
	Client grpcclient.Config `mapstructure:"client" json:"client" yaml:"client"`
	// Framer 配置
	Framer framer.Config `mapstructure:"framer" json:"framer" yaml:"framer"`
}

// DefaultLoginClientConfig 默认配置
func DefaultLoginClientConfig() *LoginClientConfig {
	return &LoginClientConfig{
		Client: *grpcclient.DefaultConfig(),
		Framer: *framer.DefaultConfig(),
	}
}

// LoginClient Login 服务客户端
type LoginClient struct {
	config   *LoginClientConfig
	client   *grpcclient.Client
	framer   framer.Framer
	logger   logger.Logger
	pbClient pb.CommonServiceClient
}

// NewLoginClient 创建 Login 客户端
func NewLoginClient(cfg *LoginClientConfig, etcdCfg *etcd.Config, l logger.Logger) (*LoginClient, error) {
	newCfg, err := config.MergeConfig(DefaultLoginClientConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge login client config: %w", err)
	}

	// 注册 etcd resolver
	if err := etcd.RegisterBuilder(etcdCfg); err != nil {
		return nil, fmt.Errorf("failed to register etcd resolver: %w", err)
	}

	// 设置 target 为 etcd 服务发现格式
	if newCfg.Client.Target == "" || newCfg.Client.Target == "localhost:50051" {
		newCfg.Client.Target = "etcd:///login"
	}

	// 创建 gRPC 客户端
	grpcClient, err := grpcclient.New(&newCfg.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	// 建立连接
	if err := grpcClient.Dial(); err != nil {
		return nil, fmt.Errorf("failed to dial login service: %w", err)
	}

	conn, err := grpcClient.GetConn()
	if err != nil {
		return nil, fmt.Errorf("failed to get connection: %w", err)
	}

	// 创建 Framer
	f, err := framer.New(&newCfg.Framer)
	if err != nil {
		return nil, fmt.Errorf("failed to create framer: %w", err)
	}

	return &LoginClient{
		config:   newCfg,
		client:   grpcClient,
		framer:   f,
		logger:   l.Named("client.login"),
		pbClient: pb.NewCommonServiceClient(conn),
	}, nil
}

// Login 执行登录
func (c *LoginClient) Login(ctx context.Context, req *login.LoginRequest) (*login.LoginResponse, error) {
	// 序列化请求
	payload, err := proto.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 封装请求
	envelope, err := c.framer.Encode(uint32(api.OpCode_OP_LOGIN_REQ), payload)
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %w", err)
	}

	// 发送请求
	resp, err := c.pbClient.Call(ctx, envelope)
	if err != nil {
		return nil, fmt.Errorf("failed to call login service: %w", err)
	}

	// 解码响应
	_, respPayload, err := c.framer.Decode(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 解析响应
	var loginResp login.LoginResponse
	if err := proto.Unmarshal(respPayload, &loginResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return &loginResp, nil
}

// Close 关闭客户端
func (c *LoginClient) Close() error {
	return c.client.Close()
}

// RegisterResolver 注册 etcd resolver（全局只需调用一次）
func RegisterResolver(cfg *etcd.Config) error {
	builder, err := etcd.NewResolverBuilder(cfg)
	if err != nil {
		return fmt.Errorf("failed to create resolver builder: %w", err)
	}
	resolver.Register(builder)
	return nil
}
