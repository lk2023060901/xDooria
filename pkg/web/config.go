package web

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Config Web 服务配置
type Config struct {
	Port         int           `mapstructure:"port"`
	Mode         string        `mapstructure:"mode"` // debug, release, test
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
	EnableTLS    bool          `mapstructure:"enable_tls"`
	CertFile     string        `mapstructure:"cert_file"`
	KeyFile      string        `mapstructure:"key_file"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Port:         8080,
		Mode:         gin.ReleaseMode,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		EnableTLS:    false,
	}
}
