package security

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/lk2023060901/xdooria/pkg/config"
)

// jwt 库错误别名
var jwtErrSignatureInvalid = jwt.ErrSignatureInvalid

// JWTConfig JWT 配置
type JWTConfig struct {
	// 签名密钥（用于 HS256 等对称算法）
	SecretKey string `mapstructure:"secret_key" json:"secret_key"`

	// 公钥文件路径（用于 RS256 等非对称算法验证）
	PublicKeyFile string `mapstructure:"public_key_file" json:"public_key_file"`

	// 私钥文件路径（用于 RS256 等非对称算法签名）
	PrivateKeyFile string `mapstructure:"private_key_file" json:"private_key_file"`

	// 签名算法（默认 HS256）
	// 支持：HS256, HS384, HS512, RS256, RS384, RS512, ES256, ES384, ES512
	Algorithm string `mapstructure:"algorithm" json:"algorithm"`

	// Token 过期时间（默认 24 小时）
	ExpiresIn time.Duration `mapstructure:"expires_in" json:"expires_in"`

	// 刷新 Token 过期时间（默认 7 天）
	RefreshExpiresIn time.Duration `mapstructure:"refresh_expires_in" json:"refresh_expires_in"`

	// 签发者
	Issuer string `mapstructure:"issuer" json:"issuer"`

	// Token 前缀（默认 "Bearer "）
	TokenPrefix string `mapstructure:"token_prefix" json:"token_prefix"`

	// Header 名称（默认 "authorization"）
	HeaderName string `mapstructure:"header_name" json:"header_name"`

	// 跳过验证的路径
	SkipPaths []string `mapstructure:"skip_paths" json:"skip_paths"`
}

// Claims 通用 JWT Claims
type Claims struct {
	jwt.RegisteredClaims

	// Payload 自定义载荷，完全由调用方决定内容
	Payload map[string]any `json:"payload,omitempty"`
}


// DefaultJWTConfig 返回默认 JWT 配置（最小可用配置）
func DefaultJWTConfig() *JWTConfig {
	return &JWTConfig{
		Algorithm:        "HS256",
		ExpiresIn:        24 * time.Hour,
		RefreshExpiresIn: 7 * 24 * time.Hour,
		TokenPrefix:      "Bearer ",
		HeaderName:       "authorization",
	}
}

// JWTManager JWT 管理器
type JWTManager struct {
	config     *JWTConfig
	publicKey  any
	privateKey any
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(cfg *JWTConfig) (*JWTManager, error) {
	newCfg, err := config.MergeConfig(DefaultJWTConfig(), cfg)
	if err != nil {
		return nil, err
	}

	m := &JWTManager{
		config: newCfg,
	}

	// 加载密钥
	if err := m.loadKeys(); err != nil {
		return nil, err
	}

	return m, nil
}

// loadKeys 加载签名密钥
func (m *JWTManager) loadKeys() error {
	alg := strings.ToUpper(m.config.Algorithm)

	switch {
	case strings.HasPrefix(alg, "HS"):
		// HMAC 算法，使用对称密钥
		if m.config.SecretKey == "" {
			return ErrSecretKeyEmpty
		}

	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "ES"):
		// RSA/ECDSA 算法，使用非对称密钥
		if m.config.PublicKeyFile != "" {
			pubKey, err := loadPublicKey(m.config.PublicKeyFile, alg)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPublicKeyLoad, err)
			}
			m.publicKey = pubKey
		}

		if m.config.PrivateKeyFile != "" {
			privKey, err := loadPrivateKey(m.config.PrivateKeyFile, alg)
			if err != nil {
				return fmt.Errorf("%w: %v", ErrPrivateKeyLoad, err)
			}
			m.privateKey = privKey
		}
	}

	return nil
}

// GenerateToken 生成 Token
func (m *JWTManager) GenerateToken(claims *Claims) (string, error) {
	now := time.Now()

	// 设置标准字段
	claims.IssuedAt = jwt.NewNumericDate(now)
	claims.NotBefore = jwt.NewNumericDate(now)
	if claims.ExpiresAt == nil {
		claims.ExpiresAt = jwt.NewNumericDate(now.Add(m.config.ExpiresIn))
	}
	if m.config.Issuer != "" && claims.Issuer == "" {
		claims.Issuer = m.config.Issuer
	}

	// 创建 Token
	token := jwt.NewWithClaims(m.getSigningMethod(), claims)

	// 签名
	return m.signToken(token)
}

// GenerateRefreshToken 生成刷新 Token
func (m *JWTManager) GenerateRefreshToken(payload map[string]any) (string, error) {
	claims := &Claims{
		Payload: payload,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.config.RefreshExpiresIn)),
		},
	}
	return m.GenerateToken(claims)
}

// ValidateToken 验证 Token
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	// 移除前缀
	tokenString = m.stripPrefix(tokenString)

	// 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (any, error) {
		// 验证算法
		if token.Method.Alg() != m.config.Algorithm {
			return nil, ErrAlgorithmMismatch
		}
		return m.getVerifyKey(), nil
	})

	if err != nil {
		return nil, m.wrapError(err)
	}

	// 提取 Claims
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// RefreshToken 刷新 Token
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		// 如果只是过期，可以刷新
		if !errors.Is(err, ErrTokenExpired) {
			return "", err
		}
	}

	// 生成新 Token
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(m.config.ExpiresIn))
	claims.IssuedAt = jwt.NewNumericDate(time.Now())

	return m.GenerateToken(claims)
}

