package etcd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/config"
	xdooriaetcd "github.com/lk2023060901/xdooria/pkg/etcd"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/registry"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// Registrar 基于 etcd 的服务注册器
type Registrar struct {
	client      *xdooriaetcd.Client
	config      *Config
	serviceInfo *registry.ServiceInfo
	leaseID     int64
	logger      logger.Logger
	pool        *conc.Pool[struct{}]
	stopCh      chan struct{}
}

// NewRegistrar 创建 etcd 服务注册器
func NewRegistrar(cfg *Config) (*Registrar, error) {
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge config: %w", err)
	}

	if err := newCfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	etcdCfg := &xdooriaetcd.Config{
		Endpoints:   newCfg.Endpoints,
		DialTimeout: newCfg.DialTimeout,
	}

	client, err := xdooriaetcd.New(etcdCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create etcd client: %w", err)
	}

	return &Registrar{
		client:  client,
		config:  newCfg,
		logger:  logger.Default().Named("registry.etcd"),
		pool:    conc.NewDefaultPool[struct{}](),
		stopCh:  make(chan struct{}),
	}, nil
}

// Register 注册服务
func (r *Registrar) Register(ctx context.Context, info *registry.ServiceInfo) error {
	r.serviceInfo = info

	// 创建租约
	leaseID, err := r.client.Lease().Grant(ctx, int64(r.config.TTL.Seconds()))
	if err != nil {
		return fmt.Errorf("failed to grant lease: %w", err)
	}
	r.leaseID = int64(leaseID)

	// 序列化服务信息
	value, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	// 注册服务
	key := r.buildKey(info.ServiceName, info.Address)
	if err := r.client.KV().PutWithLease(ctx, key, string(value), leaseID); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	r.logger.Info("service registered",
		"service", info.ServiceName,
		"address", info.Address,
		"lease_id", r.leaseID,
	)

	// 启动心跳保活（使用 conc.Pool）
	r.pool.Submit(func() (struct{}, error) {
		r.keepAlive()
		return struct{}{}, nil
	})

	return nil
}

// Deregister 取消注册
func (r *Registrar) Deregister(ctx context.Context) error {
	if r.serviceInfo == nil {
		return nil
	}

	close(r.stopCh)

	key := r.buildKey(r.serviceInfo.ServiceName, r.serviceInfo.Address)
	if _, err := r.client.KV().Delete(ctx, key); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	// 撤销租约
	if r.leaseID != 0 {
		if err := r.client.Lease().Revoke(ctx, xdooriaetcd.LeaseID(r.leaseID)); err != nil {
			r.logger.Warn("failed to revoke lease", "error", err)
		}
	}

	r.logger.Info("service deregistered",
		"service", r.serviceInfo.ServiceName,
		"address", r.serviceInfo.Address,
	)

	r.pool.Release()
	return r.client.Close()
}

// UpdateMetadata 更新元数据
func (r *Registrar) UpdateMetadata(ctx context.Context, metadata map[string]string) error {
	if r.serviceInfo == nil {
		return fmt.Errorf("service not registered")
	}

	r.serviceInfo.Metadata = metadata

	value, err := json.Marshal(r.serviceInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal service info: %w", err)
	}

	key := r.buildKey(r.serviceInfo.ServiceName, r.serviceInfo.Address)
	if err := r.client.KV().PutWithLease(ctx, key, string(value), xdooriaetcd.LeaseID(r.leaseID)); err != nil {
		return fmt.Errorf("failed to update metadata: %w", err)
	}

	r.logger.Info("metadata updated",
		"service", r.serviceInfo.ServiceName,
		"metadata", metadata,
	)

	return nil
}

// buildKey 构建服务注册 key
func (r *Registrar) buildKey(serviceName, address string) string {
	return fmt.Sprintf("%s/%s/%s", r.config.Namespace, serviceName, address)
}

// keepAlive 保持心跳
func (r *Registrar) keepAlive() {
	ctx := context.Background()
	ch, err := r.client.Lease().KeepAlive(ctx, xdooriaetcd.LeaseID(r.leaseID))
	if err != nil {
		r.logger.Error("failed to keep alive", "error", err)
		r.reRegister()
		return
	}

	for {
		select {
		case <-r.stopCh:
			return
		case _, ok := <-ch:
			if !ok {
				r.logger.Warn("keep alive channel closed, attempting to re-register")
				r.reRegister()
				return
			}
		}
	}
}

// reRegister 自动重新注册
func (r *Registrar) reRegister() {
	if r.serviceInfo == nil {
		r.logger.Warn("no service info to re-register")
		return
	}

	ctx := context.Background()

	// 创建新的租约
	leaseID, err := r.client.Lease().Grant(ctx, int64(r.config.TTL.Seconds()))
	if err != nil {
		r.logger.Error("failed to grant lease for re-register", "error", err)
		return
	}
	r.leaseID = int64(leaseID)

	// 序列化服务信息
	value, err := json.Marshal(r.serviceInfo)
	if err != nil {
		r.logger.Error("failed to marshal service info for re-register", "error", err)
		return
	}

	// 重新注册服务
	key := r.buildKey(r.serviceInfo.ServiceName, r.serviceInfo.Address)
	if err := r.client.KV().PutWithLease(ctx, key, string(value), leaseID); err != nil {
		r.logger.Error("failed to re-register service", "error", err)
		return
	}

	r.logger.Info("service re-registered successfully",
		"service", r.serviceInfo.ServiceName,
		"address", r.serviceInfo.Address,
		"new_lease_id", r.leaseID,
	)

	// 重新启动心跳保活
	r.pool.Submit(func() (struct{}, error) {
		r.keepAlive()
		return struct{}{}, nil
	})
}
