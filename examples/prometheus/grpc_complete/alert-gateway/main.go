package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/notify/feishu"
	"github.com/lk2023060901/xdooria/pkg/notify/webhook"
	"go.uber.org/zap"
)

func main() {
	log.Println("🚀 启动告警网关服务...")

	// 创建 Logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("创建 logger 失败: %v", err)
	}
	defer logger.Sync()

	// 获取配置
	port := getEnv("PORT", "8080")
	feishuWebhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	feishuSecret := getEnv("FEISHU_SECRET", "")

	if feishuWebhookURL == "" {
		log.Fatal("❌ 环境变量 FEISHU_WEBHOOK_URL 未设置")
	}

	// 创建飞书适配器
	feishuAdapter, err := feishu.NewAdapter(&feishu.Config{
		WebhookURL: feishuWebhookURL,
		Secret:     feishuSecret,
		Timeout:    30 * time.Second,
	})
	if err != nil {
		log.Fatalf("❌ 创建飞书适配器失败: %v", err)
	}

	logger.Info("✅ 飞书适配器已创建",
		zap.String("webhook_url", maskWebhookURL(feishuWebhookURL)),
		zap.Bool("has_secret", feishuSecret != ""),
	)

	// 创建 Webhook Handler
	handler := webhook.NewHandler(feishuAdapter, logger)

	// 注册路由
	mux := http.NewServeMux()

	// Grafana Webhook 路由
	mux.HandleFunc("/api/alerts/grafana", handler.HandleGrafana)

	// 健康检查路由
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// 根路径提示
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
<!DOCTYPE html>
<html>
<head>
    <title>告警网关服务</title>
    <meta charset="utf-8">
</head>
<body>
    <h1>🚨 告警网关服务</h1>
    <h2>可用端点：</h2>
    <ul>
        <li><a href="/health">/health</a> - 健康检查</li>
        <li><code>POST /api/alerts/grafana</code> - 接收 Grafana Webhook</li>
    </ul>
    <h2>配置信息：</h2>
    <ul>
        <li>监听端口: ` + port + `</li>
        <li>飞书 Webhook: ` + maskWebhookURL(feishuWebhookURL) + `</li>
        <li>签名验证: ` + boolToYesNo(feishuSecret != "") + `</li>
    </ul>
</body>
</html>
		`))
	})

	// 启动 HTTP 服务器
	addr := fmt.Sprintf(":%s", port)
	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// 启动服务器（异步）
	go func() {
		logger.Info("✅ 告警网关服务已启动",
			zap.String("addr", addr),
			zap.String("grafana_endpoint", fmt.Sprintf("http://0.0.0.0:%s/api/alerts/grafana", port)),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("❌ HTTP 服务器启动失败", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("🛑 关闭告警网关服务...")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// maskWebhookURL 隐藏 Webhook URL 的敏感部分
func maskWebhookURL(url string) string {
	if len(url) > 50 {
		return url[:30] + "***" + url[len(url)-10:]
	}
	return url
}

// boolToYesNo 将布尔值转换为中文是/否
func boolToYesNo(b bool) string {
	if b {
		return "是"
	}
	return "否"
}
