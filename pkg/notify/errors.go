package notify

import "errors"

var (
	// ErrNoNotifiers 没有可用的通知器
	ErrNoNotifiers = errors.New("no notifiers configured")

	// ErrInvalidConfig 配置无效
	ErrInvalidConfig = errors.New("invalid notifier config")

	// ErrSendFailed 发送失败
	ErrSendFailed = errors.New("failed to send notification")

	// ErrWebhookEmpty Webhook URL 为空
	ErrWebhookEmpty = errors.New("webhook url is empty")

	// ErrSecretEmpty 签名密钥为空
	ErrSecretEmpty = errors.New("secret is empty")
)
