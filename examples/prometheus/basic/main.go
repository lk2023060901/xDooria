package main

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/lk2023060901/xdooria/pkg/prometheus"
)

func main() {
	fmt.Println("=== Prometheus 基础使用示例 ===")

	// 创建 Prometheus 客户端
	// HTTP 服务器会自动启动在 :9090/metrics
	client, err := prometheus.New(&prometheus.Config{
		Namespace: "myapp",
		HTTPServer: prometheus.HTTPServerConfig{
			Enabled: true,
			Addr:    ":9090",
			Path:    "/metrics",
		},
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("✓ Prometheus 客户端创建成功")
	fmt.Println("✓ HTTP 服务器启动在 :9090/metrics")
	fmt.Println()

	// 1. Counter - 计数器
	fmt.Println("【1. Counter - 计数器】")
	requestCounter := client.MustNewCounter(
		"http_requests_total",
		"Total HTTP requests",
		[]string{"method", "path", "status"},
	)
	fmt.Println("  ✓ 创建 Counter: http_requests_total")

	// 模拟请求
	requestCounter.WithLabelValues("GET", "/api/users", "200").Inc()
	requestCounter.WithLabelValues("GET", "/api/users", "200").Add(5)
	requestCounter.WithLabelValues("POST", "/api/users", "201").Inc()
	requestCounter.WithLabelValues("GET", "/api/orders", "404").Inc()
	fmt.Println("  ✓ 记录请求: GET /api/users (200) x6")
	fmt.Println("  ✓ 记录请求: POST /api/users (201) x1")
	fmt.Println("  ✓ 记录请求: GET /api/orders (404) x1")
	fmt.Println()

	// 2. Gauge - 仪表盘
	fmt.Println("【2. Gauge - 仪表盘】")
	activeConnections := client.MustNewGauge(
		"active_connections",
		"Number of active connections",
		[]string{"server"},
	)
	fmt.Println("  ✓ 创建 Gauge: active_connections")

	// 模拟连接变化
	activeConnections.WithLabelValues("server1").Set(100)
	fmt.Println("  ✓ 设置 server1 连接数: 100")

	activeConnections.WithLabelValues("server1").Add(10)
	fmt.Println("  ✓ 增加 server1 连接数: +10")

	activeConnections.WithLabelValues("server1").Sub(5)
	fmt.Println("  ✓ 减少 server1 连接数: -5")

	activeConnections.WithLabelValues("server2").Set(50)
	fmt.Println("  ✓ 设置 server2 连接数: 50")
	fmt.Println()

	// 3. Histogram - 直方图
	fmt.Println("【3. Histogram - 直方图】")
	requestDuration := client.MustNewHistogram(
		"http_request_duration_seconds",
		"HTTP request duration in seconds",
		[]string{"method", "path"},
		[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	)
	fmt.Println("  ✓ 创建 Histogram: http_request_duration_seconds")

	// 模拟请求耗时
	durations := []float64{0.003, 0.012, 0.045, 0.089, 0.123, 0.234, 0.456, 0.789}
	for _, d := range durations {
		requestDuration.WithLabelValues("GET", "/api/users").Observe(d)
	}
	fmt.Printf("  ✓ 记录 %d 次请求耗时\n", len(durations))
	fmt.Println()

	// 4. Summary - 摘要
	fmt.Println("【4. Summary - 摘要】")
	queryDuration := client.MustNewSummary(
		"db_query_duration_seconds",
		"Database query duration in seconds",
		[]string{"query_type"},
		map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	)
	fmt.Println("  ✓ 创建 Summary: db_query_duration_seconds")

	// 模拟数据库查询
	for i := 0; i < 100; i++ {
		duration := rand.Float64() * 0.5
		queryDuration.WithLabelValues("select").Observe(duration)
	}
	fmt.Println("  ✓ 记录 100 次查询耗时")
	fmt.Println()

	// 5. 获取已注册的指标
	fmt.Println("【5. 获取已注册的指标】")
	if counter, ok := client.GetCounter("http_requests_total"); ok {
		counter.WithLabelValues("GET", "/health", "200").Inc()
		fmt.Println("  ✓ 获取并使用 Counter: http_requests_total")
	}

	if gauge, ok := client.GetGauge("active_connections"); ok {
		gauge.WithLabelValues("server3").Set(75)
		fmt.Println("  ✓ 获取并使用 Gauge: active_connections")
	}
	fmt.Println()

	// 保持运行以便访问 metrics
	fmt.Println("=== 访问 http://localhost:9090/metrics 查看指标 ===")
	fmt.Println("按 Ctrl+C 退出...")
	fmt.Println()

	// 持续更新指标模拟真实场景
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 模拟新请求
		methods := []string{"GET", "POST", "PUT", "DELETE"}
		paths := []string{"/api/users", "/api/orders", "/api/products"}
		statuses := []string{"200", "201", "400", "404", "500"}

		method := methods[rand.Intn(len(methods))]
		path := paths[rand.Intn(len(paths))]
		status := statuses[rand.Intn(len(statuses))]

		requestCounter.WithLabelValues(method, path, status).Inc()

		// 模拟请求耗时
		duration := rand.Float64() * 2
		requestDuration.WithLabelValues(method, path).Observe(duration)

		// 模拟连接数变化
		server := fmt.Sprintf("server%d", rand.Intn(3)+1)
		change := float64(rand.Intn(20) - 10)
		activeConnections.WithLabelValues(server).Add(change)

		fmt.Printf("📊 [%s] %s %s %s (%.3fs)\n",
			time.Now().Format("15:04:05"),
			method, path, status, duration)
	}
}
