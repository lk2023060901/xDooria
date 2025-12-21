package etcd

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// ClientOption gRPC Client 配置选项
type ClientOption func(*clientOptions)

type clientOptions struct {
	balancerName     string
	timeout          time.Duration
	dialOptions      []grpc.DialOption
	disableRetry     bool
	disableHealthCheck bool
}

// WithBalancer 设置负载均衡策略
func WithBalancer(name string) ClientOption {
	return func(o *clientOptions) {
		o.balancerName = name
	}
}

// WithTimeout 设置连接超时
func WithTimeout(timeout time.Duration) ClientOption {
	return func(o *clientOptions) {
		o.timeout = timeout
	}
}

// WithDialOptions 设置额外的 DialOption
func WithDialOptions(opts ...grpc.DialOption) ClientOption {
	return func(o *clientOptions) {
		o.dialOptions = append(o.dialOptions, opts...)
	}
}

// WithDisableRetry 禁用重试
func WithDisableRetry() ClientOption {
	return func(o *clientOptions) {
		o.disableRetry = true
	}
}

// WithDisableHealthCheck 禁用健康检查
func WithDisableHealthCheck() ClientOption {
	return func(o *clientOptions) {
		o.disableHealthCheck = true
	}
}

// DialService 创建 gRPC Client 连接，使用服务发现
// serviceName 格式: etcd:///service-name
func DialService(serviceName string, opts ...ClientOption) (*grpc.ClientConn, error) {
	options := &clientOptions{
		balancerName: RoundRobinBalancerName, // 默认使用 Round Robin
		timeout:      5 * time.Second,
		dialOptions:  make([]grpc.DialOption, 0),
	}

	for _, opt := range opts {
		opt(options)
	}

	// 构建 service config
	serviceConfig := fmt.Sprintf(`{
		"loadBalancingPolicy": "%s",
		"waitForReady": true
	}`, options.balancerName)

	if !options.disableHealthCheck {
		serviceConfig = fmt.Sprintf(`{
			"loadBalancingPolicy": "%s",
			"waitForReady": true,
			"healthCheckConfig": {
				"serviceName": ""
			}
		}`, options.balancerName)
	}

	// 基础 DialOption
	baseOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(serviceConfig),
	}

	// 合并用户提供的 DialOption
	allOpts := append(baseOpts, options.dialOptions...)

	// 创建连接（不阻塞，异步连接）
	// waitForReady 已在 serviceConfig 中配置，第一个 RPC 会自动等待服务就绪
	conn, err := grpc.Dial(serviceName, allOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial service %s: %w", serviceName, err)
	}

	return conn, nil
}

// DialServiceWithConsistentHash 使用一致性哈希创建连接
func DialServiceWithConsistentHash(serviceName string, opts ...ClientOption) (*grpc.ClientConn, error) {
	opts = append(opts, WithBalancer(ConsistentHashBalancerName))
	return DialService(serviceName, opts...)
}

// DialServiceWithRoundRobin 使用 Round Robin 创建连接
func DialServiceWithRoundRobin(serviceName string, opts ...ClientOption) (*grpc.ClientConn, error) {
	opts = append(opts, WithBalancer(RoundRobinBalancerName))
	return DialService(serviceName, opts...)
}

// WithHashKey 在 context 中添加一致性哈希的 hash-key
// 用于一致性哈希负载均衡时指定路由键
func WithHashKey(ctx context.Context, key string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, HashKeyHeader, key)
}
