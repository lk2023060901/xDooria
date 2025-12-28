package client

import (
	"fmt"
	"net"
	"sync"
	"time"

	common "github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/logger"
	"github.com/lk2023060901/xdooria/pkg/network/framer"
	"google.golang.org/protobuf/proto"
)

// Robot TCP 客户端
type Robot struct {
	logger  logger.Logger
	framer  framer.Framer
	addr    string

	mu       sync.RWMutex
	conn     net.Conn
	connected bool

	// 接收队列
	recvChan chan *common.Envelope
	stopCh   chan struct{}
}

// NewRobot 创建 Robot 客户端
func NewRobot(addr string, framerCfg *framer.Config) (*Robot, error) {
	f, err := framer.New(framerCfg)
	if err != nil {
		return nil, fmt.Errorf("create framer failed: %w", err)
	}

	return &Robot{
		logger:   logger.Default().Named("robot.client"),
		framer:   f,
		addr:     addr,
		recvChan: make(chan *common.Envelope, 100),
		stopCh:   make(chan struct{}),
	}, nil
}

// Connect 连接到服务器
func (r *Robot) Connect() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.connected {
		return fmt.Errorf("already connected")
	}

	conn, err := net.DialTimeout("tcp", r.addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("dial failed: %w", err)
	}

	r.conn = conn
	r.connected = true

	// 启动接收循环
	go r.recvLoop()

	r.logger.Info("connected to server", "addr", r.addr)
	return nil
}

// Close 关闭连接
func (r *Robot) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.connected {
		return nil
	}

	close(r.stopCh)

	if r.conn != nil {
		r.conn.Close()
	}

	r.connected = false
	r.logger.Info("disconnected from server")

	return nil
}

// Send 发送消息
func (r *Robot) Send(op uint32, payload []byte) error {
	r.mu.RLock()
	conn := r.conn
	connected := r.connected
	r.mu.RUnlock()

	if !connected || conn == nil {
		return fmt.Errorf("not connected")
	}

	// 使用 Framer 编码消息
	env, err := r.framer.Encode(op, payload)
	if err != nil {
		return fmt.Errorf("encode failed: %w", err)
	}

	// 序列化 Envelope
	data, err := proto.Marshal(env)
	if err != nil {
		return fmt.Errorf("marshal failed: %w", err)
	}

	// 直接发送数据（不需要长度前缀，gnet 会处理分包）
	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write data failed: %w", err)
	}

	r.logger.Debug("sent message", "op", op, "len", len(payload))
	return nil
}

// Recv 接收消息（带超时）
func (r *Robot) Recv(timeout time.Duration) (*common.Envelope, error) {
	select {
	case env := <-r.recvChan:
		return env, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("recv timeout")
	case <-r.stopCh:
		return nil, fmt.Errorf("connection closed")
	}
}

// IsConnected 是否已连接
func (r *Robot) IsConnected() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.connected
}

// recvLoop 接收循环
func (r *Robot) recvLoop() {
	defer func() {
		r.mu.Lock()
		r.connected = false
		if r.conn != nil {
			r.conn.Close()
		}
		r.mu.Unlock()
	}()

	// 使用缓冲区读取 protobuf 数据
	// 注意：由于 protobuf 是自描述格式，我们需要先peek长度或使用分隔符
	// 这里简化处理：假设每次读取一个完整消息
	buf := make([]byte, 65536) // 64KB 缓冲区

	for {
		select {
		case <-r.stopCh:
			return
		default:
		}

		r.mu.RLock()
		conn := r.conn
		r.mu.RUnlock()

		if conn == nil {
			return
		}

		// 设置读取超时
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		// 读取数据
		n, err := conn.Read(buf)
		if err != nil {
			r.logger.Warn("read failed", "error", err)
			return
		}

		data := buf[:n]

		// 反序列化 Envelope
		env := &common.Envelope{}
		if err := proto.Unmarshal(data, env); err != nil {
			r.logger.Error("unmarshal envelope failed", "error", err)
			continue
		}

		// 使用 Framer 解码
		op, payload, err := r.framer.Decode(env)
		if err != nil {
			r.logger.Error("decode failed", "error", err)
			continue
		}

		// 构建解码后的 Envelope
		decodedEnv := &common.Envelope{
			Header:  &common.MessageHeader{Op: op},
			Payload: payload,
		}

		r.logger.Info("received message", "op", op, "len", len(payload))

		// 放入接收队列
		select {
		case r.recvChan <- decodedEnv:
		case <-r.stopCh:
			return
		default:
			r.logger.Warn("recv channel full, dropping message")
		}
	}
}
