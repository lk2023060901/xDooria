# gRPC 封装设计方案

## 概述

xDooria 框架的 gRPC 封装旨在提供统一、易用的 RPC 通信能力，集成服务发现、负载均衡、可观测性等企业级特性。

## 核心功能模块

### 1. Server 和 Client 基础封装

#### Server 封装
- 统一的 Server 创建和配置接口
- 支持多种网络协议 (TCP, Unix Socket)
- 优雅关闭机制
- 自动服务注册（集成 etcd）
- 健康检查和反射服务

**核心接口设计：**
```go
type ServerConfig struct {
    Name            string
    Network         string // "tcp", "unix"
    Address         string
    MaxRecvMsgSize  int
    MaxSendMsgSize  int
    KeepAlive       *KeepAliveConfig
    ServiceRegistry *ServiceRegistryConfig
}

type Server interface {
    RegisterService(desc *grpc.ServiceDesc, impl interface{})
    Start() error
    Stop() error
    GracefulStop() error
}
```

#### Client 封装
- 统一的 Client 连接管理
- 支持服务发现和自动负载均衡
- 连接池管理
- 自动重连机制
- 超时和重试配置

**核心接口设计：**
```go
type ClientConfig struct {
    Target          string // "etcd:///service-name" or "ip:port"
    DialTimeout     time.Duration
    KeepAlive       *KeepAliveConfig
    LoadBalancer    string // "round_robin", "weighted_round_robin"
    MaxRetries      int
    RetryBackoff    time.Duration
}

type Client interface {
    Dial() (*grpc.ClientConn, error)
    Close() error
}
```

### 2. 拦截器（Interceptor）集成

提供开箱即用的拦截器和灵活的扩展机制。

#### 内置拦截器

**Server 端：**
- **Recovery Interceptor**: panic 恢复和错误上报
- **Logging Interceptor**: 结构化请求日志
- **Metrics Interceptor**: Prometheus 指标收集
- **Tracing Interceptor**: OpenTelemetry 链路追踪
- **Auth Interceptor**: 认证和鉴权
- **Rate Limiting Interceptor**: 限流保护
- **Validation Interceptor**: 请求参数校验

**Client 端：**
- **Logging Interceptor**: 请求日志
- **Metrics Interceptor**: 客户端指标
- **Tracing Interceptor**: 链路追踪传播
- **Retry Interceptor**: 自动重试
- **Timeout Interceptor**: 超时控制

#### 拦截器链管理
```go
type InterceptorChain struct {
    unary    []grpc.UnaryServerInterceptor
    stream   []grpc.StreamServerInterceptor
}

func (c *InterceptorChain) Add(interceptor interface{}) *InterceptorChain
func (c *InterceptorChain) Build() grpc.ServerOption
```

### 3. 服务发现和负载均衡

#### 服务注册
- 基于 etcd 的服务注册
- 自动健康检查和心跳
- 服务元数据管理（版本、权重等）
- 优雅上下线

**设计：**
```go
type ServiceRegistryConfig struct {
    Enabled     bool
    EtcdConfig  *etcd.Config
    ServiceName string
    Metadata    map[string]string
    TTL         time.Duration
}

type ServiceRegistry interface {
    Register(ctx context.Context) error
    Deregister(ctx context.Context) error
    UpdateMetadata(ctx context.Context, metadata map[string]string) error
}
```

#### 服务发现
- 基于 etcd 的服务发现
- 实时监听服务变化
- 自动更新连接池

**设计：**
```go
type Resolver interface {
    Resolve(serviceName string) ([]string, error)
    Watch(serviceName string) (<-chan []string, error)
}
```

#### 负载均衡策略
- **Round Robin**: 轮询
- **Weighted Round Robin**: 加权轮询
- **Random**: 随机
- **Consistent Hash**: 一致性哈希
- 支持自定义负载均衡器

### 4. 连接池管理

#### 连接池特性
- 连接复用和管理
- 连接健康检查
- 自动清理空闲连接
- 连接限流
- 故障熔断

**设计：**
```go
type PoolConfig struct {
    MaxIdle     int
    MaxActive   int
    IdleTimeout time.Duration
    MaxLifetime time.Duration
}

type ConnectionPool interface {
    Get(ctx context.Context, target string) (*grpc.ClientConn, error)
    Put(conn *grpc.ClientConn) error
    Close() error
    Stats() PoolStats
}
```

