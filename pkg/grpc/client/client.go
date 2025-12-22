package client

import (
	"context"
	"fmt"
	"sync"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/grpc/interceptor"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Client gRPC Client 封装
type Client struct {
	config *Config
	conn   *grpc.ClientConn
	logger logger.Logger

	// 选项
	dialOpts           []grpc.DialOption
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor

	// 状态管理
	mu        sync.RWMutex
	connected bool
}

// New 创建 gRPC Client
func New(cfg *Config, opts ...Option) (*Client, error) {
	// 合并配置
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		config:             newCfg,
		logger:             logger.Default().Named("grpc.client"),
		dialOpts:           make([]grpc.DialOption, 0),
		unaryInterceptors:  make([]grpc.UnaryClientInterceptor, 0),
		streamInterceptors: make([]grpc.StreamClientInterceptor, 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	// 添加默认的超时拦截器（放在最前面，优先级最高）
	if newCfg.RequestTimeout > 0 {
		c.unaryInterceptors = append(
			[]grpc.UnaryClientInterceptor{interceptor.ClientTimeoutInterceptor(newCfg.RequestTimeout)},
			c.unaryInterceptors...,
		)
		c.streamInterceptors = append(
			[]grpc.StreamClientInterceptor{interceptor.StreamClientTimeoutInterceptor(newCfg.RequestTimeout)},
			c.streamInterceptors...,
		)
	}

	return c, nil
}

// Dial 建立连接
func (c *Client) Dial() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return ErrClientAlreadyConnected
	}

	// 构建 DialOptions
	dialOpts := c.buildDialOptions()

	// 创建带超时的 Context
	ctx, cancel := context.WithTimeout(context.Background(), c.config.DialTimeout)
	defer cancel()

	c.logger.Info("dialing gRPC server",
		"target", c.config.Target,
		"timeout", c.config.DialTimeout,
	)

	conn, err := grpc.DialContext(ctx, c.config.Target, dialOpts...)
	if err != nil {
		return fmt.Errorf("%w: %s: %v", ErrDialFailed, c.config.Target, err)
	}

	c.conn = conn
	c.connected = true

	c.logger.Info("connected to gRPC server",
		"target", c.config.Target,
	)

	return nil
}

// GetConn 获取连接
func (c *Client) GetConn() (*grpc.ClientConn, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if !c.connected || c.conn == nil {
		return nil, ErrClientNotConnected
	}

	return c.conn, nil
}

// Close 关闭连接
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil
	}

	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
	}

	c.connected = false
	c.logger.Info("gRPC client closed")

	return nil
}

// buildDialOptions 构建 DialOptions
func (c *Client) buildDialOptions() []grpc.DialOption {
	opts := make([]grpc.DialOption, 0)

	// 默认使用不安全连接（TLS 后续实现）
	opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))

	// 消息大小限制
	opts = append(opts,
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(c.config.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(c.config.MaxSendMsgSize),
		),
	)

	// KeepAlive 配置
	opts = append(opts,
		grpc.WithKeepaliveParams(c.config.KeepAlive),
	)

	// 负载均衡
	if c.config.LoadBalancer != "" {
		opts = append(opts, grpc.WithDefaultServiceConfig(fmt.Sprintf(
			`{"loadBalancingPolicy":"%s"}`, c.config.LoadBalancer,
		)))
	}

	// 拦截器链
	if len(c.unaryInterceptors) > 0 {
		opts = append(opts, grpc.WithChainUnaryInterceptor(c.unaryInterceptors...))
	}
	if len(c.streamInterceptors) > 0 {
		opts = append(opts, grpc.WithChainStreamInterceptor(c.streamInterceptors...))
	}

	// 用户自定义选项
	opts = append(opts, c.dialOpts...)

	return opts
}
