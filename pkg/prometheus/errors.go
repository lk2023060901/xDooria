package prometheus

import "errors"

var (
	// ErrInvalidConfig 无效配置
	ErrInvalidConfig = errors.New("prometheus: invalid config")

	// ErrMetricExists 指标已存在
	ErrMetricExists = errors.New("prometheus: metric already exists")

	// ErrMetricNotFound 指标不存在
	ErrMetricNotFound = errors.New("prometheus: metric not found")

	// ErrClientClosed 客户端已关闭
	ErrClientClosed = errors.New("prometheus: client closed")
)
