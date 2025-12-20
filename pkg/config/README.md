# Config 配置管理模块

通用的配置管理模块，基于 Viper 实现，提供配置加载、解析、验证和热重载功能。

## 功能特性

- ✅ 多格式支持：YAML、JSON、TOML 等
- ✅ 多源配置：文件、环境变量、命令行参数、默认值
- ✅ 配置优先级：命令行 > 环境变量 > 配置文件 > 默认值
- ✅ 类型安全：支持解析到结构体或基本类型
- ✅ 配置验证：基于 struct tag 的验证规则
- ✅ 配置热重载：监听文件变化自动重载
- ✅ 线程安全：支持并发访问

## 安装依赖

```bash
go get github.com/spf13/viper
go get github.com/fsnotify/fsnotify
go get github.com/go-playground/validator/v10
```

## 快速开始

### 1. 基本使用

```go
package main

import (
    "fmt"
    "github.com/lk2023060901/xdooria/pkg/config"
)

type AppConfig struct {
    Server struct {
        Port int    `yaml:"port" validate:"required,min=1,max=65535"`
        Host string `yaml:"host" validate:"required"`
    } `yaml:"server"`
    Database struct {
        Postgres postgres.Config `yaml:"postgres"`
        Redis    redis.Config    `yaml:"redis"`
    } `yaml:"database"`
}

func main() {
    // 创建配置管理器
    mgr := config.NewManager()

    // 加载配置文件
    if err := mgr.LoadFile("config.yaml"); err != nil {
        panic(err)
    }

    // 解析到结构体
    var cfg AppConfig
    if err := mgr.Unmarshal(&cfg); err != nil {
        panic(err)
    }

    // 验证配置
    validator := config.NewValidator()
    if err := validator.Validate(cfg); err != nil {
        panic(err)
    }

    fmt.Printf("Server running on %s:%d\n", cfg.Server.Host, cfg.Server.Port)
}
```

### 2. 环境变量支持

```go
// 绑定环境变量（前缀 APP_）
mgr := config.NewManager()
mgr.BindEnv("APP")
mgr.LoadFile("config.yaml")

// 环境变量 APP_SERVER_PORT=9000 会覆盖配置文件中的 server.port
port := mgr.GetInt("server.port") // 返回 9000
```

### 3. 默认值

```go
defaults := map[string]any{
    "server.port": 8080,
    "server.host": "localhost",
}

mgr := config.NewManager(config.WithDefaults(defaults))
```

### 4. UnmarshalKey - 解析部分配置

```go
// 解析整个 postgres 配置
var pgCfg postgres.Config
mgr.UnmarshalKey("database.postgres", &pgCfg)

// 解析单个字段
var port int
mgr.UnmarshalKey("server.port", &port)

var host string
mgr.UnmarshalKey("server.host", &host)

var enabled bool
mgr.UnmarshalKey("feature.enabled", &enabled)
```

### 5. 配置热重载

```go
mgr := config.NewManager()
mgr.LoadFile("config.yaml")

// 注册回调函数
mgr.Watch(func() {
    fmt.Println("Configuration changed!")

    // 重新加载配置
    var cfg AppConfig
    mgr.Unmarshal(&cfg)

    // 应用新配置...
})
```

## 配置文件示例

```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  postgres:
    master:
      host: localhost
      port: 5432
      user: xdooria
      password: ${DB_PASSWORD}  # 支持环境变量引用
      dbname: xdooria
    query_timeout: 30s
    pool:
      max_conns: 100
      min_conns: 10

  redis:
    standalone:
      addr: localhost:6379
    pool_size: 50

logger:
  level: info     # debug, info, warn, error
  format: json    # json, text

feature:
  enabled: true
```

## 配置验证

### 标准验证规则

```go
type Config struct {
    Port     int    `validate:"required,min=1,max=65535"`
    Host     string `validate:"required"`
    Email    string `validate:"email"`
    Level    string `validate:"oneof=debug info warn error"`
    URL      string `validate:"url"`
    MinValue int    `validate:"gte=0"`
    MaxValue int    `validate:"lte=100"`
}

validator := config.NewValidator()
if err := validator.Validate(cfg); err != nil {
    // 错误信息会格式化为友好的提示
    fmt.Println(err)
}
```

### 自定义验证规则

