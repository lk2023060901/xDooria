package etcd

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Endpoints) == 0 {
		t.Error("Expected default endpoints")
	}

	if cfg.DialTimeout <= 0 {
		t.Error("Expected positive dial timeout")
	}

	if cfg.MaxCallSendMsgSize <= 0 {
		t.Error("Expected positive max send msg size")
	}

	if cfg.MaxCallRecvMsgSize <= 0 {
		t.Error("Expected positive max recv msg size")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "empty endpoints",
			config: &Config{
				Endpoints:   []string{},
				DialTimeout: 5 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "zero dial timeout",
			config: &Config{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: 0,
			},
			wantErr: true,
		},
		{
			name: "negative dial timeout",
			config: &Config{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: -1 * time.Second,
			},
			wantErr: true,
		},
		{
			name: "incomplete TLS config - missing cert",
			config: &Config{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: 5 * time.Second,
				KeyFile:     "/path/to/key",
				CAFile:      "/path/to/ca",
			},
			wantErr: true,
		},
		{
			name: "incomplete TLS config - missing key",
			config: &Config{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: 5 * time.Second,
				CertFile:    "/path/to/cert",
				CAFile:      "/path/to/ca",
			},
			wantErr: true,
		},
		{
			name: "incomplete TLS config - missing CA",
			config: &Config{
				Endpoints:   []string{"localhost:2379"},
				DialTimeout: 5 * time.Second,
				CertFile:    "/path/to/cert",
				KeyFile:     "/path/to/key",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigToClientConfig(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Username = "testuser"
	cfg.Password = "testpass"

	clientCfg, err := cfg.ToClientConfig()
	if err != nil {
		t.Fatalf("ToClientConfig() error = %v", err)
	}

	if len(clientCfg.Endpoints) == 0 {
		t.Error("Expected endpoints in client config")
	}

	if clientCfg.DialTimeout != cfg.DialTimeout {
		t.Errorf("Expected DialTimeout=%v, got %v", cfg.DialTimeout, clientCfg.DialTimeout)
	}

	if clientCfg.Username != cfg.Username {
		t.Errorf("Expected Username=%s, got %s", cfg.Username, clientCfg.Username)
	}

	if clientCfg.Password != cfg.Password {
		t.Errorf("Expected Password=%s, got %s", cfg.Password, clientCfg.Password)
	}
}
