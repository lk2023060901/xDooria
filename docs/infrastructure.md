# 基础设施开发计划

## 概述

本文档详细说明 xDooria 游戏服务端基础设施层的开发优先级、封装方案和实施计划。基础设施层为所有业务服务提供统一的技术底座，包括配置管理、日志、数据库访问、服务通信、消息队列等核心组件。

---

## 开发优先级

### 优先级划分原则

1. **最小可用系统** - 先支持最基础的功能
2. **服务依赖关系** - 被依赖多的组件优先
3. **开发调试便利** - 影响开发效率的优先

---

### P0 级（第一阶段 - 核心基础设施）

#### 1. 配置管理 (Config)
**优先级：⭐⭐⭐⭐⭐**

**开发理由：**
- 所有服务启动都需要读取配置
- 是其他组件初始化的前提条件
- 支持多环境配置（dev/test/prod）

**技术选型：**
- `github.com/spf13/viper`

**核心功能：**
- 支持 YAML 格式（服务器配置）
- 支持 JSON 格式（游戏策划配置）
- 环境变量覆盖
- 配置热加载
- 默认值设置

**配置格式说明：**
- **YAML** - 用于服务端配置（数据库、Redis、服务端口等）
- **JSON** - 用于游戏策划配置（从 xdooria-config 仓库加载）

---

#### 2. 日志系统 (Logger)
**优先级：⭐⭐⭐⭐⭐**

**开发理由：**
- 调试和问题排查必需
- 所有服务都需要记录日志
- 结构化日志便于日志分析

**技术选型：**
- `go.uber.org/zap` - 高性能日志库
- `gopkg.in/natefinch/lumberjack.v2` - 日志轮转

**核心功能：**
- 结构化日志输出（JSON/Console 格式）
- 多级别日志（Debug/Info/Warn/Error/Fatal）
- 日志轮转和归档（基于文件大小和时间）
- 带上下文的日志记录
- 自定义字段支持

---

#### 3. 数据库连接池 (PostgreSQL Client)
**优先级：⭐⭐⭐⭐⭐**

**开发理由：**
- 数据持久化基础
- 几乎所有服务都需要访问数据库
- 需要统一的查询构建和事务管理

**技术选型：**
- `github.com/jackc/pgx/v5` - PostgreSQL 驱动
- `github.com/Masterminds/squirrel` - SQL 构建器

**核心功能：**
- 连接池管理
- SQL 构建器封装
- 事务支持
- 健康检查
- 自动重连

---

#### 4. Redis 客户端 (Redis Client)
**优先级：⭐⭐⭐⭐⭐**

**开发理由：**
- 缓存和状态共享基础
- 会话管理、排行榜、匹配队列等都依赖 Redis
- 有状态服务需要 Redis 共享状态

**技术选型：**
- `github.com/redis/go-redis/v9`

**核心功能：**
- 单机和集群支持
- 会话管理封装
- 排行榜管理（Sorted Set）
- 分布式锁
- 发布订阅

---

### P1 级（第二阶段 - 服务间通信）

#### 5. gRPC 服务框架 (gRPC Server/Client)
**优先级：⭐⭐⭐⭐**

**开发理由：**
- 服务间通信基础
- 客户端与服务端通信协议
- 需要统一的服务启动和中间件管理

**技术选型：**
- `google.golang.org/grpc`
- `github.com/grpc-ecosystem/go-grpc-middleware`

**核心功能：**
- gRPC 服务端封装
- gRPC 客户端连接管理
- 中间件支持（日志、认证、限流、追踪）
- 优雅关闭
- 自动重连

---

#### 6. 服务发现 (Service Discovery - etcd)
**优先级：⭐⭐⭐⭐**

**开发理由：**
- 微服务架构必需
- 服务注册和自动发现
- 动态配置管理

**技术选型：**
- `go.etcd.io/etcd/client/v3`

**核心功能：**
- 服务注册和注销
- 服务发现和监听
- 健康检查（心跳保活）
- 与 gRPC 集成（resolver）
- 配置中心

---

#### 7. 认证中间件 (JWT Auth)
**优先级：⭐⭐⭐⭐**

**开发理由：**
- 安全认证必需
- 所有业务接口都需要验证 Token
- 统一的认证逻辑

**技术选型：**
- `github.com/golang-jwt/jwt/v5`

**核心功能：**
- JWT Token 签发
- Token 验证和解析
- Token 刷新机制
- gRPC 认证中间件

---

### P2 级（第三阶段 - 高级功能）

#### 8. 消息队列 (Kafka Client)
**优先级：⭐⭐⭐**

**开发理由：**
- 异步任务处理
- 服务间事件通知
- 聊天消息广播

**技术选型：**
- `github.com/segmentio/kafka-go`

**核心功能：**
- 生产者封装
- 消费者封装
- 消息重试机制
- 批量发送

---

#### 9. 分布式锁 (Distributed Lock)
**优先级：⭐⭐⭐**

**开发理由：**
- 并发控制需要
- 防止重复操作
- 资源竞争保护

**实现方式：**
- 基于 Redis 实现

**核心功能：**
- 获取锁和释放锁
- 锁超时自动释放
- 可重入锁（可选）
- 带回调的锁操作

---

#### 10. 定时任务 (Cron Scheduler)
**优先级：⭐⭐⭐**

**开发理由：**
- 定时任务管理
- 每日任务刷新
- 排行榜定期计算

**技术选型：**
- `github.com/robfig/cron/v3`

**核心功能：**
- Cron 表达式支持
- 任务注册和管理
- 任务执行监控
- 错误处理

---

### P3 级（第四阶段 - 可观测性）

#### 11. 监控指标 (Metrics - Prometheus)
**优先级：⭐⭐**

**开发理由：**
- 性能监控
- 问题预警
- 容量规划

**技术选型：**
- `github.com/prometheus/client_golang`

**核心功能：**
- 指标采集（QPS、延迟、错误率）
- HTTP 指标接口
- 自定义指标

---

#### 12. 分布式追踪 (Tracing - Jaeger)
**优先级：⭐⭐**

**开发理由：**
- 问题排查
- 性能分析
- 调用链追踪

**技术选型：**
- `go.opentelemetry.io/otel`
- Jaeger

**核心功能：**
- 链路追踪
- Span 创建和传递
- 与 gRPC 集成

---

## 项目结构设计

