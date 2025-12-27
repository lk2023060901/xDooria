// pkg/websocket/client.go
package websocket

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gorilla/websocket"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/serializer"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/prometheus/client_golang/prometheus"
)

// Client WebSocket 客户端
type Client struct {
	config *ClientConfig
	logger logger.Logger
	dialer *websocket.Dialer

	// 连接
	conn   *Connection
	connMu sync.RWMutex

	// 消息处理
	handler     MessageHandler
	middlewares []Middleware
	handlerFunc HandlerFunc

	// 序列化
	serializer serializer.Serializer

	// 重连
	reconnector *Reconnector

	// 心跳
	heartbeat *HeartbeatManager

	// 回调
	onReconnecting   func(attempt int)
	onReconnected    func()
	onReconnectFailed func(err error)

	// 消息 Channel（Channel 模式）
	msgChan chan *Message

	// 指标
	metrics           *ClientMetrics
	metricsRegisterer prometheus.Registerer

	// 工作池
	workerPool *conc.Pool[struct{}]

	// 状态
	mu           sync.RWMutex
	state        ConnectionState
	reconnecting atomic.Bool
	closed       atomic.Bool
	closeCh      chan struct{}
	closeOnce    sync.Once
}

// NewClient 创建客户端
func NewClient(cfg *ClientConfig, opts ...ClientOption) (*Client, error) {
	if cfg == nil {
		cfg = DefaultClientConfig()
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	c := &Client{
		config:     cfg,
		closeCh:    make(chan struct{}),
		state:      StateDisconnected,
		serializer: serializer.Default(),
		msgChan:    make(chan *Message, cfg.RecvQueueSize),
	}

	// 应用选项
	for _, opt := range opts {
		opt(c)
	}

	// 初始化 dialer
	c.dialer = &websocket.Dialer{
		HandshakeTimeout:  cfg.DialTimeout,
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		EnableCompression: cfg.EnableCompression,
	}

	// 配置 TLS
	if cfg.TLS != nil {
		tlsConfig, err := cfg.TLS.BuildTLSConfig()
		if err != nil {
			return nil, err
		}
		c.dialer.TLSClientConfig = tlsConfig
	}

	// 初始化重连器
	if cfg.Reconnect.Enable {
		c.reconnector = NewReconnector(&cfg.Reconnect, c, c.logger)
	}

	// 初始化指标
	if c.metricsRegisterer != nil {
		c.metrics = NewClientMetrics(c.metricsRegisterer)
	}

	// 初始化工作池
	c.workerPool = conc.NewPool[struct{}](10)

	// 构建处理器链
	c.buildHandlerChain()

	return c, nil
}

// buildHandlerChain 构建处理器链
func (c *Client) buildHandlerChain() {
	// 基础处理函数
	baseHandler := func(conn *Connection, msg *Message) error {
		// 发送到消息 Channel
		select {
		case c.msgChan <- msg:
		default:
			// Channel 满了，跳过
		}

		// 调用处理器
		if c.handler != nil {
			return c.handler.OnMessage(conn, msg)
		}
		return nil
	}

	// 应用中间件
	if len(c.middlewares) > 0 {
		chain := NewMiddlewareChain(c.middlewares...)
		c.handlerFunc = chain.Then(baseHandler)
	} else {
		c.handlerFunc = baseHandler
	}
}

// Connect 建立连接
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.closed.Load() {
		c.mu.Unlock()
		return ErrConnectionClosed
	}
	if c.state == StateConnected {
		c.mu.Unlock()
		return ErrAlreadyConnected
	}
	c.state = StateConnecting
	c.mu.Unlock()

	return c.connect(ctx)
}

