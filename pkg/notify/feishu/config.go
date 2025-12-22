package feishu

import (
	"fmt"
	"strings"
	"time"

	"github.com/lk2023060901/xdooria/pkg/notify"
)

// Config 飞书机器人配置
type Config struct {
	// WebhookURL Webhook 地址（必填）
	WebhookURL string `mapstructure:"webhook_url" json:"webhook_url"`

	// Secret 签名密钥（可选，生产环境建议配置）
	Secret string `mapstructure:"secret" json:"secret"`

	// Timeout HTTP 请求超时时间（默认 5 秒）
	Timeout time.Duration `mapstructure:"timeout" json:"timeout"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout: 5 * time.Second,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.WebhookURL == "" {
		return fmt.Errorf("%w: webhook_url is required", notify.ErrInvalidConfig)
	}

	// WebhookURL 必须是 HTTP(S) 协议
	if !strings.HasPrefix(c.WebhookURL, "http://") &&
		!strings.HasPrefix(c.WebhookURL, "https://") {
		return fmt.Errorf("%w: webhook_url must start with http:// or https://", notify.ErrInvalidConfig)
	}

	// Secret 可选（不配置时跳过签名）

	if c.Timeout <= 0 {
		c.Timeout = 5 * time.Second
	}

	return nil
}