```
xDooria/
├── pkg/                      # 公共基础库
│   ├── config/              # 配置管理
│   │   ├── config.go        # 配置结构定义
│   │   └── loader.go        # 配置加载器
│   ├── logger/              # 日志系统
│   │   ├── logger.go        # 日志接口
│   │   └── zap.go           # Zap 实现
│   ├── database/            # 数据库封装
│   │   ├── postgres/        # PostgreSQL 客户端
│   │   │   ├── client.go    # 数据库客户端
│   │   │   ├── tx.go        # 事务管理
│   │   │   └── builder.go   # SQL 构建器
│   │   └── redis/           # Redis 客户端
│   │       ├── client.go    # Redis 客户端
│   │       ├── session.go   # 会话管理
│   │       ├── rank.go      # 排行榜
│   │       └── lock.go      # 分布式锁
│   ├── grpcx/               # gRPC 封装
│   │   ├── server/          # gRPC 服务端
│   │   │   ├── server.go    # 服务端封装
│   │   │   └── options.go   # 配置选项
│   │   ├── client/          # gRPC 客户端
│   │   │   ├── manager.go   # 连接管理器
│   │   │   └── pool.go      # 连接池
│   │   └── middleware/      # gRPC 中间件
│   │       ├── auth.go      # 认证中间件
│   │       ├── logging.go   # 日志中间件
│   │       ├── recovery.go  # 恢复中间件
│   │       ├── ratelimit.go # 限流中间件
│   │       └── tracing.go   # 追踪中间件
│   ├── registry/            # 服务发现
│   │   ├── registry.go      # 接口定义
│   │   └── etcd/            # etcd 实现
│   │       ├── registry.go  # 服务注册
│   │       ├── resolver.go  # gRPC resolver
│   │       └── watcher.go   # 服务监听
│   ├── mq/                  # 消息队列
│   │   ├── message.go       # 消息定义
│   │   └── kafka/           # Kafka 封装
│   │       ├── producer.go  # 生产者
│   │       └── consumer.go  # 消费者
│   ├── auth/                # 认证
│   │   └── jwt/             # JWT 实现
│   │       ├── manager.go   # JWT 管理器
│   │       └── claims.go    # Claims 定义
│   ├── lock/                # 分布式锁
│   │   ├── lock.go          # 锁接口
│   │   └── redis_lock.go    # Redis 实现
│   ├── scheduler/           # 定时任务
│   │   ├── scheduler.go     # 调度器
│   │   └── job.go           # 任务定义
│   ├── metrics/             # 监控指标
│   │   ├── metrics.go       # 指标接口
│   │   └── prometheus.go    # Prometheus 实现
│   ├── tracing/             # 分布式追踪
│   │   ├── tracer.go        # 追踪器
│   │   └── jaeger.go        # Jaeger 实现
│   └── utils/               # 工具函数
│       ├── uuid/            # UUID 生成
│       ├── decimal/         # 货币计算
│       ├── crypto/          # 加密工具
│       └── hash/            # 一致性哈希
├── internal/                # 内部服务实现
│   ├── gateway/             # 网关服务
│   ├── auth/                # 认证服务
│   ├── game/                # 游戏服务
│   └── ...                  # 其他服务
├── configs/                 # 配置文件
│   ├── dev/                 # 开发环境
│   ├── test/                # 测试环境
│   └── prod/                # 生产环境
├── scripts/                 # 脚本工具
├── docs/                    # 文档
└── go.mod
```

---

## 详细封装设计

### 1. 配置管理 (pkg/config)

#### 配置结构

```go
// pkg/config/config.go
package config

import (
    "github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
    Etcd     EtcdConfig     `mapstructure:"etcd"`
    Kafka    KafkaConfig    `mapstructure:"kafka"`
    Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig 服务配置
type ServerConfig struct {
    Name string `mapstructure:"name"`     // 服务名称
    Port int    `mapstructure:"port"`     // 服务端口
    Env  string `mapstructure:"env"`      // 环境 (dev/test/prod)
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
    Host     string `mapstructure:"host"`
    Port     int    `mapstructure:"port"`
    User     string `mapstructure:"user"`
    Password string `mapstructure:"password"`
    Database string `mapstructure:"database"`
    MaxConns int    `mapstructure:"max_conns"`
    MinConns int    `mapstructure:"min_conns"`
    MaxIdleTime int `mapstructure:"max_idle_time"` // 秒
}

// RedisConfig Redis 配置
type RedisConfig struct {
    Addrs    []string `mapstructure:"addrs"`    // 支持集群
    Password string   `mapstructure:"password"`
    DB       int      `mapstructure:"db"`
    PoolSize int      `mapstructure:"pool_size"`
}

// EtcdConfig etcd 配置
type EtcdConfig struct {
    Endpoints   []string `mapstructure:"endpoints"`
    DialTimeout int      `mapstructure:"dial_timeout"` // 秒
}

// KafkaConfig Kafka 配置
type KafkaConfig struct {
    Brokers        []string `mapstructure:"brokers"`
    LookupdAddr    string `mapstructure:"lookupd_addr"`
    MaxInFlight    int    `mapstructure:"max_in_flight"`
}

// LogConfig 日志配置
type LogConfig struct {
    Level      string `mapstructure:"level"`       // debug/info/warn/error
    Format     string `mapstructure:"format"`      // json/console
    OutputPath string `mapstructure:"output_path"` // 输出路径
}
```

#### 配置加载

```go
// pkg/config/loader.go
package config

import (
    "fmt"
    "github.com/spf13/viper"
)

// Load 加载服务器配置文件（YAML 格式）
func Load(configPath string) (*Config, error) {
    v := viper.New()
    v.SetConfigFile(configPath)
    v.SetConfigType("yaml")

    // 读取配置文件
    if err := v.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("failed to read config: %w", err)
    }

    // 解析配置
    var cfg Config
    if err := v.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("failed to unmarshal config: %w", err)
    }

    return &cfg, nil
}

// LoadGameConfig 加载游戏配置文件（JSON 格式）
// 用于加载从 xdooria-config 仓库导出的策划配置
func LoadGameConfig(configPath string, target interface{}) error {
    v := viper.New()
    v.SetConfigFile(configPath)
    v.SetConfigType("json")

    // 读取配置文件
    if err := v.ReadInConfig(); err != nil {
        return fmt.Errorf("failed to read game config: %w", err)
    }

    // 解析配置
    if err := v.Unmarshal(target); err != nil {
        return fmt.Errorf("failed to unmarshal game config: %w", err)
    }

    return nil
}

// Watch 监听配置变化（用于热更新）
func Watch(configPath string, callback func(*Config)) error {
    v := viper.New()
    v.SetConfigFile(configPath)

    v.WatchConfig()
    v.OnConfigChange(func(e fsnotify.Event) {
        var cfg Config
        if err := v.Unmarshal(&cfg); err == nil {
            callback(&cfg)
        }
    })

    return nil
}
```

#### 配置文件示例