// connect 内部连接方法
func (c *Client) connect(ctx context.Context) error {
	// 构建请求头
	header := make(http.Header)
	for k, v := range c.config.Headers {
		header.Set(k, v)
	}

	// 连接
	wsConn, _, err := c.dialer.DialContext(ctx, c.config.URL, header)
	if err != nil {
		c.mu.Lock()
		c.state = StateDisconnected
		c.mu.Unlock()
		return err
	}

	// 创建连接
	conn := NewConnection(wsConn,
		WithConnectionLogger(c.logger),
		WithConnectionSerializer(c.serializer),
	)

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	c.mu.Lock()
	c.state = StateConnected
	c.mu.Unlock()

	// 更新指标
	if c.metrics != nil {
		c.metrics.OnConnected()
	}

	// 连接建立回调
	if c.handler != nil {
		if err := c.handler.OnConnect(conn); err != nil {
			c.Close()
			return err
		}
	}

	// 设置 Pong 处理器
	conn.SetPongHandler(func(appData string) error {
		if c.heartbeat != nil {
			c.heartbeat.OnPong()
		}
		return nil
	})

	// 启动写入循环
	c.workerPool.Submit(func() (struct{}, error) {
		conn.WriteLoop()
		return struct{}{}, nil
	})

	// 启动心跳
	if c.config.Heartbeat.Enable {
		c.heartbeat = NewHeartbeatManager(&c.config.Heartbeat, conn, c.logger, c.workerPool)
		c.heartbeat.SetOnTimeout(func() {
			c.handleDisconnect(ErrHeartbeatTimeout)
		})
		c.heartbeat.Start()
	}

	// 启动读取循环
	c.workerPool.Submit(func() (struct{}, error) {
		c.readLoop()
		return struct{}{}, nil
	})

	return nil
}

// readLoop 读取循环
func (c *Client) readLoop() {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return
	}

	conn.ReadLoop(c.handlerFunc)

	// 读取循环结束，处理断开
	c.handleDisconnect(conn.CloseError())
}

// handleDisconnect 处理断开连接
func (c *Client) handleDisconnect(err error) {
	if c.closed.Load() {
		return
	}

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	// 停止心跳
	if c.heartbeat != nil {
		c.heartbeat.Stop()
	}

	// 断开回调
	if c.handler != nil && conn != nil {
		c.handler.OnDisconnect(conn, err)
	}

	// 更新指标
	if c.metrics != nil {
		c.metrics.OnDisconnected()
	}

	// 尝试重连
	if c.reconnector != nil && !c.closed.Load() {
		c.mu.Lock()
		c.state = StateReconnecting
		c.mu.Unlock()

		c.workerPool.Submit(func() (struct{}, error) {
			c.reconnector.Start(context.Background())
			return struct{}{}, nil
		})
	} else {
		c.mu.Lock()
		c.state = StateDisconnected
		c.mu.Unlock()
	}
}

// Send 发送消息
func (c *Client) Send(ctx context.Context, msg *Message) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	err := conn.Send(ctx, msg)
	if err == nil && c.metrics != nil {
		c.metrics.OnMessageSent(int64(len(msg.Data)))
	}
	return err
}

// SendAsync 发送消息（异步）
func (c *Client) SendAsync(msg *Message) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return ErrNotConnected
	}

	err := conn.SendAsync(msg)
	if err == nil && c.metrics != nil {
		c.metrics.OnMessageSent(int64(len(msg.Data)))
	}
	return err
}

// SendBytes 发送字节数据
func (c *Client) SendBytes(ctx context.Context, data []byte) error {
	return c.Send(ctx, NewMessage(data))
}

// MessageChan 获取消息 Channel
func (c *Client) MessageChan() <-chan *Message {
	return c.msgChan
}

// State 获取连接状态
func (c *Client) State() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// IsConnected 检查是否已连接
func (c *Client) IsConnected() bool {
	return c.State() == StateConnected
}

// Connection 获取当前连接
func (c *Client) Connection() *Connection {
	c.connMu.RLock()
	defer c.connMu.RUnlock()
	return c.conn
}

// SetHandler 设置消息处理器
func (c *Client) SetHandler(handler MessageHandler) {
	c.handler = handler
	c.buildHandlerChain()
}

// Close 关闭客户端
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		close(c.closeCh)

		c.mu.Lock()
		c.state = StateClosed
		c.mu.Unlock()

		// 停止重连
		if c.reconnector != nil {
			c.reconnector.Stop()
		}

		// 停止心跳
		if c.heartbeat != nil {
			c.heartbeat.Stop()
		}

		// 关闭连接
		c.connMu.Lock()
		if c.conn != nil {
			err = c.conn.Close()
			c.conn = nil
		}
		c.connMu.Unlock()

		// 关闭消息 Channel
		close(c.msgChan)

		// 关闭工作池
		c.workerPool.Release()
	})
	return err
}
