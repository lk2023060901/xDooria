package validator

import (
	"reflect"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// Init 注册自定义验证器逻辑
func Init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// 注册 tag 名称转换逻辑，使错误信息显示 json tag 而非 struct 字段名
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})

		// 这里可以继续注册自定义校验逻辑，如 v.RegisterValidation("mobile", ...)
	}
}
