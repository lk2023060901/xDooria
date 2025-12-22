# 技术栈选型

## 概述

本文档详细说明 xDooria 游戏服务端的技术栈选型，包括编程语言、数据库、缓存、消息队列、服务发现、以及各种开源库的选择。

---

## 编程语言

### Go 1.24+

**选择理由：**
- 高性能、高并发
- 原生支持 gRPC
- 简洁的语法和强大的标准库
- 优秀的协程（goroutine）支持
- 适合构建微服务架构

---

## 数据库

### PostgreSQL 14+

**用途：** 主数据库，存储所有持久化数据

**存储内容：**
- 用户账号和角色信息
- 玩家属性、背包、道具
- 交易记录
- 任务进度
- 公会数据
- 家园数据
- 邮件数据
- 聊天记录（可选）

**Go 驱动库：**
```go
github.com/jackc/pgx/v5          // PostgreSQL 驱动（推荐）
github.com/jackc/pgx/v5/pgxpool  // 连接池
```

**SQL 构建器：**
```go
github.com/Masterminds/squirrel  // SQL 构建器，避免字符串拼接
```

**选择理由：**
- 功能强大，支持 JSON、数组等复杂类型
- ACID 事务支持
- 丰富的索引类型
- 优秀的性能和稳定性
- 开源且社区活跃

---

## 缓存

### Redis 7.0+

**用途：** 缓存层和实时数据存储

**使用场景：**
- 玩家会话和在线状态
- 排行榜（Redis Sorted Set）
- 房间和大厅状态共享
- 匹配队列
- 热点数据缓存（玩家信息、背包等）
- 好友列表和在线状态
- 公会信息缓存
- 家园状态缓存
- 分布式锁

**Go 客户端库：**
```go
github.com/redis/go-redis/v9     // Redis 客户端
```

**选择理由：**
- 高性能内存数据库
- 丰富的数据结构（String、Hash、List、Set、Sorted Set）
- 支持 Pub/Sub 消息模式
- 支持持久化（AOF/RDB）
- 支持集群和主从复制

---

## 消息队列

### Kafka

**用途：** 异步任务处理和消息广播

**使用场景：**
- 异步任务处理
- 服务间事件通知
- 聊天消息广播
- 邮件批量发送
- 排行榜异步计算
- 事件溯源和日志收集

**Go 客户端库：**
```go
github.com/segmentio/kafka-go    // Kafka 客户端（推荐）
github.com/IBM/sarama            // Kafka 客户端（官方推荐）
```

**选择理由：**
- 高吞吐量，低延迟
- 持久化存储，支持消息回溯
- 成熟的生态系统
- 支持事务和exactly-once语义
- 分区机制，天然支持消息顺序
- 无需依赖 ZooKeeper
- 支持消息持久化
- Go 原生实现，性能优秀

**备选方案：**
```go
github.com/nats-io/nats.go       // NATS（更轻量）
```

---

## 服务发现与配置中心

### etcd 3.5+

**用途：** 服务注册与发现、配置中心

**使用场景：**
- 服务注册与发现
- 动态配置管理
- 分布式锁
- 服务健康检查
- gRPC 负载均衡

**Go 客户端库：**
```go
go.etcd.io/etcd/client/v3                      // etcd 客户端
go.etcd.io/etcd/client/v3/naming/endpoints     // 服务注册
go.etcd.io/etcd/client/v3/naming/resolver      // gRPC resolver
```

**选择理由：**
- Kubernetes 生态原生支持
- 强一致性保证（Raft 协议）
- 与 gRPC 集成良好
- 高性能、低延迟
- 统一管理服务发现和配置

---

## gRPC 相关

### Protocol Buffers & gRPC

**用途：** 服务间通信协议

**核心库：**
```go
google.golang.org/grpc                         // gRPC 框架
google.golang.org/protobuf                     // Protocol Buffers
grpc-ecosystem/go-grpc-middleware              // gRPC 中间件
```

**中间件功能：**
- 日志记录
- 认证鉴权
- 请求追踪
- 错误处理
- 限流熔断

**选择理由：**
- 高性能二进制协议
- 强类型，自动生成代码
- 支持流式传输（聊天服务需要）
- 跨语言支持（方便客户端对接）
- 丰富的生态系统

