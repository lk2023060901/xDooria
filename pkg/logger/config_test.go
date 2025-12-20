package logger

import (
	"testing"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// TestDefaultConfig 测试默认配置
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg == nil {
		t.Fatal("DefaultConfig() returned nil")
	}

	// 验证默认值
	if cfg.Level != InfoLevel {
		t.Errorf("Expected Level=InfoLevel, got %v", cfg.Level)
	}

	if cfg.Format != ConsoleFormat {
		t.Errorf("Expected Format=ConsoleFormat, got %v", cfg.Format)
	}

	if !cfg.EnableConsole {
		t.Error("Expected EnableConsole=true")
	}

	if cfg.EnableFile {
		t.Error("Expected EnableFile=false")
	}

	if cfg.TimeFormat != "2006-01-02 15:04:05" {
		t.Errorf("Expected TimeFormat='2006-01-02 15:04:05', got %s", cfg.TimeFormat)
	}

	if cfg.Rotation.Type != RotationBySize {
		t.Errorf("Expected RotationType=size, got %v", cfg.Rotation.Type)
	}

	if cfg.Rotation.MaxSize != 100 {
		t.Errorf("Expected MaxSize=100, got %d", cfg.Rotation.MaxSize)
	}

	if cfg.EnableStacktrace != true {
		t.Error("Expected EnableStacktrace=true")
	}

	if cfg.StacktraceLevel != ErrorLevel {
		t.Errorf("Expected StacktraceLevel=ErrorLevel, got %v", cfg.StacktraceLevel)
	}
}

// TestConfigValidate 测试配置验证
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errType error
	}{
		{
			name: "valid config - console only",
			config: &Config{
				Level:         InfoLevel,
				EnableConsole: true,
				EnableFile:    false,
			},
			wantErr: false,
		},
		{
			name: "valid config - file only",
			config: &Config{
				Level:         InfoLevel,
				EnableConsole: false,
				EnableFile:    true,
				OutputPath:    "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "valid config - both outputs",
			config: &Config{
				Level:         InfoLevel,
				EnableConsole: true,
				EnableFile:    true,
				OutputPath:    "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "invalid - no output enabled",
			config: &Config{
				Level:         InfoLevel,
				EnableConsole: false,
				EnableFile:    false,
			},
			wantErr: true,
			errType: ErrNoOutputEnabled,
		},
		{
			name: "invalid - file enabled but no path",
			config: &Config{
				Level:         InfoLevel,
				EnableConsole: false,
				EnableFile:    true,
				OutputPath:    "",
			},
			wantErr: true,
			errType: ErrInvalidOutputPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.errType != nil && err != tt.errType {
				t.Errorf("Validate() error = %v, want %v", err, tt.errType)
			}
		})
	}
}

// TestConfigMerge 测试配置合并
func TestConfigMerge(t *testing.T) {
	defaultCfg := DefaultConfig()
	customCfg := &Config{
		Level:  DebugLevel,
		Format: JSONFormat,
	}

	merged, err := config.MergeConfig(defaultCfg, customCfg)
	if err != nil {
		t.Fatalf("MergeConfig() error = %v", err)
	}

	// 验证合并结果
	if merged.Level != DebugLevel {
		t.Errorf("Expected Level=DebugLevel, got %v", merged.Level)
	}

	if merged.Format != JSONFormat {
		t.Errorf("Expected Format=JSONFormat, got %v", merged.Format)
	}

	// 验证默认值被保留
	if !merged.EnableConsole {
		t.Error("Expected EnableConsole=true from default")
	}

	if merged.TimeFormat != "2006-01-02 15:04:05" {
		t.Errorf("Expected TimeFormat from default, got %s", merged.TimeFormat)
	}
}

// TestRotationConfig 测试轮换配置
func TestRotationConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Rotation.Type != RotationBySize {
		t.Errorf("Expected RotationBySize, got %v", cfg.Rotation.Type)
	}

	if cfg.Rotation.MaxSize != 100 {
		t.Errorf("Expected MaxSize=100, got %d", cfg.Rotation.MaxSize)
	}

	if cfg.Rotation.MaxBackups != 5 {
		t.Errorf("Expected MaxBackups=5, got %d", cfg.Rotation.MaxBackups)
	}

	if cfg.Rotation.MaxAge != 7 {
		t.Errorf("Expected MaxAge=7, got %d", cfg.Rotation.MaxAge)
	}

	if !cfg.Rotation.Compress {
		t.Error("Expected Compress=true")
	}

	if cfg.Rotation.RotationTime != "24h" {
		t.Errorf("Expected RotationTime=24h, got %s", cfg.Rotation.RotationTime)
	}

	if cfg.Rotation.MaxAgeTime != "168h" {
		t.Errorf("Expected MaxAgeTime=168h, got %s", cfg.Rotation.MaxAgeTime)
	}

	if cfg.Rotation.RotationPattern != ".%Y%m%d" {
		t.Errorf("Expected RotationPattern=.%%Y%%m%%d, got %s", cfg.Rotation.RotationPattern)
	}
}

