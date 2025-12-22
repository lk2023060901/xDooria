package security

import "errors"

// TLS 错误
var (
	ErrCertFileEmpty = errors.New("security: cert file is empty")
	ErrKeyFileEmpty  = errors.New("security: key file is empty")
	ErrCAFileEmpty   = errors.New("security: CA file is empty for mTLS")
	ErrCertLoad      = errors.New("security: failed to load certificate")
	ErrCALoad        = errors.New("security: failed to load CA certificate")
	ErrCAAppend      = errors.New("security: failed to append CA certificate")
)

// JWT 错误
var (
	ErrSecretKeyEmpty    = errors.New("security: secret key is empty")
	ErrPublicKeyEmpty    = errors.New("security: public key file is empty")
	ErrPrivateKeyEmpty   = errors.New("security: private key file is empty")
	ErrPublicKeyLoad     = errors.New("security: failed to load public key")
	ErrPrivateKeyLoad    = errors.New("security: failed to load private key")
	ErrTokenMissing      = errors.New("security: token is missing")
	ErrTokenInvalid      = errors.New("security: token is invalid")
	ErrTokenExpired      = errors.New("security: token has expired")
	ErrTokenNotValidYet  = errors.New("security: token is not valid yet")
	ErrTokenMalformed    = errors.New("security: token is malformed")
	ErrAlgorithmInvalid  = errors.New("security: invalid algorithm")
	ErrAlgorithmMismatch = errors.New("security: algorithm mismatch")
)

// IP 过滤错误
var (
	ErrIPDenied    = errors.New("security: IP address denied")
	ErrIPInvalid   = errors.New("security: invalid IP address")
	ErrCIDRInvalid = errors.New("security: invalid CIDR")
	ErrModeInvalid = errors.New("security: invalid mode, must be 'whitelist' or 'blacklist'")
	ErrIPListEmpty = errors.New("security: IP list is empty")
)

// 签名错误
var (
	ErrSignatureMissing = errors.New("security: signature is missing")
	ErrSignatureInvalid = errors.New("security: signature is invalid")
	ErrTimestampMissing = errors.New("security: timestamp is missing")
	ErrTimestampInvalid = errors.New("security: timestamp is invalid")
	ErrTimestampExpired = errors.New("security: timestamp expired")
	ErrNonceMissing     = errors.New("security: nonce is missing")
	ErrNonceReused      = errors.New("security: nonce has been used")
)
