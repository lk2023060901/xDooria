// pkg/crypto/hmac.go
package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"hash"
)

// HashAlgorithm HMAC 哈希算法类型
type HashAlgorithm int

const (
	// SHA256 使用 SHA-256 算法
	SHA256 HashAlgorithm = iota
	// SHA512 使用 SHA-512 算法
	SHA512
)

// HMACHasher 提供 HMAC 签名功能
type HMACHasher struct {
	key       []byte
	algorithm HashAlgorithm
}

// HMACOption HMAC 配置选项
type HMACOption func(*HMACHasher)

// WithHashAlgorithm 设置哈希算法
func WithHashAlgorithm(algo HashAlgorithm) HMACOption {
	return func(h *HMACHasher) {
		h.algorithm = algo
	}
}

// NewHMACHasher 创建 HMAC 哈希器
func NewHMACHasher(key []byte, opts ...HMACOption) *HMACHasher {
	h := &HMACHasher{
		key:       key,
		algorithm: SHA256, // 默认使用 SHA-256
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// NewHMACHasherFromString 从字符串密钥创建 HMAC 哈希器
func NewHMACHasherFromString(key string, opts ...HMACOption) *HMACHasher {
	return NewHMACHasher([]byte(key), opts...)
}

// getHashFunc 根据算法类型获取哈希函数
func (h *HMACHasher) getHashFunc() func() hash.Hash {
	switch h.algorithm {
	case SHA512:
		return sha512.New
	default:
		return sha256.New
	}
}

// Sign 生成 HMAC 签名（返回十六进制字符串）
func (h *HMACHasher) Sign(data []byte) string {
	mac := hmac.New(h.getHashFunc(), h.key)
	mac.Write(data)
	signature := mac.Sum(nil)
	return hex.EncodeToString(signature)
}

// SignString 对字符串生成 HMAC 签名
func (h *HMACHasher) SignString(s string) string {
	return h.Sign([]byte(s))
}

// SignBase64 生成 HMAC 签名（返回 base64 字符串）
func (h *HMACHasher) SignBase64(data []byte) string {
	mac := hmac.New(h.getHashFunc(), h.key)
	mac.Write(data)
	signature := mac.Sum(nil)
	return base64.StdEncoding.EncodeToString(signature)
}

// SignStringBase64 对字符串生成 HMAC 签名（base64）
func (h *HMACHasher) SignStringBase64(s string) string {
	return h.SignBase64([]byte(s))
}

// SignBytes 生成 HMAC 签名（返回原始字节）
func (h *HMACHasher) SignBytes(data []byte) []byte {
	mac := hmac.New(h.getHashFunc(), h.key)
	mac.Write(data)
	return mac.Sum(nil)
}

// Verify 验证 HMAC 签名（十六进制格式）
func (h *HMACHasher) Verify(data []byte, signature string) (bool, error) {
	expectedSignature := h.Sign(data)

	// 使用常量时间比较防止时序攻击
	signatureBytes, err := hex.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature format: %w", err)
	}

	expectedBytes, _ := hex.DecodeString(expectedSignature)

	return hmac.Equal(signatureBytes, expectedBytes), nil
}

// VerifyString 验证字符串的 HMAC 签名
func (h *HMACHasher) VerifyString(s, signature string) (bool, error) {
	return h.Verify([]byte(s), signature)
}

// VerifyBase64 验证 HMAC 签名（base64 格式）
func (h *HMACHasher) VerifyBase64(data []byte, signature string) (bool, error) {
	expectedSignature := h.SignBase64(data)

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false, fmt.Errorf("invalid signature format: %w", err)
	}

	expectedBytes, _ := base64.StdEncoding.DecodeString(expectedSignature)

	return hmac.Equal(signatureBytes, expectedBytes), nil
}

// VerifyStringBase64 验证字符串的 HMAC 签名（base64 格式）
func (h *HMACHasher) VerifyStringBase64(s, signature string) (bool, error) {
	return h.VerifyBase64([]byte(s), signature)
}

// VerifyBytes 验证原始字节的 HMAC 签名
func (h *HMACHasher) VerifyBytes(data, signature []byte) bool {
	expectedSignature := h.SignBytes(data)
	return hmac.Equal(signature, expectedSignature)
}

// 便捷函数

// HMACSign 使用 SHA-256 生成 HMAC 签名
func HMACSign(key, data []byte) string {
	return NewHMACHasher(key).Sign(data)
}

// HMACSignString 对字符串使用 SHA-256 生成 HMAC 签名
func HMACSignString(key, data string) string {
	return NewHMACHasherFromString(key).SignString(data)
}

// HMACVerify 验证 HMAC 签名
func HMACVerify(key, data []byte, signature string) (bool, error) {
	return NewHMACHasher(key).Verify(data, signature)
}

// HMACVerifyString 验证字符串的 HMAC 签名
func HMACVerifyString(key, data, signature string) (bool, error) {
	return NewHMACHasherFromString(key).VerifyString(data, signature)
}
