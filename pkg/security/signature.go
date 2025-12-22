package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// SignatureConfig 签名验证配置
type SignatureConfig struct {
	// 签名密钥
	SecretKey string `mapstructure:"secret_key" json:"secret_key"`

	// 签名算法（默认 HMAC-SHA256）
	Algorithm string `mapstructure:"algorithm" json:"algorithm"`

	// 时间戳容差（防重放攻击，默认 5 分钟）
	TimestampTolerance time.Duration `mapstructure:"timestamp_tolerance" json:"timestamp_tolerance"`

	// 签名 Header 名称（默认 X-Signature）
	SignatureHeader string `mapstructure:"signature_header" json:"signature_header"`

	// 时间戳 Header 名称（默认 X-Timestamp）
	TimestampHeader string `mapstructure:"timestamp_header" json:"timestamp_header"`

	// Nonce Header 名称（可选，防重放）
	NonceHeader string `mapstructure:"nonce_header" json:"nonce_header"`

	// 跳过检查的路径
	SkipPaths []string `mapstructure:"skip_paths" json:"skip_paths"`
}

// 签名算法常量
const (
	SignatureAlgorithmHMACSHA256 = "HMAC-SHA256"
	SignatureAlgorithmHMACSHA512 = "HMAC-SHA512"
)


// DefaultSignatureConfig 返回默认签名配置（最小可用配置）
func DefaultSignatureConfig() *SignatureConfig {
	return &SignatureConfig{
		Algorithm:          SignatureAlgorithmHMACSHA256,
		TimestampTolerance: 5 * time.Minute,
		SignatureHeader:    "X-Signature",
		TimestampHeader:    "X-Timestamp",
	}
}

// NonceValidator Nonce 验证器接口
type NonceValidator interface {
	// Validate 验证 nonce 是否有效（未被使用过）
	// 如果有效，应该将其标记为已使用
	Validate(nonce string, ttl time.Duration) bool
}

// SignatureVerifier 签名验证器
type SignatureVerifier struct {
	config         *SignatureConfig
	nonceValidator NonceValidator
}

// NewSignatureVerifier 创建签名验证器
func NewSignatureVerifier(cfg *SignatureConfig) (*SignatureVerifier, error) {
	newCfg, err := config.MergeConfig(DefaultSignatureConfig(), cfg)
	if err != nil {
		return nil, err
	}

	// 验证密钥
	if newCfg.SecretKey == "" {
		return nil, ErrSecretKeyEmpty
	}

	// 验证算法
	if newCfg.Algorithm != SignatureAlgorithmHMACSHA256 && newCfg.Algorithm != SignatureAlgorithmHMACSHA512 {
		return nil, ErrAlgorithmInvalid
	}

	return &SignatureVerifier{
		config: newCfg,
	}, nil
}

// SetNonceValidator 设置 Nonce 验证器（可选）
func (v *SignatureVerifier) SetNonceValidator(validator NonceValidator) {
	v.nonceValidator = validator
}

// Verify 验证签名
// params: 请求参数（用于生成签名的数据）
// headers: 请求头（包含签名、时间戳等）
func (v *SignatureVerifier) Verify(params map[string]string, headers map[string]string) error {
	// 获取签名
	signature, ok := headers[v.config.SignatureHeader]
	if !ok || signature == "" {
		return ErrSignatureMissing
	}

	// 获取时间戳
	timestampStr, ok := headers[v.config.TimestampHeader]
	if !ok || timestampStr == "" {
		return ErrTimestampMissing
	}

	// 验证时间戳
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return ErrTimestampInvalid
	}

	if err := v.validateTimestamp(timestamp); err != nil {
		return err
	}

	// 验证 Nonce（如果配置了）
	if v.config.NonceHeader != "" {
		nonce, ok := headers[v.config.NonceHeader]
		if !ok || nonce == "" {
			return ErrNonceMissing
		}
		if err := v.validateNonce(nonce); err != nil {
			return err
		}
	}

	// 计算期望的签名
	expectedSignature := v.Sign(params, timestampStr)

	// 比较签名（使用常量时间比较防止时序攻击）
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return ErrSignatureInvalid
	}

	return nil
}

