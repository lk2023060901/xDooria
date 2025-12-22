package webhook_test

import (
	"fmt"
	"net/http"

	"github.com/lk2023060901/xdooria/pkg/notify/feishu"
	"github.com/lk2023060901/xdooria/pkg/notify/webhook"
	"go.uber.org/zap"
)

// Example_grafanaIntegration 演示如何在 HTTP 服务中集成 Grafana Webhook
func Example_grafanaIntegration() {
	// 1. 创建飞书适配器
	feishuAdapter, err := feishu.NewAdapter(&feishu.Config{
		WebhookURL: "https://open.feishu.cn/open-apis/bot/v2/hook/your-webhook-url",
	})
	if err != nil {
		panic(err)
	}

	// 2. 创建 Webhook Handler
	logger, _ := zap.NewProduction()
	handler := webhook.NewHandler(feishuAdapter, logger)

	// 3. 注册 HTTP 路由
	http.HandleFunc("/api/alerts/grafana", handler.HandleGrafana)

	// 4. 启动 HTTP 服务
	fmt.Println("启动告警接收服务，监听 :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}

	// 5. 在 Grafana 中配置 Webhook Contact Point
	// URL: http://your-service:8080/api/alerts/grafana
}
