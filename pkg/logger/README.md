# Logger - 日志模块

基于 `uber-go/zap` 的高性能结构化日志库，提供丰富的功能和灵活的配置选项。

## 特性

- ✅ **高性能**: 基于 zap，零分配、高吞吐量
- ✅ **结构化日志**: 支持丰富的字段类型
- ✅ **文件轮换**: 支持按大小和按时间轮换
- ✅ **Context 支持**: 自动从 context 提取字段
- ✅ **Hook 机制**: 灵活的日志钩子系统
- ✅ **多种格式**: JSON / Console 格式
- ✅ **多输出**: 同时输出到控制台和文件
- ✅ **采样**: 高频日志采样降低 I/O
- ✅ **全局 Logger**: 便捷的全局函数

## 快速开始

### 基础使用

```go
package main

import (
    "github.com/lk2023060901/xdooria/pkg/logger"
    "go.uber.org/zap"
)

func main() {
    // 1. 使用默认配置
    log, _ := logger.New(nil)
    log.Info("Hello, Logger!")

    // 2. 使用全局 Logger
    logger.Info("使用全局 logger")
    logger.Info("带字段的日志",
        zap.String("user", "zhangsan"),
        zap.Int("age", 25),
    )

    // 3. 自定义配置
    cfg := &logger.Config{
        Level:  logger.DebugLevel,
        Format: logger.JSONFormat,
    }
    log, _ = logger.New(cfg)
    log.Debug("调试信息")
}
```

## 配置

### Config 结构

```go
type Config struct {
    // 基础配置
    Level            Level              // 日志级别: debug, info, warn, error, panic, fatal
    Format           Format             // 输出格式: json, console
    TimeFormat       string             // 时间格式，默认: "2006-01-02 15:04:05"
    Development      bool               // 开发模式（彩色输出）

    // 输出配置
    EnableConsole    bool               // 启用控制台输出
    EnableFile       bool               // 启用文件输出
    OutputPath       string             // 文件路径

    // 轮换配置
    Rotation         RotationConfig     // 文件轮换配置

    // 高级配置
    GlobalFields     map[string]interface{}  // 全局字段
    EnableStacktrace bool                    // 启用堆栈跟踪
    StacktraceLevel  Level                   // 堆栈跟踪级别
    EnableSampling   bool                    // 启用采样
    SamplingInitial  int                     // 采样初始值
    SamplingThereafter int                   // 采样后续值
    EnableAsync      bool                    // 启用异步写入
    BufferSize       int                     // 缓冲区大小

    // Context 提取
    ContextExtractor ContextFieldExtractor   // Context 字段提取器
}
```

### 日志级别

```go
const (
    DebugLevel Level = "debug"  // 调试信息
    InfoLevel  Level = "info"   // 一般信息
    WarnLevel  Level = "warn"   // 警告信息
    ErrorLevel Level = "error"  // 错误信息
    PanicLevel Level = "panic"  // Panic
    FatalLevel Level = "fatal"  // Fatal (会退出程序)
)
```

### 默认配置

```go
func DefaultConfig() *Config {
    return &Config{
        Level:            InfoLevel,
        Format:           ConsoleFormat,
        EnableConsole:    true,
        EnableFile:       false,
        TimeFormat:       "2006-01-02 15:04:05",
        EnableStacktrace: true,
        StacktraceLevel:  ErrorLevel,
        Rotation: RotationConfig{
            Type:            RotationBySize,
            MaxSize:         100,      // 100MB
            MaxBackups:      5,
            MaxAge:          7,        // 7 天
            Compress:        true,
            RotationTime:    "24h",
            MaxAgeTime:      "168h",   // 7 天
            RotationPattern: ".%Y%m%d",
        },
        EnableSampling:     false,
        SamplingInitial:    100,
        SamplingThereafter: 100,
        BufferSize:         256 * 1024,  // 256KB
    }
}
```

## 核心功能

### 1. 创建 Logger

```go
// 方法 1: 使用默认配置
log, err := logger.New(nil)

// 方法 2: 自定义配置
cfg := &logger.Config{
    Level:  logger.DebugLevel,
    Format: logger.JSONFormat,
}
log, err := logger.New(cfg)

// 方法 3: 使用 Option 模式
log, err := logger.New(nil,
    logger.WithLevel(logger.DebugLevel),
    logger.WithDevelopment(true),
)
```