```go
import "github.com/go-playground/validator/v10"

// 自定义验证函数
customRule := func(fl validator.FieldLevel) bool {
    value := fl.Field().String()
    return value == "expected_value"
}

rules := map[string]validator.Func{
    "custom_rule": customRule,
}

validator := config.NewValidator()
err := validator.ValidateWithCustom(cfg, rules)
```

### 函数式验证

```go
validateFunc := func(cfg any) error {
    c := cfg.(AppConfig)
    if c.Server.Port < 1024 {
        return fmt.Errorf("port must be >= 1024")
    }
    return nil
}

if err := config.ValidateWithFunc(cfg, validateFunc); err != nil {
    panic(err)
}
```

## 配置合并

```go
// 合并两个配置（src 覆盖 dst）
dst := &Config{
    Server: ServerConfig{Port: 8080, Host: "localhost"},
}

src := &Config{
    Server: ServerConfig{Port: 9090},
}

merged, err := config.MergeConfig(dst, src)
// merged.Server.Port = 9090 (来自 src)
// merged.Server.Host = "localhost" (保留 dst)
```

## 高级选项

```go
mgr := config.NewManager(
    // 设置默认值
    config.WithDefaults(map[string]any{
        "server.port": 8080,
    }),

    // 设置配置文件类型
    config.WithConfigType("yaml"),

    // 设置配置文件名（不含扩展名）
    config.WithConfigName("config"),

    // 添加搜索路径
    config.WithConfigPaths(".", "/etc/app", "$HOME/.app"),

    // 设置环境变量前缀
    config.WithEnvPrefix("APP"),
)
```

## API 参考

### Manager 接口

```go
type Manager interface {
    LoadFile(path string) error
    BindEnv(prefix string)
    Unmarshal(v any) error
    UnmarshalKey(key string, v any) error
    Get(key string) any
    GetString(key string) string
    GetInt(key string) int
    GetBool(key string) bool
    Watch(callback func()) error
    IsSet(key string) bool
    AllSettings() map[string]any
}
```

### Validator

```go
type Validator struct {
    Validate(cfg any) error
    ValidateField(field any, tag string) error
    ValidateWithCustom(cfg any, rules map[string]validator.Func) error
    MustValidate(cfg any)
}
```

## 与其他模块集成

### PostgreSQL

```go
type AppConfig struct {
    Database struct {
        Postgres postgres.Config `yaml:"postgres"`
    } `yaml:"database"`
}

mgr := config.NewManager()
mgr.LoadFile("config.yaml")

var cfg AppConfig
mgr.Unmarshal(&cfg)

// 使用 wire 注入
// wire.FieldsOf(new(*AppConfig), "Database.Postgres")
client, err := postgres.NewClient(&cfg.Database.Postgres)
```

### Redis

```go
type AppConfig struct {
    Database struct {
        Redis redis.Config `yaml:"redis"`
    } `yaml:"database"`
}

mgr := config.NewManager()
mgr.LoadFile("config.yaml")

var cfg AppConfig
mgr.Unmarshal(&cfg)

client, err := redis.NewClient(&cfg.Database.Redis)
```

## 最佳实践

1. **使用结构体标签验证**：在配置结构体中添加 `validate` 标签
2. **环境变量覆盖**：生产环境使用环境变量覆盖敏感配置
3. **配置分层**：开发/测试/生产环境使用不同的配置文件
4. **热重载谨慎使用**：仅在需要时启用，避免频繁重载
5. **Wire 依赖注入**：使用 wire 管理模块间的配置传递

## 错误处理

```go
var (
    ErrConfigFileNotFound  = errors.New("config file not found")
    ErrInvalidConfigFormat = errors.New("invalid config format")
    ErrKeyNotFound         = errors.New("config key not found")
    ErrInvalidType         = errors.New("invalid config type")
    ErrValidationFailed    = errors.New("config validation failed")
    ErrNilConfig           = errors.New("config cannot be nil")
    ErrMergeFailed         = errors.New("failed to merge configs")
)
```

## 性能考虑

- 配置读取使用读写锁，支持高并发读取
- 配置热重载会触发所有注册的回调，避免注册耗时操作
- 大配置文件建议使用 `UnmarshalKey` 只解析需要的部分

## 许可证

MIT License
