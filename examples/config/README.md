# Config 配置管理示例

本目录包含 Config 模块的各种使用示例，每个子目录演示一个特定功能。

## 目录结构

```
examples/config/
├── basic/            # 示例 1: 基础使用
├── unmarshal-key/    # 示例 2: UnmarshalKey 解析部分配置
├── env-override/     # 示例 3: 环境变量覆盖配置
├── validation/       # 示例 4: 配置验证
├── hot-reload/       # 示例 5: 配置热重载
├── merge/            # 示例 6: 配置合并
└── advanced/         # 示例 7: 高级用法
```

## 运行示例

### 1. 基础使用 (basic)

演示配置文件的加载、解析和基本获取操作。

```bash
cd basic
go run main.go
```

**学习要点**：
- `LoadFile()` - 加载配置文件
- `Unmarshal()` - 解析到结构体
- `Get/GetInt/GetString/GetBool()` - 获取配置值
- `IsSet()` - 检查配置项是否存在

---

### 2. UnmarshalKey (unmarshal-key)

演示如何解析部分配置（支持 struct 和基本类型）。

```bash
cd unmarshal-key
go run main.go
```

**学习要点**：
- `UnmarshalKey("server", &struct)` - 解析配置块到结构体
- `UnmarshalKey("server.port", &int)` - 解析单个字段到基本类型
- 灵活解析不同层级的配置

---

### 3. 环境变量覆盖 (env-override)

演示环境变量如何覆盖配置文件中的值。

```bash
cd env-override
go run main.go
```

**学习要点**：
- `BindEnv("PREFIX")` - 绑定环境变量前缀
- 环境变量映射规则：`server.port` → `PREFIX_SERVER_PORT`
- 配置优先级：环境变量 > 配置文件 > 默认值

---

### 4. 配置验证 (validation)

演示各种配置验证规则和自定义验证。

```bash
cd validation
go run main.go
```

**学习要点**：
- 基于 struct tag 的验证：`validate:"required,min=1,max=65535"`
- 标准验证规则：email, url, oneof, min, max 等
- 单字段验证：`ValidateField()`
- 自定义验证规则：`ValidateWithCustom()`
- 友好的错误提示

**配置文件**：
- `valid-config.yaml` - 有效配置
- `missing-required.yaml` - 缺少必填字段
- `invalid-port.yaml` - 端口超出范围
- `invalid-enum.yaml` - 枚举值错误
- `invalid-email.yaml` - 邮箱和 URL 格式错误

---

### 5. 配置热重载 (hot-reload)

演示配置文件变更后自动重载（需要持续运行）。

```bash
cd hot-reload
go run main.go

# 在另一个终端修改配置文件
echo 'server:
  port: 9090
  host: "0.0.0.0"
logger:
  level: "debug"' > config.yaml

# 观察程序输出，会自动检测并重载配置
# 按 Ctrl+C 退出
```

**学习要点**：
- `Watch(callback)` - 注册配置变更回调
- 自动监听文件变化
- 实时应用新配置
- 适用场景：动态调整日志级别、特性开关等

---

### 6. 配置合并 (merge)

演示如何合并多个配置源。

```bash
cd merge
go run main.go
```

**学习要点**：
- `MergeConfig(dst, src)` - 合并两个配置
- src 的非零值覆盖 dst
- 递归合并嵌套结构
- Nil 值处理

**应用场景**：
- 基础配置 + 环境配置
- 默认配置 + 用户配置
- 多个配置文件合并

---

### 7. 高级用法 (advanced)

演示高级配置选项和组合使用。

```bash
cd advanced
go run main.go
```

**学习要点**：
- `WithDefaults()` - 设置默认值
- `WithConfigName()` - 配置文件名
- `WithConfigPaths()` - 多路径搜索
- `WithEnvPrefix()` - 环境变量前缀
- `AllSettings()` - 获取所有配置
- 组合使用多个选项

---

## 完整示例对比

| 示例 | 核心功能 | 难度 | 推荐顺序 |
|------|---------|------|----------|
| basic | 基础加载和解析 | ⭐ | 1 |
| unmarshal-key | 部分配置解析 | ⭐⭐ | 2 |
| env-override | 环境变量覆盖 | ⭐⭐ | 3 |
| validation | 配置验证 | ⭐⭐⭐ | 4 |
| merge | 配置合并 | ⭐⭐⭐ | 5 |
| advanced | 高级选项 | ⭐⭐⭐ | 6 |
| hot-reload | 配置热重载 | ⭐⭐⭐⭐ | 7 |

## 快速测试所有示例

```bash
# 在 examples/config 目录下运行
for dir in basic unmarshal-key env-override validation merge advanced; do
    echo "=== Running $dir ==="
    (cd $dir && go run main.go)
    echo ""
done
```

## 常见问题

### Q: UnmarshalKey 和 Unmarshal 有什么区别？

**A**:
- `Unmarshal(&cfg)` - 解析整个配置文件到结构体
- `UnmarshalKey("database", &dbCfg)` - 只解析指定部分
- `UnmarshalKey("server.port", &port)` - 解析单个字段

### Q: 环境变量如何映射到配置项？

**A**:
- 前缀: `BindEnv("APP")` → 环境变量前缀为 `APP_`
- 分隔符: `.` → `_`
- 示例: `server.port` → `APP_SERVER_PORT`
- 示例: `database.postgres.host` → `APP_DATABASE_POSTGRES_HOST`

### Q: 配置优先级是什么？

**A**: 从高到低：
1. 命令行参数
2. 环境变量
3. 配置文件
4. 默认值

### Q: 热重载会影响性能吗？

**A**:
- 使用文件系统监听 (fsnotify)，性能影响极小
- 仅在文件变更时触发回调
- 建议：避免在回调中执行耗时操作

### Q: 如何在生产环境使用？

**A**: 推荐实践：
1. 配置文件存储基础配置
2. 敏感信息通过环境变量传递
3. 使用配置验证确保启动时检查
4. 谨慎使用热重载（仅用于日志级别等）

## 更多文档

- [完整文档](../../pkg/config/README.md)
- [API 参考](../../pkg/config/)
- [测试用例](../../pkg/config/*_test.go)

## 反馈和贡献

如有问题或建议，欢迎提交 Issue 或 Pull Request。
