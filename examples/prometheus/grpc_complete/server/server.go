package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	log.Println("🎮 启动直播间监控服务...")

	// 获取配置
	metricsPort := getEnv("METRICS_PORT", "9100")

	// 创建 Prometheus 注册器
	reg := prometheus.NewRegistry()

	// 注册 Go runtime 指标（监控 goroutine、内存等）
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))

	// 创建指标
	liveMetrics := NewLiveStreamMetrics(reg)
	streamerMetrics := NewStreamerMetrics(reg)
	viewerMetrics := NewViewerMetrics(reg)

	// 创建并启动直播间模拟器
	liveSimulator := NewLiveStreamSimulator(liveMetrics, streamerMetrics, viewerMetrics)
	liveSimulator.Start()

	// 定期打印运行时统计信息
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			log.Printf("📊 运行时统计 - Goroutines: %d, 堆内存: %.2f MB, 系统内存: %.2f MB",
				runtime.NumGoroutine(),
				float64(m.Alloc)/1024/1024,
				float64(m.Sys)/1024/1024,
			)
		}
	}()

	// 启动 Prometheus HTTP 服务器
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{
			Registry: reg,
		}))

		addr := fmt.Sprintf(":%s", metricsPort)
		log.Printf("Prometheus metrics 服务启动在 http://0.0.0.0:%s/metrics\n", metricsPort)

		if err := http.ListenAndServe(addr, mux); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 关闭服务器...")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
