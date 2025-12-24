// pkg/crypto/aes_test.go
package crypto

import (
	"testing"
)

func TestAESEncryptDecrypt(t *testing.T) {
	// 创建 32 字节的密钥
	key := []byte("12345678901234567890123456789012")
	aes, err := NewAES(key)
	if err != nil {
		t.Fatalf("Failed to create AES: %v", err)
	}

	plaintext := "Hello, World! This is a secret message."

	// 加密
	ciphertext, err := aes.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	// 解密
	decrypted, err := aes.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text doesn't match. Expected: %s, Got: %s", plaintext, decrypted)
	}
}

func TestAESEncryptDecryptBytes(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	aes, err := NewAES(key)
	if err != nil {
		t.Fatalf("Failed to create AES: %v", err)
	}

	plaintext := []byte("Binary data test")

	ciphertext, err := aes.EncryptBytes(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := aes.DecryptBytes(ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted bytes don't match")
	}
}

func TestGenerateAESKey(t *testing.T) {
	key, err := GenerateAESKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}

func TestAESInvalidKey(t *testing.T) {
	invalidKey := []byte("short")
	_, err := NewAES(invalidKey)
	if err == nil {
		t.Error("Expected error for invalid key length")
	}
}

func TestNewAESFromString(t *testing.T) {
	// 测试有效的 32 字符密钥
	key := "12345678901234567890123456789012"
	aes, err := NewAESFromString(key)
	if err != nil {
		t.Fatalf("Failed to create AES from string: %v", err)
	}

	plaintext := "Test message"
	encrypted, err := aes.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	decrypted, err := aes.DecryptString(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted text doesn't match")
	}

	// 测试无效长度的密钥
	_, err = NewAESFromString("short")
	if err == nil {
		t.Error("Expected error for invalid key length")
	}
}

func TestGenerateAESKeyString(t *testing.T) {
	keyString, err := GenerateAESKeyString()
	if err != nil {
		t.Fatalf("Failed to generate key string: %v", err)
	}

	if keyString == "" {
		t.Error("Generated key string should not be empty")
	}

	// 验证生成的密钥字符串长度合理（base64 编码的 32 字节）
	if len(keyString) < 40 {
		t.Errorf("Generated key string seems too short: %d", len(keyString))
	}
}

func TestAESDecryptInvalidData(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	aes, err := NewAES(key)
	if err != nil {
		t.Fatalf("Failed to create AES: %v", err)
	}

	// 测试无效的 base64 数据
	_, err = aes.DecryptString("invalid base64 !!!")
	if err == nil {
		t.Error("Expected error for invalid base64")
	}

	// 测试太短的密文
	_, err = aes.DecryptString("dGVzdA==") // "test" in base64, too short
	if err == nil {
		t.Error("Expected error for ciphertext too short")
	}
}

func TestAESDecryptBytesInvalid(t *testing.T) {
	key := []byte("12345678901234567890123456789012")
	aes, err := NewAES(key)
	if err != nil {
		t.Fatalf("Failed to create AES: %v", err)
	}

	// 测试太短的密文
	_, err = aes.DecryptBytes([]byte("short"))
	if err == nil {
		t.Error("Expected error for ciphertext too short")
	}

	// 测试无效的密文（正确长度但内容错误）
	invalidCiphertext := make([]byte, 50)
	_, err = aes.DecryptBytes(invalidCiphertext)
	if err == nil {
		t.Error("Expected error for invalid ciphertext")
	}
}
