package etcd

import (
	"context"
	"sync"

	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
)

// HealthChecker gRPC 健康检查器
type HealthChecker struct {
	server *health.Server
	mu     sync.RWMutex
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		server: health.NewServer(),
	}
}

// Server 返回 gRPC Health Server
func (h *HealthChecker) Server() *health.Server {
	return h.server
}

// SetServingStatus 设置服务健康状态
func (h *HealthChecker) SetServingStatus(service string, status grpc_health_v1.HealthCheckResponse_ServingStatus) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.server.SetServingStatus(service, status)
}

// SetServing 设置服务为可用状态
func (h *HealthChecker) SetServing(service string) {
	h.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_SERVING)
}

// SetNotServing 设置服务为不可用状态
func (h *HealthChecker) SetNotServing(service string) {
	h.SetServingStatus(service, grpc_health_v1.HealthCheckResponse_NOT_SERVING)
}

// Shutdown 关闭健康检查器
func (h *HealthChecker) Shutdown() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.server.Shutdown()
}

// CheckHealth 检查服务健康状态
func (h *HealthChecker) CheckHealth(ctx context.Context, service string) (grpc_health_v1.HealthCheckResponse_ServingStatus, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	resp, err := h.server.Check(ctx, &grpc_health_v1.HealthCheckRequest{
		Service: service,
	})
	if err != nil {
		return grpc_health_v1.HealthCheckResponse_UNKNOWN, err
	}

	return resp.Status, nil
}