// TestLevelConstants 测试日志级别常量
func TestLevelConstants(t *testing.T) {
	levels := []Level{
		DebugLevel,
		InfoLevel,
		WarnLevel,
		ErrorLevel,
		PanicLevel,
		FatalLevel,
	}

	expected := []string{
		"debug",
		"info",
		"warn",
		"error",
		"panic",
		"fatal",
	}

	for i, level := range levels {
		if string(level) != expected[i] {
			t.Errorf("Expected level %s, got %s", expected[i], string(level))
		}
	}
}

// TestFormatConstants 测试日志格式常量
func TestFormatConstants(t *testing.T) {
	if string(JSONFormat) != "json" {
		t.Errorf("Expected JSONFormat='json', got %s", string(JSONFormat))
	}

	if string(ConsoleFormat) != "console" {
		t.Errorf("Expected ConsoleFormat='console', got %s", string(ConsoleFormat))
	}
}

// TestRotationTypeConstants 测试轮换类型常量
func TestRotationTypeConstants(t *testing.T) {
	if string(RotationBySize) != "size" {
		t.Errorf("Expected RotationBySize='size', got %s", string(RotationBySize))
	}

	if string(RotationByTime) != "time" {
		t.Errorf("Expected RotationByTime='time', got %s", string(RotationByTime))
	}
}

// TestGlobalFields 测试全局字段配置
func TestGlobalFields(t *testing.T) {
	cfg := &Config{
		Level:         InfoLevel,
		EnableConsole: true,
		GlobalFields: map[string]interface{}{
			"app":     "test-app",
			"version": "1.0.0",
			"env":     "production",
		},
	}

	if len(cfg.GlobalFields) != 3 {
		t.Errorf("Expected 3 global fields, got %d", len(cfg.GlobalFields))
	}

	if cfg.GlobalFields["app"] != "test-app" {
		t.Errorf("Expected app='test-app', got %v", cfg.GlobalFields["app"])
	}

	if cfg.GlobalFields["version"] != "1.0.0" {
		t.Errorf("Expected version='1.0.0', got %v", cfg.GlobalFields["version"])
	}

	if cfg.GlobalFields["env"] != "production" {
		t.Errorf("Expected env='production', got %v", cfg.GlobalFields["env"])
	}
}

// TestSamplingConfig 测试采样配置
func TestSamplingConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.EnableSampling {
		t.Error("Expected EnableSampling=false by default")
	}

	if cfg.SamplingInitial != 100 {
		t.Errorf("Expected SamplingInitial=100, got %d", cfg.SamplingInitial)
	}

	if cfg.SamplingThereafter != 100 {
		t.Errorf("Expected SamplingThereafter=100, got %d", cfg.SamplingThereafter)
	}
}

// TestAsyncConfig 测试异步配置
func TestAsyncConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.EnableAsync {
		t.Error("Expected EnableAsync=false by default")
	}

	if cfg.BufferSize != 256*1024 {
		t.Errorf("Expected BufferSize=262144, got %d", cfg.BufferSize)
	}
}

// TestPartialConfig 测试部分配置合并
func TestPartialConfig(t *testing.T) {
	// 只设置部分字段
	partial := &Config{
		Level: DebugLevel,
	}

	defaultCfg := DefaultConfig()
	merged, err := config.MergeConfig(defaultCfg, partial)
	if err != nil {
		t.Fatalf("MergeConfig() error = %v", err)
	}

	// Level 应该被覆盖
	if merged.Level != DebugLevel {
		t.Errorf("Expected Level=DebugLevel, got %v", merged.Level)
	}

	// 其他字段应该使用默认值
	if merged.Format != ConsoleFormat {
		t.Errorf("Expected Format=ConsoleFormat, got %v", merged.Format)
	}

	if !merged.EnableConsole {
		t.Error("Expected EnableConsole=true")
	}
}
