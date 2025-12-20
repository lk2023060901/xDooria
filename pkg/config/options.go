package config

import "github.com/spf13/viper"

// Option 配置选项函数
type Option func(*manager)

// WithDefaults 设置默认配置值
func WithDefaults(defaults map[string]any) Option {
	return func(m *manager) {
		for key, value := range defaults {
			m.v.SetDefault(key, value)
		}
	}
}

// WithConfigType 设置配置文件类型（yaml、json、toml 等）
func WithConfigType(configType string) Option {
	return func(m *manager) {
		m.v.SetConfigType(configType)
	}
}

// WithConfigName 设置配置文件名（不含扩展名）
func WithConfigName(name string) Option {
	return func(m *manager) {
		m.v.SetConfigName(name)
	}
}

// WithConfigPaths 添加配置文件搜索路径
func WithConfigPaths(paths ...string) Option {
	return func(m *manager) {
		for _, path := range paths {
			m.v.AddConfigPath(path)
		}
	}
}

// WithEnvPrefix 设置环境变量前缀
func WithEnvPrefix(prefix string) Option {
	return func(m *manager) {
		if prefix != "" {
			m.v.SetEnvPrefix(prefix)
			m.v.AutomaticEnv()
		}
	}
}

// WithViper 使用自定义的 Viper 实例
func WithViper(v *viper.Viper) Option {
	return func(m *manager) {
		m.v = v
	}
}
