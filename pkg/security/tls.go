package security

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// TLSConfig TLS 配置
type TLSConfig struct {
	// 证书文件路径
	CertFile string `mapstructure:"cert_file" json:"cert_file"`

	// 私钥文件路径
	KeyFile string `mapstructure:"key_file" json:"key_file"`

	// CA 证书文件路径（用于 mTLS 验证客户端证书）
	CAFile string `mapstructure:"ca_file" json:"ca_file"`

	// 是否启用 mTLS（双向认证）
	MutualTLS bool `mapstructure:"mutual_tls" json:"mutual_tls"`

	// 跳过证书验证（仅用于开发环境，生产环境禁止使用）
	InsecureSkipVerify bool `mapstructure:"insecure_skip_verify" json:"insecure_skip_verify"`

	// 最低 TLS 版本（默认 TLS 1.2）
	MinVersion uint16 `mapstructure:"min_version" json:"min_version"`

	// 最高 TLS 版本（默认 TLS 1.3）
	MaxVersion uint16 `mapstructure:"max_version" json:"max_version"`

	// 服务器名称（用于客户端验证服务器证书）
	ServerName string `mapstructure:"server_name" json:"server_name"`

	// 加密套件（空表示使用默认）
	CipherSuites []uint16 `mapstructure:"cipher_suites" json:"cipher_suites"`
}

// TLS 版本常量
const (
	TLSVersion10 uint16 = tls.VersionTLS10
	TLSVersion11 uint16 = tls.VersionTLS11
	TLSVersion12 uint16 = tls.VersionTLS12
	TLSVersion13 uint16 = tls.VersionTLS13
)

// DefaultTLSConfig 返回默认 TLS 配置（最小可用配置）
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		MutualTLS:          false,
		InsecureSkipVerify: false,
		MinVersion:         TLSVersion12,
		MaxVersion:         TLSVersion13,
	}
}

// Validate 验证配置
func (c *TLSConfig) Validate() error {
	if c.CertFile == "" {
		return ErrCertFileEmpty
	}

	if c.KeyFile == "" {
		return ErrKeyFileEmpty
	}

	if c.MutualTLS && c.CAFile == "" {
		return ErrCAFileEmpty
	}

	return nil
}

// NewServerTLSConfig 创建服务端 TLS 配置
func NewServerTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	newCfg, err := config.MergeConfig(DefaultTLSConfig(), cfg)
	if err != nil {
		return nil, err
	}

	if err := newCfg.Validate(); err != nil {
		return nil, err
	}

	// 加载服务端证书
	cert, err := tls.LoadX509KeyPair(newCfg.CertFile, newCfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCertLoad, err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   newCfg.MinVersion,
		MaxVersion:   newCfg.MaxVersion,
	}

	// 设置加密套件
	if len(newCfg.CipherSuites) > 0 {
		tlsConfig.CipherSuites = newCfg.CipherSuites
	}

	// mTLS：验证客户端证书
	if newCfg.MutualTLS {
		caPool, err := loadCAPool(newCfg.CAFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.ClientCAs = caPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	return tlsConfig, nil
}

// NewClientTLSConfig 创建客户端 TLS 配置
func NewClientTLSConfig(cfg *TLSConfig) (*tls.Config, error) {
	newCfg, err := config.MergeConfig(DefaultTLSConfig(), cfg)
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{
		MinVersion:         newCfg.MinVersion,
		MaxVersion:         newCfg.MaxVersion,
		InsecureSkipVerify: newCfg.InsecureSkipVerify,
		ServerName:         newCfg.ServerName,
	}

	// 设置加密套件
	if len(newCfg.CipherSuites) > 0 {
		tlsConfig.CipherSuites = newCfg.CipherSuites
	}

	// 加载 CA 证书（用于验证服务端证书）
	if newCfg.CAFile != "" {
		caPool, err := loadCAPool(newCfg.CAFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.RootCAs = caPool
	}

	// mTLS：加载客户端证书
	if newCfg.MutualTLS {
		if newCfg.CertFile == "" {
			return nil, ErrCertFileEmpty
		}
		if newCfg.KeyFile == "" {
			return nil, ErrKeyFileEmpty
		}

		cert, err := tls.LoadX509KeyPair(newCfg.CertFile, newCfg.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrCertLoad, err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	return tlsConfig, nil
}

// loadCAPool 加载 CA 证书池
func loadCAPool(caFile string) (*x509.CertPool, error) {
	caData, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCALoad, err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caData) {
		return nil, ErrCAAppend
	}

	return caPool, nil
}

// RecommendedCipherSuites 返回推荐的加密套件
func RecommendedCipherSuites() []uint16 {
	return []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
}
