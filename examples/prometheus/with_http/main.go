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
	fmt.Println("=== Prometheus HTTP 集成示例 ===")

	// 创建 Prometheus 客户端（不启动独立服务器）
	client, err := prometheus.New(&prometheus.Config{
		Namespace: "webapp",
		HTTPServer: prometheus.HTTPServerConfig{
			Enabled: false, // 不启动独立服务器
		},
	})
	if err != nil {
		panic(err)
	}
	defer client.Close()

	fmt.Println("✓ Prometheus 客户端创建成功")
	fmt.Println()

	// 注册 HTTP 指标
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

	requestSize := client.MustNewHistogram(
		"http_request_size_bytes",
		"HTTP request size in bytes",
		[]string{"method", "path"},
		[]float64{100, 1000, 10000, 100000, 1000000},
	)

	responseSize := client.MustNewHistogram(
		"http_response_size_bytes",
		"HTTP response size in bytes",
		[]string{"method", "path"},
		[]float64{100, 1000, 10000, 100000, 1000000},
	)

	inFlightRequests := client.MustNewGauge(
		"http_requests_in_flight",
		"Current HTTP requests in flight",
		[]string{"method", "path"},
	)

	fmt.Println("✓ 注册 HTTP 指标")
	fmt.Println()

	// 创建 HTTP 中间件
	metricsMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := r.URL.Path

			// 记录请求中
			inFlightRequests.WithLabelValues(r.Method, path).Inc()
			defer inFlightRequests.WithLabelValues(r.Method, path).Dec()

			// 记录请求大小
			if r.ContentLength > 0 {
				requestSize.WithLabelValues(r.Method, path).Observe(float64(r.ContentLength))
			}

			// 包装 ResponseWriter 以获取状态码和响应大小
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// 执行请求
			next.ServeHTTP(rw, r)

			// 记录指标
			duration := time.Since(start).Seconds()
			status := strconv.Itoa(rw.statusCode)

			requestCounter.WithLabelValues(r.Method, path, status).Inc()
			requestDuration.WithLabelValues(r.Method, path).Observe(duration)
			responseSize.WithLabelValues(r.Method, path).Observe(float64(rw.size))
		})
	}

	// 创建 HTTP 路由
	mux := http.NewServeMux()

	// 集成 Prometheus metrics 端点
	mux.Handle("/metrics", client.Handler())

	// 业务 API 端点
	mux.HandleFunc("/api/users", func(w http.ResponseWriter, r *http.Request) {
		// 模拟处理时间
		time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{"users": [{"id": 1, "name": "Alice"}, {"id": 2, "name": "Bob"}]}`
		w.Write([]byte(response))
	})

	mux.HandleFunc("/api/orders", func(w http.ResponseWriter, r *http.Request) {
		// 模拟处理时间
		time.Sleep(time.Duration(rand.Intn(200)) * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{"orders": [{"id": 1, "amount": 100}, {"id": 2, "amount": 200}]}`
		w.Write([]byte(response))
	})

	mux.HandleFunc("/api/products", func(w http.ResponseWriter, r *http.Request) {
		// 模拟处理时间
		time.Sleep(time.Duration(rand.Intn(150)) * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{"products": [{"id": 1, "name": "Product A"}, {"id": 2, "name": "Product B"}]}`
		w.Write([]byte(response))
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		// 慢请求
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Slow response"))
	})

	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		// 错误响应
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal server error"))
	})

	// 应用中间件
	handler := metricsMiddleware(mux)

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
	fmt.Println("按 Ctrl+C 退出...")
	fmt.Println()

	// 启动后台任务模拟流量
	go simulateTraffic()

	log.Fatal(http.ListenAndServe(addr, handler))
}

// responseWriter 包装 ResponseWriter 以记录状态码和响应大小
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

// simulateTraffic 模拟后台流量
func simulateTraffic() {
	time.Sleep(2 * time.Second) // 等待服务器启动

	endpoints := []string{
		"http://localhost:8080/api/users",
		"http://localhost:8080/api/orders",
		"http://localhost:8080/api/products",
		"http://localhost:8080/health",
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 随机选择端点
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
	}
}
