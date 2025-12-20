package etcd

import (
	"context"
	"sync"

	clientv3 "go.etcd.io/etcd/client/v3"
)

// Watcher 监听客户端
type Watcher struct {
	client  *Client
	watcher clientv3.Watcher
	mu      sync.RWMutex
	watches map[string]context.CancelFunc
}

// newWatcher 创建 Watcher 客户端
func newWatcher(client *Client) *Watcher {
	return &Watcher{
		client:  client,
		watcher: clientv3.NewWatcher(client.rawClient()),
		watches: make(map[string]context.CancelFunc),
	}
}

// Watch 监听单个键的变化
func (w *Watcher) Watch(ctx context.Context, key string, handler func(*WatchEvent)) error {
	return w.watch(ctx, key, false, handler)
}

// WatchPrefix 监听前缀匹配的所有键
func (w *Watcher) WatchPrefix(ctx context.Context, prefix string, handler func(*WatchEvent)) error {
	return w.watch(ctx, prefix, true, handler)
}

// WatchWithRevision 从指定版本开始监听
func (w *Watcher) WatchWithRevision(ctx context.Context, key string, revision int64, handler func(*WatchEvent)) error {
	opts := []clientv3.OpOption{
		clientv3.WithRev(revision),
	}

	watchCh := w.watcher.Watch(ctx, key, opts...)
	return w.processWatchEvents(ctx, key, watchCh, handler)
}

// watch 内部监听实现
func (w *Watcher) watch(ctx context.Context, key string, isPrefix bool, handler func(*WatchEvent)) error {
	// 检查是否已经在监听
	w.mu.RLock()
	if _, exists := w.watches[key]; exists {
		w.mu.RUnlock()
		return nil // 已经在监听
	}
	w.mu.RUnlock()

	// 创建可取消的上下文
	watchCtx, cancel := context.WithCancel(ctx)

	// 注册取消函数
	w.mu.Lock()
	w.watches[key] = cancel
	w.mu.Unlock()

	// 启动监听
	opts := []clientv3.OpOption{}
	if isPrefix {
		opts = append(opts, clientv3.WithPrefix())
	}

	watchCh := w.watcher.Watch(watchCtx, key, opts...)

	// 处理事件
	go func() {
		defer func() {
			w.mu.Lock()
			delete(w.watches, key)
			w.mu.Unlock()
		}()

		if err := w.processWatchEvents(watchCtx, key, watchCh, handler); err != nil {
			// 监听出错，可以记录日志
		}
	}()

	return nil
}

// processWatchEvents 处理监听事件
func (w *Watcher) processWatchEvents(ctx context.Context, key string, watchCh clientv3.WatchChan, handler func(*WatchEvent)) error {
	for {
		select {
		case <-ctx.Done():
			return ErrContextCanceled

		case resp, ok := <-watchCh:
			if !ok {
				return ErrWatchClosed
			}

			if resp.Canceled {
				return ErrWatchClosed
			}

			if err := resp.Err(); err != nil {
				return err
			}

			// 处理所有事件
			for _, ev := range resp.Events {
				event := &WatchEvent{
					Key:      string(ev.Kv.Key),
					Value:    ev.Kv.Value,
					Revision: ev.Kv.ModRevision,
				}

				switch ev.Type {
				case clientv3.EventTypePut:
					event.Type = EventTypePut
				case clientv3.EventTypeDelete:
					event.Type = EventTypeDelete
				}

				handler(event)
			}
		}
	}
}

// StopWatch 停止监听
func (w *Watcher) StopWatch(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if cancel, exists := w.watches[key]; exists {
		cancel()
		delete(w.watches, key)
	}
}

// StopAll 停止所有监听
func (w *Watcher) StopAll() {
	w.mu.Lock()
	defer w.mu.Unlock()

	for key, cancel := range w.watches {
		cancel()
		delete(w.watches, key)
	}
}

// Close 关闭 Watcher
func (w *Watcher) Close() error {
	w.StopAll()
	return w.watcher.Close()
}

// WatchChan 返回原始监听通道（高级用法）
func (w *Watcher) WatchChan(ctx context.Context, key string, opts ...clientv3.OpOption) clientv3.WatchChan {
	return w.watcher.Watch(ctx, key, opts...)
}
