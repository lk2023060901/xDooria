package kafka

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	// 验证 Brokers
	if len(cfg.Brokers) != 1 || cfg.Brokers[0] != "localhost:9092" {
		t.Errorf("expected default brokers to be [localhost:9092], got %v", cfg.Brokers)
	}

	// 验证 Producer 配置
	if cfg.Producer.Async != false {
		t.Errorf("expected Producer.Async to be false, got %v", cfg.Producer.Async)
	}
	if cfg.Producer.BatchSize != 100 {
		t.Errorf("expected Producer.BatchSize to be 100, got %d", cfg.Producer.BatchSize)
	}
	if cfg.Producer.BatchTimeout != 1*time.Second {
		t.Errorf("expected Producer.BatchTimeout to be 1s, got %v", cfg.Producer.BatchTimeout)
	}
	if cfg.Producer.MaxRetries != 3 {
		t.Errorf("expected Producer.MaxRetries to be 3, got %d", cfg.Producer.MaxRetries)
	}
	if cfg.Producer.RequiredAcks != -1 {
		t.Errorf("expected Producer.RequiredAcks to be -1, got %d", cfg.Producer.RequiredAcks)
	}
	if cfg.Producer.Compression != "snappy" {
		t.Errorf("expected Producer.Compression to be snappy, got %s", cfg.Producer.Compression)
	}

	// 验证 Consumer 配置
	if cfg.Consumer.GroupID != "default-group" {
		t.Errorf("expected Consumer.GroupID to be default-group, got %s", cfg.Consumer.GroupID)
	}
	if cfg.Consumer.MinBytes != 10*1024 {
		t.Errorf("expected Consumer.MinBytes to be 10KB, got %d", cfg.Consumer.MinBytes)
	}
	if cfg.Consumer.MaxBytes != 10*1024*1024 {
		t.Errorf("expected Consumer.MaxBytes to be 10MB, got %d", cfg.Consumer.MaxBytes)
	}
	if cfg.Consumer.StartOffset != -2 {
		t.Errorf("expected Consumer.StartOffset to be -2 (Earliest), got %d", cfg.Consumer.StartOffset)
	}
	if cfg.Consumer.Concurrency != 1 {
		t.Errorf("expected Consumer.Concurrency to be 1, got %d", cfg.Consumer.Concurrency)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name:    "valid config",
			config:  DefaultConfig(),
			wantErr: nil,
		},
		{
			name: "no brokers",
			config: &Config{
				Brokers: []string{},
				Consumer: ConsumerConfig{
					GroupID: "test-group",
				},
			},
			wantErr: ErrNoBrokers,
		},
		{
			name: "nil brokers",
			config: &Config{
				Brokers: nil,
				Consumer: ConsumerConfig{
					GroupID: "test-group",
				},
			},
			wantErr: ErrNoBrokers,
		},
		{
			name: "empty group id",
			config: &Config{
				Brokers: []string{"localhost:9092"},
				Consumer: ConsumerConfig{
					GroupID: "",
				},
			},
			wantErr: ErrEmptyGroupID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSASLConfig(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		Consumer: ConsumerConfig{
			GroupID: "test-group",
		},
		SASL: &SASLConfig{
			Mechanism: "PLAIN",
			Username:  "user",
			Password:  "pass",
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid SASL config should pass validation, got: %v", err)
	}

	if cfg.SASL.Mechanism != "PLAIN" {
		t.Errorf("expected SASL.Mechanism to be PLAIN, got %s", cfg.SASL.Mechanism)
	}
}

func TestTLSConfig(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		Consumer: ConsumerConfig{
			GroupID: "test-group",
		},
		TLS: &TLSConfig{
			Enable:             true,
			InsecureSkipVerify: true,
		},
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("valid TLS config should pass validation, got: %v", err)
	}

	if !cfg.TLS.Enable {
		t.Errorf("expected TLS.Enable to be true")
	}
	if !cfg.TLS.InsecureSkipVerify {
		t.Errorf("expected TLS.InsecureSkipVerify to be true")
	}
}
