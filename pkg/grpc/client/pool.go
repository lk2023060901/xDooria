package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

// PoolConfig 连接池配置
type PoolConfig struct {
	// 连接池大小（默认 5）
	Size int `mapstructure:"size" json:"size"`

	// 每个连接的最大空闲时间（0 表示不限制）
	MaxIdleTime time.Duration `mapstructure:"max_idle_time" json:"max_idle_time"`

	// 健康检查间隔（0 表示不启用）
	HealthCheckInterval time.Duration `mapstructure:"health_check_interval" json:"health_check_interval"`

	// 获取连接的超时时间
	GetTimeout time.Duration `mapstructure:"get_timeout" json:"get_timeout"`

	// 是否在获取连接时等待（如果池已满）
	WaitForConn bool `mapstructure:"wait_for_conn" json:"wait_for_conn"`
}

// DefaultPoolConfig 返回默认连接池配置
func DefaultPoolConfig() *PoolConfig {
	return &PoolConfig{
		Size:                5,
		MaxIdleTime:         30 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
		GetTimeout:          5 * time.Second,
		WaitForConn:         true,
	}
}

// pooledConn 池化的连接
type pooledConn struct {
	conn         *grpc.ClientConn
	lastUsedTime time.Time
	inUse        bool
	mu           sync.Mutex
}

// Pool gRPC 连接池
type Pool struct {
	config    *PoolConfig
	clientCfg *Config
	logger    *logger.Logger

	// 连接池
	conns    []*pooledConn
	connChan chan *pooledConn // 可用连接通道

	// 拨号选项
	dialOpts           []grpc.DialOption
	unaryInterceptors  []grpc.UnaryClientInterceptor
	streamInterceptors []grpc.StreamClientInterceptor

	// 状态管理
	mu               sync.RWMutex
	closed           bool
	healthCheckFuture *conc.Future[struct{}]
	stopCh           chan struct{}
}

// NewPool 创建连接池
func NewPool(cfg *Config, poolCfg *PoolConfig, opts ...Option) (*Pool, error) {
	// 合并连接池配置
	newPoolCfg, err := config.MergeConfig(DefaultPoolConfig(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to merge pool config: %w", err)
	}

	// 验证客户端配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	p := &Pool{
		config:             newPoolCfg,
		clientCfg:          cfg,
		logger:             logger.Default().Named("grpc.pool"),
		conns:              make([]*pooledConn, 0, newPoolCfg.Size),
		connChan:           make(chan *pooledConn, newPoolCfg.Size),
		dialOpts:           make([]grpc.DialOption, 0),
		unaryInterceptors:  make([]grpc.UnaryClientInterceptor, 0),
		streamInterceptors: make([]grpc.StreamClientInterceptor, 0),
		stopCh:             make(chan struct{}),
	}

	// 应用选项（为了获取拦截器和 dialOpts）
	for _, opt := range opts {
		// 创建临时 client 来应用选项
		tmpClient := &Client{
			dialOpts:           p.dialOpts,
			unaryInterceptors:  p.unaryInterceptors,
			streamInterceptors: p.streamInterceptors,
			logger:             p.logger,
		}
		opt(tmpClient)
		p.dialOpts = tmpClient.dialOpts
		p.unaryInterceptors = tmpClient.unaryInterceptors
		p.streamInterceptors = tmpClient.streamInterceptors
	}

	// 初始化连接池
	if err := p.init(); err != nil {
		return nil, fmt.Errorf("failed to initialize pool: %w", err)
	}

	// 启动健康检查
	if newPoolCfg.HealthCheckInterval > 0 {
		p.startHealthCheck()
	}

	return p, nil
}

// init 初始化连接池
func (p *Pool) init() error {
	p.logger.Info("initializing connection pool",
		zap.String("target", p.clientCfg.Target),
		zap.Int("size", p.config.Size),
	)

	for i := 0; i < p.config.Size; i++ {
		conn, err := p.createConn()
		if err != nil {
			// 清理已创建的连接
			p.closeAll()
			return fmt.Errorf("failed to create connection %d: %w", i, err)
		}

		pc := &pooledConn{
			conn:         conn,
			lastUsedTime: time.Now(),
			inUse:        false,
		}

		p.conns = append(p.conns, pc)
		p.connChan <- pc
	}

	p.logger.Info("connection pool initialized",
		zap.Int("size", len(p.conns)),
	)

	return nil
}

// createConn 创建单个连接
func (p *Pool) createConn() (*grpc.ClientConn, error) {
	dialOpts := p.buildDialOptions()

	ctx, cancel := context.WithTimeout(context.Background(), p.clientCfg.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, p.clientCfg.Target, dialOpts...)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// buildDialOptions 构建拨号选项（复用 Client 的逻辑）
func (p *Pool) buildDialOptions() []grpc.DialOption {
	// 这里复用 Client.buildDialOptions 的逻辑
	tmpClient := &Client{
		config:             p.clientCfg,
		dialOpts:           p.dialOpts,
		unaryInterceptors:  p.unaryInterceptors,
		streamInterceptors: p.streamInterceptors,
	}
	return tmpClient.buildDialOptions()
}

// Get 从池中获取连接
func (p *Pool) Get(ctx context.Context) (*grpc.ClientConn, error) {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return nil, ErrPoolClosed
	}
	p.mu.RUnlock()

	// 设置获取超时
	if p.config.GetTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, p.config.GetTimeout)
		defer cancel()
	}

	select {
	case pc := <-p.connChan:
		// 标记为使用中
		pc.mu.Lock()
		pc.inUse = true
		pc.lastUsedTime = time.Now()
		pc.mu.Unlock()

		// 检查连接状态
		if !p.isConnHealthy(pc.conn) {
			p.logger.Warn("connection unhealthy, recreating",
				zap.String("target", p.clientCfg.Target),
			)

			// 尝试重新创建连接
			if err := p.recreateConn(pc); err != nil {
				// 如果重建失败，归还到池中并返回错误
				p.Put(pc.conn)
				return nil, fmt.Errorf("failed to recreate connection: %w", err)
			}
		}

		return pc.conn, nil

	case <-ctx.Done():
		if p.config.WaitForConn {
			return nil, fmt.Errorf("timeout waiting for connection: %w", ctx.Err())
		}
		return nil, ErrPoolExhausted

	case <-p.stopCh:
		return nil, ErrPoolClosed
	}
}

