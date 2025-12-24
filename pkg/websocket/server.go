// pkg/websocket/server.go
package websocket

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/serializer"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"github.com/prometheus/client_golang/prometheus"
)

// Server WebSocket 服务端
type Server struct {
	config   *ServerConfig
	upgrader *websocket.Upgrader
	logger   logger.Logger

	// 连接池
	pool *ConnectionPool

	// 消息处理
	handler     MessageHandler
	middlewares []Middleware
	handlerFunc HandlerFunc

	// 序列化
	serializer serializer.Serializer

	// 工作池
	workerPool *conc.Pool[struct{}]

	// 指标
	metrics            *ServerMetrics
	metricsRegisterer  prometheus.Registerer
	metricsInitialized bool

	// 状态
	mu       sync.RWMutex
	closed   bool
	closeCh  chan struct{}
	wg       sync.WaitGroup
	initOnce sync.Once
}

// NewServer 创建服务端
func NewServer(cfg *ServerConfig, opts ...ServerOption) (*Server, error) {
	if cfg == nil {
		cfg = DefaultServerConfig()
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	s := &Server{
		config:     cfg,
		closeCh:    make(chan struct{}),
		serializer: serializer.Default(),
	}

	// 应用选项
	for _, opt := range opts {
		opt(s)
	}

	// 初始化 upgrader
	s.upgrader = &websocket.Upgrader{
		ReadBufferSize:    cfg.ReadBufferSize,
		WriteBufferSize:   cfg.WriteBufferSize,
		HandshakeTimeout:  cfg.HandshakeTimeout,
		EnableCompression: cfg.EnableCompression,
		CheckOrigin:       cfg.CheckOrigin,
	}

	// 如果没有设置 CheckOrigin，使用默认的（同源检查）
	if s.upgrader.CheckOrigin == nil {
		s.upgrader.CheckOrigin = func(r *http.Request) bool {
			return r.Header.Get("Origin") == ""
		}
	}

	// 初始化连接池
	s.pool = NewConnectionPool(&cfg.Pool, s.logger)

	// 初始化工作池
	poolSize := cfg.Pool.MaxConnections / 10
	if poolSize < 10 {
		poolSize = 10
	}
	s.workerPool = conc.NewPool[struct{}](poolSize)

	// 初始化指标
	if s.metricsRegisterer != nil {
		s.metrics = NewServerMetrics(s.metricsRegisterer)
		s.metricsInitialized = true
	}

	// 构建处理器链
	s.buildHandlerChain()

	return s, nil
}

// buildHandlerChain 构建处理器链
func (s *Server) buildHandlerChain() {
	// 基础处理函数
	baseHandler := func(conn *Connection, msg *Message) error {
		if s.handler != nil {
			return s.handler.OnMessage(conn, msg)
		}
		return nil
	}

	// 应用中间件
	if len(s.middlewares) > 0 {
		chain := NewMiddlewareChain(s.middlewares...)
		s.handlerFunc = chain.Then(baseHandler)
	} else {
		s.handlerFunc = baseHandler
	}
}

// Handler 返回 http.Handler
func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(s.ServeHTTP)
}

// ServeHTTP 实现 http.Handler 接口
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := s.Upgrade(w, r)
	if err != nil {
		if s.logger != nil {
			s.logger.Warn("websocket upgrade failed", "error", err, "remote_addr", r.RemoteAddr)
		}
		return
	}

	// 启动连接处理
	s.handleConnection(conn)
}

// Upgrade 升级 HTTP 连接为 WebSocket
func (s *Server) Upgrade(w http.ResponseWriter, r *http.Request) (*Connection, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	s.mu.RUnlock()

	// 检查连接池是否已满
	if s.pool.IsFull() {
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return nil, ErrPoolFull
	}

	// 检查每 IP 连接数
	remoteIP := extractIP(r.RemoteAddr)
	if s.pool.IsIPLimitReached(remoteIP) {
		http.Error(w, "too many connections from this IP", http.StatusTooManyRequests)
		return nil, ErrMaxConnectionsPerIP
	}

	// 升级连接
	wsConn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// 创建连接
	conn := NewConnection(wsConn,
		WithConnectionLogger(s.logger),
		WithConnectionSerializer(s.serializer),
	)

	// 设置消息大小限制
	if s.config.MaxMessageSize > 0 {
		conn.SetReadLimit(s.config.MaxMessageSize)
	}

	// 添加到连接池
	if err := s.pool.Add(conn); err != nil {
		conn.Close()
		return nil, err
	}

	// 更新指标
	if s.metrics != nil {
		s.metrics.OnConnectionOpened()
	}

	return conn, nil
}

