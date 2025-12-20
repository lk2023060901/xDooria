package sentry

import (
	"os"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	defaultClient *Client
	defaultOnce   sync.Once
	defaultMu     sync.RWMutex
)

// InitDefault 初始化默认客户端
func InitDefault(cfg *Config) error {
	var err error
	defaultOnce.Do(func() {
		defaultMu.Lock()
		defer defaultMu.Unlock()

		defaultClient, err = New(cfg)
	})
	return err
}

// InitDefaultFromEnv 从环境变量初始化默认客户端
// 环境变量前缀: SENTRY_
func InitDefaultFromEnv() error {
	cfg := DefaultConfig()

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		cfg.DSN = dsn
	}
	if env := os.Getenv("SENTRY_ENVIRONMENT"); env != "" {
		cfg.Environment = env
	}
	if release := os.Getenv("SENTRY_RELEASE"); release != "" {
		cfg.Release = release
	}
	if serverName := os.Getenv("SENTRY_SERVER_NAME"); serverName != "" {
		cfg.ServerName = serverName
	}

	return InitDefault(cfg)
}

// Default 获取默认客户端
func Default() *Client {
	defaultMu.RLock()
	defer defaultMu.RUnlock()
	return defaultClient
}

// CaptureException 使用默认客户端捕获异常
func CaptureException(err error) *sentry.EventID {
	if c := Default(); c != nil {
		return c.CaptureException(err)
	}
	return nil
}

// CaptureMessage 使用默认客户端捕获消息
func CaptureMessage(message string, level Level) *sentry.EventID {
	if c := Default(); c != nil {
		return c.CaptureMessage(message, level)
	}
	return nil
}

// Recover 使用默认客户端恢复 panic
func Recover() {
	if c := Default(); c != nil {
		c.Recover()
	}
}

// RecoverWithContext 使用默认客户端恢复 panic（不重新抛出）
func RecoverWithContext(recovered interface{}) *sentry.EventID {
	if c := Default(); c != nil {
		return c.RecoverWithContext(recovered)
	}
	return nil
}

// Flush 刷新默认客户端
func Flush(timeout time.Duration) bool {
	if c := Default(); c != nil {
		return c.Flush(timeout)
	}
	return false
}

// Close 关闭默认客户端
func Close() error {
	defaultMu.Lock()
	defer defaultMu.Unlock()

	if defaultClient != nil {
		err := defaultClient.Close()
		defaultClient = nil
		return err
	}
	return nil
}
