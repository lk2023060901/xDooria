// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package consul

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/serf/serf"

	"github.com/lk2023060901/xdooria/pkg/raft/consul/libserf"
	"github.com/lk2023060901/xdooria/pkg/raft/consul/metadata"
	"github.com/lk2023060901/xdooria/pkg/raft/consul/pool"
)

// createDirIfNotExist creates a directory if it doesn't exist
func createDirIfNotExist(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

const (
	// StatusReap is used to update the status of a node if we
	// are handling a EventMemberReap
	StatusReap = serf.MemberStatus(-1)

	// maxPeerRetries limits how many invalidate attempts are made
	maxPeerRetries = 6

	// protocolVersionMap maps Consul protocol versions to Serf protocol versions.
	protocolVersionMin = 2
	protocolVersionMax = 3
)

// protocolVersionMap maps Consul protocol versions to Serf protocol versions.
var protocolVersionMap = map[int]uint8{
	2: 4,
	3: 4,
}

// SerfEventHandler defines the interface for handling Serf events
type SerfEventHandler interface {
	HandleMemberJoin(members []serf.Member)
	HandleMemberLeave(members []serf.Member)
	HandleMemberFailed(members []serf.Member)
	HandleMemberUpdate(members []serf.Member)
	HandleMemberReap(members []serf.Member)
}

// ServerLookup provides lookup for servers
type ServerLookup struct {
	mu      sync.RWMutex
	servers map[raft.ServerID]*metadata.Server
}

// NewServerLookup creates a new ServerLookup
func NewServerLookup() *ServerLookup {
	return &ServerLookup{
		servers: make(map[raft.ServerID]*metadata.Server),
	}
}

// AddServer adds a server to the lookup
func (sl *ServerLookup) AddServer(s *metadata.Server) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.servers[raft.ServerID(s.ID)] = s
}

// RemoveServer removes a server from the lookup
func (sl *ServerLookup) RemoveServer(s *metadata.Server) {
	sl.mu.Lock()
	defer sl.mu.Unlock()
	delete(sl.servers, raft.ServerID(s.ID))
}

// Server returns a server by ID
func (sl *ServerLookup) Server(id raft.ServerID) *metadata.Server {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	return sl.servers[id]
}

// Servers returns all servers
func (sl *ServerLookup) Servers() []*metadata.Server {
	sl.mu.RLock()
	defer sl.mu.RUnlock()
	servers := make([]*metadata.Server, 0, len(sl.servers))
	for _, s := range sl.servers {
		servers = append(servers, s)
	}
	return servers
}

// SerfLANConfig holds configuration for the LAN Serf cluster
type SerfLANConfig struct {
	// NodeName is the name of the local node
	NodeName string

	// NodeID is the unique identifier for this node (should match Raft server ID)
	NodeID string

	// Datacenter is the name of the local datacenter
	Datacenter string

	// BindAddr is the address to bind to for Serf
	BindAddr string

	// BindPort is the port to bind to for Serf
	BindPort int

	// RaftAddr is the Raft address to advertise
	RaftAddr string

	// RaftPort is the Raft port to advertise
	RaftPort int

	// Bootstrap indicates if this server should bootstrap the cluster
	Bootstrap bool

	// BootstrapExpect is the expected number of servers in the cluster
	BootstrapExpect int

	// ReadReplica indicates if this server is a read replica
	ReadReplica bool

	// UseTLS indicates if TLS is enabled
	UseTLS bool

	// Build is the build version string
	Build string

	// ProtocolVersion is the Consul protocol version
	ProtocolVersion int

	// RaftVersion is the Raft protocol version
	RaftVersion int

	// DataDir is the directory for Serf snapshots
	DataDir string

	// RejoinAfterLeave controls rejoin after leave
	RejoinAfterLeave bool

	// Logger is the logger to use
	Logger hclog.Logger
}

// SerfLAN manages the LAN Serf cluster for server discovery
type SerfLAN struct {
	serf   *serf.Serf
	config *SerfLANConfig

	// eventCh is the channel for Serf events
	eventCh chan serf.Event

	// shutdownCh is closed when the server is shutting down
	shutdownCh chan struct{}

	// logger is the logger to use
	logger hclog.Logger

	// serverLookup tracks known servers
	serverLookup *ServerLookup

	// raft is the Raft instance for bootstrap
	raft *raft.Raft

	// raftStore is the LogStore for checking if bootstrap has occurred
	raftStore raft.LogStore

	// connPool is the connection pool for RPC
	connPool *pool.ConnPool

	// bootstrapExpect tracks the original bootstrap expect value
	bootstrapExpect int

	// bootstrapLock protects bootstrap operations
	bootstrapLock sync.Mutex

	// reconcileCh is used to signal reconciliation
	reconcileCh chan serf.Member

	// isLeader indicates if this node is the leader
	isLeader func() bool
}

