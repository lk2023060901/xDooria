# Observability Example

这是一个完整的可观测性示例，展示如何使用 xDooria 框架的 `prometheus` 和 `sentry` 包构建可观测的应用。

## 功能演示

本示例包含：

- **示例应用**：模拟真实业务场景，生成各种指标和错误
- **Prometheus**：收集和存储应用指标
- **Grafana**：可视化指标和创建监控看板
- **Sentry**：错误追踪和崩溃分析（需单独配置，参见下文）

> **注意**: Sentry 官方镜像仅支持 x86_64 架构，ARM64/Apple Silicon 用户无法使用本地部署。
> 可选方案：使用 Sentry SaaS 服务 (sentry.io) 或在 Ubuntu/x86 服务器上部署。

## 架构

```
┌─────────────────┐
│  Example App    │
│  :9090          │──┐
│                 │  │
│  - Prometheus   │  │ (暴露 /metrics)
│    metrics      │  │
│  - Sentry       │  │ (可选，推送错误)
│    errors       │  │
└─────────────────┘  │
                     │
                     ↓
              ┌──────────────┐
              │  Prometheus  │
              │  :9091       │
              │              │
              │  (拉取指标)   │
              └──────┬───────┘
                     │ (数据源)
                     ↓
              ┌──────────────┐
              │   Grafana    │
              │   :3000      │
              │              │
              │  (可视化)     │
              └──────────────┘
```

## 快速开始

### 前置要求

- Docker & Docker Compose

### 1. 启动服务

```bash
cd examples/observability

# 启动所有服务
docker-compose up -d
```

### 2. 访问服务

- **应用指标**: http://localhost:9090/metrics
- **Prometheus**: http://localhost:9091
- **Grafana**: http://localhost:3000 (admin/admin)

### 3. 查看 Grafana 监控数据

1. 打开 Grafana: http://localhost:3000
2. 使用默认账号登录: `admin` / `admin`
3. 进入 "Dashboards"，查看预配置的两个 Dashboard：
   - **Application Metrics**: 应用业务指标
     - 请求速率、错误速率、Panic 速率
     - 请求时长分布 (p50/p95/p99)
     - 累计计数器
   - **Go Runtime Metrics**: Go 运行时指标
     - Goroutine 泄露监控
     - 内存使用监控
     - GC 频率监控

## 应用行为说明

示例应用每秒模拟一次业务请求，包含以下场景：

| 场景 | 概率 | 说明 |
|------|------|------|
| 正常请求 | 65% | 快速处理（< 50ms） |
| 慢请求 | 15% | 处理时间 500ms - 2s |
| 错误 | 10% | 触发错误（可选上报 Sentry） |
| Panic | 5% | 触发 panic 并恢复（可选上报 Sentry） |
| Goroutine 泄露 | 5% | 创建永久阻塞的 goroutine |

## 指标说明

应用暴露以下 Prometheus 指标：

| 指标名称 | 类型 | 说明 |
|---------|------|------|
| `example_app_requests_total` | Counter | 总请求数 |
| `example_app_errors_total` | Counter | 总错误数 |
| `example_app_panics_total` | Counter | 总 panic 数 |
| `example_app_request_duration_seconds` | Histogram | 请求时长分布 |

## Grafana Dashboard

预配置了两个 Dashboard：

### Application Metrics Dashboard
1. **Request Rate** - 每秒请求数（QPS）
2. **Error Rate** - 每秒错误数
3. **Panic Rate** - 每秒 panic 数
4. **Request Duration** - p50/p95/p99 延迟
5. **Total Counters** - 累计请求/错误/panic 数

### Go Runtime Metrics Dashboard
1. **Goroutines Count** - Goroutine 数量趋势（监控泄露）
2. **Memory Usage** - 内存分配和使用（监控内存泄露）
3. **GC Rate** - 垃圾回收频率
4. **Current Goroutines** - 当前 Goroutine 数量
5. **Current Memory** - 当前内存分配量

## 自定义配置

### 修改 Prometheus 抓取间隔

编辑 `prometheus/prometheus.yml`:

```yaml
global:
  scrape_interval: 5s  # 改为 5 秒
```

### 修改应用行为

编辑 `app/main.go` 中的概率分布或模拟逻辑。

### 添加自定义指标

在 `app/main.go` 中添加新的 Prometheus 指标：

```go
myMetric, _ := promClient.NewGauge(prometheus.GaugeOpts{
    Name: "my_custom_metric",
    Help: "My custom metric",
})
myMetric.Set(123)
```

## 重要配置说明

### Grafana Datasource UID 配置

本示例使用自动 provisioning 配置 Grafana datasource 和 dashboard。以下是关键配置规则：

#### 1. Datasource 配置 (`grafana/provisioning/datasources/datasources.yml`)

```yaml
apiVersion: 1

datasources:
  - name: Prometheus
    type: prometheus
    access: proxy
    url: http://prometheus:9090
    isDefault: true  # 关键：标记为默认数据源
    # 注意：不要设置 uid 字段
```

