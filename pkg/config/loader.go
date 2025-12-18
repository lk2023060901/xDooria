// pkg/config/loader.go
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Loader 配置加载器
type Loader struct {
	viper *viper.Viper
}

// NewLoader 创建配置加载器
func NewLoader() *Loader {
	return &Loader{
		viper: viper.New(),
	}
}

// LoadFile 加载配置文件
// configType: "yaml" 或 "json"
func (l *Loader) LoadFile(configPath string, configType string) error {
	// 设置配置文件
	l.viper.SetConfigFile(configPath)
	l.viper.SetConfigType(configType)

	// 设置环境变量前缀（仅 YAML 服务器配置支持）
	if configType == "yaml" {
		l.viper.SetEnvPrefix("XDOORIA")
		l.viper.AutomaticEnv()
		l.viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	}

	// 读取配置文件
	if err := l.viper.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	return nil
}

// Unmarshal 解析整个配置到结构体
func (l *Loader) Unmarshal(target interface{}) error {
	if err := l.viper.Unmarshal(target); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// UnmarshalKey 解析配置中的某个 key 到结构体
func (l *Loader) UnmarshalKey(key string, target interface{}) error {
	if err := l.viper.UnmarshalKey(key, target); err != nil {
		return fmt.Errorf("failed to unmarshal key %s: %w", key, err)
	}
	return nil
}
