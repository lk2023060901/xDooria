package kafka

import (
	"testing"
)

func TestNewDialer_NoTLSNoSASL(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
	}

	dialer, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("newDialer error: %v", err)
	}

	if dialer == nil {
		t.Fatal("expected non-nil dialer")
	}

	if dialer.TLS != nil {
		t.Error("expected TLS to be nil")
	}

	if dialer.SASLMechanism != nil {
		t.Error("expected SASLMechanism to be nil")
	}
}

func TestNewDialer_WithTLSInsecure(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		TLS: &TLSConfig{
			Enable:             true,
			InsecureSkipVerify: true,
		},
	}

	dialer, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("newDialer error: %v", err)
	}

	if dialer.TLS == nil {
		t.Fatal("expected TLS to be configured")
	}

	if !dialer.TLS.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestNewDialer_WithSASLPlain(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		SASL: &SASLConfig{
			Mechanism: "PLAIN",
			Username:  "user",
			Password:  "pass",
		},
	}

	dialer, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("newDialer error: %v", err)
	}

	if dialer.SASLMechanism == nil {
		t.Fatal("expected SASL mechanism to be configured")
	}
}

func TestNewDialer_WithSASLScramSHA256(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		SASL: &SASLConfig{
			Mechanism: "SCRAM-SHA-256",
			Username:  "user",
			Password:  "pass",
		},
	}

	dialer, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("newDialer error: %v", err)
	}

	if dialer.SASLMechanism == nil {
		t.Fatal("expected SASL mechanism to be configured")
	}
}

func TestNewDialer_WithSASLScramSHA512(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		SASL: &SASLConfig{
			Mechanism: "SCRAM-SHA-512",
			Username:  "user",
			Password:  "pass",
		},
	}

	dialer, err := newDialer(cfg)
	if err != nil {
		t.Fatalf("newDialer error: %v", err)
	}

	if dialer.SASLMechanism == nil {
		t.Fatal("expected SASL mechanism to be configured")
	}
}

func TestNewTransport_NoTLSNoSASL(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
	}

	transport, err := newTransport(cfg)
	if err != nil {
		t.Fatalf("newTransport error: %v", err)
	}

	if transport == nil {
		t.Fatal("expected non-nil transport")
	}

	if transport.TLS != nil {
		t.Error("expected TLS to be nil")
	}

	if transport.SASL != nil {
		t.Error("expected SASL to be nil")
	}
}

func TestNewTransport_WithTLSAndSASL(t *testing.T) {
	cfg := &Config{
		Brokers: []string{"localhost:9092"},
		TLS: &TLSConfig{
			Enable:             true,
			InsecureSkipVerify: true,
		},
		SASL: &SASLConfig{
			Mechanism: "PLAIN",
			Username:  "user",
			Password:  "pass",
		},
	}

	transport, err := newTransport(cfg)
	if err != nil {
		t.Fatalf("newTransport error: %v", err)
	}

	if transport.TLS == nil {
		t.Error("expected TLS to be configured")
	}

	if transport.SASL == nil {
		t.Error("expected SASL to be configured")
	}
}

func TestNewTLSConfig_InsecureSkipVerify(t *testing.T) {
	cfg := &TLSConfig{
		InsecureSkipVerify: true,
	}

	tlsConfig, err := newTLSConfig(cfg)
	if err != nil {
		t.Fatalf("newTLSConfig error: %v", err)
	}

	if !tlsConfig.InsecureSkipVerify {
		t.Error("expected InsecureSkipVerify to be true")
	}
}

func TestNewTLSConfig_InvalidCAFile(t *testing.T) {
	cfg := &TLSConfig{
		CAFile: "/nonexistent/ca.crt",
	}

	_, err := newTLSConfig(cfg)
	if err == nil {
		t.Error("expected error for nonexistent CA file")
	}
}

func TestNewTLSConfig_InvalidCertFile(t *testing.T) {
	cfg := &TLSConfig{
		CertFile: "/nonexistent/cert.crt",
		KeyFile:  "/nonexistent/key.pem",
	}

	_, err := newTLSConfig(cfg)
	if err == nil {
		t.Error("expected error for nonexistent cert file")
	}
}

func TestNewSASLMechanism_Plain(t *testing.T) {
	tests := []struct {
		mechanism string
	}{
		{"PLAIN"},
		{"plain"},
		{""},
	}

	for _, tt := range tests {
		t.Run(tt.mechanism, func(t *testing.T) {
			cfg := &SASLConfig{
				Mechanism: tt.mechanism,
				Username:  "user",
				Password:  "pass",
			}

			mechanism, err := newSASLMechanism(cfg)
			if err != nil {
				t.Fatalf("newSASLMechanism error: %v", err)
			}

			if mechanism == nil {
				t.Fatal("expected non-nil mechanism")
			}
		})
	}
}

func TestNewSASLMechanism_ScramSHA256(t *testing.T) {
	tests := []struct {
		mechanism string
	}{
		{"SCRAM-SHA-256"},
		{"scram-sha-256"},
	}

	for _, tt := range tests {
		t.Run(tt.mechanism, func(t *testing.T) {
			cfg := &SASLConfig{
				Mechanism: tt.mechanism,
				Username:  "user",
				Password:  "pass",
			}

			mechanism, err := newSASLMechanism(cfg)
			if err != nil {
				t.Fatalf("newSASLMechanism error: %v", err)
			}

			if mechanism == nil {
				t.Fatal("expected non-nil mechanism")
			}
		})
	}
}

func TestNewSASLMechanism_ScramSHA512(t *testing.T) {
	tests := []struct {
		mechanism string
	}{
		{"SCRAM-SHA-512"},
		{"scram-sha-512"},
	}

	for _, tt := range tests {
		t.Run(tt.mechanism, func(t *testing.T) {
			cfg := &SASLConfig{
				Mechanism: tt.mechanism,
				Username:  "user",
				Password:  "pass",
			}

			mechanism, err := newSASLMechanism(cfg)
			if err != nil {
				t.Fatalf("newSASLMechanism error: %v", err)
			}

			if mechanism == nil {
				t.Fatal("expected non-nil mechanism")
			}
		})
	}
}

func TestNewSASLMechanism_DefaultToPlain(t *testing.T) {
	cfg := &SASLConfig{
		Mechanism: "unknown-mechanism",
		Username:  "user",
		Password:  "pass",
	}

	mechanism, err := newSASLMechanism(cfg)
	if err != nil {
		t.Fatalf("newSASLMechanism error: %v", err)
	}

	if mechanism == nil {
		t.Fatal("expected non-nil mechanism (defaulting to PLAIN)")
	}
}
