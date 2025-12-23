package raft

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, "127.0.0.1:7000", cfg.BindAddr)
	assert.Equal(t, "./raft-data", cfg.DataDir)
	assert.Equal(t, 1000*time.Millisecond, cfg.HeartbeatTimeout)
	assert.Equal(t, 1000*time.Millisecond, cfg.ElectionTimeout)
	assert.Equal(t, uint64(8192), cfg.SnapshotThreshold)
	assert.Equal(t, 3, cfg.SnapshotRetain)
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config",
			modify:  func(c *Config) {},
			wantErr: false,
		},
		{
			name: "missing bind_addr",
			modify: func(c *Config) {
				c.BindAddr = ""
			},
			wantErr: true,
			errMsg:  "bind_addr is required",
		},
		{
			name: "invalid bind_addr format",
			modify: func(c *Config) {
				c.BindAddr = "invalid-addr"
			},
			wantErr: true,
			errMsg:  "invalid bind_addr",
		},
		{
			name: "missing data_dir",
			modify: func(c *Config) {
				c.DataDir = ""
			},
			wantErr: true,
			errMsg:  "data_dir is required",
		},
		{
			name: "zero heartbeat_timeout",
			modify: func(c *Config) {
				c.HeartbeatTimeout = 0
			},
			wantErr: true,
			errMsg:  "heartbeat_timeout must be positive",
		},
		{
			name: "zero election_timeout",
			modify: func(c *Config) {
				c.ElectionTimeout = 0
			},
			wantErr: true,
			errMsg:  "election_timeout must be positive",
		},
		{
			name: "election_timeout less than heartbeat_timeout",
			modify: func(c *Config) {
				c.HeartbeatTimeout = 2 * time.Second
				c.ElectionTimeout = 1 * time.Second
			},
			wantErr: true,
			errMsg:  "election_timeout must be >= heartbeat_timeout",
		},
		{
			name: "zero snapshot_threshold",
			modify: func(c *Config) {
				c.SnapshotThreshold = 0
			},
			wantErr: true,
			errMsg:  "snapshot_threshold must be positive",
		},
		{
			name: "zero snapshot_retain",
			modify: func(c *Config) {
				c.SnapshotRetain = 0
			},
			wantErr: true,
			errMsg:  "snapshot_retain must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(cfg)

			err := cfg.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestConfig_ToRaftConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.HeartbeatTimeout = 500 * time.Millisecond
	cfg.ElectionTimeout = 1 * time.Second
	cfg.LogLevel = "debug"

	raftCfg := cfg.ToRaftConfig("test-node")

	assert.Equal(t, "test-node", string(raftCfg.LocalID))
	assert.Equal(t, 500*time.Millisecond, raftCfg.HeartbeatTimeout)
	assert.Equal(t, 1*time.Second, raftCfg.ElectionTimeout)
	assert.Equal(t, "DEBUG", raftCfg.LogLevel)
}

func TestParsePeers(t *testing.T) {
	tests := []struct {
		name    string
		peers   []string
		want    []Peer
		wantErr bool
		errMsg  string
	}{
		{
			name:  "valid peers",
			peers: []string{"node1=127.0.0.1:7001", "node2=127.0.0.1:7002"},
			want: []Peer{
				{ID: "node1", Address: "127.0.0.1:7001"},
				{ID: "node2", Address: "127.0.0.1:7002"},
			},
			wantErr: false,
		},
		{
			name:    "empty peers",
			peers:   []string{},
			want:    []Peer{},
			wantErr: false,
		},
		{
			name:    "invalid format - no equals",
			peers:   []string{"node1:127.0.0.1:7001"},
			wantErr: true,
			errMsg:  "invalid peer format",
		},
		{
			name:    "invalid format - no port",
			peers:   []string{"node1=127.0.0.1"},
			wantErr: true,
			errMsg:  "invalid peer address",
		},
		{
			name:    "invalid format - empty",
			peers:   []string{""},
			wantErr: true,
			errMsg:  "invalid peer format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePeers(tt.peers)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}