// Sign 生成签名
func (v *SignatureVerifier) Sign(params map[string]string, timestamp string) string {
	// 构建待签名字符串
	data := v.buildSignData(params, timestamp)

	// 计算签名
	return v.computeSignature(data)
}

// buildSignData 构建待签名数据
func (v *SignatureVerifier) buildSignData(params map[string]string, timestamp string) string {
	// 按 key 排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建字符串：key1=value1&key2=value2&timestamp=xxx
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(k)
		builder.WriteByte('=')
		builder.WriteString(params[k])
	}

	// 添加时间戳
	if builder.Len() > 0 {
		builder.WriteByte('&')
	}
	builder.WriteString("timestamp=")
	builder.WriteString(timestamp)

	return builder.String()
}

// computeSignature 计算签名
func (v *SignatureVerifier) computeSignature(data string) string {
	var h hash.Hash
	switch v.config.Algorithm {
	case SignatureAlgorithmHMACSHA512:
		h = hmac.New(sha512.New, []byte(v.config.SecretKey))
	default:
		h = hmac.New(sha256.New, []byte(v.config.SecretKey))
	}
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// validateTimestamp 验证时间戳
func (v *SignatureVerifier) validateTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	diff := now - timestamp
	if diff < 0 {
		diff = -diff
	}

	if diff > int64(v.config.TimestampTolerance.Seconds()) {
		return ErrTimestampExpired
	}

	return nil
}

// validateNonce 验证 Nonce
func (v *SignatureVerifier) validateNonce(nonce string) error {
	if v.nonceValidator == nil {
		// 没有设置验证器，跳过验证
		return nil
	}

	if !v.nonceValidator.Validate(nonce, v.config.TimestampTolerance*2) {
		return ErrNonceReused
	}

	return nil
}

// ShouldSkip 检查路径是否需要跳过验证
func (v *SignatureVerifier) ShouldSkip(path string) bool {
	for _, skipPath := range v.config.SkipPaths {
		if matchPath(skipPath, path) {
			return true
		}
	}
	return false
}

// GetConfig 获取配置
func (v *SignatureVerifier) GetConfig() *SignatureConfig {
	return v.config
}

// SignatureHelper 签名辅助工具（供客户端使用）
type SignatureHelper struct {
	secretKey string
	algorithm string
}

// NewSignatureHelper 创建签名辅助工具
func NewSignatureHelper(secretKey, algorithm string) *SignatureHelper {
	if algorithm == "" {
		algorithm = SignatureAlgorithmHMACSHA256
	}
	return &SignatureHelper{
		secretKey: secretKey,
		algorithm: algorithm,
	}
}

// Sign 生成签名，返回签名和时间戳
func (h *SignatureHelper) Sign(params map[string]string) (signature string, timestamp string) {
	timestamp = strconv.FormatInt(time.Now().Unix(), 10)

	// 按 key 排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 构建字符串
	var builder strings.Builder
	for i, k := range keys {
		if i > 0 {
			builder.WriteByte('&')
		}
		builder.WriteString(k)
		builder.WriteByte('=')
		builder.WriteString(params[k])
	}

	if builder.Len() > 0 {
		builder.WriteByte('&')
	}
	builder.WriteString("timestamp=")
	builder.WriteString(timestamp)

	// 计算签名
	var mac hash.Hash
	switch h.algorithm {
	case SignatureAlgorithmHMACSHA512:
		mac = hmac.New(sha512.New, []byte(h.secretKey))
	default:
		mac = hmac.New(sha256.New, []byte(h.secretKey))
	}
	mac.Write([]byte(builder.String()))
	signature = hex.EncodeToString(mac.Sum(nil))

	return signature, timestamp
}

// GenerateNonce 生成随机 Nonce
func GenerateNonce() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
