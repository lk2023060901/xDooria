package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// BindAndValidate 绑定请求参数并进行校验
func BindAndValidate(c *gin.Context, obj any) bool {
	if err := c.ShouldBind(obj); err != nil {
		// 如果是参数校验错误
		if errs, ok := err.(validator.ValidationErrors); ok {
			Error(c, http.StatusBadRequest, 400, errs.Error())
			return false
		}
		// 其他绑定错误
		Error(c, http.StatusBadRequest, 400, "invalid request parameters: "+err.Error())
		return false
	}
	return true
}

// GetQuery 获取查询参数，带默认值
func GetQuery(c *gin.Context, key, defaultValue string) string {
	val := c.Query(key)
	if val == "" {
		return defaultValue
	}
	return val
}