// NewSerfLAN creates a new SerfLAN instance
func NewSerfLAN(config *SerfLANConfig, r *raft.Raft, store raft.LogStore, connPool *pool.ConnPool, isLeader func() bool) (*SerfLAN, error) {
	if config.Logger == nil {
		config.Logger = hclog.Default()
	}

	sl := &SerfLAN{
		config:          config,
		eventCh:         make(chan serf.Event, 256),
		shutdownCh:      make(chan struct{}),
		logger:          config.Logger.Named("serf.lan"),
		serverLookup:    NewServerLookup(),
		raft:            r,
		raftStore:       store,
		connPool:        connPool,
		bootstrapExpect: config.BootstrapExpect,
		reconcileCh:     make(chan serf.Member, 32),
		isLeader:        isLeader,
	}

	// Create Serf configuration
	serfConfig, err := sl.setupSerfConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to setup serf config: %w", err)
	}

	// Create Serf instance
	s, err := serf.Create(serfConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create serf: %w", err)
	}
	sl.serf = s

	// Start event handler
	go sl.lanEventHandler()

	return sl, nil
}

// setupSerfConfig creates the Serf configuration
func (sl *SerfLAN) setupSerfConfig() (*serf.Config, error) {
	conf := libserf.DefaultConfig()
	conf.Init()

	// Use NodeID as the memberlist node name to avoid conflicts with memberlist's internal ID tracking
	// memberlist v0.5.x uses the "id" tag internally, so we must ensure consistency
	serfNodeName := sl.config.NodeID
	if serfNodeName == "" {
		serfNodeName = sl.config.NodeName // Fallback to NodeName if not set
	}
	conf.NodeName = serfNodeName
	conf.Tags = map[string]string{
		"role":       "consul",
		"dc":         sl.config.Datacenter,
		"port":       strconv.Itoa(sl.config.RaftPort),
		"id":         serfNodeName, // Must match memberlist node name
		"node_name":  sl.config.NodeName, // Store original node name for display
		"vsn":        strconv.Itoa(sl.config.ProtocolVersion),
		"vsn_min":    strconv.Itoa(protocolVersionMin),
		"vsn_max":    strconv.Itoa(protocolVersionMax),
		"raft_vsn":   strconv.Itoa(sl.config.RaftVersion),
		"build":      sl.config.Build,
	}

	if sl.config.Bootstrap {
		conf.Tags["bootstrap"] = "1"
	}
	if sl.config.BootstrapExpect != 0 {
		conf.Tags["expect"] = strconv.Itoa(sl.config.BootstrapExpect)
	}
	if sl.config.ReadReplica {
		conf.Tags["read_replica"] = "1"
	}
	if sl.config.UseTLS {
		conf.Tags["use_tls"] = "1"
	}

	conf.MemberlistConfig.BindAddr = sl.config.BindAddr
	conf.MemberlistConfig.BindPort = sl.config.BindPort
	conf.MemberlistConfig.AdvertiseAddr = sl.config.BindAddr
	conf.MemberlistConfig.AdvertisePort = sl.config.BindPort

	if sl.config.Logger != nil {
		conf.Logger = sl.config.Logger.StandardLogger(&hclog.StandardLoggerOptions{InferLevels: true})
		conf.MemberlistConfig.Logger = conf.Logger
	}

	conf.EventCh = sl.eventCh
	if pv, ok := protocolVersionMap[sl.config.ProtocolVersion]; ok {
		conf.ProtocolVersion = pv
	} else {
		conf.ProtocolVersion = 4
	}
	conf.RejoinAfterLeave = sl.config.RejoinAfterLeave

	// Merge delegate for LAN
	// Use serfNodeName (which is the NodeID/UUID) for validation
	conf.Merge = &lanMergeDelegate{
		dc:       sl.config.Datacenter,
		nodeID:   serfNodeName, // Use the same ID as serf node name
		nodeName: serfNodeName, // Serf node name is now the UUID
		logger:   sl.logger,
	}

	// Disable automatic name conflict resolution
	conf.EnableNameConflictResolution = false

	// Set snapshot path
	if sl.config.DataDir != "" {
		serfDir := filepath.Join(sl.config.DataDir, "serf")
		if err := createDirIfNotExist(serfDir); err != nil {
			return nil, fmt.Errorf("failed to create serf directory: %w", err)
		}
		conf.SnapshotPath = filepath.Join(serfDir, "local.snapshot")
	}

	conf.ReconnectTimeoutOverride = libserf.NewReconnectOverride(sl.logger)

	return conf, nil
}

