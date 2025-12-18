# Viper 配置管理示例

本目录包含 Viper 配置管理的示例代码，演示如何加载 YAML 和 JSON 格式的配置文件。

## 目录结构

```
examples/viper/
├── yaml/           # YAML 配置示例（服务器配置）
│   ├── config.yaml
│   └── main.go
├── json/           # JSON 配置示例（游戏策划配置）
│   ├── items.json
│   ├── levels.json
│   └── main.go
└── README.md
```

## YAML 示例

YAML 格式用于服务器配置，包含数据库、Redis、etcd 等基础设施配置。

### 运行 YAML 示例

```bash
cd examples/viper/yaml
go run main.go
```

### 功能演示

1. **加载整个配置** - 使用 `Unmarshal()` 加载完整配置
2. **UnmarshalKey** - 使用 `UnmarshalKey()` 只加载配置的某个部分
3. **环境变量覆盖** - 通过环境变量覆盖配置文件中的值

### 环境变量规则

- 前缀: `XDOORIA_`
- 配置路径中的 `.` 替换为 `_`
- 示例:
  - `server.port` → `XDOORIA_SERVER_PORT`
  - `database.host` → `XDOORIA_DATABASE_HOST`

## JSON 示例

JSON 格式用于游戏策划配置，从 xdooria-config 仓库导出。

### 运行 JSON 示例

```bash
cd examples/viper/json
go run main.go
```

### 功能演示

1. **加载道具配置** - 加载 `items.json`
2. **加载关卡配置** - 加载 `levels.json`
3. **UnmarshalKey** - 直接解析 JSON 中的某个数组

## 核心 API

### 加载配置的基本步骤

```go
// 1. 创建 Viper 实例
v := viper.New()

// 2. 设置配置文件
v.SetConfigFile("config.yaml")
v.SetConfigType("yaml")  // 或 "json"

// 3. 读取配置文件
if err := v.ReadInConfig(); err != nil {
    log.Fatal(err)
}

// 4. 解析到结构体
var cfg Config
if err := v.Unmarshal(&cfg); err != nil {
    log.Fatal(err)
}
```

### Unmarshal vs UnmarshalKey

- **Unmarshal**: 解析整个配置文件到结构体
- **UnmarshalKey**: 只解析配置文件中的某个 key

```go
// Unmarshal - 解析整个配置
var cfg Config
v.Unmarshal(&cfg)

// UnmarshalKey - 只解析 server 部分
var serverCfg ServerConfig
v.UnmarshalKey("server", &serverCfg)
```

## 注意事项

1. **struct tag**: 使用 `mapstructure` tag 而不是 `json` 或 `yaml`
   ```go
   type Config struct {
       Name string `mapstructure:"name"`
   }
   ```

2. **环境变量**: 只在 YAML 服务器配置中使用，JSON 游戏配置不需要

3. **嵌套配置**: Viper 支持深层嵌套结构，使用 `.` 分隔路径
   ```go
   v.UnmarshalKey("database.connection", &dbConn)
   ```
