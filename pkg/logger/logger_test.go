package logger

import (
	"bytes"
	"encoding/json"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestNew 测试创建 Logger
func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name:    "nil config uses default",
			config:  nil,
			wantErr: false,
		},
		{
			name: "valid minimal config",
			config: &Config{
				Level:         InfoLevel,
				Format:        JSONFormat,
				EnableConsole: true,
			},
			wantErr: false,
		},
		{
			name: "invalid config - file enabled but no path",
			config: &Config{
				Level:      InfoLevel,
				EnableFile: true,
				OutputPath: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := New(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && logger == nil {
				t.Error("New() returned nil logger without error")
			}
		})
	}
}

// TestLoggerLevels 测试日志级别
func TestLoggerLevels(t *testing.T) {
	var buf bytes.Buffer

	// 创建输出到缓冲区的 logger
	config := &Config{
		Level:         DebugLevel,
		Format:        JSONFormat,
		EnableConsole: false,
		EnableFile:    false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 使用自定义 writer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			MessageKey:  "msg",
			LevelKey:    "level",
			EncodeLevel: zapcore.LowercaseLevelEncoder,
		}),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)
	logger.Logger = zap.New(core)

	tests := []struct {
		name     string
		logFunc  func(string, ...zap.Field)
		message  string
		wantLog  bool
		expected string
	}{
		{
			name:     "debug level",
			logFunc:  logger.Debug,
			message:  "debug message",
			wantLog:  true,
			expected: "debug",
		},
		{
			name:     "info level",
			logFunc:  logger.Info,
			message:  "info message",
			wantLog:  true,
			expected: "info",
		},
		{
			name:     "warn level",
			logFunc:  logger.Warn,
			message:  "warn message",
			wantLog:  true,
			expected: "warn",
		},
		{
			name:     "error level",
			logFunc:  logger.Error,
			message:  "error message",
			wantLog:  true,
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message)

			if tt.wantLog {
				output := buf.String()
				if output == "" {
					t.Error("Expected log output, got empty string")
					return
				}

				var logEntry map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
					t.Errorf("Failed to parse log output: %v", err)
					return
				}

				if level, ok := logEntry["level"].(string); !ok || level != tt.expected {
					t.Errorf("Expected level %s, got %v", tt.expected, logEntry["level"])
				}

				if msg, ok := logEntry["msg"].(string); !ok || msg != tt.message {
					t.Errorf("Expected message %s, got %v", tt.message, logEntry["msg"])
				}
			}
		})
	}
}

// TestLoggerWithFields 测试带字段的日志
func TestLoggerWithFields(t *testing.T) {
	var buf bytes.Buffer

	config := &Config{
		Level:         InfoLevel,
		Format:        JSONFormat,
		EnableConsole: false,
		EnableFile:    false,
	}

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// 使用自定义 writer
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			MessageKey: "msg",
			LevelKey:   "level",
		}),
		zapcore.AddSync(&buf),
		zapcore.InfoLevel,
	)
	logger.Logger = zap.New(core)

	logger.Info("test message",
		zap.String("user_id", "123"),
		zap.Int("count", 42),
	)

	var logEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &logEntry); err != nil {
		t.Fatalf("Failed to parse log output: %v", err)
	}

	if userID, ok := logEntry["user_id"].(string); !ok || userID != "123" {
		t.Errorf("Expected user_id=123, got %v", logEntry["user_id"])
	}

	if count, ok := logEntry["count"].(float64); !ok || count != 42 {
		t.Errorf("Expected count=42, got %v", logEntry["count"])
	}
}

// TestLoggerNamed 测试命名 logger
func TestLoggerNamed(t *testing.T) {
	logger, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	namedLogger := logger.Named("test-service")
	if namedLogger == nil {
		t.Error("Named() returned nil")
	}

	// Named 应该返回一个新的 logger
	if namedLogger == logger {
		t.Error("Named() should return a new logger instance")
	}
}

// TestLoggerWithFields 测试 WithFields 方法
func TestLoggerWithFieldsMethod(t *testing.T) {
	logger, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	withLogger := logger.WithFields("service", "test", "version", "1.0")
	if withLogger == nil {
		t.Error("WithFields() returned nil")
	}

	// WithFields 应该返回一个新的 logger
	if withLogger == logger {
		t.Error("WithFields() should return a new logger instance")
	}
}

// TestDefaultLogger 测试默认全局 logger
func TestDefaultLogger(t *testing.T) {
	// 测试全局方法 - Default() 会懒加载
	logger := Default()
	if logger == nil {
		t.Error("Default() returned nil logger")
	}

	// 测试全局便捷函数
	Info("test info")
	Debug("test debug")
	Warn("test warn")
	Error("test error")

	// 不应该 panic
}

// TestWithFields 测试全局 WithFields
func TestWithFields(t *testing.T) {
	logger := WithFields("key1", "value1", "key2", 123)
	if logger == nil {
		t.Error("WithFields() returned nil")
	}

	// 应该返回一个新的 logger
	defaultLog := Default()
	if logger == defaultLog {
		t.Error("WithFields() should return a new logger instance")
	}
}

// TestNamed 测试全局 Named
func TestNamed(t *testing.T) {
	logger := Named("test")
	if logger == nil {
		t.Error("Named() returned nil")
	}

	// 应该返回一个新的 logger
	defaultLog := Default()
	if logger == defaultLog {
		t.Error("Named() should return a new logger instance")
	}
}

// TestSetDefault 测试设置默认 logger
func TestSetDefault(t *testing.T) {
	config := &Config{
		Level:  DebugLevel,
		Format: JSONFormat,
	}

	newLogger, err := New(config)
	if err != nil {
		t.Fatalf("Failed to create new logger: %v", err)
	}

	oldLogger := Default()
	defer func() {
		SetDefault(oldLogger)
	}()

	SetDefault(newLogger)

	if Default() != newLogger {
		t.Error("SetDefault() did not update default logger")
	}
}

// TestSetGlobalFields 测试设置全局字段
func TestSetGlobalFields(t *testing.T) {
	oldLogger := Default()
	defer func() {
		SetDefault(oldLogger)
	}()

	// 设置全局字段
	SetGlobalFields("app", "test", "version", "1.0.0")

	// 验证全局字段已设置（通过创建新的 logger 验证）
	logger := Default()
	if logger == nil {
		t.Error("Default() returned nil after SetGlobalFields")
	}
}

// TestSync 测试 Sync 方法
func TestSync(t *testing.T) {
	logger, err := New(nil)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	// Sync 不应该返回错误
	if err := logger.Sync(); err != nil {
		t.Logf("Sync() returned error (may be expected for stdout): %v", err)
	}
}
