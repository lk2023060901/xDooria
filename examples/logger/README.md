# Logger 日志模块示例

本目录包含 Logger 模块的各种使用示例，每个子目录演示一个特定功能。

## 目录结构

```
examples/logger/
├── basic/            # 示例 1: 基础使用
├── context/          # 示例 2: Context 字段提取
├── structured/       # 示例 3: 结构化日志
├── file-rotation/    # 示例 4: 文件轮换
└── hook/             # 示例 5: Hook 钩子
```

## 运行示例

### 1. 基础使用 (basic)

演示 Logger 的基本功能和使用方法。

```bash
cd basic
go run main.go
```

**学习要点**：
- `New()` - 创建 Logger 实例
- 默认配置 vs 自定义配置
- 全局 Logger 使用
- 带字段的日志
- `WithFields()` - 添加公共字段
- `Named()` - 创建命名 Logger

---

### 2. Context 字段提取 (context)

演示如何从 `context.Context` 中自动提取字段。

```bash
cd context
go run main.go
```

**学习要点**：
- `ContextExtractor` - 自定义字段提取器
- 通过 Config 设置提取器
- 通过 Option 设置提取器
- `*Context()` 方法 - DebugContext, InfoContext 等
- 全局 Logger 的 Context 支持
- 混合使用 Context 字段和手动字段

**应用场景**：
- 链路追踪 (trace_id, span_id)
- 请求追踪 (request_id)
- 用户上下文 (user_id, tenant_id)
- 分布式系统中的上下文传递

---

### 3. 结构化日志 (structured)

演示结构化日志的强大功能。

```bash
cd structured
go run main.go
```

**学习要点**：
- JSON 格式 vs Console 格式
- 多种字段类型：String, Int, Bool, Time, Duration, Error 等
- `WithFields()` - 预设公共字段
- `Named()` - 命名 Logger 和层级结构
- 全局字段 (`GlobalFields`)
- 复杂对象序列化 (`zap.Any()`)

**字段类型**：
- 基本类型: `zap.String()`, `zap.Int()`, `zap.Bool()`, `zap.Float64()`
- 时间类型: `zap.Time()`, `zap.Duration()`
- 数组类型: `zap.Strings()`, `zap.Ints()`
- 错误类型: `zap.Error()`
- 通用类型: `zap.Any()`

---

### 4. 文件轮换 (file-rotation)

演示日志文件的轮换策略。

```bash
cd file-rotation
go run main.go

# 查看生成的日志文件
ls -lh ./logs/
```

**学习要点**：
- 按大小轮换 (`RotationBySize`)
  - `MaxSize` - 最大文件大小
  - `MaxBackups` - 保留备份数
  - `MaxAge` - 保留天数
  - `Compress` - 压缩旧文件
- 按时间轮换 (`RotationByTime`)
  - `RotationTime` - 轮换间隔 (1h, 24h 等)
  - `MaxAgeTime` - 保留时长
  - `RotationPattern` - 文件名模式
- 同时输出到控制台和文件

**配置示例**：
```go
Rotation: logger.RotationConfig{
    Type:       logger.RotationBySize,
    MaxSize:    100,  // 100MB
    MaxBackups: 5,    // 保留 5 个备份
    MaxAge:     7,    // 保留 7 天
    Compress:   true, // 压缩
}
```

---

### 5. Hook 钩子 (hook)

演示日志钩子的使用和自定义 Hook。

```bash
cd hook
go run main.go
```

**学习要点**：
- 内置 Hook: `SensitiveDataHook()` - 敏感数据脱敏
- 自定义 Hook 实现 `Hook` 接口
- `HookFunc` - 函数式 Hook
- 多个 Hook 组合使用
- Hook 的执行顺序

**自定义 Hook 示例**：
```go
type CustomHook struct{}

func (h *CustomHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
    // 处理日志
    // 返回 false 跳过该日志
    return true
}
```

**应用场景**：
- 敏感数据脱敏
- 日志过滤
- 日志监控统计
- 日志级别动态调整
- 日志告警

---

## 完整示例对比

