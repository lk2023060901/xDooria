package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/lk2023060901/xdooria/pkg/prometheus"
)

func main() {
	fmt.Println("=== Prometheus 完整栈示例应用 ===")

	// 创建 Prometheus 客户端（不启动独立服务器，使用集成模式）
	client, err := prometheus.New(&prometheus.Config{
		Namespace: "example_app",
		HTTPServer: prometheus.HTTPServerConfig{
			Enabled: false, // 集成到应用的 HTTP 服务器
		},
		EnableGoCollector:      true,
		EnableProcessCollector: true,
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("✓ Prometheus 客户端创建成功")
	fmt.Println()

	// 注册应用指标
	requestCounter := client.MustNewCounter(
		"http_requests_total",
		"Total HTTP requests",
		[]string{"method", "path", "status"},
	)

	requestDuration := client.MustNewHistogram(
		"http_request_duration_seconds",
		"HTTP request duration in seconds",
		[]string{"method", "path"},
		[]float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5},
	)

	activeRequests := client.MustNewGauge(
		"http_requests_in_flight",
		"Current HTTP requests in flight",
		[]string{"method"},
	)

	businessCounter := client.MustNewCounter(
		"business_operations_total",
		"Total business operations",
		[]string{"operation", "result"},
	)

	businessDuration := client.MustNewSummary(
		"business_operation_duration_seconds",
		"Business operation duration in seconds",
		[]string{"operation"},
		map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	)

	fmt.Println("✓ 注册应用指标")
	fmt.Println()

	// 创建 HTTP 中间件
	metricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := r.URL.Path

			// 记录活跃请求
			activeRequests.WithLabelValues(r.Method).Inc()
			defer activeRequests.WithLabelValues(r.Method).Dec()

			// 包装 ResponseWriter
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 执行请求
			next.ServeHTTP(rw, r)

			// 记录指标
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.statusCode)

			requestCounter.WithLabelValues(r.Method, path, status).Inc()
			requestDuration.WithLabelValues(r.Method, path).Observe(duration)
		})
	}

	// 创建路由
	mux := http.NewServeMux()

	// Prometheus metrics 端点
	mux.Handle("/metrics", client.Handler())

	// API 端点
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// 模拟业务逻辑
		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
		duration := time.Since(start).Seconds()

		businessCounter.WithLabelValues("get_users", "success").Inc()
		businessDuration.WithLabelValues("get_users").Observe(duration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`))
	})

	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)
		duration := time.Since(start).Seconds()

		businessCounter.WithLabelValues("get_orders", "success").Inc()
		businessDuration.WithLabelValues("get_orders").Observe(duration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"orders": [{"id": 1, "amount": 100}, {"id": 2, "amount": 200}]}`))
	})

	mux.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)
		duration := time.Since(start).Seconds()

		businessCounter.WithLabelValues("get_products", "success").Inc()
		businessDuration.WithLabelValues("get_products").Observe(duration)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"products": [{"id": 1, "name": "Product A"}, {"id": 2, "name": "Product B"}]}`))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		time.Sleep(2 * time.Second)
		duration := time.Since(start).Seconds()

		businessCounter.WithLabelValues("slow_operation", "success").Inc()
		businessDuration.WithLabelValues("slow_operation").Observe(duration)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Slow response"))
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		businessCounter.WithLabelValues("error_operation", "failure").Inc()

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	})

	// 应用中间件
	handler := metricsMiddleware(mux)

	// 启动后台流量模拟
	go simulateTraffic()

	// 启动 HTTP 服务器
	addr := ":8080"
	fmt.Printf("✓ HTTP 服务器启动在 %s\n", addr)
	fmt.Println()
	fmt.Println("访问端点:")
	fmt.Println("  - http://localhost:8080/metrics       (Prometheus 指标)")
	fmt.Println("  - http://localhost:8080/api/users     (用户 API)")
	fmt.Println("  - http://localhost:8080/api/orders    (订单 API)")
	fmt.Println("  - http://localhost:8080/api/products  (产品 API)")
	fmt.Println("  - http://localhost:8080/health        (健康检查)")
	fmt.Println("  - http://localhost:8080/slow          (慢请求)")
	fmt.Println("  - http://localhost:8080/error         (错误响应)")
	fmt.Println()
	fmt.Println("监控端点:")
	fmt.Println("  - http://localhost:9090               (Prometheus Server)")
	fmt.Println("  - http://localhost:3000               (Grafana Dashboard)")
	fmt.Println()

	log.Fatal(http.ListenAndServe(addr, handler))
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.size += n
	return n, err
}

func simulateTraffic() {
	time.Sleep(3 * time.Second) // 等待服务器启动

	endpoints := []string{
		"http://localhost:8080/api/users",
		"http://localhost:8080/api/orders",
		"http://localhost:8080/api/products",
		"http://localhost:8080/health",
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		endpoint := endpoints[rand.Intn(len(endpoints))]

		go func(url string) {
			resp, err := http.Get(url)
			if err != nil {
				return
			}
			defer resp.Body.Close()

			fmt.Printf("📊 [%s] GET %s → %d\n",
				time.Now().Format("15:04:05"),
				url,
				resp.StatusCode)
		}(endpoint)

		// 偶尔访问慢端点和错误端点
		if rand.Intn(10) == 0 {
			go func() {
				resp, err := http.Get("http://localhost:8080/slow")
				if err != nil {
					return
				}
				defer resp.Body.Close()
			}()
		}

		if rand.Intn(15) == 0 {
			go func() {
				resp, err := http.Get("http://localhost:8080/error")
				if err != nil {
					return
				}
				defer resp.Body.Close()
			}()
		}
	}
}
