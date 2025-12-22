package feishu

import "errors"

var (
	// ErrRequestFailed HTTP 请求失败
	ErrRequestFailed = errors.New("feishu: http request failed")

	// ErrResponseInvalid 响应格式无效
	ErrResponseInvalid = errors.New("feishu: invalid response")

	// ErrAPIError 飞书 API 返回错误
	ErrAPIError = errors.New("feishu: api error")
)
