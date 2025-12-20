package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Manager 配置管理器接口
type Manager interface {
	// LoadFile 加载配置文件
	LoadFile(path string) error
	// BindEnv 绑定环境变量（支持自动映射）
	BindEnv(prefix string)
	// Unmarshal 解析整个配置到结构体
	Unmarshal(v any) error
	// UnmarshalKey 解析指定路径的配置到结构体或基本类型
	// key 可以是 "database.postgres" (获取 struct)
	// 也可以是 "server.port" (获取 int)
	UnmarshalKey(key string, v any) error
	// Get 获取配置值（返回 any）
	Get(key string) any
	// GetString 获取字符串配置
	GetString(key string) string
	// GetInt 获取整数配置
	GetInt(key string) int
	// GetBool 获取布尔配置
	GetBool(key string) bool
	// Watch 监听配置文件变化
	Watch(callback func()) error
	// IsSet 检查配置项是否存在
	IsSet(key string) bool
	// AllSettings 获取所有配置（以 map 形式）
	AllSettings() map[string]any
}

// manager 配置管理器实现
type manager struct {
	v         *viper.Viper
	mu        sync.RWMutex
	callbacks []func()
}

// NewManager 创建配置管理器
func NewManager(opts ...Option) Manager {
	m := &manager{
		v:         viper.New(),
		callbacks: make([]func(), 0),
	}

	// 应用选项
	for _, opt := range opts {
		opt(m)
	}

	return m
}

// LoadFile 加载配置文件（支持 YAML、JSON、TOML 等）
func (m *manager) LoadFile(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.v.SetConfigFile(path)

	if err := m.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	return nil
}

// BindEnv 绑定环境变量
// prefix: 环境变量前缀，如 "APP" 会匹配 APP_DATABASE_HOST
func (m *manager) BindEnv(prefix string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if prefix != "" {
		m.v.SetEnvPrefix(prefix)
	}
	m.v.AutomaticEnv()
	m.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

// Unmarshal 解析整个配置到结构体
func (m *manager) Unmarshal(v any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.v.Unmarshal(v); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// UnmarshalKey 解析指定路径的配置
// 支持解析到 struct 或基本类型
func (m *manager) UnmarshalKey(key string, v any) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := m.v.UnmarshalKey(key, v); err != nil {
		return fmt.Errorf("failed to unmarshal key %s: %w", key, err)
	}
	return nil
}

// Get 获取配置值
func (m *manager) Get(key string) any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.Get(key)
}

// GetString 获取字符串配置
func (m *manager) GetString(key string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.GetString(key)
}

// GetInt 获取整数配置
func (m *manager) GetInt(key string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.GetInt(key)
}

// GetBool 获取布尔配置
func (m *manager) GetBool(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.GetBool(key)
}

// Watch 监听配置文件变化
func (m *manager) Watch(callback func()) error {
	m.mu.Lock()
	m.callbacks = append(m.callbacks, callback)
	m.mu.Unlock()

	m.v.WatchConfig()
	m.v.OnConfigChange(func(e fsnotify.Event) {
		m.mu.RLock()
		callbacks := m.callbacks
		m.mu.RUnlock()

		// 触发所有回调
		for _, cb := range callbacks {
			cb()
		}
	})

	return nil
}

// IsSet 检查配置项是否存在
func (m *manager) IsSet(key string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.IsSet(key)
}

// AllSettings 获取所有配置
func (m *manager) AllSettings() map[string]any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.v.AllSettings()
}
