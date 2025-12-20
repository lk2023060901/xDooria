package config

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Validator 配置验证器
type Validator struct {
	validate *validator.Validate
}

// NewValidator 创建验证器
func NewValidator() *Validator {
	return &Validator{
		validate: validator.New(),
	}
}

// Validate 验证配置结构体
// 支持标准的 validator tag，如：
// - required: 必填字段
// - min=1,max=100: 数值范围
// - oneof=debug info warn error: 枚举值
// - email: 邮箱格式
// - url: URL 格式
func (v *Validator) Validate(cfg any) error {
	if cfg == nil {
		return ErrNilConfig
	}

	if err := v.validate.Struct(cfg); err != nil {
		return fmt.Errorf("%w: %v", ErrValidationFailed, formatValidationErrors(err))
	}

	return nil
}

// ValidateWithCustom 使用自定义验证规则验证配置
func (v *Validator) ValidateWithCustom(cfg any, rules map[string]validator.Func) error {
	// 注册自定义验证规则
	for tag, fn := range rules {
		if err := v.validate.RegisterValidation(tag, fn); err != nil {
			return fmt.Errorf("failed to register custom validation %s: %w", tag, err)
		}
	}

	return v.Validate(cfg)
}

// formatValidationErrors 格式化验证错误信息
func formatValidationErrors(err error) string {
	var sb strings.Builder

	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		for i, fieldErr := range validationErrors {
			if i > 0 {
				sb.WriteString("; ")
			}

			field := fieldErr.Field()
			tag := fieldErr.Tag()
			param := fieldErr.Param()

			switch tag {
			case "required":
				sb.WriteString(fmt.Sprintf("field '%s' is required", field))
			case "min":
				sb.WriteString(fmt.Sprintf("field '%s' must be at least %s", field, param))
			case "max":
				sb.WriteString(fmt.Sprintf("field '%s' must be at most %s", field, param))
			case "oneof":
				sb.WriteString(fmt.Sprintf("field '%s' must be one of [%s]", field, param))
			case "email":
				sb.WriteString(fmt.Sprintf("field '%s' must be a valid email", field))
			case "url":
				sb.WriteString(fmt.Sprintf("field '%s' must be a valid URL", field))
			case "gte":
				sb.WriteString(fmt.Sprintf("field '%s' must be greater than or equal to %s", field, param))
			case "lte":
				sb.WriteString(fmt.Sprintf("field '%s' must be less than or equal to %s", field, param))
			default:
				sb.WriteString(fmt.Sprintf("field '%s' failed validation '%s'", field, tag))
			}
		}
	} else {
		sb.WriteString(err.Error())
	}

	return sb.String()
}

// ValidateFunc 验证函数类型
type ValidateFunc func(cfg any) error

// ValidateWithFunc 使用自定义函数验证配置
func ValidateWithFunc(cfg any, fn ValidateFunc) error {
	if cfg == nil {
		return ErrNilConfig
	}
	return fn(cfg)
}

// MustValidate 验证配置，失败则 panic
func (v *Validator) MustValidate(cfg any) {
	if err := v.Validate(cfg); err != nil {
		panic(err)
	}
}

// ValidateField 验证单个字段
func (v *Validator) ValidateField(field any, tag string) error {
	if err := v.validate.Var(field, tag); err != nil {
		return fmt.Errorf("%w: %v", ErrValidationFailed, formatValidationErrors(err))
	}
	return nil
}

// CheckRequired 检查必填字段（递归检查结构体）
func CheckRequired(cfg any) error {
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Errorf("expected struct, got %s", v.Kind())
	}

	return checkRequiredFields(v, "")
}

// checkRequiredFields 递归检查必填字段
func checkRequiredFields(v reflect.Value, prefix string) error {
	t := v.Type()

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		// 跳过未导出字段
		if !fieldType.IsExported() {
			continue
		}

		fieldName := fieldType.Name
		if prefix != "" {
			fieldName = prefix + "." + fieldName
		}

		// 检查 required tag
		tag := fieldType.Tag.Get("validate")
		if strings.Contains(tag, "required") {
			if isZeroValueValidator(field) {
				return fmt.Errorf("%w: field '%s' is required but empty", ErrValidationFailed, fieldName)
			}
		}

		// 递归检查嵌套结构体
		if field.Kind() == reflect.Struct {
			if err := checkRequiredFields(field, fieldName); err != nil {
				return err
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() && field.Elem().Kind() == reflect.Struct {
			if err := checkRequiredFields(field.Elem(), fieldName); err != nil {
				return err
			}
		}
	}

	return nil
}

// isZeroValueValidator 检查是否为零值（验证器专用）
func isZeroValueValidator(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Ptr, reflect.Interface:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Struct:
		return reflect.DeepEqual(v.Interface(), reflect.Zero(v.Type()).Interface())
	default:
		return false
	}
}
