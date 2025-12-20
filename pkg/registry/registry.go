package registry

import "context"

// ServiceInfo 服务信息
type ServiceInfo struct {
	// ServiceName 服务名称
	ServiceName string
	// Address 服务地址（如 192.168.1.10:50051）
	Address string
	// Metadata 元数据（如 version, weight, region 等）
	Metadata map[string]string
}

// Registrar 服务注册接口
type Registrar interface {
	// Register 注册服务
	Register(ctx context.Context, info *ServiceInfo) error
	// Deregister 取消注册
	Deregister(ctx context.Context) error
	// UpdateMetadata 更新元数据
	UpdateMetadata(ctx context.Context, metadata map[string]string) error
}

// Resolver 服务发现接口
type Resolver interface {
	// Resolve 解析服务地址列表
	Resolve(ctx context.Context, serviceName string) ([]*ServiceInfo, error)
	// Watch 监听服务变化
	Watch(ctx context.Context, serviceName string) (<-chan []*ServiceInfo, error)
}
