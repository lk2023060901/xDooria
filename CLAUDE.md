# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**xDooria** 是一款使用 Go 语言构建的多人在线游戏服务端，采用微服务架构。系统支持每个房间/大厅 20-30 人同时在线进行实时交互。

## 常用开发命令

### 测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./pkg/logger/...
go test ./pkg/database/postgres/...

# 运行测试并显示详细输出
go test -v ./pkg/logger/...

# 运行测试并显示覆盖率
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # 查看覆盖率报告

# 运行单个测试函数
go test -v ./pkg/logger -run TestNew
```

### 构建

```bash
# 编译所有包（检查编译错误）
go build ./...

# 构建示例程序
go build -o bin/logger-example ./examples/logger/basic
go build -o bin/grpc-server ./examples/grpc/helloworld/server
go build -o bin/grpc-client ./examples/grpc/helloworld/client

# 运行示例
go run ./examples/logger/basic/main.go
go run ./examples/grpc/helloworld/server/main.go
```

### Protocol Buffers 代码生成

```bash
# 在 examples/grpc/proto/ 目录下生成 gRPC 代码
cd examples/grpc/proto
buf generate

# 注意: 实际的 proto 定义在独立仓库中
# - xdooria-proto-common: 公共类型定义
# - xdooria-proto-api: 客户端-服务端通信协议
# - xdooria-proto-internal: 服务端内部通信协议
```

### 依赖管理

```bash
# 添加依赖
go get github.com/some/package@latest

# 更新依赖
go get -u ./...

# 整理依赖
go mod tidy

# 查看依赖树
go mod graph
```

### 代码格式化和检查

```bash
# 格式化代码
go fmt ./...
gofmt -w .

# 运行 vet 检查
go vet ./...

# 使用 golangci-lint（如果安装）
golangci-lint run
```

## 技术栈

- **编程语言**: Go 1.24+
- **RPC 框架**: gRPC + Protocol Buffers
- **数据库**: PostgreSQL 14+ (主从复制，读写分离)
- **缓存**: Redis 7.0+ (会话、排行榜、实时状态)
- **消息队列**: Kafka 3.6+ (异步任务、聊天广播)
- **服务发现**: etcd 3.5+ (服务注册、配置中心)
- **日志**: Zap (结构化日志)
- **可观测性**: OpenTelemetry + Jaeger (分布式追踪), Prometheus (指标)

### 核心依赖库

```go
// 数据库
github.com/jackc/pgx/v5              // PostgreSQL 驱动 (支持连接池)
github.com/Masterminds/squirrel      // SQL 查询构建器

// gRPC 和 RPC
google.golang.org/grpc               // gRPC 框架
google.golang.org/protobuf           // Protocol Buffers
buf.build/go/protovalidate          // Proto 验证

// 基础设施
go.etcd.io/etcd/client/v3           // etcd 服务发现
github.com/redis/go-redis/v9         // Redis 客户端
github.com/segmentio/kafka-go        // Kafka 客户端

// Raft 共识算法（用于分布式协调）
github.com/hashicorp/raft            // Raft 实现
github.com/hashicorp/raft-boltdb/v2  // BoltDB 存储后端

// 配置与工具
github.com/spf13/viper               // 配置管理（支持多格式、环境变量）
go.uber.org/zap                      // 结构化日志
github.com/robfig/cron/v3            // 定时任务
github.com/golang-jwt/jwt/v5         // JWT 认证
github.com/google/uuid               // UUID 生成

// 可观测性
go.opentelemetry.io/otel            // OpenTelemetry SDK
github.com/prometheus/client_golang  // Prometheus 指标
github.com/getsentry/sentry-go      // 错误监控

