package config

import "errors"

var (
	// ErrConfigFileNotFound 配置文件未找到
	ErrConfigFileNotFound = errors.New("config file not found")

	// ErrInvalidConfigFormat 配置格式无效
	ErrInvalidConfigFormat = errors.New("invalid config format")

	// ErrKeyNotFound 配置键不存在
	ErrKeyNotFound = errors.New("config key not found")

	// ErrInvalidType 配置类型不匹配
	ErrInvalidType = errors.New("invalid config type")

	// ErrValidationFailed 配置验证失败
	ErrValidationFailed = errors.New("config validation failed")

	// ErrNilConfig 配置为 nil
	ErrNilConfig = errors.New("config cannot be nil")

	// ErrMergeFailed 配置合并失败
	ErrMergeFailed = errors.New("failed to merge configs")
)
