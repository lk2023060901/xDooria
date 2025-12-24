// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package consul

import (
	"context"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	autopilot "github.com/hashicorp/raft-autopilot"
	"github.com/hashicorp/serf/serf"

	"github.com/lk2023060901/xdooria/pkg/raft/consul/metadata"
)

// AutopilotConfig holds the Autopilot configuration
type AutopilotConfig struct {
	// CleanupDeadServers controls whether dead servers are removed from
	// the Raft peer set when new servers join the cluster.
	CleanupDeadServers bool

	// LastContactThreshold is the limit on the amount of time a server can go
	// without leader contact before being considered unhealthy.
	LastContactThreshold time.Duration

	// MaxTrailingLogs is the amount of entries in the Raft log that a server
	// can be behind before being considered unhealthy.
	MaxTrailingLogs uint64

	// MinQuorum sets the minimum number of servers necessary in a cluster
	// to maintain quorum. Autopilot will not prune servers below this number.
	MinQuorum uint

	// ServerStabilizationTime is the minimum amount of time a server must be
	// in a stable, healthy state before it can be added to the cluster.
	ServerStabilizationTime time.Duration

	// AutopilotInterval is the interval for Autopilot reconciliation.
	AutopilotInterval time.Duration

	// ServerHealthInterval is the interval for server health checks.
	ServerHealthInterval time.Duration
}

// DefaultAutopilotConfig returns a default Autopilot configuration
func DefaultAutopilotConfig() *AutopilotConfig {
	return &AutopilotConfig{
		CleanupDeadServers:      true,
		LastContactThreshold:    200 * time.Millisecond,
		MaxTrailingLogs:         250,
		MinQuorum:               0,
		ServerStabilizationTime: 10 * time.Second,
		AutopilotInterval:       2 * time.Second,
		ServerHealthInterval:    2 * time.Second,
	}
}

// ToAutopilotLibraryConfig converts to the autopilot library Config
func (c *AutopilotConfig) ToAutopilotLibraryConfig() *autopilot.Config {
	return &autopilot.Config{
		CleanupDeadServers:      c.CleanupDeadServers,
		LastContactThreshold:    c.LastContactThreshold,
		MaxTrailingLogs:         c.MaxTrailingLogs,
		MinQuorum:               c.MinQuorum,
		ServerStabilizationTime: c.ServerStabilizationTime,
	}
}

// AutopilotDelegate implements autopilot.ApplicationIntegration
type AutopilotDelegate struct {
	config       *AutopilotConfig
	raft         *raft.Raft
	serfLAN      *SerfLAN
	statsFetcher *StatsFetcher
	logger       hclog.Logger

	// removeFailedServerFunc is called when autopilot wants to remove a failed server
	removeFailedServerFunc func(serverID string, serverName string) error

	// stateNotifyFunc is called when autopilot state changes
	stateNotifyFunc func(state *autopilot.State)
}

// AutopilotDelegateConfig holds configuration for creating an AutopilotDelegate
type AutopilotDelegateConfig struct {
	Config                 *AutopilotConfig
	Raft                   *raft.Raft
	SerfLAN                *SerfLAN
	StatsFetcher           *StatsFetcher
	Logger                 hclog.Logger
	RemoveFailedServerFunc func(serverID string, serverName string) error
	StateNotifyFunc        func(state *autopilot.State)
}

// NewAutopilotDelegate creates a new AutopilotDelegate
func NewAutopilotDelegate(cfg AutopilotDelegateConfig) *AutopilotDelegate {
	if cfg.Config == nil {
		cfg.Config = DefaultAutopilotConfig()
	}
	if cfg.Logger == nil {
		cfg.Logger = hclog.Default()
	}

	return &AutopilotDelegate{
		config:                 cfg.Config,
		raft:                   cfg.Raft,
		serfLAN:                cfg.SerfLAN,
		statsFetcher:           cfg.StatsFetcher,
		logger:                 cfg.Logger.Named("autopilot"),
		removeFailedServerFunc: cfg.RemoveFailedServerFunc,
		stateNotifyFunc:        cfg.StateNotifyFunc,
	}
}

// AutopilotConfig returns the current Autopilot configuration
func (d *AutopilotDelegate) AutopilotConfig() *autopilot.Config {
	return d.config.ToAutopilotLibraryConfig()
}

// KnownServers returns the known servers from Serf membership
func (d *AutopilotDelegate) KnownServers() map[raft.ServerID]*autopilot.Server {
	return d.autopilotServers()
}

// FetchServerStats fetches server stats from all known servers
func (d *AutopilotDelegate) FetchServerStats(ctx context.Context, servers map[raft.ServerID]*autopilot.Server) map[raft.ServerID]*autopilot.ServerStats {
	if d.statsFetcher == nil {
		return nil
	}
	return d.statsFetcher.Fetch(ctx, servers)
}

// NotifyState is called when the autopilot state changes
func (d *AutopilotDelegate) NotifyState(state *autopilot.State) {
	if d.stateNotifyFunc != nil {
		d.stateNotifyFunc(state)
	}

	// Log state information
	if state.Healthy {
		d.logger.Debug("Cluster is healthy",
			"failure_tolerance", state.FailureTolerance,
			"voters", len(state.Voters),
		)
	} else {
		d.logger.Warn("Cluster is unhealthy",
			"failure_tolerance", state.FailureTolerance,
			"voters", len(state.Voters),
		)
	}
}

// RemoveFailedServer is called when autopilot wants to remove a failed server
func (d *AutopilotDelegate) RemoveFailedServer(srv *autopilot.Server) {
	if d.removeFailedServerFunc == nil {
		d.logger.Warn("No removeFailedServerFunc configured, cannot remove server",
			"server", srv.Name, "id", srv.ID)
		return
	}

	go func() {
		if err := d.removeFailedServerFunc(string(srv.ID), srv.Name); err != nil {
			d.logger.Error("failed to remove server", "name", srv.Name, "id", srv.ID, "error", err)
		}
	}()
}