```yaml
# configs/dev/game.yaml
server:
  name: game-service
  port: 50051
  env: dev

database:
  host: localhost
  port: 5432
  user: xdooria
  password: xdooria123
  database: xdooria_game
  max_conns: 20
  min_conns: 5
  max_idle_time: 300

redis:
  addrs:
    - localhost:6379
  password: ""
  db: 0
  pool_size: 10

etcd:
  endpoints:
    - localhost:2379
  dial_timeout: 5

kafka:
  brokers:
    - localhost:9092
  lookupd_addr: localhost:4161
  max_in_flight: 200

log:
  level: debug
  format: console
  output_path: logs/game.log
```

---

### 2. 日志系统 (pkg/logger)

```go
// pkg/logger/logger.go
package logger

import (
    "context"
    "os"

    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "gopkg.in/natefinch/lumberjack.v2"
)

var globalLogger *zap.Logger

// Config 日志配置
type Config struct {
    Level      string // debug/info/warn/error
    Format     string // json/console
    OutputPath string // 日志文件路径
    MaxSize    int    // 单个文件最大 MB
    MaxBackups int    // 保留旧文件数量
    MaxAge     int    // 保留天数
    Compress   bool   // 是否压缩
}

// Init 初始化日志系统
func Init(serviceName string, cfg Config) error {
    // 日志级别
    level := zapcore.InfoLevel
    switch cfg.Level {
    case "debug":
        level = zapcore.DebugLevel
    case "warn":
        level = zapcore.WarnLevel
    case "error":
        level = zapcore.ErrorLevel
    }

    // 编码配置
    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "time",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    // 编码器
    var encoder zapcore.Encoder
    if cfg.Format == "json" {
        encoder = zapcore.NewJSONEncoder(encoderConfig)
    } else {
        encoder = zapcore.NewConsoleEncoder(encoderConfig)
    }

    // 日志轮转
    writer := &lumberjack.Logger{
        Filename:   cfg.OutputPath,
        MaxSize:    cfg.MaxSize,    // MB
        MaxBackups: cfg.MaxBackups,
        MaxAge:     cfg.MaxAge,     // days
        Compress:   cfg.Compress,
    }

    // 同时输出到文件和控制台
    core := zapcore.NewTee(
        zapcore.NewCore(encoder, zapcore.AddSync(writer), level),
        zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level),
    )

    // 创建 logger
    globalLogger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))

    // 添加服务名称字段
    globalLogger = globalLogger.With(zap.String("service", serviceName))

    return nil
}

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
    globalLogger.Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
    globalLogger.Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
    globalLogger.Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
    globalLogger.Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
    globalLogger.Fatal(msg, fields...)
}

// WithContext 带上下文的日志
func WithContext(ctx context.Context) *Logger {
    // 从 context 中提取 trace_id、player_id 等信息
    fields := extractFieldsFromContext(ctx)
    return &Logger{
        logger: globalLogger.With(fields...),
    }
}

// Logger 带上下文的日志记录器
type Logger struct {
    logger *zap.Logger
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
    l.logger.Debug(msg, fields...)
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
    l.logger.Info(msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
    l.logger.Warn(msg, fields...)
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
    l.logger.Error(msg, fields...)
}

// extractFieldsFromContext 从 context 中提取字段
func extractFieldsFromContext(ctx context.Context) []zap.Field {
    var fields []zap.Field

    // 提取 trace_id
    if traceID := ctx.Value("trace_id"); traceID != nil {
        fields = append(fields, zap.String("trace_id", traceID.(string)))
    }

    // 提取 player_id
    if playerID := ctx.Value("player_id"); playerID != nil {
        fields = append(fields, zap.String("player_id", playerID.(string)))
    }

    return fields
}

// Sync 同步日志缓冲区
func Sync() error {
    return globalLogger.Sync()
}
```

---

### 3. PostgreSQL 封装 (pkg/database/postgres)

```go
// pkg/database/postgres/client.go
package postgres

import (
    "context"
    "fmt"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/Masterminds/squirrel"
)

// Client PostgreSQL 客户端
type Client struct {
    pool *pgxpool.Pool
    sb   squirrel.StatementBuilderType
}

// New 创建 PostgreSQL 客户端
func New(cfg DatabaseConfig) (*Client, error) {
    // 构建连接字符串
    dsn := fmt.Sprintf(
        "host=%s port=%d user=%s password=%s dbname=%s "+
        "pool_max_conns=%d pool_min_conns=%d pool_max_conn_idle_time=%ds",
        cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.Database,
        cfg.MaxConns, cfg.MinConns, cfg.MaxIdleTime,
    )

    // 创建连接池
    poolConfig, err := pgxpool.ParseConfig(dsn)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
    if err != nil {
        return nil, fmt.Errorf("failed to create pool: %w", err)
    }

    // 测试连接
    if err := pool.Ping(context.Background()); err != nil {
        return nil, fmt.Errorf("failed to ping database: %w", err)
    }

    return &Client{
        pool: pool,
        sb:   squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
    }, nil
}

// Select 创建 SELECT 查询
func (c *Client) Select(columns ...string) squirrel.SelectBuilder {
    return c.sb.Select(columns...)
}

// Insert 创建 INSERT 查询
func (c *Client) Insert(table string) squirrel.InsertBuilder {
    return c.sb.Insert(table)
}

// Update 创建 UPDATE 查询
func (c *Client) Update(table string) squirrel.UpdateBuilder {
    return c.sb.Update(table)
}

// Delete 创建 DELETE 查询
func (c *Client) Delete(table string) squirrel.DeleteBuilder {
    return c.sb.Delete(table)
}

// BeginTx 开启事务
func (c *Client) BeginTx(ctx context.Context) (*Tx, error) {
    tx, err := c.pool.Begin(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to begin transaction: %w", err)
    }

    return &Tx{
        tx: tx,
        sb: c.sb,
    }, nil
}

// QueryRow 查询单行
func (c *Client) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
    return c.pool.QueryRow(ctx, sql, args...)
}

// Query 查询多行
func (c *Client) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
    return c.pool.Query(ctx, sql, args...)
}

// Exec 执行 SQL
func (c *Client) Exec(ctx context.Context, sql string, args ...interface{}) (CommandTag, error) {
    return c.pool.Exec(ctx, sql, args...)
}

// Ping 健康检查
func (c *Client) Ping(ctx context.Context) error {
    return c.pool.Ping(ctx)
}

// Close 关闭连接池
func (c *Client) Close() {
    c.pool.Close()
}

// Stats 连接池统计
func (c *Client) Stats() *pgxpool.Stat {
    return c.pool.Stat()
}
```