### 2. 结构化日志

```go
log.Info("用户登录",
    zap.String("user_id", "12345"),
    zap.String("ip", "192.168.1.100"),
    zap.Int("login_count", 42),
    zap.Bool("success", true),
    zap.Time("timestamp", time.Now()),
)
```

**支持的字段类型**：
- `zap.String(key, val)` - 字符串
- `zap.Int(key, val)` - 整数
- `zap.Int64(key, val)` - 64位整数
- `zap.Float64(key, val)` - 浮点数
- `zap.Bool(key, val)` - 布尔值
- `zap.Time(key, val)` - 时间
- `zap.Duration(key, val)` - 时长
- `zap.Error(err)` - 错误
- `zap.Any(key, val)` - 任意类型（会序列化）

### 3. WithFields 预设字段

```go
// 创建带预设字段的 logger
requestLogger := log.WithFields(
    "request_id", "req-abc123",
    "service", "api-server",
)

// 后续日志都会包含这些字段
requestLogger.Info("请求开始")
requestLogger.Info("请求完成")
```

### 4. Named Logger

```go
// 创建命名 logger
dbLogger := log.Named("database")
dbLogger.Info("数据库连接成功")

// 子命名
pgLogger := dbLogger.Named("postgres")
pgLogger.Info("PostgreSQL 初始化")
// 输出: {"logger":"database.postgres","msg":"PostgreSQL 初始化"}
```

### 5. Context 字段提取

```go
// 配置 Context 提取器
cfg := &logger.Config{
    ContextExtractor: func(ctx context.Context) []zap.Field {
        fields := make([]zap.Field, 0)
        if traceID, ok := ctx.Value("trace_id").(string); ok {
            fields = append(fields, zap.String("trace_id", traceID))
        }
        return fields
    },
}

log, _ := logger.New(cfg)

// 使用 *Context 方法
ctx := context.WithValue(context.Background(), "trace_id", "abc123")
log.InfoContext(ctx, "处理请求")
// 输出: {"trace_id":"abc123","msg":"处理请求"}
```

**Context 方法**：
- `DebugContext(ctx, msg, fields...)`
- `InfoContext(ctx, msg, fields...)`
- `WarnContext(ctx, msg, fields...)`
- `ErrorContext(ctx, msg, fields...)`
- `PanicContext(ctx, msg, fields...)`
- `FatalContext(ctx, msg, fields...)`

### 6. 文件轮换

#### 按大小轮换

```go
cfg := &logger.Config{
    EnableFile: true,
    OutputPath: "/var/log/app.log",
    Rotation: logger.RotationConfig{
        Type:       logger.RotationBySize,
        MaxSize:    100,      // 100MB
        MaxBackups: 5,        // 保留 5 个备份
        MaxAge:     7,        // 保留 7 天
        Compress:   true,     // 压缩旧文件
    },
}
```

#### 按时间轮换

```go
cfg := &logger.Config{
    EnableFile: true,
    OutputPath: "/var/log/app.log",
    Rotation: logger.RotationConfig{
        Type:            logger.RotationByTime,
        RotationTime:    "24h",        // 每天轮换
        MaxAgeTime:      "168h",       // 保留 7 天
        RotationPattern: ".%Y%m%d",    // 文件名: app.log.20250120
    },
}
```

### 7. Hook 钩子

#### 使用内置 Hook

```go
log, _ := logger.New(cfg,
    logger.WithHooks(
        logger.SensitiveDataHook([]string{"password", "token"}),
    ),
)

log.Info("登录", zap.String("password", "secret123"))
// 输出: {"msg":"登录","password":"***REDACTED***"}
```

#### 自定义 Hook

```go
type CustomHook struct{}

func (h *CustomHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
    // 处理日志
    // 返回 false 跳过该日志
    return true
}

log, _ := logger.New(cfg, logger.WithHooks(&CustomHook{}))
```

#### 使用 HookFunc

```go
hook := logger.HookFunc(func(entry zapcore.Entry, fields []zapcore.Field) bool {
    // 自定义处理逻辑
    return true
})

log, _ := logger.New(cfg, logger.WithHooks(hook))
```