// getSigningMethod 获取签名方法
func (m *JWTManager) getSigningMethod() jwt.SigningMethod {
	switch strings.ToUpper(m.config.Algorithm) {
	case "HS256":
		return jwt.SigningMethodHS256
	case "HS384":
		return jwt.SigningMethodHS384
	case "HS512":
		return jwt.SigningMethodHS512
	case "RS256":
		return jwt.SigningMethodRS256
	case "RS384":
		return jwt.SigningMethodRS384
	case "RS512":
		return jwt.SigningMethodRS512
	case "ES256":
		return jwt.SigningMethodES256
	case "ES384":
		return jwt.SigningMethodES384
	case "ES512":
		return jwt.SigningMethodES512
	default:
		return jwt.SigningMethodHS256
	}
}

// signToken 签名 Token
func (m *JWTManager) signToken(token *jwt.Token) (string, error) {
	alg := strings.ToUpper(m.config.Algorithm)

	switch {
	case strings.HasPrefix(alg, "HS"):
		return token.SignedString([]byte(m.config.SecretKey))
	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "ES"):
		return token.SignedString(m.privateKey)
	default:
		return token.SignedString([]byte(m.config.SecretKey))
	}
}

// getVerifyKey 获取验证密钥
func (m *JWTManager) getVerifyKey() any {
	alg := strings.ToUpper(m.config.Algorithm)

	switch {
	case strings.HasPrefix(alg, "HS"):
		return []byte(m.config.SecretKey)
	case strings.HasPrefix(alg, "RS"), strings.HasPrefix(alg, "ES"):
		return m.publicKey
	default:
		return []byte(m.config.SecretKey)
	}
}

// stripPrefix 移除 Token 前缀
func (m *JWTManager) stripPrefix(tokenString string) string {
	if m.config.TokenPrefix != "" && strings.HasPrefix(tokenString, m.config.TokenPrefix) {
		return strings.TrimPrefix(tokenString, m.config.TokenPrefix)
	}
	return tokenString
}

// wrapError 包装错误
func (m *JWTManager) wrapError(err error) error {
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return ErrTokenExpired
	case errors.Is(err, jwt.ErrTokenNotValidYet):
		return ErrTokenNotValidYet
	case errors.Is(err, jwt.ErrTokenMalformed):
		return ErrTokenMalformed
	case errors.Is(err, jwtErrSignatureInvalid):
		return ErrSignatureInvalid
	default:
		return fmt.Errorf("%w: %v", ErrTokenInvalid, err)
	}
}

// ShouldSkip 检查路径是否需要跳过验证
func (m *JWTManager) ShouldSkip(path string) bool {
	for _, skipPath := range m.config.SkipPaths {
		if matchPath(skipPath, path) {
			return true
		}
	}
	return false
}

// GetConfig 获取配置
func (m *JWTManager) GetConfig() *JWTConfig {
	return m.config
}

// matchPath 路径匹配（支持通配符 *）
func matchPath(pattern, path string) bool {
	if pattern == path {
		return true
	}

	// 支持前缀匹配，如 /api/public/*
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	return false
}

// loadPublicKey 加载公钥
func loadPublicKey(file string, alg string) (any, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(alg, "RS") {
		return jwt.ParseRSAPublicKeyFromPEM(data)
	} else if strings.HasPrefix(alg, "ES") {
		return jwt.ParseECPublicKeyFromPEM(data)
	}

	return nil, errors.New("unsupported algorithm")
}

// loadPrivateKey 加载私钥
func loadPrivateKey(file string, alg string) (any, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	if strings.HasPrefix(alg, "RS") {
		return jwt.ParseRSAPrivateKeyFromPEM(data)
	} else if strings.HasPrefix(alg, "ES") {
		return jwt.ParseECPrivateKeyFromPEM(data)
	}

	return nil, errors.New("unsupported algorithm")
}

// Context key 类型
type contextKey string

const (
	// ClaimsContextKey 用于在 context 中存储 Claims
	ClaimsContextKey contextKey = "jwt_claims"
)

// SetClaimsToContext 将 Claims 存入 context
func SetClaimsToContext(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, ClaimsContextKey, claims)
}

// GetClaimsFromContext 从 context 获取 Claims
func GetClaimsFromContext(ctx context.Context) (*Claims, bool) {
	claims, ok := ctx.Value(ClaimsContextKey).(*Claims)
	return claims, ok
}

// Unmarshal 将整个 Payload 解析到结构体
func (c *Claims) Unmarshal(v any) error {
	if c.Payload == nil {
		return nil
	}
	return mapstructure.Decode(c.Payload, v)
}

// UnmarshalKey 将指定 key 的值解析到结构体或基本类型
func (c *Claims) UnmarshalKey(key string, v any) error {
	if c.Payload == nil {
		return nil
	}
	val := c.getNestedValue(key)
	if val == nil {
		return nil
	}
	return mapstructure.Decode(val, v)
}

// Get 获取指定 key 的值（支持点号分隔的嵌套 key）
func (c *Claims) Get(key string) any {
	if c.Payload == nil {
		return nil
	}
	return c.getNestedValue(key)
}

// getNestedValue 支持点号分隔的嵌套 key，如 "user.profile.name"
func (c *Claims) getNestedValue(key string) any {
	keys := strings.Split(key, ".")
	var current any = c.Payload

	for _, k := range keys {
		if m, ok := current.(map[string]any); ok {
			current = m[k]
		} else {
			return nil
		}
	}
	return current
}

// 确保类型断言
var (
	_ any = (*rsa.PublicKey)(nil)
	_ any = (*rsa.PrivateKey)(nil)
	_ any = (*ecdsa.PublicKey)(nil)
	_ any = (*ecdsa.PrivateKey)(nil)
)
