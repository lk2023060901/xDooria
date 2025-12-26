package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应结构
type Response struct {
	Code    int    `json:"code"`    // 业务错误码
	Message string `json:"message"` // 提示信息
	Data    any    `json:"data"`    // 数据载体
	TraceID string `json:"trace_id,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Response{
		Code:    0, // 0 表示成功
		Message: "ok",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}

// AbortWithError 中断并返回错误
func AbortWithError(c *gin.Context, httpStatus int, code int, message string) {
	c.AbortWithStatusJSON(httpStatus, Response{
		Code:    code,
		Message: message,
		Data:    nil,
	})
}