### 8. 全局 Logger

```go
// 初始化全局 logger
logger.InitDefault(&logger.Config{
    Level:  logger.InfoLevel,
    Format: logger.JSONFormat,
})

// 或从环境变量初始化
logger.InitDefaultFromEnv()  // 读取 XDOORIA_LOG_* 环境变量

// 使用全局函数
logger.Info("信息")
logger.Debug("调试")
logger.Warn("警告")
logger.Error("错误")

// 全局 WithFields
userLogger := logger.WithFields("service", "user")
userLogger.Info("用户服务启动")

// 全局 Named
dbLogger := logger.Named("database")
dbLogger.Info("数据库连接")

// 设置全局字段
logger.SetGlobalFields("app", "myapp", "version", "1.0.0")
```

## 高级功能

### 1. 采样 (Sampling)

用于高频日志场景，减少 I/O 压力：

```go
cfg := &logger.Config{
    EnableSampling:     true,
    SamplingInitial:    100,   // 前 100 条全部记录
    SamplingThereafter: 100,   // 之后每 100 条记录 1 条
}
```

### 2. 堆栈跟踪

```go
cfg := &logger.Config{
    EnableStacktrace: true,
    StacktraceLevel:  logger.ErrorLevel,  // Error 及以上级别记录堆栈
}
```

### 3. 异步写入

```go
cfg := &logger.Config{
    EnableAsync: true,
    BufferSize:  256 * 1024,  // 256KB 缓冲区
}

log, _ := logger.New(cfg)
defer log.Sync()  // ⚠️ 异步模式下必须调用 Sync()
```

### 4. 全局字段

```go
cfg := &logger.Config{
    GlobalFields: map[string]interface{}{
        "app":     "myapp",
        "env":     "production",
        "version": "1.0.0",
        "region":  "cn-north-1",
    },
}
```

### 5. 开发模式

```go
cfg := &logger.Config{
    Development: true,  // 彩色输出、更详细的错误信息
}
```

## Option 模式

```go
log, _ := logger.New(nil,
    logger.WithLevel(logger.DebugLevel),
    logger.WithFormat(logger.JSONFormat),
    logger.WithDevelopment(true),
    logger.WithGlobalFields(map[string]interface{}{
        "app": "myapp",
    }),
    logger.WithHooks(customHook),
    logger.WithContextExtractor(extractorFunc),
)
```

**可用 Options**：
- `WithLevel(level Level)`
- `WithFormat(format Format)`
- `WithDevelopment(dev bool)`
- `WithGlobalFields(fields map[string]interface{})`
- `WithHooks(hooks ...Hook)`
- `WithContextExtractor(extractor ContextFieldExtractor)`

## 环境变量

使用 `InitDefaultFromEnv()` 时支持以下环境变量：

```bash
XDOORIA_LOG_LEVEL=debug          # 日志级别
XDOORIA_LOG_FORMAT=json          # 输出格式
XDOORIA_LOG_PATH=/var/log/app.log  # 文件路径
XDOORIA_LOG_CONSOLE=false        # 关闭控制台
XDOORIA_LOG_DEVELOPMENT=true     # 开发模式
```

## 最佳实践

### 1. 生产环境配置

```go
cfg := &logger.Config{
    Level:         logger.InfoLevel,
    Format:        logger.JSONFormat,
    EnableConsole: false,
    EnableFile:    true,
    OutputPath:    "/var/log/app.log",
    GlobalFields: map[string]interface{}{
        "app":     "myapp",
        "env":     "production",
        "version": os.Getenv("APP_VERSION"),
    },
    Rotation: logger.RotationConfig{
        Type:       logger.RotationBySize,
        MaxSize:    100,
        MaxBackups: 10,
        MaxAge:     30,
        Compress:   true,
    },
    EnableStacktrace: true,
    StacktraceLevel:  logger.ErrorLevel,
}
```

### 2. 开发环境配置

```go
cfg := &logger.Config{
    Level:         logger.DebugLevel,
    Format:        logger.ConsoleFormat,
    Development:   true,  // 彩色输出
    EnableConsole: true,
    EnableFile:    false,
}
```

