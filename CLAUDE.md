# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**xDooria** 是一款使用 Go 语言构建的多人在线游戏服务端，采用微服务架构。系统支持每个房间/大厅 20-30 人同时在线进行实时交互。

## 技术栈

- **编程语言**: Go 1.24+
- **RPC 框架**: gRPC + Protocol Buffers
- **数据库**: PostgreSQL 14+ (持久化存储)
- **缓存**: Redis 7.0+ (会话、排行榜、实时状态)
- **消息队列**: Kafka 3.6+ (异步任务、聊天广播)
- **服务发现**: etcd 3.5+ (服务注册、配置中心)
- **日志**: Zap (结构化日志)

### 核心依赖库

```go
// 数据库
github.com/jackc/pgx/v5              // PostgreSQL 驱动
github.com/Masterminds/squirrel      // SQL 构建器

// gRPC
google.golang.org/grpc
google.golang.org/protobuf
github.com/grpc-ecosystem/go-grpc-middleware

// 基础设施
go.etcd.io/etcd/client/v3           // 服务发现
github.com/redis/go-redis/v9         // Redis 客户端
github.com/segmentio/kafka-go        // Kafka 客户端

// 配置与工具
github.com/spf13/viper               // 配置管理
go.uber.org/zap                      // 日志
github.com/robfig/cron/v3            // 定时任务
github.com/golang-jwt/jwt/v5         // JWT 认证
github.com/google/uuid               // UUID 生成
github.com/shopspring/decimal        // 精确货币计算
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

## 开发工作流

### 项目结构
```
xDooria/
├── docs/               # 架构和设计文档
│   ├── architecture.md # 系统架构
│   ├── services.md     # 服务说明
│   └── tech-stack.md   # 技术栈
├── README.md
└── (服务代码目录将随开发进度逐步添加)
```

### 外部平台对接

- **账号平台**: 自有账号系统 + 第三方登录 (微信、QQ、Apple ID)
- **GM 平台**: 玩家数据管理、配置管理、运营工具、数据统计
- **支付平台**: 微信支付、支付宝
- **物流平台**: 玩偶兑换实体商品发货