## 可观测性集成

### 1. Prometheus 指标

#### Server 端指标
- `grpc_server_handled_total`: 请求总数（按方法、状态码）
- `grpc_server_handling_seconds`: 请求处理时长直方图
- `grpc_server_msg_received_total`: 接收消息总数
- `grpc_server_msg_sent_total`: 发送消息总数
- `grpc_server_started_total`: 启动的请求总数
- `grpc_server_connections_total`: 当前连接数

#### Client 端指标
- `grpc_client_handled_total`: 请求总数（按方法、状态码）
- `grpc_client_handling_seconds`: 请求时长直方图
- `grpc_client_msg_received_total`: 接收消息总数
- `grpc_client_msg_sent_total`: 发送消息总数
- `grpc_client_started_total`: 启动的请求总数

#### 连接池指标
- `grpc_pool_connections_total`: 连接池大小
- `grpc_pool_connections_idle`: 空闲连接数
- `grpc_pool_connections_active`: 活跃连接数
- `grpc_pool_wait_duration_seconds`: 等待连接时长

### 2. 链路追踪（OpenTelemetry）

- 自动注入和提取 trace context
- 支持 Jaeger、Zipkin 等后端
- Span 自动标记（方法名、状态码、错误信息）
- 支持自定义 span 属性
- 与 xDooria 的 logger 集成（trace ID 自动关联）

**集成示例：**
```go
type TracingConfig struct {
    Enabled      bool
    ServiceName  string
    ExporterType string // "jaeger", "zipkin", "otlp"
    Endpoint     string
    SamplingRate float64
}
```

### 3. 结构化日志

- 集成 xDooria 的 logger 包
- 自动记录请求信息（方法、参数、响应、错误）
- 支持日志级别配置
- 自动关联 trace ID 和 request ID
- 敏感信息脱敏

**日志格式：**
```json
{
  "timestamp": "2025-01-01T12:00:00Z",
  "level": "info",
  "service": "user-service",
  "trace_id": "abc123...",
  "span_id": "def456...",
  "grpc_method": "/user.UserService/GetUser",
  "grpc_code": "OK",
  "duration_ms": 45,
  "peer": "192.168.1.100:50051",
  "message": "request completed"
}
```

### 4. Sentry 错误上报

- 自动捕获 panic 和错误
- 上报 gRPC 错误详情（方法、状态码、错误消息）
- 上报调用栈和上下文信息
- 支持自定义错误过滤规则
- 集成 trace ID 关联

**配置：**
```go
type SentryConfig struct {
    Enabled     bool
    DSN         string
    Environment string
    SampleRate  float64
    IgnoreCodes []codes.Code // 忽略特定状态码
}
```

## 其他建议功能

### 1. 中间件链式调用

提供灵活的中间件机制，允许业务自定义处理逻辑：

```go
type Middleware func(ctx context.Context, req interface{}, handler grpc.UnaryHandler) (interface{}, error)

type MiddlewareChain struct {
    middlewares []Middleware
}

func (c *MiddlewareChain) Use(middleware Middleware) *MiddlewareChain
func (c *MiddlewareChain) Build() grpc.UnaryServerInterceptor
```

### 2. 配置管理

- 集成 xDooria 的 config 包
- 支持配置热加载
- 支持环境变量覆盖
- 配置验证

**配置文件示例：**
```yaml
grpc:
  server:
    name: user-service
    address: :50051
    max_recv_msg_size: 4194304  # 4MB
    max_send_msg_size: 4194304
    keep_alive:
      max_idle: 5m
      timeout: 10s

  client:
    user_service:
      target: etcd:///user-service
      dial_timeout: 5s
      load_balancer: round_robin
      max_retries: 3

  interceptors:
    - logging
    - metrics
    - tracing
    - recovery

  observability:
    prometheus:
      enabled: true
    tracing:
      enabled: true
      exporter: jaeger
      endpoint: http://jaeger:14268/api/traces
    sentry:
      enabled: true
      dsn: ${SENTRY_DSN}
```

### 3. 测试支持

- 提供 mock server 和 mock client
- 提供测试拦截器和断言工具
- 集成 gRPC reflection 用于调试