```go
// pkg/database/postgres/tx.go
package postgres

import (
    "context"
    "fmt"

    "github.com/jackc/pgx/v5"
    "github.com/Masterminds/squirrel"
)

// Tx 事务
type Tx struct {
    tx pgx.Tx
    sb squirrel.StatementBuilderType
}

// Select 创建 SELECT 查询
func (t *Tx) Select(columns ...string) squirrel.SelectBuilder {
    return t.sb.Select(columns...)
}

// Insert 创建 INSERT 查询
func (t *Tx) Insert(table string) squirrel.InsertBuilder {
    return t.sb.Insert(table)
}

// Update 创建 UPDATE 查询
func (t *Tx) Update(table string) squirrel.UpdateBuilder {
    return t.sb.Update(table)
}

// Delete 创建 DELETE 查询
func (t *Tx) Delete(table string) squirrel.DeleteBuilder {
    return t.sb.Delete(table)
}

// Commit 提交事务
func (t *Tx) Commit(ctx context.Context) error {
    if err := t.tx.Commit(ctx); err != nil {
        return fmt.Errorf("failed to commit transaction: %w", err)
    }
    return nil
}

// Rollback 回滚事务
func (t *Tx) Rollback(ctx context.Context) error {
    if err := t.tx.Rollback(ctx); err != nil {
        return fmt.Errorf("failed to rollback transaction: %w", err)
    }
    return nil
}

// QueryRow 查询单行
func (t *Tx) QueryRow(ctx context.Context, sql string, args ...interface{}) Row {
    return t.tx.QueryRow(ctx, sql, args...)
}

// Query 查询多行
func (t *Tx) Query(ctx context.Context, sql string, args ...interface{}) (Rows, error) {
    return t.tx.Query(ctx, sql, args...)
}

// Exec 执行 SQL
func (t *Tx) Exec(ctx context.Context, sql string, args ...interface{}) (CommandTag, error) {
    return t.tx.Exec(ctx, sql, args...)
}
```

---

### 4. Redis 封装 (pkg/database/redis)

```go
// pkg/database/redis/client.go
package redis

import (
    "context"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// Client Redis 客户端
type Client struct {
    client redis.UniversalClient
}

// New 创建 Redis 客户端（支持单机和集群）
func New(cfg RedisConfig) (*Client, error) {
    var client redis.UniversalClient

    if len(cfg.Addrs) == 1 {
        // 单机模式
        client = redis.NewClient(&redis.Options{
            Addr:     cfg.Addrs[0],
            Password: cfg.Password,
            DB:       cfg.DB,
            PoolSize: cfg.PoolSize,
        })
    } else {
        // 集群模式
        client = redis.NewClusterClient(&redis.ClusterOptions{
            Addrs:    cfg.Addrs,
            Password: cfg.Password,
            PoolSize: cfg.PoolSize,
        })
    }

    // 测试连接
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := client.Ping(ctx).Err(); err != nil {
        return nil, fmt.Errorf("failed to ping redis: %w", err)
    }

    return &Client{
        client: client,
    }, nil
}

// GetClient 获取原生客户端
func (c *Client) GetClient() redis.UniversalClient {
    return c.client
}

// Session 会话管理器
func (c *Client) Session() *SessionManager {
    return &SessionManager{client: c.client}
}

// Leaderboard 排行榜管理器
func (c *Client) Leaderboard() *LeaderboardManager {
    return &LeaderboardManager{client: c.client}
}

// Lock 分布式锁
func (c *Client) Lock(key string, ttl time.Duration) *Lock {
    return &Lock{
        client: c.client,
        key:    key,
        ttl:    ttl,
    }
}

// Close 关闭连接
func (c *Client) Close() error {
    return c.client.Close()
}

// Ping 健康检查
func (c *Client) Ping(ctx context.Context) error {
    return c.client.Ping(ctx).Err()
}
```

```go
// pkg/database/redis/session.go
package redis

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/redis/go-redis/v9"
)

// SessionManager 会话管理器
type SessionManager struct {
    client redis.UniversalClient
}

// SessionData 会话数据
type SessionData struct {
    PlayerID  string    `json:"player_id"`
    Token     string    `json:"token"`
    Gateway   string    `json:"gateway"`
    LoginTime time.Time `json:"login_time"`
}

// Set 设置会话
func (sm *SessionManager) Set(ctx context.Context, playerID string, data *SessionData, ttl time.Duration) error {
    key := fmt.Sprintf("session:%s", playerID)

    dataBytes, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("failed to marshal session data: %w", err)
    }

    if err := sm.client.Set(ctx, key, dataBytes, ttl).Err(); err != nil {
        return fmt.Errorf("failed to set session: %w", err)
    }

    return nil
}

// Get 获取会话
func (sm *SessionManager) Get(ctx context.Context, playerID string) (*SessionData, error) {
    key := fmt.Sprintf("session:%s", playerID)

    dataBytes, err := sm.client.Get(ctx, key).Bytes()
    if err != nil {
        if err == redis.Nil {
            return nil, fmt.Errorf("session not found")
        }
        return nil, fmt.Errorf("failed to get session: %w", err)
    }

    var data SessionData
    if err := json.Unmarshal(dataBytes, &data); err != nil {
        return nil, fmt.Errorf("failed to unmarshal session data: %w", err)
    }

    return &data, nil
}

// Delete 删除会话
func (sm *SessionManager) Delete(ctx context.Context, playerID string) error {
    key := fmt.Sprintf("session:%s", playerID)
    return sm.client.Del(ctx, key).Err()
}

// SetOnline 设置在线状态
func (sm *SessionManager) SetOnline(ctx context.Context, playerID string, ttl time.Duration) error {
    key := fmt.Sprintf("online:%s", playerID)
    return sm.client.Set(ctx, key, time.Now().Unix(), ttl).Err()
}

// IsOnline 检查在线状态
func (sm *SessionManager) IsOnline(ctx context.Context, playerID string) (bool, error) {
    key := fmt.Sprintf("online:%s", playerID)
    exists, err := sm.client.Exists(ctx, key).Result()
    if err != nil {
        return false, err
    }
    return exists > 0, nil
}
```

