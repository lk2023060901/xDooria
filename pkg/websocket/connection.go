// pkg/websocket/connection.go
package websocket

import (
	"context"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/serializer"
)

// Connection WebSocket 连接封装
type Connection struct {
	id   string
	conn *websocket.Conn

	// 配置
	readTimeout   time.Duration
	writeTimeout  time.Duration
	sendQueueSize int

	// 发送队列
	sendChan chan *Message

	// 日志和序列化
	logger     logger.Logger
	serializer serializer.Serializer

	// 元数据
	metadata sync.Map

	// 状态
	mu         sync.RWMutex
	state      ConnectionState
	closed     atomic.Bool
	closeChan  chan struct{}
	closeOnce  sync.Once
	closeError error

	// 连接信息
	remoteAddr  string
	localAddr   string
	connectedAt time.Time

	// 订阅者（Channel 模式）
	subscribers   sync.Map // map[string]chan *Message
	subscriberSeq atomic.Int64
}

// NewConnection 创建连接
func NewConnection(conn *websocket.Conn, opts ...ConnectionOption) *Connection {
	c := &Connection{
		id:            uuid.New().String(),
		conn:          conn,
		readTimeout:   60 * time.Second,
		writeTimeout:  10 * time.Second,
		sendQueueSize: 256,
		sendChan:      make(chan *Message, 256),
		closeChan:     make(chan struct{}),
		state:         StateConnected,
		remoteAddr:    conn.RemoteAddr().String(),
		localAddr:     conn.LocalAddr().String(),
		connectedAt:   time.Now(),
		serializer:    serializer.Default(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	return c
}

// ID 返回连接 ID
func (c *Connection) ID() string {
	return c.id
}

// State 返回连接状态
func (c *Connection) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetState 设置连接状态
func (c *Connection) SetState(state ConnectionState) {
	c.mu.Lock()
	c.state = state
	c.mu.Unlock()
}

// RemoteAddr 返回远程地址
func (c *Connection) RemoteAddr() string {
	return c.remoteAddr
}

// LocalAddr 返回本地地址
func (c *Connection) LocalAddr() string {
	return c.localAddr
}

// ConnectedAt 返回连接时间
func (c *Connection) ConnectedAt() time.Time {
	return c.connectedAt
}

// IsClosed 检查连接是否已关闭
func (c *Connection) IsClosed() bool {
	return c.closed.Load()
}

// SetMetadata 设置元数据
func (c *Connection) SetMetadata(key string, value interface{}) {
	c.metadata.Store(key, value)
}

// GetMetadata 获取元数据
func (c *Connection) GetMetadata(key string) (interface{}, bool) {
	return c.metadata.Load(key)
}

// DeleteMetadata 删除元数据
func (c *Connection) DeleteMetadata(key string) {
	c.metadata.Delete(key)
}

// RangeMetadata 遍历元数据
func (c *Connection) RangeMetadata(fn func(key, value interface{}) bool) {
	c.metadata.Range(fn)
}

// Info 返回连接信息
func (c *Connection) Info() ConnectionInfo {
	info := ConnectionInfo{
		ID:          c.id,
		RemoteAddr:  c.remoteAddr,
		LocalAddr:   c.localAddr,
		State:       c.State(),
		ConnectedAt: c.connectedAt,
		Metadata:    make(map[string]interface{}),
	}
	c.metadata.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			info.Metadata[k] = value
		}
		return true
	})
	return info
}

// Send 发送消息（同步）
func (c *Connection) Send(ctx context.Context, msg *Message) error {
	if c.IsClosed() {
		return ErrConnectionClosed
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case c.sendChan <- msg:
		return nil
	case <-c.closeChan:
		return ErrConnectionClosed
	}
}

// SendAsync 发送消息（异步，非阻塞）
func (c *Connection) SendAsync(msg *Message) error {
	if c.IsClosed() {
		return ErrConnectionClosed
	}

	select {
	case c.sendChan <- msg:
		return nil
	default:
		return ErrSendQueueFull
	}
}

// SendText 发送文本消息
func (c *Connection) SendText(ctx context.Context, data string) error {
	return c.Send(ctx, NewTextMessageString(data))
}

// SendBinary 发送二进制消息
func (c *Connection) SendBinary(ctx context.Context, data []byte) error {
	return c.Send(ctx, NewBinaryMessage(data))
}

// SendJSON 发送 JSON 消息
func (c *Connection) SendJSON(ctx context.Context, v interface{}) error {
	msg, err := NewJSONMessage(v)
	if err != nil {
		return err
	}
	return c.Send(ctx, msg)
}

