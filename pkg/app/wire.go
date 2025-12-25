package app

import (
	"github.com/google/wire"
)

// AppComponents 用于收集 Wire 注入的所有组件
type AppComponents struct {
	Servers []Server
	Closers []Closer
}

// ProviderSet 导出给 Wire 使用
var ProviderSet = wire.NewSet(
	NewBaseApp,
)

// InitApp 是一个辅助函数，用于将 Wire 注入的组件绑定到 BaseApp
func InitApp(app *BaseApp, comps AppComponents) Application {
	app.AppendServer(comps.Servers...)
	app.AppendCloser(comps.Closers...)
	return app
}

// MapCloser 将实现了 Close() error 的对象转换为 Closer 接口
func MapCloser(c interface{ Close() error }) Closer {
	return closerWrapper{c}
}

// MapServer 将 Server 实例收集到 slice 中（用于 Wire）
func MapServer(s Server) Server {
	return s
}

type closerWrapper struct {
	obj interface{ Close() error }
}

func (w closerWrapper) Close() error {
	return w.obj.Close()
}
