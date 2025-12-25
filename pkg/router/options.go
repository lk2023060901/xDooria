package router

// Options 路由与桥接配置
type Options struct {
	// SendQueueSize 发送队列（Channel）的长度
	SendQueueSize int 
	// RecvQueueSize 接收队列（Channel）的长度
	RecvQueueSize int 
}

type Option func(*Options)

func DefaultOptions() Options {
	return Options{
		SendQueueSize: 1024,
		RecvQueueSize: 1024,
	}
}

func WithSendQueueSize(size int) Option {
	return func(o *Options) { o.SendQueueSize = size }
}

func WithRecvQueueSize(size int) Option {
	return func(o *Options) { o.RecvQueueSize = size }
}