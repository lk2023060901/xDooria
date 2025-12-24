// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package consul

import (
	"fmt"
	"io"
	"net"
	"net/rpc"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/yamux"

	"github.com/lk2023060901/xdooria/pkg/raft/consul/pool"
	"github.com/lk2023060901/xdooria/pkg/util/conc"
)

// RPCServer handles incoming RPC connections
type RPCServer struct {
	rpcServer  *rpc.Server
	raft       *raft.Raft
	logger     hclog.Logger
	shutdownCh chan struct{}
	shutdown   bool
	mu         sync.Mutex

	// 使用 conc.Pool 管理 goroutine
	connPool *conc.Pool[struct{}]
}

// RPCServerConfig holds configuration for the RPC server
type RPCServerConfig struct {
	// Raft is the Raft instance to query for stats
	Raft *raft.Raft

	// Logger is the logger to use
	Logger hclog.Logger

	// MaxConnections is the maximum number of concurrent connections
	MaxConnections int
}

// NewRPCServer creates a new RPC server (no listener - connections are routed from main listener)
func NewRPCServer(config *RPCServerConfig) (*RPCServer, error) {
	if config.Logger == nil {
		config.Logger = hclog.Default()
	}

	if config.MaxConnections <= 0 {
		config.MaxConnections = 256
	}

	// Create RPC server
	rpcServer := rpc.NewServer()

	// Create and register Status endpoint
	status := &Status{raft: config.Raft}
	if err := rpcServer.RegisterName("Status", status); err != nil {
		return nil, fmt.Errorf("failed to register Status endpoint: %w", err)
	}

	// Create connection pool using conc.Pool
	connPool := conc.NewPool[struct{}](config.MaxConnections, conc.WithPreAlloc(false))

	s := &RPCServer{
		rpcServer:  rpcServer,
		raft:       config.Raft,
		logger:     config.Logger.Named("rpc"),
		shutdownCh: make(chan struct{}),
		connPool:   connPool,
	}

	return s, nil
}

// HandleConn handles a single RPC connection (called from main listener)
func (s *RPCServer) HandleConn(conn net.Conn) {
	// 检查是否正在关闭
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		conn.Close()
		return
	}
	s.mu.Unlock()

	s.connPool.Submit(func() (struct{}, error) {
		s.handleConsulConn(conn)
		return struct{}{}, nil
	})
}

// HandleMultiplexV2 handles multiplexed connections (called from main listener)
func (s *RPCServer) HandleMultiplexV2(conn net.Conn) {
	// 检查是否正在关闭
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		conn.Close()
		return
	}
	s.mu.Unlock()

	s.connPool.Submit(func() (struct{}, error) {
		s.handleMultiplexV2(conn)
		return struct{}{}, nil
	})
}

// handleConsulConn handles a single Consul RPC connection
func (s *RPCServer) handleConsulConn(conn net.Conn) {
	defer conn.Close()

	codec := pool.NewMsgpackServerCodec(conn)
	for {
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		if err := s.rpcServer.ServeRequest(codec); err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "closed") {
				return
			}
			s.logger.Error("RPC error", "error", err)
			return
		}
	}
}

// handleMultiplexV2 handles multiplexed connections using yamux
func (s *RPCServer) handleMultiplexV2(conn net.Conn) {
	defer conn.Close()

	conf := yamux.DefaultConfig()
	conf.LogOutput = nil
	conf.Logger = s.logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true})

	server, err := yamux.Server(conn, conf)
	if err != nil {
		s.logger.Error("failed to create yamux server", "error", err)
		return
	}
	defer server.Close()

	// 监听 shutdown 信号，关闭 yamux server 以中断 Accept
	go func() {
		<-s.shutdownCh
		server.Close()
	}()

	for {
		// 检查是否正在关闭
		select {
		case <-s.shutdownCh:
			return
		default:
		}

		sub, err := server.Accept()
		if err != nil {
			if err != io.EOF && !strings.Contains(err.Error(), "closed") {
				s.logger.Error("multiplex conn accept failed", "error", err)
			}
			return
		}

		// 检查 shutdown 后再提交任务
		s.mu.Lock()
		if s.shutdown {
			s.mu.Unlock()
			sub.Close()
			return
		}
		s.mu.Unlock()

		// 使用 conc.Pool 处理 yamux stream
		s.connPool.Submit(func() (struct{}, error) {
			s.handleConsulConn(sub)
			return struct{}{}, nil
		})
	}
}

// Shutdown stops the RPC server
func (s *RPCServer) Shutdown() error {
	s.mu.Lock()
	if s.shutdown {
		s.mu.Unlock()
		return nil
	}
	s.shutdown = true
	close(s.shutdownCh)
	s.mu.Unlock()

	// Release the connection pool
	s.connPool.Release()

	return nil
}

// Status endpoint for RPC
type Status struct {
	raft *raft.Raft
}

// Ping is used to check connectivity
func (s *Status) Ping(args struct{}, reply *struct{}) error {
	return nil
}

// RaftStats returns the Raft stats for autopilot
func (s *Status) RaftStats(args EmptyReadRequest, reply *RaftStats) error {
	stats := s.raft.Stats()

	reply.LastContact = stats["last_contact"]

	var err error
	if lastIndex := stats["last_log_index"]; lastIndex != "" {
		_, err = fmt.Sscanf(lastIndex, "%d", &reply.LastIndex)
		if err != nil {
			return fmt.Errorf("error parsing last_log_index: %w", err)
		}
	}

	if lastTerm := stats["last_log_term"]; lastTerm != "" {
		_, err = fmt.Sscanf(lastTerm, "%d", &reply.LastTerm)
		if err != nil {
			return fmt.Errorf("error parsing last_log_term: %w", err)
		}
	}

	return nil
}

// Leader returns the address of the current leader
func (s *Status) Leader(args struct{}, reply *string) error {
	*reply = string(s.raft.Leader())
	return nil
}

// Peers returns the list of peers in the Raft cluster
func (s *Status) Peers(args struct{}, reply *[]string) error {
	future := s.raft.GetConfiguration()
	if err := future.Error(); err != nil {
		return err
	}

	for _, server := range future.Configuration().Servers {
		*reply = append(*reply, string(server.Address))
	}
	return nil
}
