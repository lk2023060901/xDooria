# Crypto Package

提供常用的加密算法实现，用于游戏服务端的数据加密、密码存储、签名验证等场景。

## 功能模块

### 1. AES-256-GCM 加密 (`aes.go`)

AES-256-GCM 模式的对称加密，用于加密敏感数据。

**特点**:
- 使用 AES-256（32 字节密钥）
- GCM 模式提供认证加密（防篡改）
- 自动生成随机 nonce
- 支持 base64 编码输出

**使用示例**:

```go
// 创建加密器
key := []byte("12345678901234567890123456789012") // 32 字节
aes, err := crypto.NewAES(key)

// 加密字符串
ciphertext, err := aes.EncryptString("Hello, World!")

// 解密字符串
plaintext, err := aes.DecryptString(ciphertext)

// 生成随机密钥
key, err := crypto.GenerateAESKey()
```

**应用场景**:
- 加密玩家存档
- 加密配置文件
- 加密敏感通信数据

### 2. bcrypt 密码哈希 (`bcrypt.go`)

bcrypt 密码哈希算法，用于安全存储用户密码。

**特点**:
- 自动加盐
- 可配置工作因子（防暴力破解）
- 行业标准密码存储方案

**使用示例**:

```go
// 创建哈希器
hasher := crypto.NewBcryptHasher()

// 哈希密码
hashed, err := hasher.Hash("myPassword123")

// 验证密码
err := hasher.Verify("myPassword123", hashed)
if err == nil {
    // 密码正确
}

// 检查密码是否匹配
isMatch := hasher.IsMatch("myPassword123", hashed)

// 使用自定义工作因子
hasher := crypto.NewBcryptHasher(crypto.WithCost(12))

// 便捷函数
hashed, _ := crypto.HashPassword("myPassword")
isMatch := crypto.IsPasswordMatch("myPassword", hashed)
```

**应用场景**:
- 用户密码存储
- 管理员密码存储
- API 密钥存储

### 3. MD5 哈希 (`md5.go`)

MD5 哈希算法，用于数据完整性校验。

**警告**: MD5 已不安全，**不应用于密码存储或安全场景**，仅用于数据完整性校验。

**使用示例**:

```go
// 创建哈希器
hasher := crypto.NewMD5Hasher()

// 计算字符串哈希
hash := hasher.HashString("Hello, World!")

// 计算文件哈希
hash, err := hasher.HashFile("/path/to/file")

// 验证哈希
isValid := hasher.VerifyString("Hello, World!", hash)

// 带盐值哈希
hash := hasher.HashStringWithSalt("data", "salt")

// 便捷函数
hash := crypto.MD5HashString("data")
isValid := crypto.MD5Verify([]byte("data"), hash)
```

**应用场景**:
- 文件完整性校验
- 资源包校验和
- 数据去重（非安全场景）

### 4. HMAC 签名 (`hmac.go`)

HMAC (Hash-based Message Authentication Code) 用于数据签名和验证。

**特点**:
- 支持 SHA-256 和 SHA-512
- 防篡改、防重放攻击
- 支持十六进制和 base64 编码

**使用示例**:

```go
// 创建签名器（默认 SHA-256）
key := []byte("secret-key")
hasher := crypto.NewHMACHasher(key)

// 生成签名
signature := hasher.SignString("Important message")

// 验证签名
valid, err := hasher.VerifyString("Important message", signature)

// 使用 SHA-512
hasher := crypto.NewHMACHasher(key, crypto.WithHashAlgorithm(crypto.SHA512))

// Base64 编码签名
signature := hasher.SignStringBase64("data")
valid, err := hasher.VerifyStringBase64("data", signature)

// 原始字节签名
signature := hasher.SignBytes([]byte("data"))
valid := hasher.VerifyBytes([]byte("data"), signature)

// 便捷函数
signature := crypto.HMACSignString("secret-key", "data")
valid, _ := crypto.HMACVerifyString("secret-key", "data", signature)
```

**应用场景**:
- API 请求签名
- 游戏数据包签名（防篡改）
- Webhook 签名验证
- Token 签名

## 安全建议

### 密码存储
- ✅ 使用 **bcrypt** 或 argon2 存储密码
- ❌ **不要使用** MD5 或 SHA-256 存储密码

### 数据加密
- ✅ 使用 **AES-256-GCM** 加密敏感数据
- ✅ 密钥应从环境变量或密钥管理系统获取
- ❌ **不要硬编码**密钥在代码中

### 数据签名
- ✅ 使用 **HMAC-SHA256** 或更强的算法
- ✅ 密钥应保密且足够随机
- ✅ 使用常量时间比较防止时序攻击（已内置）

### 哈希校验
- ✅ 文件校验可使用 MD5（仅完整性，非安全）
- ✅ 安全场景使用 SHA-256 或更强算法

## 测试

```bash
# 运行所有测试
go test ./pkg/crypto/...

# 运行特定测试
go test -v ./pkg/crypto -run TestAES

# 查看覆盖率
go test -cover ./pkg/crypto/...
```

## 性能考虑

### bcrypt
- 工作因子越大越安全，但越慢
- 默认值 10 适合大多数场景
- 生产环境可考虑 12-14

### AES-GCM
- 高性能对称加密
- 硬件加速支持（AES-NI）

### HMAC
- 非常快速
- SHA-256 通常足够且比 SHA-512 快

## 依赖

- Go 标准库 `crypto/*`
- `golang.org/x/crypto/bcrypt`

## 示例场景

### 场景 1: 用户注册和登录

```go
// 注册: 哈希密码
hasher := crypto.NewBcryptHasher()
hashedPassword, _ := hasher.Hash(password)
// 存储到数据库

// 登录: 验证密码
err := hasher.Verify(inputPassword, storedHashedPassword)
if err == nil {
    // 登录成功
}
```

### 场景 2: 加密玩家存档

```go
// 加密存档
aes, _ := crypto.NewAES([]byte(encryptionKey))
encryptedData, _ := aes.EncryptString(jsonData)
// 存储加密数据

// 解密存档
decryptedData, _ := aes.DecryptString(encryptedData)
```

### 场景 3: API 请求签名

```go
// 生成签名
hmac := crypto.NewHMACHasher([]byte(apiSecret))
signature := hmac.SignString(requestBody)
// 发送 signature 到服务端

// 验证签名
valid, _ := hmac.VerifyString(requestBody, receivedSignature)
if valid {
    // 请求合法
}
```

### 场景 4: 资源文件校验

```go
// 计算文件 MD5
hasher := crypto.NewMD5Hasher()
hash, _ := hasher.HashFile("resources/model.fbx")

// 客户端下载后验证
valid, _ := hasher.VerifyFile("downloaded.fbx", expectedHash)
```

## 相关文档

- [Go crypto 标准库](https://pkg.go.dev/crypto)
- [golang.org/x/crypto](https://pkg.go.dev/golang.org/x/crypto)
- [OWASP 密码存储指南](https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html)
