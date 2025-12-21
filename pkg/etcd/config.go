package etcd

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Endpoints:          []string{"localhost:2379"},
		DialTimeout:        5 * time.Second,
		AutoSyncInterval:   0,
		MaxCallSendMsgSize: 2 * 1024 * 1024,  // 2MB
		MaxCallRecvMsgSize: 4 * 1024 * 1024,  // 4MB
		EnableRetry:        true,
		MaxRetries:         3,
		RetryInterval:      100 * time.Millisecond,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if len(c.Endpoints) == 0 {
		return fmt.Errorf("%w: endpoints cannot be empty", ErrInvalidConfig)
	}

	if c.DialTimeout <= 0 {
		return fmt.Errorf("%w: dial timeout must be positive", ErrInvalidConfig)
	}

	// 如果启用 TLS，必须提供证书文件
	if c.CertFile != "" || c.KeyFile != "" || c.CAFile != "" {
		if c.CertFile == "" || c.KeyFile == "" || c.CAFile == "" {
			return fmt.Errorf("%w: all TLS files (cert, key, ca) must be provided", ErrInvalidConfig)
		}
	}

	return nil
}

// ToClientConfig 转换为 etcd client 配置
func (c *Config) ToClientConfig() (*clientv3.Config, error) {
	config := &clientv3.Config{
		Endpoints:            c.Endpoints,
		DialTimeout:          c.DialTimeout,
		AutoSyncInterval:     c.AutoSyncInterval,
		MaxCallSendMsgSize:   c.MaxCallSendMsgSize,
		MaxCallRecvMsgSize:   c.MaxCallRecvMsgSize,
		RejectOldCluster:     true,
		PermitWithoutStream:  true,
	}

	// 认证
	if c.Username != "" && c.Password != "" {
		config.Username = c.Username
		config.Password = c.Password
	}

	// TLS 配置
	if c.CertFile != "" {
		tlsConfig, err := c.buildTLSConfig()
		if err != nil {
			return nil, err
		}
		config.TLS = tlsConfig
	}

	// gRPC 配置
	dialOpts := []grpc.DialOption{
		// 禁用 gRPC 的默认服务配置解析，避免 etcd 内部 resolver 冲突
		grpc.WithDisableServiceConfig(),
	}

	// 如果没有配置 TLS，使用 insecure credentials
	if config.TLS == nil {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	config.DialOptions = dialOpts

	return config, nil
}

// buildTLSConfig 构建 TLS 配置
func (c *Config) buildTLSConfig() (*tls.Config, error) {
	// 加载客户端证书
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("load client cert failed: %w", err)
	}

	// 加载 CA 证书
	caData, err := os.ReadFile(c.CAFile)
	if err != nil {
		return nil, fmt.Errorf("read ca file failed: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caData) {
		return nil, fmt.Errorf("failed to parse ca certificate")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      pool,
		MinVersion:   tls.VersionTLS12,
	}, nil
}
