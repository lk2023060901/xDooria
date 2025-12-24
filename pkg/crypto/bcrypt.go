// pkg/crypto/bcrypt.go
package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher 提供 bcrypt 密码哈希功能
type BcryptHasher struct {
	cost int
}

// BcryptOption bcrypt 配置选项
type BcryptOption func(*BcryptHasher)

// WithCost 设置 bcrypt 工作因子 (4-31，默认 10)
// 值越大越安全，但计算越慢
func WithCost(cost int) BcryptOption {
	return func(h *BcryptHasher) {
		h.cost = cost
	}
}

// NewBcryptHasher 创建 bcrypt 哈希器
func NewBcryptHasher(opts ...BcryptOption) *BcryptHasher {
	h := &BcryptHasher{
		cost: bcrypt.DefaultCost, // 默认值为 10
	}
	for _, opt := range opts {
		opt(h)
	}
	return h
}

// Hash 对密码进行哈希
func (h *BcryptHasher) Hash(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedPassword), nil
}

// Verify 验证密码是否匹配
func (h *BcryptHasher) Verify(password, hashedPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return fmt.Errorf("password mismatch")
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}
	return nil
}

// IsMatch 检查密码是否匹配（返回布尔值）
func (h *BcryptHasher) IsMatch(password, hashedPassword string) bool {
	return h.Verify(password, hashedPassword) == nil
}

// 便捷函数

// HashPassword 使用默认配置哈希密码
func HashPassword(password string) (string, error) {
	return NewBcryptHasher().Hash(password)
}

// VerifyPassword 使用默认配置验证密码
func VerifyPassword(password, hashedPassword string) error {
	return NewBcryptHasher().Verify(password, hashedPassword)
}

// IsPasswordMatch 检查密码是否匹配
func IsPasswordMatch(password, hashedPassword string) bool {
	return NewBcryptHasher().IsMatch(password, hashedPassword)
}
