// pkg/config/watcher.go
package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Watcher 泛型配置监听器（用于热更新）
//
// 适用场景：业务应用、明确配置结构、需要类型安全
//
// 使用示例：
//
//	type AppConfig struct {
//	    Server ServerConfig `mapstructure:"server"`
//	    DB     DBConfig     `mapstructure:"db"`
//	}
//
//	watcher, err := config.NewWatcher[AppConfig]("config.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// 获取配置（线程安全）
//	cfg := watcher.GetConfig()
//
//	// 监听配置变化
//	watcher.OnChange(func(cfg *AppConfig) {
//	    log.Println("config changed:", cfg.Server.Port)
//	})
type Watcher[T any] struct {
	v         *viper.Viper
	callbacks []func(*T)
	mu        sync.RWMutex
	config    *T
}

// NewWatcher 创建泛型配置监听器
// configPath: 配置文件路径（支持 yaml、json、toml）
func NewWatcher[T any](configPath string, opts ...Option) (*Watcher[T], error) {
	v := viper.New()
	v.SetConfigFile(configPath)

	// 应用选项
	m := &manager{v: v}
	for _, opt := range opts {
		opt(m)
	}

	// 读取配置
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	watcher := &Watcher[T]{
		v:         v,
		callbacks: make([]func(*T), 0),
		config:    &cfg,
	}

	// 启动监听
	watcher.watch()

	return watcher, nil
}

// GetConfig 获取当前配置（线程安全）
func (w *Watcher[T]) GetConfig() *T {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}

// OnChange 注册配置变化回调
func (w *Watcher[T]) OnChange(callback func(*T)) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.callbacks = append(w.callbacks, callback)
}

// watch 监听配置变化
func (w *Watcher[T]) watch() {
	w.v.WatchConfig()
	w.v.OnConfigChange(func(e fsnotify.Event) {
		var newCfg T
		if err := w.v.Unmarshal(&newCfg); err != nil {
			fmt.Printf("config: failed to unmarshal on change: %v\n", err)
			return
		}

		// 更新配置
		w.mu.Lock()
		w.config = &newCfg
		callbacks := w.callbacks
		w.mu.Unlock()

		// 触发回调
		for _, callback := range callbacks {
			callback(&newCfg)
		}
	})
}

// Reload 手动重新加载配置
func (w *Watcher[T]) Reload() error {
	if err := w.v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	var newCfg T
	if err := w.v.Unmarshal(&newCfg); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	w.mu.Lock()
	w.config = &newCfg
	callbacks := w.callbacks
	w.mu.Unlock()

	// 触发回调
	for _, callback := range callbacks {
		callback(&newCfg)
	}

	return nil
}
