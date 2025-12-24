// pkg/websocket/server_pool.go
package websocket

import (
	"sync"
	"sync/atomic"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// ConnectionPool 连接池
type ConnectionPool struct {
	config *PoolConfig
	logger logger.Logger

	// 连接存储
	connections sync.Map // map[connID]*Connection

	// IP 计数
	ipCount sync.Map // map[ip]*int64

	// 统计
	totalCount  atomic.Int64
	activeCount atomic.Int64

	// 状态
	mu     sync.RWMutex
	closed bool
}

// NewConnectionPool 创建连接池
func NewConnectionPool(cfg *PoolConfig, log logger.Logger) *ConnectionPool {
	if cfg == nil {
		defaultCfg := DefaultPoolConfig()
		cfg = &defaultCfg
	}
	return &ConnectionPool{
		config: cfg,
		logger: log,
	}
}

// Add 添加连接
func (p *ConnectionPool) Add(conn *Connection) error {
	p.mu.RLock()
	if p.closed {
		p.mu.RUnlock()
		return ErrPoolClosed
	}
	p.mu.RUnlock()

	// 检查连接数限制
	if p.config.MaxConnections > 0 && p.activeCount.Load() >= int64(p.config.MaxConnections) {
		return ErrPoolFull
	}

	// 检查每 IP 连接数
	remoteIP := extractIP(conn.RemoteAddr())
	if p.config.MaxConnectionsPerIP > 0 {
		count := p.getIPCount(remoteIP)
		if count >= int64(p.config.MaxConnectionsPerIP) {
			return ErrMaxConnectionsPerIP
		}
	}

	// 添加连接
	p.connections.Store(conn.ID(), conn)
	p.activeCount.Add(1)
	p.totalCount.Add(1)

	// 增加 IP 计数
	p.incrementIPCount(remoteIP)

	return nil
}

// Remove 移除连接
func (p *ConnectionPool) Remove(connID string) {
	if val, ok := p.connections.LoadAndDelete(connID); ok {
		conn := val.(*Connection)
		p.activeCount.Add(-1)

		// 减少 IP 计数
		remoteIP := extractIP(conn.RemoteAddr())
		p.decrementIPCount(remoteIP)

		// 关闭连接
		conn.Close()
	}
}

// Get 获取连接
func (p *ConnectionPool) Get(connID string) (*Connection, bool) {
	if val, ok := p.connections.Load(connID); ok {
		return val.(*Connection), true
	}
	return nil, false
}

// GetAll 获取所有连接
func (p *ConnectionPool) GetAll() []*Connection {
	var conns []*Connection
	p.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			conns = append(conns, conn)
		}
		return true
	})
	return conns
}

// Range 遍历连接
func (p *ConnectionPool) Range(fn func(conn *Connection) bool) {
	p.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			return fn(conn)
		}
		return true
	})
}

// Count 获取连接数
func (p *ConnectionPool) Count() int {
	return int(p.activeCount.Load())
}

// IsFull 检查连接池是否已满
func (p *ConnectionPool) IsFull() bool {
	if p.config.MaxConnections <= 0 {
		return false
	}
	return p.activeCount.Load() >= int64(p.config.MaxConnections)
}

// IsIPLimitReached 检查 IP 是否达到连接限制
func (p *ConnectionPool) IsIPLimitReached(ip string) bool {
	if p.config.MaxConnectionsPerIP <= 0 {
		return false
	}
	return p.getIPCount(ip) >= int64(p.config.MaxConnectionsPerIP)
}

// getIPCount 获取 IP 连接数
func (p *ConnectionPool) getIPCount(ip string) int64 {
	if val, ok := p.ipCount.Load(ip); ok {
		return val.(*atomic.Int64).Load()
	}
	return 0
}

// incrementIPCount 增加 IP 连接数
func (p *ConnectionPool) incrementIPCount(ip string) {
	val, _ := p.ipCount.LoadOrStore(ip, &atomic.Int64{})
	val.(*atomic.Int64).Add(1)
}

// decrementIPCount 减少 IP 连接数
func (p *ConnectionPool) decrementIPCount(ip string) {
	if val, ok := p.ipCount.Load(ip); ok {
		counter := val.(*atomic.Int64)
		if counter.Add(-1) <= 0 {
			p.ipCount.Delete(ip)
		}
	}
}

// Stats 获取统计信息
func (p *ConnectionPool) Stats() Stats {
	stats := Stats{
		TotalConnections:  p.totalCount.Load(),
		ActiveConnections: p.activeCount.Load(),
		ConnectionsPerIP:  make(map[string]int),
	}

	p.ipCount.Range(func(key, value interface{}) bool {
		if ip, ok := key.(string); ok {
			if counter, ok := value.(*atomic.Int64); ok {
				count := int(counter.Load())
				if count > 0 {
					stats.ConnectionsPerIP[ip] = count
				}
			}
		}
		return true
	})

	return stats
}

// Close 关闭连接池
func (p *ConnectionPool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// 关闭所有连接
	p.connections.Range(func(key, value interface{}) bool {
		if conn, ok := value.(*Connection); ok {
			conn.Close()
		}
		p.connections.Delete(key)
		return true
	})

	p.activeCount.Store(0)
	return nil
}
