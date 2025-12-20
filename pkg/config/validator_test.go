package config

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

// TestValidatorValidate 测试基本验证
func TestValidatorValidate(t *testing.T) {
	type Config struct {
		Port  int    `validate:"required,min=1,max=65535"`
		Host  string `validate:"required"`
		Email string `validate:"email"`
		Level string `validate:"oneof=debug info warn error"`
	}

	v := NewValidator()

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				Port:  8080,
				Host:  "localhost",
				Email: "test@example.com",
				Level: "info",
			},
			wantErr: false,
		},
		{
			name: "missing required field",
			config: Config{
				Port:  8080,
				Email: "test@example.com",
				Level: "info",
			},
			wantErr: true,
		},
		{
			name: "invalid port range",
			config: Config{
				Port:  70000,
				Host:  "localhost",
				Email: "test@example.com",
				Level: "info",
			},
			wantErr: true,
		},
		{
			name: "invalid email",
			config: Config{
				Port:  8080,
				Host:  "localhost",
				Email: "invalid-email",
				Level: "info",
			},
			wantErr: true,
		},
		{
			name: "invalid enum value",
			config: Config{
				Port:  8080,
				Host:  "localhost",
				Email: "test@example.com",
				Level: "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.Validate(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidatorValidateField 测试单字段验证
func TestValidatorValidateField(t *testing.T) {
	v := NewValidator()

	tests := []struct {
		name    string
		field   any
		tag     string
		wantErr bool
	}{
		{
			name:    "valid port",
			field:   8080,
			tag:     "min=1,max=65535",
			wantErr: false,
		},
		{
			name:    "invalid port",
			field:   0,
			tag:     "min=1,max=65535",
			wantErr: true,
		},
		{
			name:    "valid email",
			field:   "test@example.com",
			tag:     "email",
			wantErr: false,
		},
		{
			name:    "invalid email",
			field:   "invalid-email",
			tag:     "email",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateField(tt.field, tt.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateField() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidatorWithCustom 测试自定义验证规则
func TestValidatorWithCustom(t *testing.T) {
	type Config struct {
		Value string `validate:"custom_rule"`
	}

	v := NewValidator()

	// 自定义验证规则：值必须是 "valid"
	customRule := func(fl validator.FieldLevel) bool {
		return fl.Field().String() == "valid"
	}

	rules := map[string]validator.Func{
		"custom_rule": customRule,
	}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "valid custom rule",
			config:  Config{Value: "valid"},
			wantErr: false,
		},
		{
			name:    "invalid custom rule",
			config:  Config{Value: "invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.ValidateWithCustom(tt.config, rules)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithCustom() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestCheckRequired 测试必填字段检查
func TestCheckRequired(t *testing.T) {
	type NestedConfig struct {
		Value string `validate:"required"`
	}

	type Config struct {
		Name   string        `validate:"required"`
		Port   int           `validate:"required"`
		Nested *NestedConfig `validate:"required"`
	}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "all fields set",
			config: Config{
				Name: "test",
				Port: 8080,
				Nested: &NestedConfig{
					Value: "nested",
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			config: Config{
				Port: 8080,
				Nested: &NestedConfig{
					Value: "nested",
				},
			},
			wantErr: true,
		},
		{
			name: "missing nested value",
			config: Config{
				Name: "test",
				Port: 8080,
				Nested: &NestedConfig{
					Value: "",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckRequired(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckRequired() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestValidateWithFunc 测试函数验证
func TestValidateWithFunc(t *testing.T) {
	type Config struct {
		Value int
	}

	// 自定义验证函数：值必须是偶数
	validateEven := func(cfg any) error {
		c := cfg.(Config)
		if c.Value%2 != 0 {
			return ErrValidationFailed
		}
		return nil
	}

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name:    "even value",
			config:  Config{Value: 42},
			wantErr: false,
		},
		{
			name:    "odd value",
			config:  Config{Value: 43},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWithFunc(tt.config, validateEven)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWithFunc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestMustValidate 测试 MustValidate
func TestMustValidate(t *testing.T) {
	type Config struct {
		Port int `validate:"required,min=1"`
	}

	v := NewValidator()

	// 有效配置不应 panic
	validCfg := Config{Port: 8080}
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("MustValidate() panicked on valid config: %v", r)
		}
	}()
	v.MustValidate(validCfg)

	// 无效配置应该 panic
	invalidCfg := Config{Port: 0}
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustValidate() should have panicked on invalid config")
		}
	}()
	v.MustValidate(invalidCfg)
}
