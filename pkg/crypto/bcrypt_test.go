// pkg/crypto/bcrypt_test.go
package crypto

import (
	"testing"
)

func TestBcryptHashVerify(t *testing.T) {
	hasher := NewBcryptHasher()
	password := "mySecretPassword123"

	// 哈希密码
	hashed, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// 验证正确的密码
	err = hasher.Verify(password, hashed)
	if err != nil {
		t.Errorf("Failed to verify correct password: %v", err)
	}

	// 验证错误的密码
	err = hasher.Verify("wrongPassword", hashed)
	if err == nil {
		t.Error("Expected error for wrong password")
	}
}

func TestBcryptIsMatch(t *testing.T) {
	hasher := NewBcryptHasher()
	password := "testPassword"

	hashed, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !hasher.IsMatch(password, hashed) {
		t.Error("Password should match")
	}

	if hasher.IsMatch("wrongPassword", hashed) {
		t.Error("Wrong password should not match")
	}
}

func TestBcryptWithCost(t *testing.T) {
	hasher := NewBcryptHasher(WithCost(12))
	password := "testPassword"

	hashed, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("Failed to hash with custom cost: %v", err)
	}

	if !hasher.IsMatch(password, hashed) {
		t.Error("Password should match with custom cost")
	}
}

func TestBcryptConvenienceFunctions(t *testing.T) {
	password := "myPassword"

	hashed, err := HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !IsPasswordMatch(password, hashed) {
		t.Error("Password should match using convenience function")
	}

	// 测试 VerifyPassword 函数
	err = VerifyPassword(password, hashed)
	if err != nil {
		t.Errorf("VerifyPassword should succeed: %v", err)
	}

	err = VerifyPassword("wrongPassword", hashed)
	if err == nil {
		t.Error("VerifyPassword should fail for wrong password")
	}
}

func TestBcryptHashError(t *testing.T) {
	// 测试无效的 cost 值会导致错误
	hasher := NewBcryptHasher(WithCost(32)) // bcrypt max cost is 31
	_, err := hasher.Hash("password")
	if err == nil {
		t.Error("Expected error for invalid cost")
	}
}

func TestBcryptVerifyInvalidHash(t *testing.T) {
	hasher := NewBcryptHasher()

	// 测试无效的哈希格式
	err := hasher.Verify("password", "invalid-hash")
	if err == nil {
		t.Error("Expected error for invalid hash format")
	}
}
