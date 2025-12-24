// pkg/crypto/aes.go
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

// AES 提供 AES-256-GCM 加密解密功能
type AES struct {
	key []byte
}

// NewAES 创建 AES 加密器
// key 必须是 32 字节 (AES-256)
func NewAES(key []byte) (*AES, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 bytes for AES-256, got %d bytes", len(key))
	}
	return &AES{key: key}, nil
}

// NewAESFromString 从字符串创建 AES 加密器
func NewAESFromString(key string) (*AES, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key must be 32 characters for AES-256, got %d characters", len(key))
	}
	return &AES{key: []byte(key)}, nil
}

// Encrypt 加密数据，返回 base64 编码的密文
func (a *AES) Encrypt(plaintext []byte) (string, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 使用 GCM 模式
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据（nonce 会被添加到密文前面）
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// 返回 base64 编码的密文
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// EncryptString 加密字符串
func (a *AES) EncryptString(plaintext string) (string, error) {
	return a.Encrypt([]byte(plaintext))
}

// Decrypt 解密 base64 编码的密文
func (a *AES) Decrypt(ciphertext string) ([]byte, error) {
	// 解码 base64
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	// 提取 nonce 和密文
	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]

	// 解密
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// DecryptString 解密为字符串
func (a *AES) DecryptString(ciphertext string) (string, error) {
	plaintext, err := a.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// EncryptBytes 加密数据，返回原始字节（不经过 base64 编码）
func (a *AES) EncryptBytes(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// DecryptBytes 解密原始字节
func (a *AES) DecryptBytes(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateKey 生成随机 AES-256 密钥
func GenerateAESKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

// GenerateKeyString 生成随机 AES-256 密钥字符串
func GenerateAESKeyString() (string, error) {
	key, err := GenerateAESKey()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key), nil
}
