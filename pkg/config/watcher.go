// pkg/config/watcher.go
package config

import (
	"fmt"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// Watcher 配置监听器（用于热更新）
type Watcher[T any] struct {
	loader     *Loader
	configPath string
	configType string
	callbacks  []func(*T)
	mu         sync.RWMutex
	config     *T
}

// NewWatcher 创建配置监听器（泛型版本）
// configPath: 配置文件路径
// configType: 配置类型 "yaml" 或 "json"
func NewWatcher[T any](configPath string, configType string) (*Watcher[T], error) {
	loader := NewLoader()

	// 使用 Loader 加载配置
	if err := loader.LoadFile(configPath, configType); err != nil {
		return nil, err
	}

	var cfg T
	if err := loader.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	watcher := &Watcher[T]{
		loader:     loader,
		configPath: configPath,
		configType: configType,
		callbacks:  make([]func(*T), 0),
		config:     &cfg,
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
	w.loader.viper.WatchConfig()
	w.loader.viper.OnConfigChange(func(e fsnotify.Event) {
		// 重新加载配置
		newLoader := NewLoader()
		if err := newLoader.LoadFile(w.configPath, w.configType); err != nil {
			fmt.Printf("failed to reload config: %v\n", err)
			return
		}

		var newCfg T
		if err := newLoader.Unmarshal(&newCfg); err != nil {
			fmt.Printf("failed to unmarshal config: %v\n", err)
			return
		}

		// 更新配置
		w.mu.Lock()
		w.config = &newCfg
		w.loader = newLoader
		callbacks := w.callbacks
		w.mu.Unlock()

		// 触发回调
		for _, callback := range callbacks {
			callback(&newCfg)
		}
	})
}

// Stop 停止监听
func (w *Watcher[T]) Stop() {
	// Viper 没有提供停止监听的方法
	// 实际使用中可以通过 context 控制
}
