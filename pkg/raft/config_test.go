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

func TestConfig_SerfConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.NodeName = "test-node"
	cfg.Datacenter = "dc1"
	cfg.SerfLANAddr = "127.0.0.1:8301"
	cfg.JoinAddrs = []string{"127.0.0.1:8302", "127.0.0.1:8303"}
	cfg.ExpectNodes = 3

	assert.Equal(t, "test-node", cfg.NodeName)
	assert.Equal(t, "dc1", cfg.Datacenter)
	assert.Equal(t, "127.0.0.1:8301", cfg.SerfLANAddr)
	assert.Equal(t, []string{"127.0.0.1:8302", "127.0.0.1:8303"}, cfg.JoinAddrs)
	assert.Equal(t, 3, cfg.ExpectNodes)
}

func TestConfig_TLSConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.TLSEnabled = true
	cfg.TLSCAFile = "/path/to/ca.pem"
	cfg.TLSCertFile = "/path/to/cert.pem"
	cfg.TLSKeyFile = "/path/to/key.pem"
	cfg.TLSVerify = true

	assert.True(t, cfg.TLSEnabled)
	assert.Equal(t, "/path/to/ca.pem", cfg.TLSCAFile)
	assert.Equal(t, "/path/to/cert.pem", cfg.TLSCertFile)
	assert.Equal(t, "/path/to/key.pem", cfg.TLSKeyFile)
	assert.True(t, cfg.TLSVerify)
}