// 工具库
github.com/panjf2000/ants/v2        // Goroutine 池
github.com/fsnotify/fsnotify        // 文件监控
github.com/cockroachdb/errors       // 增强错误处理
```

## 系统架构

### 微服务列表 (共 15 个服务)

#### 客户端直连服务 (2个)
- **Gateway** (gateway): 长连接 gRPC Stream，请求路由、负载均衡、限流
- **Auth** (auth): 账号认证、JWT Token 签发、第三方登录对接

#### 核心游戏服务 (6个)
- **Game** (game): 游戏核心逻辑 - 关卡、角色、玩家、背包、商店、任务
- **Hall** (hall): 大厅场景，支持 20-30 人同时在线
- **Room** (room): 游戏房间，支持 20-30 人同屏交互，实时战斗逻辑
- **Match** (match): 匹配系统，支持 ELO/MMR 算法
- **Team** (team): 组队系统
- **Rank** (rank): 排行榜系统

#### 社交服务 (4个)
- **Friend** (friend): 好友管理、拉黑、在线状态
- **Guild** (guild): 公会创建、成员管理、活动
- **Chat** (chat): 世界/私聊/附近/公会聊天频道
- **Mail** (mail): 系统邮件和玩家邮件，支持附件

#### 经济服务 (3个)
- **Trading** (trading): 玩家交易行，官方收取交易税
- **Home** (home): 家园系统 (开垦、种植、建造、道具出售)
- **Doll** (doll): 盲盒、玩偶融合、实体商品兑换与物流

### 服务状态特征

**有状态服务 (5个)** - 需要特殊扩展策略:
- **Gateway**: 维护客户端连接 (通过负载均衡扩展)
- **Game**: 内存缓存玩家数据 (一致性哈希 + Redis)
- **Hall**: 维护大厅状态 (一致性哈希 + Redis)
- **Room**: 维护房间状态 (一致性哈希 + Redis)
- **Chat**: 维护聊天连接 (Kafka 消息广播)

**无状态服务 (10个)** - 可直接水平扩展:
- Auth, Match, Team, Rank, Mail, Trading, Guild, Doll, Home, Friend

### 数据流

```
客户端 → Gateway → Auth (JWT) → 业务服务 → 基础设施 (PostgreSQL/Redis/Kafka)
                                          ↓
                                  外部平台 (账号/GM/支付/物流)
```

### 存储策略

**PostgreSQL** 存储内容:
- 账号和角色数据
- 玩家属性、背包、任务
- 社交数据 (好友、公会、邮件)
- 经济数据 (交易订单、流水、家园、玩偶)

**Redis** 存储内容:
- 会话: `session:{player_id}`, `online:{player_id}`
- 游戏状态: `hall:{hall_id}`, `room:{room_id}`, `team:{team_id}`
- 匹配队列: `match_queue:{mode}`
- 排行榜: `rank:{type}` (Sorted Set)
- 缓存: `cache:player:{player_id}`, `cache:guild:{guild_id}` 等

## Protocol Buffers 架构

### 独立的 Proto 仓库

- **xdooria-proto-common**: 公共类型定义
- **xdooria-proto-api**: 客户端与服务端通信协议
- **xdooria-proto-internal**: 服务端内部通信协议

依赖关系:
```
xdooria-proto-api → xdooria-proto-common
xdooria-proto-internal → xdooria-proto-common
xdooria (服务端) → xdooria-proto-api + xdooria-proto-internal
xdooria-client (客户端) → xdooria-proto-api
```

### 配置仓库

**xdooria-config**: 游戏策划配置
- Excel 原始表格
- 导出的 JSON 配置
- 导表工具脚本
- 服务端启动时加载，支持热更新

## 核心设计原则

1. **无状态优先**: 大部分服务设计为无状态，方便水平扩展
2. **Redis 共享状态**: 有状态服务使用 Redis 实现跨实例状态共享
3. **异步解耦**: 使用 Kafka 处理异步任务和事件驱动通信
4. **一致性哈希**: 有状态服务 (Game/Hall/Room) 按 ID 分配到不同实例
5. **服务发现**: 通过 etcd 实现服务注册和 gRPC 负载均衡

## 服务通信模式

### gRPC RPC 类型

- **Unary RPC (一元调用)**: 大部分服务调用 (请求-响应)
- **Server Streaming (服务端流)**: 推送实时数据 (房间状态更新)
- **Client Streaming (客户端流)**: 批量数据上传
- **Bidirectional Streaming (双向流)**: 实时聊天

### 负载均衡

```go
// 服务启动时注册到 etcd
endpoint.AddEndpoint(ctx, "game-instance-1", endpoints.Endpoint{
    Addr: "192.168.1.10:50051",
})