```go
// pkg/database/redis/rank.go
package redis

import (
    "context"
    "fmt"

    "github.com/redis/go-redis/v9"
)

// LeaderboardManager 排行榜管理器
type LeaderboardManager struct {
    client redis.UniversalClient
}

// RankItem 排行榜项
type RankItem struct {
    Member string
    Score  float64
    Rank   int64
}

// Add 添加或更新排行榜项
func (lm *LeaderboardManager) Add(ctx context.Context, key string, score float64, member string) error {
    return lm.client.ZAdd(ctx, key, redis.Z{
        Score:  score,
        Member: member,
    }).Err()
}

// Remove 移除排行榜项
func (lm *LeaderboardManager) Remove(ctx context.Context, key string, member string) error {
    return lm.client.ZRem(ctx, key, member).Err()
}

// Top 获取前 N 名
func (lm *LeaderboardManager) Top(ctx context.Context, key string, limit int64) ([]RankItem, error) {
    // 倒序获取（分数从高到低）
    results, err := lm.client.ZRevRangeWithScores(ctx, key, 0, limit-1).Result()
    if err != nil {
        return nil, fmt.Errorf("failed to get top ranks: %w", err)
    }

    items := make([]RankItem, 0, len(results))
    for i, z := range results {
        items = append(items, RankItem{
            Member: z.Member.(string),
            Score:  z.Score,
            Rank:   int64(i + 1),
        })
    }

    return items, nil
}

// GetRank 获取成员排名
func (lm *LeaderboardManager) GetRank(ctx context.Context, key string, member string) (int64, error) {
    // ZRevRank 返回的是从 0 开始的排名
    rank, err := lm.client.ZRevRank(ctx, key, member).Result()
    if err != nil {
        if err == redis.Nil {
            return -1, fmt.Errorf("member not found")
        }
        return -1, fmt.Errorf("failed to get rank: %w", err)
    }

    return rank + 1, nil // 转换为从 1 开始
}

// GetScore 获取成员分数
func (lm *LeaderboardManager) GetScore(ctx context.Context, key string, member string) (float64, error) {
    return lm.client.ZScore(ctx, key, member).Result()
}

// Count 获取排行榜总数
func (lm *LeaderboardManager) Count(ctx context.Context, key string) (int64, error) {
    return lm.client.ZCard(ctx, key).Result()
}
```

---

### 5. gRPC 服务端封装 (pkg/grpcx/server)

```go
// pkg/grpcx/server/server.go
package server

import (
    "context"
    "fmt"
    "net"
    "time"

    "google.golang.org/grpc"
    "google.golang.org/grpc/health"
    "google.golang.org/grpc/health/grpc_health_v1"

    "github.com/lk2023060901/xdooria/pkg/logger"
    "github.com/lk2023060901/xdooria/pkg/registry"
)

// Server gRPC 服务端
type Server struct {
    grpcServer *grpc.Server
    registry   registry.Registry
    opts       Options
}

// Options 服务端选项
type Options struct {
    ServiceName string                         // 服务名称
    Port        int                            // 服务端口
    Registry    registry.Registry              // 服务注册中心
    Middlewares []grpc.UnaryServerInterceptor  // 中间件
}

// New 创建 gRPC 服务端
func New(opts Options) (*Server, error) {
    // 创建 gRPC 服务器
    grpcServer := grpc.NewServer(
        grpc.ChainUnaryInterceptor(opts.Middlewares...),
    )

    // 注册健康检查服务
    healthServer := health.NewServer()
    grpc_health_v1.RegisterHealthServer(grpcServer, healthServer)

    return &Server{
        grpcServer: grpcServer,
        registry:   opts.Registry,
        opts:       opts,
    }, nil
}

// RegisterService 注册服务
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
    s.grpcServer.RegisterService(desc, impl)
}

// GetServer 获取原生 gRPC 服务器
func (s *Server) GetServer() *grpc.Server {
    return s.grpcServer
}

// Start 启动服务
func (s *Server) Start() error {
    // 监听端口
    lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.opts.Port))
    if err != nil {
        return fmt.Errorf("failed to listen: %w", err)
    }

    // 注册服务到注册中心
    if s.registry != nil {
        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        if err := s.registry.Register(ctx, registry.Service{
            Name: s.opts.ServiceName,
            Addr: fmt.Sprintf("localhost:%d", s.opts.Port),
        }); err != nil {
            return fmt.Errorf("failed to register service: %w", err)
        }

        logger.Info("service registered",
            zap.String("service", s.opts.ServiceName),
            zap.Int("port", s.opts.Port),
        )
    }

    // 启动服务器
    logger.Info("grpc server starting",
        zap.String("service", s.opts.ServiceName),
        zap.Int("port", s.opts.Port),
    )

    if err := s.grpcServer.Serve(lis); err != nil {
        return fmt.Errorf("failed to serve: %w", err)
    }

    return nil
}

// Shutdown 优雅关闭
func (s *Server) Shutdown(ctx context.Context) error {
    // 注销服务
    if s.registry != nil {
        if err := s.registry.Deregister(ctx, s.opts.ServiceName); err != nil {
            logger.Error("failed to deregister service", zap.Error(err))
        }
    }

    // 优雅关闭 gRPC 服务器
    stopped := make(chan struct{})
    go func() {
        s.grpcServer.GracefulStop()
        close(stopped)
    }()

    select {
    case <-ctx.Done():
        // 超时强制关闭
        s.grpcServer.Stop()
        return fmt.Errorf("shutdown timeout")
    case <-stopped:
        logger.Info("grpc server stopped gracefully")
        return nil
    }
}
```

---

### 6. gRPC 中间件 (pkg/grpcx/middleware)

```go
// pkg/grpcx/middleware/logging.go
package middleware

import (
    "context"
    "time"

    "google.golang.org/grpc"
    "go.uber.org/zap"

    "github.com/lk2023060901/xdooria/pkg/logger"
)

// LoggingInterceptor 日志中间件
func LoggingInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        start := time.Now()

        // 调用处理器
        resp, err := handler(ctx, req)

        // 记录日志
        duration := time.Since(start)

        fields := []zap.Field{
            zap.String("method", info.FullMethod),
            zap.Duration("duration", duration),
        }

        if err != nil {
            fields = append(fields, zap.Error(err))
            logger.Error("grpc call failed", fields...)
        } else {
            logger.Info("grpc call success", fields...)
        }

        return resp, err
    }
}
```

```go
// pkg/grpcx/middleware/recovery.go
package middleware

import (
    "context"
    "fmt"
    "runtime/debug"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"

    "github.com/lk2023060901/xdooria/pkg/logger"
)

// RecoveryInterceptor 恢复中间件（捕获 panic）
func RecoveryInterceptor() grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (resp interface{}, err error) {
        defer func() {
            if r := recover(); r != nil {
                // 记录 panic 信息
                logger.Error("grpc panic recovered",
                    zap.String("method", info.FullMethod),
                    zap.Any("panic", r),
                    zap.String("stack", string(debug.Stack())),
                )

                // 返回错误
                err = status.Errorf(codes.Internal, "internal server error: %v", r)
            }
        }()

        return handler(ctx, req)
    }
}
```

