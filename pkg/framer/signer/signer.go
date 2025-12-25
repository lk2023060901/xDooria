// pkg/framer/signer/signer.go
// 消息签名和验证
package signer

import (
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/crypto"
)

// Signer 签名器接口
type Signer interface {
	// Sign 对数据进行签名
	Sign(data []byte) ([]byte, error)

	// Verify 验证签名
	Verify(data []byte, signature []byte) error
}

// hmacSigner 基于 HMAC 的签名器
type hmacSigner struct {
	hmac *crypto.HMACHasher
}

// NewHMACSigner 创建基于 HMAC 的签名器
func NewHMACSigner(hmac *crypto.HMACHasher) Signer {
	return &hmacSigner{
		hmac: hmac,
	}
}

// Sign 对数据进行签名
func (s *hmacSigner) Sign(data []byte) ([]byte, error) {
	if s.hmac == nil {
		return nil, fmt.Errorf("hmac hasher not initialized")
	}

	signature := s.hmac.SignBytes(data)
	return signature, nil
}

// Verify 验证签名
func (s *hmacSigner) Verify(data []byte, signature []byte) error {
	if s.hmac == nil {
		return fmt.Errorf("hmac hasher not initialized")
	}

	if !s.hmac.VerifyBytes(data, signature) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}
