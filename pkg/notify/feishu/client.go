package feishu

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// Client 飞书客户端
type Client struct {
	config *Config
	client *http.Client
}

// NewClient 创建飞书客户端
func NewClient(cfg *Config) (*Client, error) {
	// 合并默认配置
	newCfg, err := config.MergeConfig(DefaultConfig(), cfg)
	if err != nil {
		return nil, err
	}

	// 验证配置
	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		config: newCfg,
		client: &http.Client{
			Timeout: newCfg.Timeout,
		},
	}, nil
}

// Send 发送消息
func (c *Client) Send(msg Message) error {
	// 1. 构建请求体
	payload := map[string]interface{}{
		"msg_type": msg.Type(),
		"content":  msg.Content(),
	}

	// 2. 添加签名（如果配置了 Secret）
	if c.config.Secret != "" {
		timestamp := time.Now().Unix()
		sign := c.genSign(timestamp)
		payload["timestamp"] = fmt.Sprintf("%d", timestamp)
		payload["sign"] = sign
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}

	// 3. 发送 HTTP 请求
	resp, err := c.client.Post(c.config.WebhookURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRequestFailed, err)
	}
	defer resp.Body.Close()

	// 4. 检查响应
	respBody, _ := io.ReadAll(resp.Body)

	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("%w: %v", ErrResponseInvalid, err)
	}

	if result.Code != 0 {
		return fmt.Errorf("%w: %s (code=%d)", ErrAPIError, result.Msg, result.Code)
	}

	return nil
}

// genSign 生成签名（符合飞书官方规范）
// 参考: https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
func (c *Client) genSign(timestamp int64) string {
	// 拼接 timestamp + "\n" + secret
	stringToSign := fmt.Sprintf("%d\n%s", timestamp, c.config.Secret)

	// 使用 HmacSHA256 算法计算签名（密钥作为 HMAC key）
	h := hmac.New(sha256.New, []byte(stringToSign))
	h.Write([]byte{})

	// Base64 编码
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature
}