```go
// pkg/grpcx/middleware/auth.go
package middleware

import (
    "context"
    "strings"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/metadata"
    "google.golang.org/grpc/status"

    "github.com/lk2023060901/xdooria/pkg/auth/jwt"
)

// AuthInterceptor JWT 认证中间件
func AuthInterceptor(jwtManager *jwt.Manager, skipMethods []string) grpc.UnaryServerInterceptor {
    return func(
        ctx context.Context,
        req interface{},
        info *grpc.UnaryServerInfo,
        handler grpc.UnaryHandler,
    ) (interface{}, error) {
        // 检查是否跳过认证
        for _, method := range skipMethods {
            if info.FullMethod == method {
                return handler(ctx, req)
            }
        }

        // 获取 metadata
        md, ok := metadata.FromIncomingContext(ctx)
        if !ok {
            return nil, status.Errorf(codes.Unauthenticated, "missing metadata")
        }

        // 获取 authorization header
        values := md["authorization"]
        if len(values) == 0 {
            return nil, status.Errorf(codes.Unauthenticated, "missing authorization header")
        }

        // 提取 token
        token := strings.TrimPrefix(values[0], "Bearer ")

        // 验证 token
        claims, err := jwtManager.Verify(token)
        if err != nil {
            return nil, status.Errorf(codes.Unauthenticated, "invalid token: %v", err)
        }

        // 将 player_id 注入到 context
        ctx = context.WithValue(ctx, "player_id", claims.PlayerID)

        return handler(ctx, req)
    }
}
```

---

### 7. 服务发现 (pkg/registry/etcd)

```go
// pkg/registry/registry.go
package registry

import "context"

// Service 服务信息
type Service struct {
    Name string
    Addr string
    Metadata map[string]string
}

// Registry 服务注册中心接口
type Registry interface {
    // Register 注册服务
    Register(ctx context.Context, service Service) error

    // Deregister 注销服务
    Deregister(ctx context.Context, serviceName string) error

    // Discover 发现服务
    Discover(ctx context.Context, serviceName string) ([]Service, error)

    // Watch 监听服务变化
    Watch(ctx context.Context, serviceName string) <-chan []Service
}
```

```go
// pkg/registry/etcd/registry.go
package etcd

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    clientv3 "go.etcd.io/etcd/client/v3"

    "github.com/lk2023060901/xdooria/pkg/registry"
)

// Registry etcd 服务注册中心
type Registry struct {
    client  *clientv3.Client
    leaseID clientv3.LeaseID
}

// New 创建 etcd 服务注册中心
func New(endpoints []string, dialTimeout time.Duration) (*Registry, error) {
    client, err := clientv3.New(clientv3.Config{
        Endpoints:   endpoints,
        DialTimeout: dialTimeout,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to create etcd client: %w", err)
    }

    return &Registry{
        client: client,
    }, nil
}

// Register 注册服务
func (r *Registry) Register(ctx context.Context, service registry.Service) error {
    // 创建租约（TTL 10 秒）
    lease, err := r.client.Grant(ctx, 10)
    if err != nil {
        return fmt.Errorf("failed to grant lease: %w", err)
    }
    r.leaseID = lease.ID

    // 序列化服务信息
    data, err := json.Marshal(service)
    if err != nil {
        return fmt.Errorf("failed to marshal service: %w", err)
    }

    // 注册服务
    key := fmt.Sprintf("/xdooria/services/%s/%s", service.Name, service.Addr)
    _, err = r.client.Put(ctx, key, string(data), clientv3.WithLease(lease.ID))
    if err != nil {
        return fmt.Errorf("failed to register service: %w", err)
    }

    // 保持租约（心跳）
    go r.keepAlive()

    return nil
}

// Deregister 注销服务
func (r *Registry) Deregister(ctx context.Context, serviceName string) error {
    // 撤销租约
    if r.leaseID != 0 {
        _, err := r.client.Revoke(ctx, r.leaseID)
        if err != nil {
            return fmt.Errorf("failed to revoke lease: %w", err)
        }
    }

    return nil
}

// Discover 发现服务
func (r *Registry) Discover(ctx context.Context, serviceName string) ([]registry.Service, error) {
    key := fmt.Sprintf("/xdooria/services/%s/", serviceName)

    resp, err := r.client.Get(ctx, key, clientv3.WithPrefix())
    if err != nil {
        return nil, fmt.Errorf("failed to discover service: %w", err)
    }

    services := make([]registry.Service, 0, len(resp.Kvs))
    for _, kv := range resp.Kvs {
        var service registry.Service
        if err := json.Unmarshal(kv.Value, &service); err != nil {
            continue
        }
        services = append(services, service)
    }

    return services, nil
}

// Watch 监听服务变化
func (r *Registry) Watch(ctx context.Context, serviceName string) <-chan []registry.Service {
    key := fmt.Sprintf("/xdooria/services/%s/", serviceName)

    ch := make(chan []registry.Service)

    go func() {
        defer close(ch)

        watchChan := r.client.Watch(ctx, key, clientv3.WithPrefix())
        for wresp := range watchChan {
            for range wresp.Events {
                // 服务变化，重新获取服务列表
                services, err := r.Discover(ctx, serviceName)
                if err == nil {
                    ch <- services
                }
            }
        }
    }()

    return ch
}

// keepAlive 保持租约
func (r *Registry) keepAlive() {
    ch, err := r.client.KeepAlive(context.Background(), r.leaseID)
    if err != nil {
        return
    }

    for range ch {
        // 心跳成功
    }
}

// Close 关闭连接
func (r *Registry) Close() error {
    return r.client.Close()
}
```

---

### 8. JWT 认证 (pkg/auth/jwt)

```go
// pkg/auth/jwt/manager.go
package jwt

import (
    "fmt"
    "time"

    "github.com/golang-jwt/jwt/v5"
)

// Manager JWT 管理器
type Manager struct {
    secretKey     []byte
    tokenDuration time.Duration
}

// Claims JWT Claims
type Claims struct {
    PlayerID string `json:"player_id"`
    jwt.RegisteredClaims
}

// NewManager 创建 JWT 管理器
func NewManager(secretKey string, tokenDuration time.Duration) *Manager {
    return &Manager{
        secretKey:     []byte(secretKey),
        tokenDuration: tokenDuration,
    }
}

// Generate 生成 Token
func (m *Manager) Generate(playerID string) (string, error) {
    claims := Claims{
        PlayerID: playerID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.tokenDuration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

    tokenString, err := token.SignedString(m.secretKey)
    if err != nil {
        return "", fmt.Errorf("failed to sign token: %w", err)
    }

    return tokenString, nil
}

// Verify 验证 Token
func (m *Manager) Verify(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(
        tokenString,
        &Claims{},
        func(token *jwt.Token) (interface{}, error) {
            if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
                return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
            }
            return m.secretKey, nil
        },
    )

    if err != nil {
        return nil, fmt.Errorf("failed to parse token: %w", err)
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, fmt.Errorf("invalid token")
    }

    return claims, nil
}

// Refresh 刷新 Token
func (m *Manager) Refresh(tokenString string) (string, error) {
    claims, err := m.Verify(tokenString)
    if err != nil {
        return "", err
    }

    // 生成新 Token
    return m.Generate(claims.PlayerID)
}
```

