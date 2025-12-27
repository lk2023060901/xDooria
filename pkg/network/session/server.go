package session

import (
	"context"
	"errors"
	"sync"

	"github.com/lk2023060901/xdooria-proto-common"
	"github.com/lk2023060901/xdooria/pkg/config"
)

// Server 通用的服务端实现。
type Server struct {
	config *ServerConfig
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewServer 完全由 Config 驱动进行初始化。
// 使用 MergeConfig 确保 newCfg 具备最小可用条件（默认 Manager, NopHandler 等）。
func NewServer(cfg *ServerConfig) *Server {
	// 1. 深度合并配置，补全缺失的参数和默认组件
	newCfg, _ := config.MergeConfig(DefaultServerConfig(), cfg)

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config: newCfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动服务端。
func (s *Server) Start() error {
	if s.config.Acceptor == nil {
		return errors.New("acceptor is required in config")
	}
	return s.config.Acceptor.Start()
}

// Stop 优雅停止服务端。
func (s *Server) Stop() error {
	s.cancel()

	var err error
	if s.config.Acceptor != nil {
		err = s.config.Acceptor.Stop()
	}

	if s.config.Manager != nil {
		_ = s.config.Manager.Close()
	}

	return err
}

// SessionManager 返回会话管理器。
func (s *Server) SessionManager() SessionManager {
	return s.config.Manager
}

// Handler 返回会话处理器。
func (s *Server) Handler() SessionHandler {
	return s.config.Handler
}

// Config 返回服务端配置。
func (s *Server) Config() *ServerConfig {
	return s.config
}

// 内部包装一个 SessionHandler，用于自动将会话加入/移除管理器。
type autoManagerHandler struct {
	manager SessionManager
	handler SessionHandler
}

func (h *autoManagerHandler) OnOpened(s Session) {
	h.manager.Add(s)
	h.handler.OnOpened(s)
}

func (h *autoManagerHandler) OnClosed(s Session, err error) {
	h.manager.Remove(s.ID())
	h.handler.OnClosed(s, err)
}

func (h *autoManagerHandler) OnMessage(s Session, env *common.Envelope) {
	h.handler.OnMessage(s, env)
}

func (h *autoManagerHandler) OnError(s Session, err error) {
	h.handler.OnError(s, err)
}

// ManagedAcceptor 返回一个包装了自动管理逻辑的 Acceptor。
// 它会将会话事件透明地路由到 Config.Manager 中。
func (s *Server) ManagedAcceptor(factory func(SessionHandler) Acceptor) Acceptor {
	mHandler := &autoManagerHandler{
		manager: s.config.Manager,
		handler: s.config.Handler,
	}
	return factory(mHandler)
}