// Put 归还连接到池中
func (p *Pool) Put(conn *grpc.ClientConn) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return
	}

	// 找到对应的 pooledConn
	for _, pc := range p.conns {
		if pc.conn == conn {
			pc.mu.Lock()
			pc.inUse = false
			pc.lastUsedTime = time.Now()
			pc.mu.Unlock()

			// 归还到通道
			select {
			case p.connChan <- pc:
			default:
				p.logger.Warn("connection channel full, dropping connection")
			}
			return
		}
	}

	p.logger.Warn("attempted to return unknown connection to pool")
}

// Close 关闭连接池
func (p *Pool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return nil
	}

	p.closed = true
	close(p.stopCh)

	// 等待健康检查任务完成
	if p.healthCheckFuture != nil {
		p.healthCheckFuture.Await()
	}

	// 关闭所有连接
	return p.closeAll()
}

// closeAll 关闭所有连接
func (p *Pool) closeAll() error {
	var errs []error

	for _, pc := range p.conns {
		if err := pc.conn.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	p.logger.Info("connection pool closed",
		zap.Int("connections", len(p.conns)),
	)

	if len(errs) > 0 {
		return fmt.Errorf("failed to close %d connections", len(errs))
	}

	return nil
}

// isConnHealthy 检查连接是否健康
func (p *Pool) isConnHealthy(conn *grpc.ClientConn) bool {
	state := conn.GetState()
	return state == connectivity.Ready || state == connectivity.Idle
}

// recreateConn 重新创建连接
func (p *Pool) recreateConn(pc *pooledConn) error {
	// 关闭旧连接
	if err := pc.conn.Close(); err != nil {
		p.logger.Warn("failed to close old connection", zap.Error(err))
	}

	// 创建新连接
	newConn, err := p.createConn()
	if err != nil {
		return err
	}

	pc.mu.Lock()
	pc.conn = newConn
	pc.lastUsedTime = time.Now()
	pc.mu.Unlock()

	p.logger.Info("connection recreated",
		zap.String("target", p.clientCfg.Target),
	)

	return nil
}

// startHealthCheck 启动健康检查
func (p *Pool) startHealthCheck() {
	ticker := time.NewTicker(p.config.HealthCheckInterval)

	p.healthCheckFuture = conc.Go(func() (struct{}, error) {
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.performHealthCheck()
			case <-p.stopCh:
				return struct{}{}, nil
			}
		}
	})

	p.logger.Info("health check started",
		zap.Duration("interval", p.config.HealthCheckInterval),
	)
}

// performHealthCheck 执行健康检查
func (p *Pool) performHealthCheck() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.closed {
		return
	}

	for _, pc := range p.conns {
		pc.mu.Lock()
		inUse := pc.inUse
		lastUsed := pc.lastUsedTime
		conn := pc.conn
		pc.mu.Unlock()

		// 跳过正在使用的连接
		if inUse {
			continue
		}

		// 检查空闲时间
		if p.config.MaxIdleTime > 0 {
			idleTime := time.Since(lastUsed)
			if idleTime > p.config.MaxIdleTime {
				p.logger.Debug("connection idle too long, closing",
					zap.Duration("idle_time", idleTime),
				)
				// 重新创建连接
				if err := p.recreateConn(pc); err != nil {
					p.logger.Error("failed to recreate idle connection", zap.Error(err))
				}
				continue
			}
		}

		// 检查连接状态
		if !p.isConnHealthy(conn) {
			p.logger.Warn("unhealthy connection detected during health check")
			if err := p.recreateConn(pc); err != nil {
				p.logger.Error("failed to recreate unhealthy connection", zap.Error(err))
			}
		}
	}
}

// Stats 返回连接池统计信息
func (p *Pool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var inUseCount, readyCount, idleCount int

	for _, pc := range p.conns {
		pc.mu.Lock()
		if pc.inUse {
			inUseCount++
		}
		state := pc.conn.GetState()
		pc.mu.Unlock()

		if state == connectivity.Ready {
			readyCount++
		} else if state == connectivity.Idle {
			idleCount++
		}
	}

	return PoolStats{
		Total:     len(p.conns),
		InUse:     inUseCount,
		Available: len(p.connChan),
		Ready:     readyCount,
		Idle:      idleCount,
	}
}

// PoolStats 连接池统计信息
type PoolStats struct {
	Total     int // 总连接数
	InUse     int // 使用中的连接数
	Available int // 可用连接数
	Ready     int // Ready 状态的连接数
	Idle      int // Idle 状态的连接数
}