### 3. 链路追踪集成

```go
cfg := &logger.Config{
    ContextExtractor: func(ctx context.Context) []zap.Field {
        fields := make([]zap.Field, 0, 3)

        // OpenTelemetry
        if span := trace.SpanFromContext(ctx); span.SpanContext().IsValid() {
            fields = append(fields,
                zap.String("trace_id", span.SpanContext().TraceID().String()),
                zap.String("span_id", span.SpanContext().SpanID().String()),
            )
        }

        // 自定义字段
        if reqID, ok := ctx.Value("request_id").(string); ok {
            fields = append(fields, zap.String("request_id", reqID))
        }

        return fields
    },
}
```

### 4. 错误处理

```go
if err != nil {
    log.Error("操作失败",
        zap.Error(err),
        zap.String("operation", "create_user"),
        zap.String("user_id", userID),
    )
    return err
}
```

### 5. 性能敏感场景

```go
// 避免
log.Info("处理请求", zap.Any("request", complexObject))  // 昂贵的序列化

// 推荐
log.Info("处理请求",
    zap.String("request_id", req.ID),
    zap.String("method", req.Method),
)

// 或使用采样
cfg := &logger.Config{
    EnableSampling:     true,
    SamplingInitial:    100,
    SamplingThereafter: 1000,
}
```

## API 参考

### Logger 方法

```go
type Logger struct {
    *zap.Logger
    // ...
}

// 日志方法
func (l *Logger) Debug(msg string, fields ...zap.Field)
func (l *Logger) Info(msg string, fields ...zap.Field)
func (l *Logger) Warn(msg string, fields ...zap.Field)
func (l *Logger) Error(msg string, fields ...zap.Field)
func (l *Logger) Panic(msg string, fields ...zap.Field)
func (l *Logger) Fatal(msg string, fields ...zap.Field)

// Context 方法
func (l *Logger) DebugContext(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) InfoContext(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) WarnContext(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) ErrorContext(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) PanicContext(ctx context.Context, msg string, fields ...zap.Field)
func (l *Logger) FatalContext(ctx context.Context, msg string, fields ...zap.Field)

// 辅助方法
func (l *Logger) Named(name string) *Logger
func (l *Logger) WithFields(fields ...interface{}) *Logger
func (l *Logger) Sync() error
```

### 全局函数

```go
// 初始化
func InitDefault(config *Config, opts ...Option) error
func InitDefaultFromEnv() error
func SetDefault(logger *Logger)
func Default() *Logger

// 全局日志
func Debug(msg string, fields ...zap.Field)
func Info(msg string, fields ...zap.Field)
func Warn(msg string, fields ...zap.Field)
func Error(msg string, fields ...zap.Field)
func Panic(msg string, fields ...zap.Field)
func Fatal(msg string, fields ...zap.Field)

// 全局 Context
func DebugContext(ctx context.Context, msg string, fields ...zap.Field)
func InfoContext(ctx context.Context, msg string, fields ...zap.Field)
func WarnContext(ctx context.Context, msg string, fields ...zap.Field)
func ErrorContext(ctx context.Context, msg string, fields ...zap.Field)
func PanicContext(ctx context.Context, msg string, fields ...zap.Field)
func FatalContext(ctx context.Context, msg string, fields ...zap.Field)

// 辅助
func Named(name string) *Logger
func WithFields(fields ...interface{}) *Logger
func SetGlobalFields(fields ...interface{})
func Sync() error
```

## 示例代码

完整示例请参见 [examples/logger/](../../examples/logger/) 目录：

- [basic](../../examples/logger/basic/) - 基础使用
- [context](../../examples/logger/context/) - Context 提取
- [structured](../../examples/logger/structured/) - 结构化日志
- [file-rotation](../../examples/logger/file-rotation/) - 文件轮换
- [hook](../../examples/logger/hook/) - Hook 钩子

## 测试

运行测试：

```bash
cd pkg/logger
go test -v -cover
```

当前测试覆盖率：**48.3%**

## 依赖

- `go.uber.org/zap` - 核心日志库
- `gopkg.in/natefinch/lumberjack.v2` - 按大小轮换
- `github.com/lestrrat-go/file-rotatelogs` - 按时间轮换

## License

MIT