// lanMergeDelegate is used to handle node joins in the LAN pool
type lanMergeDelegate struct {
	dc       string
	nodeID   string
	nodeName string
	logger   hclog.Logger
}

// NotifyMerge is called when a member is about to be merged
func (d *lanMergeDelegate) NotifyMerge(members []*serf.Member) error {
	for _, m := range members {
		// Validate the member is in the same datacenter
		if m.Tags["dc"] != d.dc {
			return fmt.Errorf("member '%s' part of wrong datacenter '%s'; expected '%s'",
				m.Name, m.Tags["dc"], d.dc)
		}

		// Validate the node name doesn't conflict
		if m.Name == d.nodeName && m.Tags["id"] != d.nodeID {
			return fmt.Errorf("member '%s' has conflicting node ID; expected '%s' but got '%s'",
				m.Name, d.nodeID, m.Tags["id"])
		}
	}
	return nil
}

// lanEventHandler handles LAN Serf events
func (sl *SerfLAN) lanEventHandler() {
	for {
		select {
		case e := <-sl.eventCh:
			switch e.EventType() {
			case serf.EventMemberJoin:
				sl.lanNodeJoin(e.(serf.MemberEvent))
				sl.localMemberEvent(e.(serf.MemberEvent))

			case serf.EventMemberLeave, serf.EventMemberFailed, serf.EventMemberReap:
				sl.lanNodeFailed(e.(serf.MemberEvent))
				sl.localMemberEvent(e.(serf.MemberEvent))

			case serf.EventMemberUpdate:
				sl.lanNodeUpdate(e.(serf.MemberEvent))
				sl.localMemberEvent(e.(serf.MemberEvent))

			case serf.EventUser:
				// Handle user events if needed
			case serf.EventQuery:
				// Handle queries if needed
			default:
				sl.logger.Warn("Unhandled LAN Serf Event", "event", e)
			}

		case <-sl.shutdownCh:
			return
		}
	}
}

// localMemberEvent is used to reconcile Serf events with the strongly
// consistent store if we are the current leader
func (sl *SerfLAN) localMemberEvent(me serf.MemberEvent) {
	// Do nothing if we are not the leader
	if sl.isLeader == nil || !sl.isLeader() {
		return
	}

	// Check if this is a reap event
	isReap := me.EventType() == serf.EventMemberReap

	// Queue the members for reconciliation
	for _, m := range me.Members {
		// Change the status if this is a reap event
		if isReap {
			m.Status = StatusReap
		}
		select {
		case sl.reconcileCh <- m:
		default:
		}
	}
}

// lanNodeJoin handles join events on the LAN pool
func (sl *SerfLAN) lanNodeJoin(me serf.MemberEvent) {
	for _, m := range me.Members {
		ok, serverMeta := metadata.IsConsulServer(m)
		if !ok || serverMeta.Segment != "" {
			continue
		}
		sl.logger.Info("Adding LAN server", "server", serverMeta.String())

		// Update server lookup
		sl.serverLookup.AddServer(serverMeta)

		// If we're still expecting to bootstrap, may need to handle this
		sl.bootstrapLock.Lock()
		if sl.bootstrapExpect != 0 {
			sl.maybeBootstrap()
		}
		sl.bootstrapLock.Unlock()
	}
}

// lanNodeUpdate handles update events on the LAN pool
func (sl *SerfLAN) lanNodeUpdate(me serf.MemberEvent) {
	for _, m := range me.Members {
		ok, serverMeta := metadata.IsConsulServer(m)
		if !ok || serverMeta.Segment != "" {
			continue
		}
		sl.logger.Info("Updating LAN server", "server", serverMeta.String())

		// Update server lookup
		sl.serverLookup.AddServer(serverMeta)
	}
}

// lanNodeFailed handles fail/leave/reap events on the LAN pool
func (sl *SerfLAN) lanNodeFailed(me serf.MemberEvent) {
	for _, m := range me.Members {
		ok, serverMeta := metadata.IsConsulServer(m)
		if !ok || serverMeta.Segment != "" {
			continue
		}
		sl.logger.Info("Removing LAN server", "server", serverMeta.String())

		// Update server lookup
		sl.serverLookup.RemoveServer(serverMeta)
	}
}

