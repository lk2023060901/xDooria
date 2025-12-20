package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/lk2023060901/xdooria/pkg/prometheus"
	"github.com/lk2023060901/xdooria/pkg/sentry"
)

func main() {
	log.Println("Starting observability example application...")

	// 初始化 Prometheus 客户端
	promClient, err := initPrometheus()
	if err != nil {
		log.Fatalf("Failed to initialize Prometheus: %v", err)
	}
	defer promClient.Close()
	log.Println("Prometheus initialized, metrics exposed at :9090/metrics")

	// 初始化 Sentry 客户端
	sentryClient, err := initSentry()
	if err != nil {
		log.Fatalf("Failed to initialize Sentry: %v", err)
	}
	defer sentryClient.Close()
	log.Println("Sentry initialized")

	// 创建 Prometheus 指标
	metrics := createMetrics(promClient)

	// 运行业务模拟循环
	log.Println("Starting business simulation...")
	runBusinessSimulation(metrics, sentryClient)
}

// initPrometheus 初始化 Prometheus 客户端
func initPrometheus() (*prometheus.Client, error) {
	cfg := &prometheus.Config{
		Namespace: "example_app",
		HTTPServer: prometheus.HTTPServerConfig{
			Enabled: true,
			Addr:    ":9090",
			Path:    "/metrics",
		},
		EnableGoCollector:      true,
		EnableProcessCollector: true,
	}

	return prometheus.New(cfg)
}

// initSentry 初始化 Sentry 客户端
func initSentry() (*sentry.Client, error) {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		log.Println("SENTRY_DSN not set, Sentry will be disabled")
		// 返回一个 nil 客户端，后续代码需要处理
		return nil, nil
	}

	cfg := &sentry.Config{
		DSN:              dsn,
		Environment:      "observability-example",
		Release:          "v1.0.0",
		ServerName:       "example-app",
		SampleRate:       1.0,
		AttachStacktrace: true,
		Debug:            true,
	}

	return sentry.New(cfg)
}

// Metrics 指标集合
type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	errorsTotal     *prometheus.CounterVec
	panicsTotal     *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
}

// createMetrics 创建 Prometheus 指标
func createMetrics(client *prometheus.Client) *Metrics {
	requestsTotal, _ := client.NewCounter(
		"requests_total",
		"Total number of requests processed",
		nil,
	)

	errorsTotal, _ := client.NewCounter(
		"errors_total",
		"Total number of errors occurred",
		nil,
	)

	panicsTotal, _ := client.NewCounter(
		"panics_total",
		"Total number of panics recovered",
		nil,
	)

	requestDuration, _ := client.NewHistogram(
		"request_duration_seconds",
		"Request duration in seconds",
		nil,
		[]float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0},
	)

	return &Metrics{
		requestsTotal:   requestsTotal,
		errorsTotal:     errorsTotal,
		panicsTotal:     panicsTotal,
		requestDuration: requestDuration,
	}
}

// runBusinessSimulation 运行业务模拟
func runBusinessSimulation(metrics *Metrics, sentryClient *sentry.Client) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// 模拟业务请求
		simulateRequest(metrics, sentryClient)
	}
}

// simulateRequest 模拟单个业务请求
func simulateRequest(metrics *Metrics, sentryClient *sentry.Client) {
	start := time.Now()

	// 增加请求计数
	metrics.requestsTotal.WithLabelValues().Inc()

	// 随机选择场景
	scenario := rand.Intn(100)

	switch {
	case scenario < 65:
		// 65% 正常请求
		simulateNormalRequest()
		log.Println("✓ Normal request processed")

	case scenario < 80:
		// 15% 慢请求
		simulateSlowRequest()
		log.Println("⚠ Slow request processed")

	case scenario < 90:
		// 10% 错误
		err := simulateError()
		metrics.errorsTotal.WithLabelValues().Inc()
		if sentryClient != nil {
			sentryClient.CaptureException(err)
		}
		log.Printf("✗ Error occurred: %v\n", err)

	case scenario < 95:
		// 5% panic
		simulatePanic(metrics, sentryClient)

	default:
		// 5% Goroutine 泄露
		simulateGoroutineLeak()
		log.Println("⚠ Goroutine leak triggered")
	}

	// 记录请求时长
	duration := time.Since(start).Seconds()
	metrics.requestDuration.WithLabelValues().Observe(duration)
}

// simulateNormalRequest 模拟正常请求
func simulateNormalRequest() {
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)
}

// simulateSlowRequest 模拟慢请求
func simulateSlowRequest() {
	time.Sleep(time.Duration(500+rand.Intn(1500)) * time.Millisecond)
}

// simulateError 模拟错误
func simulateError() error {
	errorTypes := []string{
		"database connection timeout",
		"invalid user input",
		"external API failed",
		"cache miss",
	}
	errorType := errorTypes[rand.Intn(len(errorTypes))]
	return errors.New(fmt.Sprintf("simulated error: %s", errorType))
}

// simulatePanic 模拟 panic
func simulatePanic(metrics *Metrics, sentryClient *sentry.Client) {
	defer func() {
		if r := recover(); r != nil {
			metrics.panicsTotal.WithLabelValues().Inc()
			if sentryClient != nil {
				sentryClient.RecoverWithContext(r)
			}
			log.Printf("✗ Panic recovered: %v\n", r)
		}
	}()

	panicTypes := []string{
		"nil pointer dereference",
		"index out of range",
		"division by zero",
	}
	panicMsg := panicTypes[rand.Intn(len(panicTypes))]
	panic(fmt.Sprintf("simulated panic: %s", panicMsg))
}

// simulateGoroutineLeak 模拟 Goroutine 泄露
// 创建一个永不退出的 goroutine（阻塞在 channel 接收）
func simulateGoroutineLeak() {
	// 创建一个永远不会被写入的 channel
	ch := make(chan struct{})

	// 启动一个会永久阻塞的 goroutine
	go func() {
		<-ch // 永远阻塞在这里，导致 goroutine 泄露
	}()

	// 不关闭 channel，也不向其发送数据
	// 这个 goroutine 将永远存活，占用内存
}
