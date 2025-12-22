package feishu

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/lk2023060901/xdooria/pkg/notify"
	"github.com/stretchr/testify/require"
)

// getWebhookURL 从环境变量获取 Webhook URL
func getWebhookURL(t *testing.T) string {
	webhookURL := os.Getenv("FEISHU_WEBHOOK_URL")
	if webhookURL == "" {
		t.Skip("跳过测试：请设置环境变量 FEISHU_WEBHOOK_URL")
	}
	return webhookURL
}

// TestAdapter_SendToRealFeishu 发送真实消息到飞书
// 使用 FEISHU_WEBHOOK_URL=https://... go test -v -run TestAdapter_SendToRealFeishu 运行
func TestAdapter_SendToRealFeishu(t *testing.T) {
	webhookURL := getWebhookURL(t)

	// 创建适配器（不配置 Secret，测试无签名模式）
	adapter, err := NewAdapter(&Config{
		WebhookURL: webhookURL,
		Secret:     "", // 如果你的机器人配置了签名验证，请填写 Secret
	})
	require.NoError(t, err)

	// 构建测试告警
	alert := &notify.Alert{
		Level:   notify.AlertLevelCritical,
		Service: "游戏服务",
		Summary: "Kafka 消费者延迟告警",
		Description: "聊天消息队列堆积严重，当前延迟: 15,234 条消息\n" +
			"建议立即检查消费者进程状态并考虑扩容。",
		Labels: map[string]string{
			"topic":          "chat.world",
			"consumer_group": "chat-handler",
			"severity":       "critical",
			"env":            "production",
		},
		StartsAt:     time.Now(),
		DashboardURL: "http://grafana.example.com/d/kafka-overview",
		RunbookURL:   "http://wiki.example.com/runbook/kafka-lag",
		AtUsers: []notify.User{
			{
				ID:   "ou_backend_oncall", // 如果知道飞书 user_id，可以填写
				Name: "后端值班工程师",
			},
		},
	}

	// 发送告警
	t.Log("正在发送消息到飞书...")
	err = adapter.Send(context.Background(), alert)
	require.NoError(t, err)
	t.Log("✅ 消息发送成功！请检查飞书群聊")
}

// TestAdapter_SendSimpleText 发送简单文本消息
func TestAdapter_SendSimpleText(t *testing.T) {
	webhookURL := getWebhookURL(t)

	adapter, err := NewAdapter(&Config{
		WebhookURL: webhookURL,
		Timeout:    30 * time.Second, // 增加超时时间
	})
	require.NoError(t, err)

	// 简单告警
	alert := &notify.Alert{
		Level:   notify.AlertLevelInfo,
		Service: "测试服务",
		Summary: "这是一条来自 pkg/notify 的测试消息 🎉",
	}

	t.Log("正在发送简单消息...")
	err = adapter.Send(context.Background(), alert)
	require.NoError(t, err)
	t.Log("✅ 消息发送成功！")
}

// TestAdapter_SendBatch 批量发送测试
func TestAdapter_SendBatch(t *testing.T) {
	webhookURL := getWebhookURL(t)

	adapter, err := NewAdapter(&Config{
		WebhookURL: webhookURL,
		Timeout:    30 * time.Second,
	})
	require.NoError(t, err)

	alerts := []*notify.Alert{
		{
			Level:   notify.AlertLevelCritical,
			Service: "订单服务",
			Summary: "订单处理失败率超过 5%",
		},
		{
			Level:   notify.AlertLevelWarning,
			Service: "库存服务",
			Summary: "库存同步延迟 30 秒",
		},
		{
			Level:   notify.AlertLevelInfo,
			Service: "支付服务",
			Summary: "支付成功率恢复正常",
		},
	}

	t.Log("正在批量发送 3 条消息...")
	err = adapter.SendBatch(context.Background(), alerts)
	require.NoError(t, err)
	t.Log("✅ 批量发送成功！")
}

// TestAdapter_SendWithMultipleAt 测试 @ 多个具体用户
func TestAdapter_SendWithMultipleAt(t *testing.T) {
	webhookURL := getWebhookURL(t)

	adapter, err := NewAdapter(&Config{
		WebhookURL: webhookURL,
		Timeout:    30 * time.Second,
	})
	require.NoError(t, err)

	// 测试 @ 多个具体用户（user_id）
	alert := &notify.Alert{
		Level:   notify.AlertLevelCritical,
		Service: "数据库服务",
		Summary: "Redis 主节点宕机",
		Description: "主节点连接失败，已自动切换到从节点\n需要立即检查并恢复主节点",
		Labels: map[string]string{
			"cluster": "redis-prod-01",
			"node":    "master",
		},
		StartsAt: time.Now(),
		// @ 多个用户（如果你知道飞书群成员的 user_id，可以替换这里）
		AtUsers: []notify.User{
			{ID: "ou_user_001", Name: "张三"},
			{ID: "ou_user_002", Name: "李四"},
			{ID: "ou_user_003", Name: "王五"},
		},
	}

	t.Log("正在发送 @ 多个用户的消息（张三、李四、王五）...")
	t.Log("注意：如果这些 user_id 不存在，@ 不会生效，但消息会正常发送")
	err = adapter.Send(context.Background(), alert)
	require.NoError(t, err)
	t.Log("✅ 消息发送成功！")
}

// TestAdapter_SendAtAll 测试 @ 所有人
func TestAdapter_SendAtAll(t *testing.T) {
	webhookURL := getWebhookURL(t)

	adapter, err := NewAdapter(&Config{
		WebhookURL: webhookURL,
		Timeout:    30 * time.Second,
	})
	require.NoError(t, err)

	// 使用 AtAll 标志
	alert := &notify.Alert{
		Level:   notify.AlertLevelCritical,
		Service: "生产环境",
		Summary: "紧急：系统全局故障",
		Description: "所有微服务无法访问数据库\n正在排查中...",
		AtAll:   true, // @ 所有人
	}

	t.Log("正在发送 @ 所有人的紧急告警...")
	err = adapter.Send(context.Background(), alert)
	require.NoError(t, err)
	t.Log("✅ 消息发送成功！")
}