// maybeBootstrap is used to handle bootstrapping when a new server joins.
// MUST be called with bootstrapLock held.
func (sl *SerfLAN) maybeBootstrap() {
	// Bootstrap can only be done if there are no committed logs, remove our
	// expectations of bootstrapping. This is slightly cheaper than the full
	// check that BootstrapCluster will do, so this is a good pre-filter.
	index, err := sl.raftStore.LastIndex()
	if err != nil {
		sl.logger.Error("Failed to read last raft index", "error", err)
		return
	}
	if index != 0 {
		sl.logger.Info("Raft data found, disabling bootstrap mode")
		sl.bootstrapExpect = 0
		return
	}

	if sl.config.ReadReplica {
		sl.logger.Info("Read replicas cannot bootstrap raft")
		return
	}

	// Scan for all the known servers
	members := sl.serf.Members()
	var servers []metadata.Server
	voters := 0
	for _, member := range members {
		valid, p := metadata.IsConsulServer(member)
		if !valid {
			continue
		}
		if p.Datacenter != sl.config.Datacenter {
			sl.logger.Warn("Member has a conflicting datacenter, ignoring", "member", member)
			continue
		}
		if p.Expect != 0 && p.Expect != sl.bootstrapExpect {
			sl.logger.Error("Member has a conflicting expect value. All nodes should expect the same number.", "member", member)
			return
		}
		if p.Bootstrap {
			sl.logger.Error("Member has bootstrap mode. Expect disabled.", "member", member)
			return
		}
		if !p.ReadReplica {
			voters++
		}
		servers = append(servers, *p)
	}

	// Skip if we haven't met the minimum expect count
	if voters < sl.bootstrapExpect {
		return
	}

	// Query each of the servers and make sure they report no Raft peers
	for _, server := range servers {
		var peers []string

		// Retry with exponential backoff to get peer status from this server
		for attempt := uint(0); attempt < maxPeerRetries; attempt++ {
			if err := sl.connPool.RPC(sl.config.Datacenter, server.ShortName, server.Addr,
				"Status.Peers", &EmptyReadRequest{}, &peers); err != nil {
				nextRetry := (1 << attempt) * time.Second
				sl.logger.Error("Failed to confirm peer status for server (will retry).",
					"server", server.Name,
					"retry_interval", nextRetry.String(),
					"error", err,
				)
				time.Sleep(nextRetry)
			} else {
				break
			}
		}

		// Found a node with some Raft peers, stop bootstrap since there's
		// evidence of an existing cluster. We should get folded in by the
		// existing servers if that's the case, so it's cleaner to sit as a
		// candidate with no peers so we don't cause spurious elections.
		if len(peers) > 0 {
			sl.logger.Info("Existing Raft peers reported by server, disabling bootstrap mode", "server", server.Name)
			sl.bootstrapExpect = 0
			return
		}
	}

	// Attempt a live bootstrap!
	var configuration raft.Configuration
	var addrs []string

	for _, server := range servers {
		addr := server.Addr.String()
		addrs = append(addrs, addr)
		id := raft.ServerID(server.ID)

		suffrage := raft.Voter
		if server.ReadReplica {
			suffrage = raft.Nonvoter
		}
		peer := raft.Server{
			ID:       id,
			Address:  raft.ServerAddress(addr),
			Suffrage: suffrage,
		}
		configuration.Servers = append(configuration.Servers, peer)
	}
	sl.logger.Info("Found expected number of peers, attempting bootstrap",
		"peers", strings.Join(addrs, ","),
	)
	future := sl.raft.BootstrapCluster(configuration)
	if err := future.Error(); err != nil {
		sl.logger.Error("Failed to bootstrap cluster", "error", err)
	}

	// Bootstrapping complete, or failed for some reason, don't enter this again
	sl.bootstrapExpect = 0
}

// Join attempts to join the given addresses
func (sl *SerfLAN) Join(addrs []string) (int, error) {
	return sl.serf.Join(addrs, true)
}

// Members returns the current members of the Serf cluster
func (sl *SerfLAN) Members() []serf.Member {
	return sl.serf.Members()
}

// LocalMember returns the local member
func (sl *SerfLAN) LocalMember() serf.Member {
	return sl.serf.LocalMember()
}

// NumNodes returns the number of nodes in the cluster
func (sl *SerfLAN) NumNodes() int {
	return sl.serf.NumNodes()
}

// ServerLookup returns the server lookup
func (sl *SerfLAN) ServerLookup() *ServerLookup {
	return sl.serverLookup
}

// ReconcileCh returns the reconciliation channel
func (sl *SerfLAN) ReconcileCh() <-chan serf.Member {
	return sl.reconcileCh
}

// Shutdown gracefully shuts down the Serf cluster
func (sl *SerfLAN) Shutdown() error {
	close(sl.shutdownCh)
	return sl.serf.Shutdown()
}

// Leave gracefully leaves the Serf cluster
func (sl *SerfLAN) Leave() error {
	return sl.serf.Leave()
}

// GetServer returns server info by address
func (sl *SerfLAN) GetServer(addr net.Addr) *metadata.Server {
	for _, member := range sl.serf.Members() {
		ok, server := metadata.IsConsulServer(member)
		if !ok {
			continue
		}
		if server.Addr.String() == addr.String() {
			return server
		}
	}
	return nil
}