// 客户端通过 etcd resolver 自动发现服务
conn, err := grpc.Dial(
    "etcd:///xdooria/services/game",
    grpc.WithResolvers(resolver),
    grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`),
)
```

## 扩展策略

### 水平扩展

**无状态服务 (11个)**: 通过 Kubernetes 直接扩展
```bash
kubectl scale deployment game-service --replicas=5
```

**有状态服务**:
- **Game/Hall/Room**: 按玩家/大厅/房间 ID 一致性哈希，状态存 Redis
- **Chat**: 所有实例订阅 Kafka Topic；消息广播到所有连接的客户端

### 数据库扩展

- **PostgreSQL**: 读写分离、按玩家 ID 分库分表
- **Redis**: Redis Cluster (自动分片)、哨兵模式 (高可用)

## 高可用设计

- **容错机制**: 超时控制、重试机制 (幂等操作)、熔断器、降级策略
- **数据容错**: PostgreSQL 主从复制、Redis AOF+RDB、定期备份、Kafka 消息持久化和副本机制
- **监控**: Prometheus (QPS、延迟、错误率)、Zap 结构化日志、ELK Stack、Jaeger 分布式追踪

## 安全设计

- **认证鉴权**: JWT Token 有过期时间、支持 Refresh Token、每个接口验证权限
- **数据安全**: 敏感数据加密、参数化查询 (防 SQL 注入)、输入验证
- **防作弊**: 服务端验证、日志审计、异常检测

## 代码库架构

### 项目结构

```
xDooria/
├── docs/               # 架构和设计文档
│   ├── architecture.md    # 系统架构设计
│   ├── services.md        # 微服务说明
│   ├── tech-stack.md      # 技术栈详解
│   ├── infrastructure.md  # 基础设施设计
│   ├── grpc-design.md     # gRPC 设计指南
│   └── citus.md           # Citus 分布式数据库
│
├── pkg/                # 可复用的基础库（框架层）
│   ├── logger/            # 日志模块 (基于 Zap)
│   ├── config/            # 配置管理 (基于 Viper)
│   ├── database/          # 数据库客户端
│   │   ├── postgres/      # PostgreSQL 客户端（支持主从、连接池）
│   │   └── redis/         # Redis 客户端（支持集群、Pipeline）
│   ├── grpc/              # gRPC 工具
│   │   ├── server/        # gRPC Server 封装
│   │   ├── client/        # gRPC Client 封装
│   │   └── interceptor/   # 拦截器（日志、追踪、认证等）
│   ├── registry/          # 服务注册发现
│   │   ├── etcd/          # etcd 注册器
│   │   └── balancer/      # 负载均衡器
│   ├── mq/                # 消息队列（Kafka）
│   ├── raft/              # Raft 共识算法封装
│   ├── websocket/         # WebSocket 支持
│   ├── otel/              # OpenTelemetry 追踪
│   ├── jaeger/            # Jaeger 分布式追踪
│   ├── prometheus/        # Prometheus 指标采集
│   ├── sentry/            # Sentry 错误监控
│   ├── scheduler/         # 定时任务调度
│   ├── security/          # 安全工具（JWT、加密）
│   ├── pool/              # 协程池
│   ├── compress/          # 压缩工具
│   ├── checksum/          # 校验和工具
│   ├── serializer/        # 序列化工具
│   ├── notify/            # 通知系统
│   └── util/              # 通用工具
│
└── examples/           # 示例代码（展示如何使用 pkg 中的模块）
    ├── logger/            # 日志使用示例
    ├── grpc/              # gRPC 示例
    ├── postgres/          # PostgreSQL 示例
    ├── redis/             # Redis 示例
    ├── etcd/              # etcd 示例
    ├── kafka/             # Kafka 示例
    ├── raft/              # Raft 示例
    ├── observability/     # 可观测性示例
    └── ...
```

### pkg/ 包设计理念

**pkg/** 目录包含所有可复用的基础组件，每个包都是独立的、可测试的模块：

1. **logger**: 结构化日志，支持文件轮换、Context 字段提取、Hook 扩展
2. **config**: 配置管理，支持多种格式（YAML/JSON/TOML）、环境变量、热更新
3. **database/postgres**: PostgreSQL 客户端，支持主从模式、读写分离、事务、超时控制
4. **database/redis**: Redis 客户端，支持集群、Pipeline、分布式锁（Redlock）
5. **grpc/interceptor**: gRPC 拦截器，包括日志、追踪、认证、限流、熔断等
6. **registry/etcd**: 基于 etcd 的服务注册发现，支持健康检查、负载均衡
7. **raft**: Raft 共识算法封装，用于分布式状态管理
8. **otel/jaeger**: 分布式追踪集成
9. **prometheus**: 指标采集和监控

### 配置管理模式

该项目提供两种配置管理方式：

**1. Manager (通用接口)** - 用于框架内部和动态 key 访问：
```go
mgr := config.NewManager()
mgr.LoadFile("config.yaml")
port := mgr.GetInt("server.port")  // 动态访问
```

**2. Watcher[T] (强类型)** - 推荐用于业务代码：
```go
type AppConfig struct {
    Server ServerConfig `mapstructure:"server"`
    DB     DBConfig     `mapstructure:"database"`
}

watcher, _ := config.NewWatcher[AppConfig]("config.yaml")
cfg := watcher.Get()  // 类型安全
watcher.OnChange(func(new, old AppConfig) {
    // 热更新回调
})
```

### gRPC 拦截器链

拦截器按以下顺序执行（[pkg/grpc/interceptor/](pkg/grpc/interceptor/)）：

**服务端拦截器顺序**:
1. Recovery - 恢复 panic
2. Logging - 请求日志
3. Tracing - 分布式追踪
4. Metrics - 指标采集
5. Auth - 身份认证
6. Validation - 参数验证
7. RateLimiting - 限流
8. Timeout - 超时控制

**客户端拦截器顺序**:
1. Logging - 请求日志
2. Tracing - 分布式追踪
3. Metrics - 指标采集
4. Retry - 重试逻辑
5. Timeout - 超时控制

### 数据库访问模式

**PostgreSQL 主从模式** ([pkg/database/postgres/](pkg/database/postgres/)):
- `Client.Master()` - 获取主库连接（用于写操作）
- `Client.Slave()` - 获取从库连接（用于读操作，自动负载均衡）
- 支持事务、超时控制、连接池管理
- 使用 Squirrel 构建 SQL 查询（类型安全）

**Redis 客户端** ([pkg/database/redis/](pkg/database/redis/)):
- 支持单机、集群、主从模式
- Pipeline 批量操作
- Redlock 分布式锁
- 支持自定义序列化器（JSON/Msgpack）

### 日志记录最佳实践

使用结构化日志（[pkg/logger/](pkg/logger/)）：

```go
// 使用 Context 传递 trace_id 等字段
logger.InfoContext(ctx, "处理请求",
    zap.String("user_id", userID),
    zap.Int("room_id", roomID))

// 使用 WithFields 创建带公共字段的 logger
reqLogger := logger.WithFields(
    zap.String("request_id", reqID),
    zap.String("method", method))
reqLogger.Info("开始处理")
reqLogger.Info("处理完成")

// 错误日志
logger.Error("数据库查询失败",
    zap.Error(err),
    zap.String("query", query))
```

### 示例代码参考

[examples/](examples/) 目录包含各模块的完整使用示例：
- 每个示例都可以独立运行 (`go run ./examples/xxx/main.go`)
- 演示了从基础用法到高级特性的完整流程
- 包含详细的注释和 README 文档

### 外部仓库依赖

- **xdooria-proto-common**: 公共 proto 类型定义
- **xdooria-proto-api**: 客户端-服务端 API 协议
- **xdooria-proto-internal**: 服务端内部 RPC 协议
- **xdooria-config**: 游戏策划配置（Excel → JSON）

### 外部平台对接

- **账号平台**: 自有账号系统 + 第三方登录 (微信、QQ、Apple ID)
- **GM 平台**: 玩家数据管理、配置管理、运营工具、数据统计
- **支付平台**: 微信支付、支付宝
- **物流平台**: 玩偶兑换实体商品发货

## 重要实现模式

### 错误处理

使用 `github.com/cockroachdb/errors` 进行错误包装：
```go
if err != nil {
    return errors.Wrap(err, "failed to load config")
}
```

### 并发控制

使用 Goroutine 池避免过度创建协程：
```go
pool, _ := ants.NewPool(100)
defer pool.Release()

pool.Submit(func() {
    // 执行任务
})
```

### 服务注册与发现

```go
// 服务注册到 etcd
registry, _ := etcd.NewRegistry(etcdClient)
registry.Register(ctx, "game-service", "192.168.1.10:50051")

// 客户端通过 etcd resolver 发现服务
conn, _ := grpc.Dial(
    "etcd:///xdooria/services/game",
    grpc.WithResolvers(etcdResolver),
    grpc.WithDefaultServiceConfig(`{"loadBalancingPolicy":"round_robin"}`))
```