// handleConnection 处理连接
func (s *Server) handleConnection(conn *Connection) {
	s.wg.Add(1)
	defer s.wg.Done()

	// 连接建立回调
	if s.handler != nil {
		if err := s.handler.OnConnect(conn); err != nil {
			if s.logger != nil {
				s.logger.Warn("websocket OnConnect error", "error", err, "conn_id", conn.ID())
			}
			s.removeConnection(conn, err)
			return
		}
	}

	// 设置 Pong 处理器
	conn.SetPongHandler(func(appData string) error {
		if s.config.PongTimeout > 0 {
			conn.conn.SetReadDeadline(time.Now().Add(s.config.PongTimeout))
		}
		return nil
	})

	// 启动写入循环
	s.workerPool.Submit(func() (struct{}, error) {
		conn.WriteLoop()
		return struct{}{}, nil
	})

	// 启动 Ping 循环
	if s.config.PingInterval > 0 {
		s.workerPool.Submit(func() (struct{}, error) {
			s.pingLoop(conn)
			return struct{}{}, nil
		})
	}

	// 读取循环（阻塞）
	conn.ReadLoop(s.handlerFunc)

	// 连接断开
	s.removeConnection(conn, conn.CloseError())
}

// pingLoop Ping 循环
func (s *Server) pingLoop(conn *Connection) {
	ticker := time.NewTicker(s.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if conn.IsClosed() {
				return
			}
			if err := conn.Ping(); err != nil {
				if s.logger != nil {
					s.logger.Debug("websocket ping error", "error", err, "conn_id", conn.ID())
				}
				conn.Close()
				return
			}
		case <-conn.closeChan:
			return
		case <-s.closeCh:
			return
		}
	}
}

// removeConnection 移除连接
func (s *Server) removeConnection(conn *Connection, err error) {
	// 从连接池移除
	s.pool.Remove(conn.ID())

	// 断开回调
	if s.handler != nil {
		s.handler.OnDisconnect(conn, err)
	}

	// 更新指标
	if s.metrics != nil {
		s.metrics.OnConnectionClosed()
	}
}

// Broadcast 向所有连接广播消息
func (s *Server) Broadcast(msg *Message, exclude ...string) error {
	excludeMap := make(map[string]struct{}, len(exclude))
	for _, id := range exclude {
		excludeMap[id] = struct{}{}
	}

	s.pool.Range(func(conn *Connection) bool {
		if _, ok := excludeMap[conn.ID()]; ok {
			return true
		}
		if err := conn.SendAsync(msg); err != nil {
			if s.logger != nil {
				s.logger.Debug("broadcast send error", "error", err, "conn_id", conn.ID())
			}
		}
		return true
	})

	// 更新指标
	if s.metrics != nil {
		s.metrics.OnMessageSent(msg.Type, int64(len(msg.Data)))
	}

	return nil
}

// BroadcastText 广播文本消息
func (s *Server) BroadcastText(data string, exclude ...string) error {
	return s.Broadcast(NewTextMessageString(data), exclude...)
}

// BroadcastBinary 广播二进制消息
func (s *Server) BroadcastBinary(data []byte, exclude ...string) error {
	return s.Broadcast(NewBinaryMessage(data), exclude...)
}

// BroadcastJSON 广播 JSON 消息
func (s *Server) BroadcastJSON(v interface{}, exclude ...string) error {
	msg, err := NewJSONMessage(v)
	if err != nil {
		return err
	}
	return s.Broadcast(msg, exclude...)
}

// GetConnection 获取指定连接
func (s *Server) GetConnection(connID string) (*Connection, bool) {
	return s.pool.Get(connID)
}

// GetConnectionCount 获取连接数
func (s *Server) GetConnectionCount() int {
	return s.pool.Count()
}

// GetConnections 获取所有连接
func (s *Server) GetConnections() []*Connection {
	return s.pool.GetAll()
}

// Stats 获取统计信息
func (s *Server) Stats() Stats {
	return s.pool.Stats()
}

// Close 关闭服务端
func (s *Server) Close() error {
	return s.CloseWithContext(context.Background())
}

// CloseWithContext 带上下文关闭服务端
func (s *Server) CloseWithContext(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	close(s.closeCh)
	s.mu.Unlock()

	// 关闭所有连接
	s.pool.Close()

	// 等待所有连接处理完成
	done := make(chan struct{})
	s.workerPool.Submit(func() (struct{}, error) {
		s.wg.Wait()
		close(done)
		return struct{}{}, nil
	})

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}

	// 关闭工作池
	s.workerPool.Release()

	return nil
}

// extractIP 从地址中提取 IP
func extractIP(addr string) string {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i]
		}
	}
	return addr
}
