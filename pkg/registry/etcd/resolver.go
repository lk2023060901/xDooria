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
	"go.uber.org/zap"
)

// Resolver 基于 etcd 的服务发现器
type Resolver struct {
	client *xdooriaetcd.Client
	config *Config
	logger *logger.Logger
	pool   *conc.Pool[struct{}]
}

// NewResolver 创建 etcd 服务发现器
func NewResolver(cfg *Config) (*Resolver, error) {
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

	return &Resolver{
		client: client,
		config: newCfg,
		logger: logger.Default().Named("resolver.etcd"),
		pool:   conc.NewDefaultPool[struct{}](),
	}, nil
}

// Resolve 解析服务地址列表
func (r *Resolver) Resolve(ctx context.Context, serviceName string) ([]*registry.ServiceInfo, error) {
	prefix := fmt.Sprintf("%s/%s/", r.config.Namespace, serviceName)

	kvs, err := r.client.KV().GetWithPrefix(ctx, prefix)
	if err != nil {
		// 如果没有找到，返回空列表而非错误
		if err == xdooriaetcd.ErrKeyNotFound {
			return []*registry.ServiceInfo{}, nil
		}
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	services := make([]*registry.ServiceInfo, 0, len(kvs))
	for _, kv := range kvs {
		var info registry.ServiceInfo
		if err := json.Unmarshal([]byte(kv.Value), &info); err != nil {
			r.logger.Warn("failed to unmarshal service info",
				zap.String("key", kv.Key),
				zap.Error(err),
			)
			continue
		}
		services = append(services, &info)
	}

	r.logger.Debug("services resolved",
		zap.String("service", serviceName),
		zap.Int("count", len(services)),
	)

	return services, nil
}

// Watch 监听服务变化
func (r *Resolver) Watch(ctx context.Context, serviceName string) (<-chan []*registry.ServiceInfo, error) {
	prefix := fmt.Sprintf("%s/%s/", r.config.Namespace, serviceName)

	resultCh := make(chan []*registry.ServiceInfo, 1)

	// 使用 conc.Pool 而不是原始 goroutine
	r.pool.Submit(func() (struct{}, error) {
		defer close(resultCh)

		// 使用 Watcher.WatchPrefix 监听服务变化
		err := r.client.Watcher().WatchPrefix(ctx, prefix, func(event *xdooriaetcd.WatchEvent) {
			// 当有变化时，重新获取所有服务
			services, err := r.Resolve(context.Background(), serviceName)
			if err != nil {
				r.logger.Error("failed to resolve services on watch event",
					zap.Error(err),
				)
				return
			}

			r.logger.Debug("service list updated",
				zap.String("service", serviceName),
				zap.String("event_type", string(event.Type)),
				zap.Int("count", len(services)),
			)

			select {
			case resultCh <- services:
			case <-ctx.Done():
			}
		})

		if err != nil {
			r.logger.Error("watch failed", zap.Error(err))
		}

		return struct{}{}, nil
	})

	return resultCh, nil
}

// Close 关闭 Resolver
func (r *Resolver) Close() error {
	r.pool.Release()
	return r.client.Close()
}
