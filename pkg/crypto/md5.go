// pkg/crypto/md5.go
package crypto

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// MD5Hasher 提供 MD5 哈希功能
// 注意: MD5 已不安全，仅用于数据完整性校验，不应用于密码存储或安全场景
type MD5Hasher struct{}

// NewMD5Hasher 创建 MD5 哈希器
func NewMD5Hasher() *MD5Hasher {
	return &MD5Hasher{}
}

// Hash 计算数据的 MD5 哈希值
func (h *MD5Hasher) Hash(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// HashString 计算字符串的 MD5 哈希值
func (h *MD5Hasher) HashString(s string) string {
	return h.Hash([]byte(s))
}

// HashFile 计算文件的 MD5 哈希值
func (h *MD5Hasher) HashFile(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// HashWithSalt 计算带盐值的 MD5 哈希
// 注意: 即使加盐，MD5 仍不适合密码存储，请使用 bcrypt 或 argon2
func (h *MD5Hasher) HashWithSalt(data []byte, salt string) string {
	combined := append(data, []byte(salt)...)
	return h.Hash(combined)
}

// HashStringWithSalt 计算带盐值的字符串 MD5 哈希
func (h *MD5Hasher) HashStringWithSalt(s, salt string) string {
	return h.HashWithSalt([]byte(s), salt)
}

// Verify 验证数据的 MD5 哈希值是否匹配
func (h *MD5Hasher) Verify(data []byte, expectedHash string) bool {
	actualHash := h.Hash(data)
	return actualHash == expectedHash
}

// VerifyString 验证字符串的 MD5 哈希值是否匹配
func (h *MD5Hasher) VerifyString(s, expectedHash string) bool {
	return h.Verify([]byte(s), expectedHash)
}

// VerifyFile 验证文件的 MD5 哈希值是否匹配
func (h *MD5Hasher) VerifyFile(filepath, expectedHash string) (bool, error) {
	actualHash, err := h.HashFile(filepath)
	if err != nil {
		return false, err
	}
	return actualHash == expectedHash, nil
}

// 便捷函数

// MD5Hash 计算数据的 MD5 哈希值
func MD5Hash(data []byte) string {
	return NewMD5Hasher().Hash(data)
}

// MD5HashString 计算字符串的 MD5 哈希值
func MD5HashString(s string) string {
	return NewMD5Hasher().HashString(s)
}

// MD5HashFile 计算文件的 MD5 哈希值
func MD5HashFile(filepath string) (string, error) {
	return NewMD5Hasher().HashFile(filepath)
}

// MD5Verify 验证数据的 MD5 哈希值
func MD5Verify(data []byte, expectedHash string) bool {
	return NewMD5Hasher().Verify(data, expectedHash)
}