| 示例 | 核心功能 | 难度 | 推荐顺序 |
|------|---------|------|----------|
| basic | 基础使用和配置 | ⭐ | 1 |
| structured | 结构化日志字段 | ⭐⭐ | 2 |
| context | Context 提取 | ⭐⭐⭐ | 3 |
| file-rotation | 文件轮换 | ⭐⭐ | 4 |
| hook | 钩子扩展 | ⭐⭐⭐ | 5 |

## 快速测试所有示例

```bash
# 在 examples/logger 目录下运行
for dir in basic structured context file-rotation hook; do
    echo "=== Running $dir ==="
    (cd $dir && go run main.go)
    echo ""
done
```

## 常见问题

### Q: 如何选择日志格式？

**A**:
- **JSONFormat**: 生产环境推荐，便于日志采集和分析
- **ConsoleFormat**: 开发环境推荐，可读性好
- 开发模式 (`Development: true`) 可启用彩色输出

### Q: 如何选择轮换策略？

**A**:
- **RotationBySize**: 适合日志量不固定的场景
- **RotationByTime**: 适合需要按时间归档的场景
- 可以根据需求组合 `MaxSize` 和 `MaxAge`

### Q: GlobalFields 和 WithFields 有什么区别？

**A**:
- **GlobalFields**: 在创建 Logger 时设置，所有日志都包含
- **WithFields**: 创建新的 Logger 实例，只影响该实例的日志
- GlobalFields 适合应用级别的字段（app, env, version）
- WithFields 适合请求级别的字段（request_id, user_id）

### Q: 如何在生产环境使用？

**A**: 推荐配置：
```go
cfg := &logger.Config{
    Level:         logger.InfoLevel,    // 生产用 Info
    Format:        logger.JSONFormat,   // JSON 便于采集
    EnableConsole: false,               // 关闭控制台
    EnableFile:    true,                // 启用文件
    OutputPath:    "/var/log/app.log",
    GlobalFields: map[string]interface{}{
        "app":     "myapp",
        "env":     "production",
        "version": "1.0.0",
    },
    Rotation: logger.RotationConfig{
        Type:       logger.RotationBySize,
        MaxSize:    100,  // 100MB
        MaxBackups: 10,
        MaxAge:     30,   // 30 天
        Compress:   true,
    },
}
```

### Q: 如何实现链路追踪？

**A**: 使用 ContextExtractor:
```go
cfg := &logger.Config{
    ContextExtractor: func(ctx context.Context) []zap.Field {
        fields := make([]zap.Field, 0)
        if traceID, ok := ctx.Value("trace_id").(string); ok {
            fields = append(fields, zap.String("trace_id", traceID))
        }
        if spanID, ok := ctx.Value("span_id").(string); ok {
            fields = append(fields, zap.String("span_id", spanID))
        }
        return fields
    },
}

// 使用
ctx := context.WithValue(ctx, "trace_id", "abc123")
logger.InfoContext(ctx, "处理请求")
```

### Q: 如何实现日志采样？

**A**: 使用 Sampling 配置:
```go
cfg := &logger.Config{
    EnableSampling:     true,
    SamplingInitial:    100,  // 前 100 条全部记录
    SamplingThereafter: 100,  // 之后每 100 条记录 1 条
}
```

### Q: 如何动态调整日志级别？

**A**: 可以通过创建新的 Logger 或使用 Hook:
```go
// 方法 1: 创建新 Logger
newLogger, _ := logger.New(&logger.Config{Level: logger.DebugLevel})
logger.SetDefault(newLogger)

// 方法 2: 使用环境变量
logger.InitDefaultFromEnv()  // 读取 XDOORIA_LOG_LEVEL
```

## 性能优化建议

1. **减少字段分配**: 使用 `WithFields()` 预设字段，避免每次都创建
2. **避免昂贵的序列化**: 使用 `zap.String()` 而不是 `zap.Any()` 对于简单类型
3. **采样**: 高频日志使用 Sampling 减少 I/O
4. **异步写入**: 可选开启 `EnableAsync` (需要注意 `Sync()` 调用)

## 更多文档

- [完整文档](../../pkg/logger/README.md)
- [API 参考](../../pkg/logger/)
- [测试用例](../../pkg/logger/*_test.go)

## 反馈和贡献

如有问题或建议，欢迎提交 Issue 或 Pull Request。