// Subscribe 订阅消息（Channel 模式）
func (c *Connection) Subscribe(bufferSize int) (<-chan *Message, string) {
	if bufferSize <= 0 {
		bufferSize = 256
	}
	ch := make(chan *Message, bufferSize)
	subID := uuid.New().String()
	c.subscribers.Store(subID, ch)
	return ch, subID
}

// Unsubscribe 取消订阅
func (c *Connection) Unsubscribe(subID string) {
	if val, ok := c.subscribers.LoadAndDelete(subID); ok {
		if ch, ok := val.(chan *Message); ok {
			close(ch)
		}
	}
}

// publishToSubscribers 发布消息给订阅者
func (c *Connection) publishToSubscribers(msg *Message) {
	c.subscribers.Range(func(key, value interface{}) bool {
		if ch, ok := value.(chan *Message); ok {
			select {
			case ch <- msg:
			default:
				// Channel 满了，跳过
			}
		}
		return true
	})
}

// ReadLoop 读取循环
func (c *Connection) ReadLoop(handler HandlerFunc) {
	defer c.Close()

	for {
		if c.IsClosed() {
			return
		}

		// 设置读取超时
		if c.readTimeout > 0 {
			c.conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		}

		// 读取消息
		msgType, data, err := c.conn.ReadMessage()
		if err != nil {
			if c.IsClosed() {
				return
			}
			// 检查是否为正常关闭
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			if err == io.EOF {
				return
			}
			if c.logger != nil {
				c.logger.Debug("websocket read error", "error", err, "conn_id", c.id)
			}
			return
		}

		// 构造消息
		msg := &Message{
			Type:      MessageType(msgType),
			Data:      data,
			Timestamp: time.Now(),
		}

		// 发布给订阅者
		c.publishToSubscribers(msg)

		// 调用处理器
		if handler != nil {
			if err := handler(c, msg); err != nil {
				if c.logger != nil {
					c.logger.Warn("websocket handler error", "error", err, "conn_id", c.id)
				}
			}
		}
	}
}

// WriteLoop 写入循环
func (c *Connection) WriteLoop() {
	defer c.Close()

	for {
		select {
		case msg, ok := <-c.sendChan:
			if !ok {
				// Channel 已关闭
				return
			}

			// 设置写入超时
			if c.writeTimeout > 0 {
				c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout))
			}

			// 写入消息
			if err := c.conn.WriteMessage(int(msg.Type), msg.Data); err != nil {
				if c.logger != nil {
					c.logger.Debug("websocket write error", "error", err, "conn_id", c.id)
				}
				return
			}

		case <-c.closeChan:
			return
		}
	}
}

// Close 关闭连接
func (c *Connection) Close() error {
	return c.CloseWithError(nil)
}

// CloseWithError 带错误关闭连接
func (c *Connection) CloseWithError(err error) error {
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.closeError = err
		c.SetState(StateClosed)
		close(c.closeChan)

		// 关闭所有订阅者 Channel
		c.subscribers.Range(func(key, value interface{}) bool {
			if ch, ok := value.(chan *Message); ok {
				close(ch)
			}
			c.subscribers.Delete(key)
			return true
		})

		// 发送关闭帧
		c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)

		// 关闭底层连接
		c.conn.Close()
	})
	return nil
}

// CloseError 返回关闭错误
func (c *Connection) CloseError() error {
	return c.closeError
}

// Ping 发送 Ping
func (c *Connection) Ping() error {
	if c.IsClosed() {
		return ErrConnectionClosed
	}
	return c.conn.WriteControl(
		websocket.PingMessage,
		[]byte{},
		time.Now().Add(c.writeTimeout),
	)
}

// SetPongHandler 设置 Pong 处理器
func (c *Connection) SetPongHandler(h func(appData string) error) {
	c.conn.SetPongHandler(h)
}

// SetPingHandler 设置 Ping 处理器
func (c *Connection) SetPingHandler(h func(appData string) error) {
	c.conn.SetPingHandler(h)
}

// SetCloseHandler 设置关闭处理器
func (c *Connection) SetCloseHandler(h func(code int, text string) error) {
	c.conn.SetCloseHandler(h)
}

// SetReadLimit 设置读取限制
func (c *Connection) SetReadLimit(limit int64) {
	c.conn.SetReadLimit(limit)
}

// UnderlyingConn 返回底层 websocket.Conn（谨慎使用）
func (c *Connection) UnderlyingConn() *websocket.Conn {
	return c.conn
}