**测试工具：**
```go
type MockServer struct {
    handlers map[string]grpc.UnaryHandler
}

func NewMockServer() *MockServer
func (s *MockServer) RegisterHandler(method string, handler grpc.UnaryHandler)
func (s *MockServer) Start() (string, error) // 返回临时地址
func (s *MockServer) Stop()
```

### 4. 错误处理

- 统一的错误码定义
- 错误码与 HTTP 状态码映射
- 错误详情封装（支持 gRPC status 和 details）
- 错误链追踪

**错误封装：**
```go
type Error struct {
    Code    codes.Code
    Message string
    Details []proto.Message
    Cause   error
}

func NewError(code codes.Code, message string) *Error
func (e *Error) WithDetails(details ...proto.Message) *Error
func (e *Error) WithCause(cause error) *Error
func (e *Error) ToStatus() *status.Status
```

### 5. 流式调用支持

- Server streaming 封装
- Client streaming 封装
- Bidirectional streaming 封装
- 流式调用的可观测性

### 6. 安全特性

- TLS/mTLS 支持
- Token 认证（JWT）
- API Key 认证
- IP 白名单
- 请求签名验证

## 目录结构建议

```
pkg/grpc/
├── server/
│   ├── server.go              # Server 核心实现
│   ├── config.go              # Server 配置
│   └── registry.go            # 服务注册
├── client/
│   ├── client.go              # Client 核心实现
│   ├── config.go              # Client 配置
│   ├── pool.go                # 连接池
│   └── resolver.go            # 服务发现
├── interceptor/
│   ├── logging.go             # 日志拦截器
│   ├── metrics.go             # 指标拦截器
│   ├── tracing.go             # 追踪拦截器
│   ├── recovery.go            # 恢复拦截器
│   ├── auth.go                # 认证拦截器
│   ├── ratelimit.go           # 限流拦截器
│   ├── retry.go               # 重试拦截器
│   └── chain.go               # 拦截器链
├── middleware/
│   ├── middleware.go          # 中间件接口
│   └── chain.go               # 中间件链
├── balancer/
│   ├── roundrobin.go          # 轮询负载均衡
│   ├── weighted.go            # 加权负载均衡
│   ├── random.go              # 随机负载均衡
│   └── consistent_hash.go     # 一致性哈希
├── observability/
│   ├── metrics.go             # Prometheus 指标
│   ├── tracing.go             # OpenTelemetry 集成
│   ├── logging.go             # 日志集成
│   └── sentry.go              # Sentry 集成
├── errors/
│   └── errors.go              # 错误处理
├── security/
│   ├── tls.go                 # TLS 配置
│   ├── auth.go                # 认证
│   └── token.go               # Token 验证
└── testing/
    ├── mock_server.go         # Mock Server
    └── mock_client.go         # Mock Client

examples/grpc/
├── simple/                    # 简单示例
│   ├── server/
│   └── client/
├── observability/             # 可观测性示例
│   ├── server/
│   └── client/
├── service_discovery/         # 服务发现示例
│   ├── server/
│   └── client/
├── streaming/                 # 流式调用示例
│   ├── server/
│   └── client/
└── secure/                    # 安全特性示例
    ├── server/
    └── client/
```

## 与 xDooria-proto-* 仓库集成

### Proto 仓库划分

- **xDooria-proto-api**: 对外暴露的公共 API 定义
- **xDooria-proto-common**: 通用消息和枚举定义
- **xDooria-proto-internal**: 内部服务间通信的 proto 定义

### 集成方案

#### 1. Proto 文件管理

**目录结构：**
```
xDooria-proto-api/
├── user/v1/
│   ├── user.proto
│   └── user_service.proto
├── order/v1/
│   ├── order.proto
│   └── order_service.proto
└── buf.yaml

xDooria-proto-common/
├── types/
│   ├── timestamp.proto
│   ├── pagination.proto
│   └── error.proto
└── buf.yaml

xDooria-proto-internal/
├── cache/v1/
│   └── cache_service.proto
├── queue/v1/
│   └── queue_service.proto
└── buf.yaml
```

#### 2. 代码生成

使用 Buf 进行统一的代码生成：