**重要事项：**
- ✅ **必须设置** `isDefault: true` 来标记默认数据源
- ❌ **不要设置** `uid` 字段，让 Grafana 自动生成
- 如果手动指定 `uid`，必须确保 dashboard JSON 中的 `uid` 完全匹配

#### 2. Dashboard 配置 (dashboard JSON 文件)

在 dashboard JSON 文件中，datasource 引用必须使用 `null` 值：

```json
{
  "datasource": {
    "type": "prometheus",
    "uid": null  // 关键：使用 null 自动匹配默认数据源
  }
}
```

**重要事项：**
- ✅ **必须使用** `"uid": null` 来引用默认数据源
- ❌ **不要使用** `"uid": ""` (空字符串) - 会导致匹配失败
- ❌ **不要使用** `"uid": "prometheus"` (硬编码字符串) - 除非 datasource 配置中明确设置了相同的 uid

#### 3. 常见问题排查

**问题：Dashboard 显示 "No data"**

排查步骤：
1. 检查 Prometheus 是否正常抓取数据
   ```bash
   curl http://localhost:9091/api/v1/query?query=up
   ```

2. 检查 Grafana datasource 配置
   - 访问 Grafana → Configuration → Data sources
   - 确认 Prometheus 数据源状态为 "Working"
   - 记下自动生成的 UID（如果有）

3. 检查 Dashboard JSON 配置
   - 确认所有 panel 的 `datasource.uid` 都是 `null`
   - 确认 datasource type 为 `"prometheus"`

4. 重新加载 provisioning 配置
   ```bash
   docker-compose restart grafana
   ```

**问题：Datasource UID 不匹配**

如果看到错误 "data source not found"：
- 检查 `datasources.yml` 中是否手动设置了 `uid`
- 检查 dashboard JSON 中的 `uid` 值
- 确保两者一致，或者都使用自动匹配方式（datasources.yml 不设置 uid，dashboard 使用 `null`）

### Sentry 集成配置（可选）

Sentry 用于错误和崩溃追踪。示例代码已集成 Sentry 支持，但需要单独配置。

#### 方案 1: 使用 Sentry SaaS 服务（推荐）

1. 注册 Sentry 账号: https://sentry.io
2. 创建新项目（Platform: Go）
3. 获取项目 DSN
4. 配置环境变量并重启应用：
   ```bash
   export SENTRY_DSN="https://your-dsn@sentry.io/project-id"
   docker-compose restart app
   ```

#### 方案 2: 本地部署 Sentry（仅限 x86_64/Ubuntu）

**重要**: Sentry 官方镜像不支持 ARM64 架构（Apple Silicon）。

如需在 Ubuntu/x86 服务器上部署本地 Sentry，请参考：
- `docker-compose.sentry.yml` - Sentry 服务配置模板
- [Sentry 官方文档](https://develop.sentry.dev/self-hosted/)

使用方法：
```bash
# 仅在 x86_64 架构服务器上执行
docker-compose -f docker-compose.yml -f docker-compose.sentry.yml up -d
```

## 清理

```bash
# 停止并删除容器
docker-compose down

# 同时删除数据卷
docker-compose down -v
```

## 学习要点

通过这个示例，你可以学习：

1. ✅ 如何使用 `pkg/prometheus` 暴露应用指标
2. ✅ 如何使用 `pkg/sentry` 追踪错误和崩溃
3. ✅ 如何配置 Prometheus 拉取指标
4. ✅ 如何在 Grafana 中配置数据源和 Dashboard
5. ✅ 如何部署完整的可观测性监控栈
6. ✅ 可观测性的最佳实践：指标、日志、追踪

## 故障排查

### 应用无法启动

```bash
# 查看应用日志
docker-compose logs app

# 检查是否端口冲突
lsof -i :9090
```

### Prometheus 无法抓取指标

```bash
# 检查 Prometheus 日志
docker-compose logs prometheus

# 访问 Prometheus Targets 页面
# http://localhost:9091/targets
```

### Grafana 无数据

1. 检查 Prometheus 数据源配置
2. 确认 Prometheus 正在抓取数据
3. 检查 Dashboard 的查询语句

## 进阶

### 集成告警

在 Prometheus 中配置告警规则，当错误率或 panic 率超过阈值时触发告警：

```yaml
# prometheus/alerts.yml
groups:
  - name: example_alerts
    rules:
      - alert: HighErrorRate
        expr: rate(example_app_errors_total[1m]) > 0.1
        annotations:
          summary: "错误率过高"
```

### 添加更多数据源

在 Grafana 中添加 Loki（日志）、Jaeger（追踪）等数据源，构建完整的可观测性平台。

## 参考

- [Prometheus 文档](https://prometheus.io/docs/)
- [Grafana 文档](https://grafana.com/docs/)
- [Sentry 文档](https://docs.sentry.io/)