---

## 配置管理

### Viper

**用途：** 配置文件管理

**库：**
```go
github.com/spf13/viper           // 配置管理
```

**功能：**
- 支持多种配置格式（YAML、JSON、TOML）
- 环境变量覆盖
- 配置热加载
- 默认值设置

---

## 日志

### Zap + Lumberjack

**用途：** 高性能日志库 + 日志轮转

**库：**
```go
go.uber.org/zap                           // 高性能日志库
gopkg.in/natefinch/lumberjack.v2          // 日志轮转
```

**选择理由：**
- **Zap**: 性能极高（结构化日志）、零内存分配、支持多种输出格式、支持日志级别和采样
- **Lumberjack**: 自动日志轮转、基于文件大小和时间归档、支持日志压缩

---

## 定时任务

### Cron

**用途：** 定时任务调度

**库：**
```go
github.com/robfig/cron/v3        // 定时任务
```

**使用场景：**
- 每日任务刷新
- 排行榜定期计算
- 数据清理
- 活动定时开启/关闭

---

## 认证与安全

### JWT

**用途：** Token 签发和验证

**库：**
```go
github.com/golang-jwt/jwt/v5     // JWT 认证
```

**功能：**
- Game Token 签发
- Token 验证和解析
- Token 刷新机制

---

## 工具库

### UUID

**用途：** 生成唯一标识符

**库：**
```go
github.com/google/uuid           // UUID 生成
```

**使用场景：**
- 玩家 ID
- 订单 ID
- 道具唯一 ID

---

### Decimal

**用途：** 精确货币计算

**库：**
```go
github.com/shopspring/decimal    // 高精度数值计算
```

**使用场景：**
- 钻石、金币计算
- 交易税计算
- 避免浮点数精度问题

**选择理由：**
- 精确的十进制运算
- 避免 float64 精度丢失
- 货币计算必备

---

## 完整依赖列表

```go
// go.mod
module github.com/lk2023060901/xdooria

go 1.24

require (
    // 数据库
    github.com/jackc/pgx/v5 v5.5.0
    github.com/Masterminds/squirrel v1.5.4

    // gRPC
    google.golang.org/grpc v1.59.0
    google.golang.org/protobuf v1.31.0
    github.com/grpc-ecosystem/go-grpc-middleware v1.4.0

    // 服务发现和配置
    go.etcd.io/etcd/client/v3 v3.5.10

    // 缓存
    github.com/redis/go-redis/v9 v9.3.0

    // 消息队列
    github.com/segmentio/kafka-go v0.4.45

    // 配置管理
    github.com/spf13/viper v1.17.0

    // 日志
    go.uber.org/zap v1.26.0
    gopkg.in/natefinch/lumberjack.v2 v2.2.1

    // 定时任务
    github.com/robfig/cron/v3 v3.0.1

    // JWT
    github.com/golang-jwt/jwt/v5 v5.2.0

    // 工具库
    github.com/google/uuid v1.4.0
    github.com/shopspring/decimal v1.3.1
)
```

---

## 基础设施总结

| 组件 | 选型 | 版本 | 用途 |
|------|------|------|------|
| 编程语言 | Go | 1.21+ | 服务端开发 |
| 数据库 | PostgreSQL | 14+ | 持久化存储 |
| 缓存 | Redis | 7.0+ | 缓存和实时数据 |
| 消息队列 | Kafka | 3.6+ | 异步任务和消息广播 |
| 服务发现 | etcd | 3.5+ | 服务注册和配置中心 |
| RPC 框架 | gRPC | latest | 服务间通信 |
| 日志 | Zap + Lumberjack | latest | 结构化日志 + 日志轮转 |

---

## 部署相关

### 容器化

```
Docker + Docker Compose (开发环境)
Kubernetes (生产环境)
```

### 监控

```
Prometheus (指标采集)
Grafana (可视化)
Jaeger (分布式追踪)
```

### CI/CD

```
GitHub Actions
GitLab CI
```

---

## 开发工具推荐

```bash
# Protocol Buffers 编译器
brew install protobuf

# buf (Proto 管理工具)
brew install buf

# 数据库迁移工具
github.com/golang-migrate/migrate

# API 文档生成
github.com/swaggo/swag
```