**buf.gen.yaml:**
```yaml
version: v1
plugins:
  - plugin: go
    out: gen/go
    opt:
      - paths=source_relative
  - plugin: go-grpc
    out: gen/go
    opt:
      - paths=source_relative
      - require_unimplemented_servers=false
  - plugin: grpc-gateway
    out: gen/go
    opt:
      - paths=source_relative
  - plugin: openapiv2
    out: gen/openapiv2
```

#### 3. 集成到 xDooria 框架

**生成的代码集成：**
```go
// pkg/grpc/proto/loader.go
package proto

import (
    userv1 "github.com/lk2023060901/xDooria-proto-api/gen/go/user/v1"
    orderv1 "github.com/lk2023060901/xDooria-proto-api/gen/go/order/v1"
)

// 提供统一的 proto 访问入口
type ProtoRegistry struct {
    // API protos
    UserV1  *userv1.Package
    OrderV1 *orderv1.Package
}

func NewProtoRegistry() *ProtoRegistry {
    return &ProtoRegistry{
        UserV1:  &userv1.Package{},
        OrderV1: &orderv1.Package{},
    }
}
```

#### 4. Server 自动注册

```go
// pkg/grpc/server/auto_register.go
package server

import (
    "google.golang.org/grpc"
    userv1 "github.com/lk2023060901/xDooria-proto-api/gen/go/user/v1"
)

type ServiceRegistrar interface {
    RegisterServices(s *grpc.Server)
}

// 示例：自动注册 User Service
func (srv *Server) RegisterUserService(impl userv1.UserServiceServer) {
    userv1.RegisterUserServiceServer(srv.grpcServer, impl)
}
```

#### 5. Client 自动生成

```go
// pkg/grpc/client/auto_client.go
package client

import (
    userv1 "github.com/lk2023060901/xDooria-proto-api/gen/go/user/v1"
)

type Clients struct {
    User  userv1.UserServiceClient
    Order orderv1.OrderServiceClient
}

func NewClients(clientManager *ClientManager) (*Clients, error) {
    userConn, err := clientManager.Dial("user-service")
    if err != nil {
        return nil, err
    }

    orderConn, err := clientManager.Dial("order-service")
    if err != nil {
        return nil, err
    }

    return &Clients{
        User:  userv1.NewUserServiceClient(userConn),
        Order: orderv1.NewOrderServiceClient(orderConn),
    }, nil
}
```

### 版本管理

- Proto 仓库使用语义化版本（v1, v2）
- 支持多版本共存
- 使用 Buf 进行破坏性变更检测

### CI/CD 集成

**Proto 仓库 CI：**
```yaml
# .github/workflows/buf.yml
name: Buf CI
on: [push, pull_request]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: bufbuild/buf-setup-action@v1
      - run: buf lint
      - run: buf breaking --against '.git#branch=main'
      - run: buf generate
```

## 实现优先级

### Phase 1: 核心功能（2-3 周）
1. Server 和 Client 基础封装
2. 基本拦截器（logging, recovery, metrics）
3. 与 xDooria-proto-* 仓库集成
4. 基础示例

### Phase 2: 服务发现（1-2 周）
1. etcd 服务注册和发现
2. 负载均衡器实现
3. 连接池管理
4. 服务发现示例

### Phase 3: 可观测性（1-2 周）
1. Prometheus 指标完善
2. OpenTelemetry 集成
3. Sentry 错误上报
4. 可观测性示例

### Phase 4: 高级特性（2-3 周）
1. 中间件链
2. 安全特性（TLS, Auth）
3. 流式调用支持
4. 测试工具

## 注意事项

1. **向后兼容**: 保持 API 稳定，避免破坏性变更
2. **性能优化**: 注意拦截器和中间件的性能开销
3. **错误处理**: 统一错误码和错误信息
4. **文档完善**: 提供详细的使用文档和示例
5. **测试覆盖**: 确保核心功能有完善的单元测试和集成测试

## 参考资料

- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [grpc-ecosystem/go-grpc-middleware](https://github.com/grpc-ecosystem/go-grpc-middleware)
- [OpenTelemetry Go](https://opentelemetry.io/docs/instrumentation/go/)
- [Prometheus Client Go](https://github.com/prometheus/client_golang)
- [etcd Client v3](https://etcd.io/docs/v3.5/dev-guide/api_reference_v3/)
