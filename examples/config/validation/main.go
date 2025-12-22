package main

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	"github.com/lk2023060901/xdooria/pkg/config"
)

// ValidConfig 有效的配置结构
type ValidConfig struct {
	Server struct {
		Port int    `yaml:"port" validate:"required,min=1,max=65535"`
		Host string `yaml:"host" validate:"required"`
	} `yaml:"server"`

	Database struct {
		Host     string `yaml:"host" validate:"required"`
		Port     int    `yaml:"port" validate:"required,min=1,max=65535"`
		User     string `yaml:"user" validate:"required"`
		Password string `yaml:"password" validate:"required,min=8"`
		DBName   string `yaml:"dbname" validate:"required"`
	} `yaml:"database"`

	Logger struct {
		Level  string `yaml:"level" validate:"required,oneof=debug info warn error"`
		Format string `yaml:"format" validate:"oneof=json text"`
	} `yaml:"logger"`

	Admin struct {
		Email string `yaml:"email" validate:"email"`
		URL   string `yaml:"url" validate:"url"`
	} `yaml:"admin"`
}

func main() {
	fmt.Println("=== 示例 4：配置验证 ===")

	v := config.NewValidator()

	// 1. 验证有效配置
	fmt.Println("【1. 验证有效配置】")
	testValidConfig(v)

	// 2. 验证缺少必填字段
	fmt.Println("\n【2. 验证缺少必填字段】")
	testMissingRequired(v)

	// 3. 验证端口范围
	fmt.Println("\n【3. 验证端口范围】")
	testInvalidPort(v)

	// 4. 验证枚举值
	fmt.Println("\n【4. 验证枚举值】")
	testInvalidEnum(v)

	// 5. 验证邮箱格式
	fmt.Println("\n【5. 验证邮箱和 URL 格式】")
	testEmailAndURL(v)

	// 6. 单字段验证
	fmt.Println("\n【6. 单字段验证】")
	testFieldValidation(v)

	// 7. 自定义验证规则
	fmt.Println("\n【7. 自定义验证规则】")
	testCustomValidation(v)

	fmt.Println("\n✅ 示例完成")
}

func testValidConfig(v *config.Validator) {
	mgr := config.NewManager()
	mgr.LoadFile("valid-config.yaml")

	var cfg ValidConfig
	mgr.Unmarshal(&cfg)

	if err := v.Validate(cfg); err != nil {
		fmt.Printf("  ❌ 验证失败: %v\n", err)
	} else {
		fmt.Println("  ✅ 配置验证通过")
	}
}

func testMissingRequired(v *config.Validator) {
	mgr := config.NewManager()
	mgr.LoadFile("missing-required.yaml")

	var cfg ValidConfig
	mgr.Unmarshal(&cfg)

	if err := v.Validate(cfg); err != nil {
		fmt.Printf("  ❌ 验证失败（符合预期）:\n")
		fmt.Printf("     %v\n", err)
	} else {
		fmt.Println("  ❌ 应该验证失败但通过了")
	}
}

func testInvalidPort(v *config.Validator) {
	mgr := config.NewManager()
	mgr.LoadFile("invalid-port.yaml")

	var cfg ValidConfig
	mgr.Unmarshal(&cfg)

	if err := v.Validate(cfg); err != nil {
		fmt.Printf("  ❌ 验证失败（符合预期）:\n")
		fmt.Printf("     %v\n", err)
	} else {
		fmt.Println("  ❌ 应该验证失败但通过了")
	}
}

func testInvalidEnum(v *config.Validator) {
	mgr := config.NewManager()
	mgr.LoadFile("invalid-enum.yaml")

	var cfg ValidConfig
	mgr.Unmarshal(&cfg)

	if err := v.Validate(cfg); err != nil {
		fmt.Printf("  ❌ 验证失败（符合预期）:\n")
		fmt.Printf("     %v\n", err)
	} else {
		fmt.Println("  ❌ 应该验证失败但通过了")
	}
}

func testEmailAndURL(v *config.Validator) {
	mgr := config.NewManager()
	mgr.LoadFile("invalid-email.yaml")

	var cfg ValidConfig
	mgr.Unmarshal(&cfg)

	if err := v.Validate(cfg); err != nil {
		fmt.Printf("  ❌ 验证失败（符合预期）:\n")
		fmt.Printf("     %v\n", err)
	} else {
		fmt.Println("  ❌ 应该验证失败但通过了")
	}
}

func testFieldValidation(v *config.Validator) {
	// 验证有效端口
	port := 8080
	if err := v.ValidateField(port, "min=1,max=65535"); err != nil {
		fmt.Printf("  ❌ 端口 %d 验证失败: %v\n", port, err)
	} else {
		fmt.Printf("  ✅ 端口 %d 验证通过\n", port)
	}

	// 验证无效端口
	invalidPort := 70000
	if err := v.ValidateField(invalidPort, "min=1,max=65535"); err != nil {
		fmt.Printf("  ❌ 端口 %d 验证失败（符合预期）: %v\n", invalidPort, err)
	} else {
		fmt.Printf("  ❌ 端口 %d 应该验证失败但通过了\n", invalidPort)
	}

	// 验证邮箱
	email := "admin@example.com"
	if err := v.ValidateField(email, "email"); err != nil {
		fmt.Printf("  ❌ 邮箱 %s 验证失败: %v\n", email, err)
	} else {
		fmt.Printf("  ✅ 邮箱 %s 验证通过\n", email)
	}
}

func testCustomValidation(v *config.Validator) {
	type CustomConfig struct {
		Environment string `validate:"custom_env"`
	}

	// 自定义验证规则：环境必须是 dev, test, prod 之一
	customEnvRule := func(fl validator.FieldLevel) bool {
		env := fl.Field().String()
		return env == "dev" || env == "test" || env == "prod"
	}

	rules := map[string]validator.Func{
		"custom_env": customEnvRule,
	}

	// 有效环境
	validCfg := CustomConfig{Environment: "prod"}
	if err := v.ValidateWithCustom(validCfg, rules); err != nil {
		fmt.Printf("  ❌ 验证失败: %v\n", err)
	} else {
		fmt.Printf("  ✅ 自定义规则验证通过（environment=prod）\n")
	}

	// 无效环境
	invalidCfg := CustomConfig{Environment: "invalid"}
	if err := v.ValidateWithCustom(invalidCfg, rules); err != nil {
		fmt.Printf("  ❌ 自定义规则验证失败（符合预期，environment=invalid）\n")
	} else {
		fmt.Printf("  ❌ 应该验证失败但通过了\n")
	}
}