---

### 9. Kafka 消息队列 (pkg/mq/kafka)

```go
// pkg/mq/kafka/producer.go
package kafka

import (
    "context"
    "fmt"

    "github.com/segmentio/kafka-go"
)

// Producer Kafka 生产者
type Producer struct {
    writer *kafka.Writer
}

// NewProducer 创建生产者
func NewProducer(brokers []string) (*Producer, error) {
    writer := &kafka.Writer{
        Addr:     kafka.TCP(brokers...),
        Balancer: &kafka.LeastBytes{},
    }

    return &Producer{
        writer: writer,
    }, nil
}

// Publish 发布消息
func (p *Producer) Publish(ctx context.Context, topic string, message []byte) error {
    return p.writer.WriteMessages(ctx, kafka.Message{
        Topic: topic,
        Value: message,
    })
}

// PublishWithKey 发布带 Key 的消息（保证同一 Key 的消息到同一分区）
func (p *Producer) PublishWithKey(ctx context.Context, topic string, key, message []byte) error {
    return p.writer.WriteMessages(ctx, kafka.Message{
        Topic: topic,
        Key:   key,
        Value: message,
    })
}

// Close 关闭生产者
func (p *Producer) Close() error {
    return p.writer.Close()
}
```

```go
// pkg/mq/kafka/consumer.go
package kafka

import (
    "context"
    "fmt"

    "github.com/segmentio/kafka-go"

    "github.com/lk2023060901/xdooria/pkg/logger"
    "go.uber.org/zap"
)

// Consumer Kafka 消费者
type Consumer struct {
    reader *kafka.Reader
}

// Handler 消息处理器
type Handler func(ctx context.Context, message kafka.Message) error

// NewConsumer 创建消费者
func NewConsumer(brokers []string, groupID, topic string, handler Handler) (*Consumer, error) {
    reader := kafka.NewReader(kafka.ReaderConfig{
        Brokers:  brokers,
        GroupID:  groupID,
        Topic:    topic,
        MinBytes: 10e3, // 10KB
        MaxBytes: 10e6, // 10MB
    })

    consumer := &Consumer{
        reader: reader,
    }

    // 启动消费协程
    go consumer.consume(handler)

    return consumer, nil
}

// consume 消费消息
func (c *Consumer) consume(handler Handler) {
    ctx := context.Background()
    for {
        message, err := c.reader.FetchMessage(ctx)
        if err != nil {
            logger.Error("failed to fetch message", zap.Error(err))
            continue
        }

        // 调用业务处理器
        if err := handler(ctx, message); err != nil {
            logger.Error("failed to handle message",
                zap.String("topic", message.Topic),
                zap.Int("partition", message.Partition),
                zap.Error(err),
            )
            continue
        }

        // 提交 offset
        if err := c.reader.CommitMessages(ctx, message); err != nil {
            logger.Error("failed to commit message", zap.Error(err))
        }
    }
}

// Close 关闭消费者
func (c *Consumer) Close() error {
    return c.reader.Close()
}

// Stop 停止消费者
func (c *Consumer) Stop() {
    c.consumer.Stop()
}
```

---

## 开发时间线

### Week 1-2: P0 级基础设施

**目标：** 实现核心基础设施，支持基本的服务启动和数据访问

| 组件 | 预计时间 | 关键任务 |
|------|---------|---------|
| 配置管理 | 2 天 | 配置结构定义、加载器、多环境支持 |
| 日志系统 | 2 天 | Zap 封装、结构化日志、上下文日志 |
| PostgreSQL | 3 天 | 连接池、SQL 构建器、事务管理 |
| Redis | 3 天 | 客户端封装、会话管理、排行榜、分布式锁 |

**里程碑：** 可以启动一个简单的服务，连接数据库和 Redis，记录日志

---

### Week 3-4: P1 级服务间通信

**目标：** 实现微服务架构的核心通信能力

| 组件 | 预计时间 | 关键任务 |
|------|---------|---------|
| gRPC 服务端 | 3 天 | 服务封装、中间件支持、优雅关闭 |
| gRPC 客户端 | 2 天 | 连接管理、负载均衡、自动重连 |
| gRPC 中间件 | 2 天 | 日志、认证、恢复、限流 |
| 服务发现 | 3 天 | etcd 封装、服务注册、服务发现、健康检查 |
| JWT 认证 | 2 天 | Token 签发、验证、刷新 |

**里程碑：** 多个服务可以通过 gRPC 通信，支持服务发现和认证

---

### Week 5: P2 级高级功能

**目标：** 实现异步任务和定时任务能力

| 组件 | 预计时间 | 关键任务 |
|------|---------|---------|
| Kafka 消息队列 | 3 天 | 生产者、消费者、消息重试 |
| 定时任务 | 2 天 | Cron 封装、任务管理 |

**里程碑：** 支持异步任务处理和定时任务调度

---

### Week 6+: P3 级可观测性

**目标：** 实现监控和追踪能力

| 组件 | 预计时间 | 关键任务 |
|------|---------|---------|
| Prometheus 监控 | 3 天 | 指标采集、HTTP 接口 |
| Jaeger 追踪 | 3 天 | 链路追踪、Span 管理 |

**里程碑：** 完整的可观测性体系

---

## 使用示例

### 完整的服务启动示例

