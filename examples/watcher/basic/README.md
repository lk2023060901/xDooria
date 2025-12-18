# Config Watcher 基础示例

演示如何使用 `config.Watcher` 实现配置热更新。

## 功能

- 自动监听配置文件变化
- 配置变化时触发回调
- 线程安全的配置读取
- 支持泛型，类型安全

## 运行示例

```bash
cd examples/watcher/basic
go run main.go
```

## 测试热更新

程序运行后：

1. 修改 `config.yaml` 文件，例如：
   ```yaml
   server:
     port: 9090  # 改成 9090
   ```

2. 保存文件

3. 程序会自动检测变化并输出：
   ```
   🔄 配置文件已变化！
   📋 当前配置:
     服务端口: 9090
   ```

## 关键代码

### 创建监听器

```go
// 使用泛型创建监听器
watcher, err := config.NewWatcher[AppConfig]("config.yaml", "yaml")
if err != nil {
    log.Fatal(err)
}
defer watcher.Stop()
```

### 获取配置

```go
// 线程安全地获取当前配置
cfg := watcher.GetConfig()
fmt.Printf("Port: %d\n", cfg.Server.Port)
```

### 注册回调

```go
// 注册配置变化回调
watcher.OnChange(func(newCfg *AppConfig) {
    fmt.Println("配置已更新！")
    // 使用新配置更新应用状态
})
```

## 注意事项

1. **配置验证**: 如果需要验证配置，在回调中实现验证逻辑
2. **错误处理**: 配置加载失败时，会保持使用旧配置
3. **并发安全**: `GetConfig()` 和 `OnChange()` 都是线程安全的
4. **多次回调**: 可以注册多个回调函数，它们会按注册顺序执行

## 使用场景

- 动态调整日志级别
- 更新数据库连接池大小
- 修改限流阈值
- 调整缓存配置
- 开关功能特性
