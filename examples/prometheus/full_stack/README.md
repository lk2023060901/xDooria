# Prometheus 完整监控栈示例

这个示例演示了如何在真实环境中使用 xDooria 的 Prometheus 客户端，包括完整的监控栈：应用、Prometheus Server 和 Grafana。

## 架构

```
┌─────────────────┐     scrape      ┌──────────────────┐
│  Example App    │ ←───────────── │  Prometheus      │
│  (Go + Client)  │                 │  Server          │
│  :8080/metrics  │                 │  :9090           │
└─────────────────┘                 └──────────────────┘
                                            │
                                            │ query
                                            ↓
                                    ┌──────────────────┐
                                    │  Grafana         │
                                    │  Dashboard       │
                                    │  :3000           │
                                    └──────────────────┘
```

## 组件

1. **Example App** - 使用 xDooria Prometheus 客户端的示例应用
   - 暴露 `/metrics` 端点供 Prometheus 采集
   - 提供多个 API 端点用于测试
   - 自动生成测试流量

2. **Prometheus Server** - 指标采集和存储
   - 每 15 秒采集一次应用指标
   - 提供查询接口

3. **Grafana** - 可视化仪表板
   - 预配置仪表板展示应用指标
   - 默认用户名/密码: admin/admin

## 快速开始

### 启动完整栈

```bash
cd examples/prometheus/full_stack
docker-compose up -d
```

### 访问服务

- **应用**: http://localhost:8080
  - `/metrics` - Prometheus 指标
  - `/api/users` - 用户 API
  - `/api/orders` - 订单 API
  - `/api/products` - 产品 API
  - `/health` - 健康检查

- **Prometheus**: http://localhost:9090
  - 查询和浏览指标

- **Grafana**: http://localhost:3000
  - 用户名: `admin`
  - 密码: `admin`
  - 预配置仪表板: "Example App Metrics"

### 停止

```bash
docker-compose down

# 停止并删除数据卷
docker-compose down -v
```

## 验证

### 1. 查看应用指标

```bash
curl http://localhost:8080/metrics
```

### 2. 检查 Prometheus 采集状态

访问 http://localhost:9090/targets，应该看到 `example_app` 状态为 `UP`

### 3. 查询指标

在 Prometheus UI 中查询：

```promql
# 请求速率
rate(example_app_http_requests_total[1m])

# P95 延迟
histogram_quantile(0.95, rate(example_app_http_request_duration_seconds_bucket[5m]))

# 当前活跃请求
sum(example_app_http_requests_in_flight)
```

### 4. 查看 Grafana 仪表板

1. 访问 http://localhost:3000
2. 登录 (admin/admin)
3. 导航到 "Dashboards" → "Example App Metrics"

## 目录结构

```
full_stack/
├── docker-compose.yaml           # Docker Compose 配置
├── prometheus.yml                # Prometheus 配置
├── app/
│   ├── main.go                  # 示例应用
│   └── Dockerfile               # 应用容器镜像
├── grafana/
│   ├── provisioning/
│   │   ├── datasources/
│   │   │   └── prometheus.yml   # Prometheus 数据源
│   │   └── dashboards/
│   │       └── dashboard.yml    # 仪表板配置
│   └── dashboards/
│       └── app-metrics.json     # 预配置仪表板
└── README.md                     # 本文档
```

## 指标说明

### HTTP 指标

- `example_app_http_requests_total` - 总请求计数
- `example_app_http_request_duration_seconds` - 请求耗时分布
- `example_app_http_requests_in_flight` - 当前处理中的请求数

### 业务指标

- `example_app_business_operations_total` - 业务操作计数
- `example_app_business_operation_duration_seconds` - 业务操作耗时

### Go 运行时指标

- `go_goroutines` - Goroutine 数量
- `go_memstats_*` - 内存统计
- `go_gc_*` - GC 统计

### 进程指标

- `process_cpu_seconds_total` - CPU 使用时间
- `process_resident_memory_bytes` - 内存使用
- `process_open_fds` - 打开的文件描述符