```go
// internal/game/main.go
package main

import (
    "context"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go.uber.org/zap"
    "google.golang.org/grpc"

    "github.com/lk2023060901/xdooria/pkg/config"
    "github.com/lk2023060901/xdooria/pkg/logger"
    "github.com/lk2023060901/xdooria/pkg/database/postgres"
    "github.com/lk2023060901/xdooria/pkg/database/redis"
    "github.com/lk2023060901/xdooria/pkg/grpcx/server"
    "github.com/lk2023060901/xdooria/pkg/grpcx/middleware"
    "github.com/lk2023060901/xdooria/pkg/registry/etcd"
    "github.com/lk2023060901/xdooria/pkg/auth/jwt"

    pb "github.com/lk2023060901/xdooria-proto-api/gen/go/game"
)

func main() {
    // 1. 加载配置
    cfg, err := config.Load("configs/dev/game.yaml")
    if err != nil {
        panic(err)
    }

    // 2. 初始化日志
    if err := logger.Init(cfg.Server.Name, cfg.Log); err != nil {
        panic(err)
    }
    defer logger.Sync()

    logger.Info("starting game service", zap.String("env", cfg.Server.Env))

    // 3. 连接数据库
    db, err := postgres.New(cfg.Database)
    if err != nil {
        logger.Fatal("failed to connect database", zap.Error(err))
    }
    defer db.Close()
    logger.Info("database connected")

    // 4. 连接 Redis
    rdb, err := redis.New(cfg.Redis)
    if err != nil {
        logger.Fatal("failed to connect redis", zap.Error(err))
    }
    defer rdb.Close()
    logger.Info("redis connected")

    // 5. 创建服务发现
    registry, err := etcd.New(cfg.Etcd.Endpoints, time.Duration(cfg.Etcd.DialTimeout)*time.Second)
    if err != nil {
        logger.Fatal("failed to create registry", zap.Error(err))
    }
    defer registry.Close()
    logger.Info("registry created")

    // 6. 创建 JWT 管理器
    jwtManager := jwt.NewManager("your-secret-key", 24*time.Hour)

    // 7. 创建 gRPC 服务
    grpcServer, err := server.New(server.Options{
        ServiceName: cfg.Server.Name,
        Port:        cfg.Server.Port,
        Registry:    registry,
        Middlewares: []grpc.UnaryServerInterceptor{
            middleware.LoggingInterceptor(),
            middleware.RecoveryInterceptor(),
            middleware.AuthInterceptor(jwtManager, []string{
                "/game.GameService/HealthCheck",
            }),
        },
    })
    if err != nil {
        logger.Fatal("failed to create grpc server", zap.Error(err))
    }

    // 8. 注册业务服务
    gameService := NewGameService(db, rdb)
    pb.RegisterGameServiceServer(grpcServer.GetServer(), gameService)
    logger.Info("game service registered")

    // 9. 启动服务
    go func() {
        if err := grpcServer.Start(); err != nil {
            logger.Fatal("failed to start server", zap.Error(err))
        }
    }()

    // 10. 等待退出信号
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("shutting down server...")

    // 11. 优雅关闭
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := grpcServer.Shutdown(ctx); err != nil {
        logger.Error("server shutdown error", zap.Error(err))
    }

    logger.Info("server stopped")
}

// NewGameService 创建游戏服务
func NewGameService(db *postgres.Client, rdb *redis.Client) *GameService {
    return &GameService{
        db:  db,
        rdb: rdb,
    }
}

// GameService 游戏服务实现
type GameService struct {
    pb.UnimplementedGameServiceServer
    db  *postgres.Client
    rdb *redis.Client
}

// GetPlayer 获取玩家信息
func (s *GameService) GetPlayer(ctx context.Context, req *pb.GetPlayerRequest) (*pb.GetPlayerResponse, error) {
    // 从 context 中获取 player_id
    playerID := ctx.Value("player_id").(string)

    logger.WithContext(ctx).Info("get player",
        zap.String("player_id", playerID),
    )

    // 查询数据库
    sql, args, _ := s.db.Select("id", "name", "level", "exp").
        From("players").
        Where("id = ?", playerID).
        ToSql()

    var player pb.Player
    err := s.db.QueryRow(ctx, sql, args...).Scan(
        &player.Id,
        &player.Name,
        &player.Level,
        &player.Exp,
    )
    if err != nil {
        return nil, err
    }

    return &pb.GetPlayerResponse{
        Player: &player,
    }, nil
}
```

---

## 测试策略

### 单元测试

每个基础组件都需要编写单元测试：

```go
// pkg/database/redis/session_test.go
package redis_test

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/lk2023060901/xdooria/pkg/database/redis"
)

func TestSessionManager(t *testing.T) {
    // 创建测试客户端
    client, err := redis.New(redis.RedisConfig{
        Addrs:    []string{"localhost:6379"},
        Password: "",
        DB:       0,
        PoolSize: 10,
    })
    assert.NoError(t, err)
    defer client.Close()

    sm := client.Session()
    ctx := context.Background()

    // 测试设置会话
    data := &redis.SessionData{
        PlayerID:  "player123",
        Token:     "token123",
        Gateway:   "gateway1",
        LoginTime: time.Now(),
    }

    err = sm.Set(ctx, "player123", data, 10*time.Minute)
    assert.NoError(t, err)

    // 测试获取会话
    result, err := sm.Get(ctx, "player123")
    assert.NoError(t, err)
    assert.Equal(t, "player123", result.PlayerID)
    assert.Equal(t, "token123", result.Token)

    // 测试删除会话
    err = sm.Delete(ctx, "player123")
    assert.NoError(t, err)

    // 验证已删除
    _, err = sm.Get(ctx, "player123")
    assert.Error(t, err)
}
```

### 集成测试

使用 Docker Compose 启动测试环境：

```yaml
# docker-compose.test.yaml
version: '3.8'

services:
  postgres:
    image: postgres:14
    environment:
      POSTGRES_USER: xdooria
      POSTGRES_PASSWORD: xdooria123
      POSTGRES_DB: xdooria_test
    ports:
      - "5432:5432"

  redis:
    image: redis:7
    ports:
      - "6379:6379"

  etcd:
    image: quay.io/coreos/etcd:v3.5.10
    environment:
      ETCD_ADVERTISE_CLIENT_URLS: http://0.0.0.0:2379
      ETCD_LISTEN_CLIENT_URLS: http://0.0.0.0:2379
    ports:
      - "2379:2379"

  kafka:
    image: confluentinc/cp-kafka:latest
    environment:
      KAFKA_ADVERTISED_LISTENERS: PLAINTEXT://kafka:9092
    ports:
      - "4150:4150"
      - "4151:4151"
```

---

## 文档和示例

每个基础组件需要提供：

1. **README.md** - 组件说明和快速开始
2. **example/** - 使用示例代码
3. **godoc** - 完整的 API 文档
4. **测试覆盖率** - 目标 80% 以上

---

## 总结

### 开发原则

1. **简洁易用** - 提供简单的 API，隐藏复杂实现
2. **统一接口** - 同类组件提供一致的接口
3. **可配置** - 通过配置文件控制行为
4. **可测试** - 支持单元测试和集成测试
5. **文档完善** - 每个包提供清晰的文档和示例

### 质量保证

- 代码审查（Code Review）
- 单元测试覆盖率 > 80%
- 集成测试覆盖核心流程
- 性能测试和基准测试
- 文档和示例完整

### 持续改进

- 根据业务需求持续优化
- 收集使用反馈
- 定期重构和优化
- 保持技术栈更新

这套基础设施将为 xDooria 的 15 个微服务提供坚实的技术底座，大大提高开发效率和代码质量。
