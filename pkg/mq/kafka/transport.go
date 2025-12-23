package kafka

import (
	"crypto/tls"
	"crypto/x509"
	"os"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl"
	"github.com/segmentio/kafka-go/sasl/plain"
	"github.com/segmentio/kafka-go/sasl/scram"
)

// newDialer 创建 kafka.Dialer（用于连接时的 TLS 和 SASL）
func newDialer(cfg *Config) (*kafka.Dialer, error) {
	dialer := &kafka.Dialer{}

	// 配置 TLS
	if cfg.TLS != nil && cfg.TLS.Enable {
		tlsConfig, err := newTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		dialer.TLS = tlsConfig
	}

	// 配置 SASL
	if cfg.SASL != nil && cfg.SASL.Username != "" {
		mechanism, err := newSASLMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		dialer.SASLMechanism = mechanism
	}

	return dialer, nil
}

// newTransport 创建 kafka.Transport（用于 Writer）
func newTransport(cfg *Config) (*kafka.Transport, error) {
	transport := &kafka.Transport{}

	// 配置 TLS
	if cfg.TLS != nil && cfg.TLS.Enable {
		tlsConfig, err := newTLSConfig(cfg.TLS)
		if err != nil {
			return nil, err
		}
		transport.TLS = tlsConfig
	}

	// 配置 SASL
	if cfg.SASL != nil && cfg.SASL.Username != "" {
		mechanism, err := newSASLMechanism(cfg.SASL)
		if err != nil {
			return nil, err
		}
		transport.SASL = mechanism
	}

	return transport, nil
}

// newTLSConfig 创建 TLS 配置
func newTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	// 加载 CA 证书
	if cfg.CAFile != "" {
		caCert, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)
		tlsConfig.RootCAs = caCertPool
	}

	// 加载客户端证书
	if cfg.CertFile != "" && cfg.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// newSASLMechanism 创建 SASL 认证机制
func newSASLMechanism(cfg *SASLConfig) (sasl.Mechanism, error) {
	switch cfg.Mechanism {
	case "PLAIN", "plain", "":
		return plain.Mechanism{
			Username: cfg.Username,
			Password: cfg.Password,
		}, nil
	case "SCRAM-SHA-256", "scram-sha-256":
		return scram.Mechanism(scram.SHA256, cfg.Username, cfg.Password)
	case "SCRAM-SHA-512", "scram-sha-512":
		return scram.Mechanism(scram.SHA512, cfg.Username, cfg.Password)
	default:
		return plain.Mechanism{
			Username: cfg.Username,
			Password: cfg.Password,
		}, nil
	}
}
