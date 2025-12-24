package main

import (
	"fmt"
	"log"

	"github.com/lk2023060901/xdooria/pkg/crypto"
)

func main() {
	fmt.Println("=== Crypto Package Examples ===\n")

	// 1. AES-256 加密示例
	fmt.Println("1. AES-256-GCM 加密示例:")
	aesExample()
	fmt.Println()

	// 2. bcrypt 密码哈希示例
	fmt.Println("2. bcrypt 密码哈希示例:")
	bcryptExample()
	fmt.Println()

	// 3. MD5 哈希示例
	fmt.Println("3. MD5 哈希示例:")
	md5Example()
	fmt.Println()

	// 4. HMAC 签名示例
	fmt.Println("4. HMAC 签名示例:")
	hmacExample()
	fmt.Println()
}

func aesExample() {
	// 创建 32 字节的密钥（AES-256）
	key := []byte("12345678901234567890123456789012")
	aes, err := crypto.NewAES(key)
	if err != nil {
		log.Fatal(err)
	}

	// 加密玩家存档数据
	playerData := `{"player_id": 12345, "level": 50, "gold": 10000}`
	fmt.Printf("原始数据: %s\n", playerData)

	encrypted, err := aes.EncryptString(playerData)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("加密后: %s...\n", encrypted[:50])

	// 解密
	decrypted, err := aes.DecryptString(encrypted)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("解密后: %s\n", decrypted)

	// 生成随机密钥
	randomKey, _ := crypto.GenerateAESKey()
	fmt.Printf("随机生成的密钥长度: %d 字节\n", len(randomKey))
}

func bcryptExample() {
	// 用户注册 - 哈希密码
	password := "mySecurePassword123"
	fmt.Printf("原始密码: %s\n", password)

	hasher := crypto.NewBcryptHasher()
	hashed, err := hasher.Hash(password)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("哈希后: %s\n", hashed)

	// 用户登录 - 验证密码
	err = hasher.Verify(password, hashed)
	if err == nil {
		fmt.Println("✓ 密码验证成功")
	} else {
		fmt.Println("✗ 密码验证失败")
	}

	// 错误密码验证
	err = hasher.Verify("wrongPassword", hashed)
	if err != nil {
		fmt.Println("✓ 错误密码被正确拒绝")
	}

	// 使用便捷函数
	isMatch := crypto.IsPasswordMatch(password, hashed)
	fmt.Printf("密码匹配: %v\n", isMatch)
}

func md5Example() {
	// 文件完整性校验
	data := "Game resource file content"
	fmt.Printf("数据: %s\n", data)

	hasher := crypto.NewMD5Hasher()
	hash := hasher.HashString(data)
	fmt.Printf("MD5 哈希: %s\n", hash)

	// 验证数据完整性
	isValid := hasher.VerifyString(data, hash)
	fmt.Printf("数据完整性验证: %v\n", isValid)

	// 带盐值的哈希（增加安全性）
	salt := "randomSalt123"
	hashedWithSalt := hasher.HashStringWithSalt(data, salt)
	fmt.Printf("带盐值的哈希: %s\n", hashedWithSalt)

	// 使用便捷函数
	quickHash := crypto.MD5HashString("Quick test")
	fmt.Printf("快速哈希: %s\n", quickHash)
}

func hmacExample() {
	// API 请求签名
	apiSecret := []byte("my-api-secret-key")
	requestBody := `{"user_id": 12345, "action": "buy_item", "item_id": 999}`

	fmt.Printf("API 请求: %s\n", requestBody)

	// 生成签名
	hasher := crypto.NewHMACHasher(apiSecret)
	signature := hasher.SignString(requestBody)
	fmt.Printf("HMAC 签名: %s\n", signature)

	// 服务端验证签名
	valid, err := hasher.VerifyString(requestBody, signature)
	if err != nil {
		log.Fatal(err)
	}
	if valid {
		fmt.Println("✓ 签名验证成功 - 请求合法")
	} else {
		fmt.Println("✗ 签名验证失败 - 请求被拒绝")
	}

	// 检测篡改
	tamperedBody := `{"user_id": 12345, "action": "buy_item", "item_id": 1}`
	valid, _ = hasher.VerifyString(tamperedBody, signature)
	if !valid {
		fmt.Println("✓ 检测到数据被篡改，签名验证失败")
	}

	// Base64 格式签名
	base64Signature := hasher.SignStringBase64(requestBody)
	fmt.Printf("Base64 签名: %s\n", base64Signature)

	// 使用 SHA-512 算法
	hasher512 := crypto.NewHMACHasher(apiSecret, crypto.WithHashAlgorithm(crypto.SHA512))
	signature512 := hasher512.SignString(requestBody)
	fmt.Printf("SHA-512 签名: %s\n", signature512)

	// 便捷函数
	quickSignature := crypto.HMACSignString("my-key", "quick test")
	fmt.Printf("快速签名: %s\n", quickSignature)
}
