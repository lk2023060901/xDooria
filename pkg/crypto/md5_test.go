// pkg/crypto/md5_test.go
package crypto

import (
	"os"
	"testing"
)

func TestMD5Hash(t *testing.T) {
	hasher := NewMD5Hasher()
	data := "Hello, World!"

	hash := hasher.HashString(data)

	// MD5 of "Hello, World!" is known
	expected := "65a8e27d8879283831b664bd8b7f0ad4"
	if hash != expected {
		t.Errorf("Expected hash %s, got %s", expected, hash)
	}
}

func TestMD5Verify(t *testing.T) {
	hasher := NewMD5Hasher()
	data := "Test data"

	hash := hasher.HashString(data)

	if !hasher.VerifyString(data, hash) {
		t.Error("Hash verification failed")
	}

	if hasher.VerifyString("Different data", hash) {
		t.Error("Hash should not match different data")
	}
}

func TestMD5HashWithSalt(t *testing.T) {
	hasher := NewMD5Hasher()
	data := "password"
	salt := "randomsalt123"

	hash1 := hasher.HashStringWithSalt(data, salt)
	hash2 := hasher.HashStringWithSalt(data, salt)

	if hash1 != hash2 {
		t.Error("Same data and salt should produce same hash")
	}

	hash3 := hasher.HashStringWithSalt(data, "differentsalt")
	if hash1 == hash3 {
		t.Error("Different salt should produce different hash")
	}
}

func TestMD5HashFile(t *testing.T) {
	hasher := NewMD5Hasher()

	// 创建临时文件
	tmpfile, err := os.CreateTemp("", "md5test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("File content for MD5 testing")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// 计算文件哈希
	hash, err := hasher.HashFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to hash file: %v", err)
	}

	// 验证文件哈希
	valid, err := hasher.VerifyFile(tmpfile.Name(), hash)
	if err != nil {
		t.Fatalf("Failed to verify file: %v", err)
	}

	if !valid {
		t.Error("File hash verification failed")
	}
}

func TestMD5ConvenienceFunctions(t *testing.T) {
	data := "Test data"

	// 测试 MD5HashString
	hash := MD5HashString(data)
	if !MD5Verify([]byte(data), hash) {
		t.Error("Convenience function verification failed")
	}

	// 测试 MD5Hash
	hash2 := MD5Hash([]byte(data))
	if hash != hash2 {
		t.Error("MD5Hash and MD5HashString should produce same result")
	}
}

func TestMD5HashFileConvenience(t *testing.T) {
	// 创建临时文件
	tmpfile, err := os.CreateTemp("", "md5test2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("Test file content")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	// 测试 MD5HashFile 便捷函数
	hash, err := MD5HashFile(tmpfile.Name())
	if err != nil {
		t.Fatalf("MD5HashFile failed: %v", err)
	}

	if hash == "" {
		t.Error("Hash should not be empty")
	}
}

func TestMD5HashFileError(t *testing.T) {
	hasher := NewMD5Hasher()

	// 测试不存在的文件
	_, err := hasher.HashFile("/nonexistent/file/path.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}

	// 测试便捷函数的错误处理
	_, err = MD5HashFile("/nonexistent/file.txt")
	if err == nil {
		t.Error("MD5HashFile should return error for nonexistent file")
	}
}

func TestMD5VerifyFileError(t *testing.T) {
	hasher := NewMD5Hasher()

	// 测试不存在的文件
	_, err := hasher.VerifyFile("/nonexistent/file.txt", "somehash")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}