// autopilotServers returns all known servers in autopilot format
func (d *AutopilotDelegate) autopilotServers() map[raft.ServerID]*autopilot.Server {
	servers := make(map[raft.ServerID]*autopilot.Server)

	for _, member := range d.serfLAN.Members() {
		srv, err := d.autopilotServer(member)
		if err != nil {
			d.logger.Warn("Error parsing server info", "name", member.Name, "error", err)
			continue
		} else if srv == nil {
			// this member was a client or non-server
			continue
		}

		servers[srv.ID] = srv
	}

	return servers
}

// autopilotServer converts a Serf member to an autopilot.Server
func (d *AutopilotDelegate) autopilotServer(m serf.Member) (*autopilot.Server, error) {
	ok, srv := metadata.IsConsulServer(m)
	if !ok {
		return nil, nil
	}

	return d.autopilotServerFromMetadata(srv)
}

// autopilotServerFromMetadata converts server metadata to autopilot.Server
func (d *AutopilotDelegate) autopilotServerFromMetadata(srv *metadata.Server) (*autopilot.Server, error) {
	server := &autopilot.Server{
		Name:        srv.ShortName,
		ID:          raft.ServerID(srv.ID),
		Address:     raft.ServerAddress(srv.Addr.String()),
		Version:     srv.Build.String(),
		RaftVersion: srv.RaftVersion,
	}

	switch srv.Status {
	case serf.StatusLeft:
		server.NodeStatus = autopilot.NodeLeft
	case serf.StatusAlive, serf.StatusLeaving:
		// we want to treat leaving as alive to prevent autopilot from
		// prematurely removing the node.
		server.NodeStatus = autopilot.NodeAlive
	case serf.StatusFailed:
		server.NodeStatus = autopilot.NodeFailed
	default:
		server.NodeStatus = autopilot.NodeUnknown
	}

	return server, nil
}

// Autopilot wraps the autopilot.Autopilot with our configuration
type Autopilot struct {
	autopilot *autopilot.Autopilot
	delegate  *AutopilotDelegate
	raft      *raft.Raft
	logger    hclog.Logger
}

// AutopilotOptions holds options for creating an Autopilot
type AutopilotOptions struct {
	Config                 *AutopilotConfig
	Raft                   *raft.Raft
	SerfLAN                *SerfLAN
	StatsFetcher           *StatsFetcher
	Logger                 hclog.Logger
	RemoveFailedServerFunc func(serverID string, serverName string) error
	StateNotifyFunc        func(state *autopilot.State)
}

// NewAutopilot creates a new Autopilot instance
func NewAutopilot(opts AutopilotOptions) (*Autopilot, error) {
	if opts.Config == nil {
		opts.Config = DefaultAutopilotConfig()
	}
	if opts.Logger == nil {
		opts.Logger = hclog.Default()
	}

	delegate := NewAutopilotDelegate(AutopilotDelegateConfig{
		Config:                 opts.Config,
		Raft:                   opts.Raft,
		SerfLAN:                opts.SerfLAN,
		StatsFetcher:           opts.StatsFetcher,
		Logger:                 opts.Logger,
		RemoveFailedServerFunc: opts.RemoveFailedServerFunc,
		StateNotifyFunc:        opts.StateNotifyFunc,
	})

	ap := autopilot.New(
		opts.Raft,
		delegate,
		autopilot.WithLogger(opts.Logger),
		autopilot.WithReconcileInterval(opts.Config.AutopilotInterval),
		autopilot.WithUpdateInterval(opts.Config.ServerHealthInterval),
		autopilot.WithReconciliationDisabled(),
	)

	return &Autopilot{
		autopilot: ap,
		delegate:  delegate,
		raft:      opts.Raft,
		logger:    opts.Logger.Named("autopilot"),
	}, nil
}

// Start starts the Autopilot
func (a *Autopilot) Start(ctx context.Context) {
	a.autopilot.Start(ctx)
}

// Stop stops the Autopilot
func (a *Autopilot) Stop() {
	// Note: autopilot doesn't have a Stop method, it uses context cancellation
}

// EnableReconciliation enables Raft reconciliation
func (a *Autopilot) EnableReconciliation() {
	a.autopilot.EnableReconciliation()
}

// DisableReconciliation disables Raft reconciliation
func (a *Autopilot) DisableReconciliation() {
	a.autopilot.DisableReconciliation()
}

// GetState returns the current autopilot state
func (a *Autopilot) GetState() *autopilot.State {
	return a.autopilot.GetState()
}

// IsHealthy returns whether the cluster is healthy
func (a *Autopilot) IsHealthy() bool {
	state := a.GetState()
	return state != nil && state.Healthy
}

// FailureTolerance returns the current failure tolerance
func (a *Autopilot) FailureTolerance() int {
	state := a.GetState()
	if state == nil {
		return 0
	}
	return state.FailureTolerance
}

// AddServer manually adds a server to the Raft configuration.
// This is typically handled by autopilot automatically, but can be called
// manually if needed.
func (a *Autopilot) AddServer(id raft.ServerID, address raft.ServerAddress) error {
	future := a.raft.AddVoter(id, address, 0, 0)
	return future.Error()
}

// RemoveServer manually removes a server from the Raft configuration.
// This is typically handled by autopilot automatically, but can be called
// manually if needed.
func (a *Autopilot) RemoveServer(id raft.ServerID) error {
	future := a.raft.RemoveServer(id, 0, 0)
	return future.Error()
}
