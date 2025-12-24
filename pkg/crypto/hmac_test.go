// pkg/crypto/hmac_test.go
package crypto

import (
	"testing"
)

func TestHMACSign(t *testing.T) {
	key := []byte("secret-key")
	hasher := NewHMACHasher(key)
	data := "Important message"

	signature := hasher.SignString(data)

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	// 验证签名
	valid, err := hasher.VerifyString(data, signature)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	if !valid {
		t.Error("Signature verification failed")
	}
}

func TestHMACVerifyInvalid(t *testing.T) {
	key := []byte("secret-key")
	hasher := NewHMACHasher(key)
	data := "Important message"

	signature := hasher.SignString(data)

	// 验证被篡改的数据
	valid, err := hasher.VerifyString("Tampered message", signature)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if valid {
		t.Error("Tampered message should not verify")
	}
}

func TestHMACWithSHA512(t *testing.T) {
	key := []byte("secret-key")
	hasher := NewHMACHasher(key, WithHashAlgorithm(SHA512))
	data := "Test data"

	signature := hasher.SignString(data)

	valid, err := hasher.VerifyString(data, signature)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if !valid {
		t.Error("SHA-512 signature verification failed")
	}
}

func TestHMACBase64(t *testing.T) {
	key := []byte("secret-key")
	hasher := NewHMACHasher(key)
	data := "Test message"

	signature := hasher.SignStringBase64(data)

	valid, err := hasher.VerifyStringBase64(data, signature)
	if err != nil {
		t.Fatalf("Failed to verify base64 signature: %v", err)
	}

	if !valid {
		t.Error("Base64 signature verification failed")
	}
}

func TestHMACBytes(t *testing.T) {
	key := []byte("secret-key")
	hasher := NewHMACHasher(key)
	data := []byte("Binary data")

	signature := hasher.SignBytes(data)

	if !hasher.VerifyBytes(data, signature) {
		t.Error("Bytes signature verification failed")
	}

	tamperedData := []byte("Tampered data")
	if hasher.VerifyBytes(tamperedData, signature) {
		t.Error("Tampered data should not verify")
	}
}

func TestHMACConvenienceFunctions(t *testing.T) {
	key := "secret-key"
	data := "Test data"

	// 测试 HMACSignString
	signature := HMACSignString(key, data)

	valid, err := HMACVerifyString(key, data, signature)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if !valid {
		t.Error("Convenience function verification failed")
	}

	// 测试 HMACSign 字节数组版本
	keyBytes := []byte(key)
	dataBytes := []byte(data)
	signature2 := HMACSign(keyBytes, dataBytes)

	if signature2 == "" {
		t.Error("HMACSign should not return empty string")
	}

	// 测试 HMACVerify 字节数组版本
	valid2, err := HMACVerify(keyBytes, dataBytes, signature2)
	if err != nil {
		t.Fatalf("HMACVerify failed: %v", err)
	}

	if !valid2 {
		t.Error("HMACVerify should succeed")
	}

	// 测试错误数据
	valid3, err := HMACVerify(keyBytes, []byte("wrong data"), signature2)
	if err != nil {
		t.Fatalf("HMACVerify failed: %v", err)
	}

	if valid3 {
		t.Error("HMACVerify should fail for wrong data")
	}
}

func TestHMACVerifyInvalidSignature(t *testing.T) {
	key := []byte("secret-key")
	hasher := NewHMACHasher(key)
	data := "Test data"

	// 测试无效的十六进制签名
	_, err := hasher.VerifyString(data, "invalid-hex-signature!!!")
	if err == nil {
		t.Error("Expected error for invalid hex signature")
	}

	// 测试无效的 base64 签名
	_, err = hasher.VerifyStringBase64(data, "invalid-base64!!!")
	if err == nil {
		t.Error("Expected error for invalid base64 signature")
	}
}

func TestHMACDifferentKeys(t *testing.T) {
	key1 := []byte("key1")
	key2 := []byte("key2")
	data := "Test data"

	hasher1 := NewHMACHasher(key1)
	hasher2 := NewHMACHasher(key2)

	signature1 := hasher1.SignString(data)
	signature2 := hasher2.SignString(data)

	if signature1 == signature2 {
		t.Error("Different keys should produce different signatures")
	}

	// key1 的签名不应该被 key2 验证通过
	valid, _ := hasher2.VerifyString(data, signature1)
	if valid {
		t.Error("Signature from key1 should not verify with key2")
	}
}
