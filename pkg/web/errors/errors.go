package errors

import "net/http"

// 常见业务错误码
const (
	CodeOK             = 0
	CodeInvalidParams  = 40001
	CodeUnAuthorized   = 40002
	CodeForbidden      = 40003
	CodeNotFound       = 40004
	CodeRateLimited    = 40029
	CodeInternalError  = 50000
	CodeExternalError  = 50001
)

// CodeToStatus 将业务错误码映射为 HTTP 状态码
func CodeToStatus(code int) int {
	switch {
	case code == CodeOK:
		return http.StatusOK
	case code >= 40000 && code < 50000:
		if code == CodeUnAuthorized {
			return http.StatusUnauthorized
		}
		if code == CodeForbidden {
			return http.StatusForbidden
		}
		if code == CodeRateLimited {
			return http.StatusTooManyRequests
		}
		return http.StatusBadRequest
	case code >= 50000:
		return http.StatusInternalServerError
	default:
		return http.StatusOK
	}
}
