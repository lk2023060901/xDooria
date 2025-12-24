// pkg/websocket/handler.go
package websocket

// MessageHandler 消息处理器接口
type MessageHandler interface {
	// OnConnect 连接建立时的回调
	OnConnect(conn *Connection) error

	// OnMessage 收到消息时的回调
	OnMessage(conn *Connection, msg *Message) error

	// OnDisconnect 连接断开时的回调
	OnDisconnect(conn *Connection, err error)

	// OnError 发生错误时的回调
	OnError(conn *Connection, err error)
}

// BaseHandler 基础处理器（提供空实现，可嵌入使用）
type BaseHandler struct{}

// OnConnect 连接建立
func (h *BaseHandler) OnConnect(conn *Connection) error {
	return nil
}

// OnMessage 收到消息
func (h *BaseHandler) OnMessage(conn *Connection, msg *Message) error {
	return nil
}

// OnDisconnect 连接断开
func (h *BaseHandler) OnDisconnect(conn *Connection, err error) {}

// OnError 发生错误
func (h *BaseHandler) OnError(conn *Connection, err error) {}

// HandlerFunc 消息处理函数类型
type HandlerFunc func(conn *Connection, msg *Message) error

// FuncHandler 函数式处理器
type FuncHandler struct {
	onConnect    func(conn *Connection) error
	onMessage    func(conn *Connection, msg *Message) error
	onDisconnect func(conn *Connection, err error)
	onError      func(conn *Connection, err error)
}

// NewFuncHandler 创建函数式处理器
func NewFuncHandler() *FuncHandler {
	return &FuncHandler{}
}

// OnConnectFunc 设置连接回调
func (h *FuncHandler) OnConnectFunc(fn func(conn *Connection) error) *FuncHandler {
	h.onConnect = fn
	return h
}

// OnMessageFunc 设置消息回调
func (h *FuncHandler) OnMessageFunc(fn func(conn *Connection, msg *Message) error) *FuncHandler {
	h.onMessage = fn
	return h
}

// OnDisconnectFunc 设置断开回调
func (h *FuncHandler) OnDisconnectFunc(fn func(conn *Connection, err error)) *FuncHandler {
	h.onDisconnect = fn
	return h
}

// OnErrorFunc 设置错误回调
func (h *FuncHandler) OnErrorFunc(fn func(conn *Connection, err error)) *FuncHandler {
	h.onError = fn
	return h
}

// OnConnect 实现 MessageHandler 接口
func (h *FuncHandler) OnConnect(conn *Connection) error {
	if h.onConnect != nil {
		return h.onConnect(conn)
	}
	return nil
}

// OnMessage 实现 MessageHandler 接口
func (h *FuncHandler) OnMessage(conn *Connection, msg *Message) error {
	if h.onMessage != nil {
		return h.onMessage(conn, msg)
	}
	return nil
}

// OnDisconnect 实现 MessageHandler 接口
func (h *FuncHandler) OnDisconnect(conn *Connection, err error) {
	if h.onDisconnect != nil {
		h.onDisconnect(conn, err)
	}
}

// OnError 实现 MessageHandler 接口
func (h *FuncHandler) OnError(conn *Connection, err error) {
	if h.onError != nil {
		h.onError(conn, err)
	}
}
