package logger

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/natefinch/lumberjack.v2"
)

// TestNewRotationWriter 测试创建轮换 writer
func TestNewRotationWriter(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "test.log")

	tests := []struct {
		name    string
		config  *RotationConfig
		wantErr bool
	}{
		{
			name: "size rotation",
			config: &RotationConfig{
				Type:       RotationBySize,
				MaxSize:    100,
				MaxBackups: 5,
				MaxAge:     7,
				Compress:   true,
			},
			wantErr: false,
		},
		{
			name: "time rotation",
			config: &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "24h",
				MaxAgeTime:      "168h",
				RotationPattern: ".%Y%m%d",
			},
			wantErr: false,
		},
		{
			name: "default rotation (size)",
			config: &RotationConfig{
				Type: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := NewRotationWriter(tt.config, outputPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewRotationWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && writer == nil {
				t.Error("NewRotationWriter() returned nil writer without error")
			}
		})
	}
}

// TestSizeRotationWriter 测试按大小轮换的 writer
func TestSizeRotationWriter(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "size-test.log")

	cfg := &RotationConfig{
		Type:       RotationBySize,
		MaxSize:    10, // 10MB
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
	}

	writer := newSizeRotationWriter(cfg, outputPath)
	if writer == nil {
		t.Fatal("newSizeRotationWriter() returned nil")
	}

	// 验证返回的是 lumberjack.Logger
	lumberLogger, ok := writer.(*lumberjack.Logger)
	if !ok {
		t.Fatal("Expected *lumberjack.Logger")
	}

	// 验证配置
	if lumberLogger.Filename != outputPath {
		t.Errorf("Expected Filename=%s, got %s", outputPath, lumberLogger.Filename)
	}

	if lumberLogger.MaxSize != cfg.MaxSize {
		t.Errorf("Expected MaxSize=%d, got %d", cfg.MaxSize, lumberLogger.MaxSize)
	}

	if lumberLogger.MaxBackups != cfg.MaxBackups {
		t.Errorf("Expected MaxBackups=%d, got %d", cfg.MaxBackups, lumberLogger.MaxBackups)
	}

	if lumberLogger.MaxAge != cfg.MaxAge {
		t.Errorf("Expected MaxAge=%d, got %d", cfg.MaxAge, lumberLogger.MaxAge)
	}

	if lumberLogger.Compress != cfg.Compress {
		t.Errorf("Expected Compress=%v, got %v", cfg.Compress, lumberLogger.Compress)
	}

	if !lumberLogger.LocalTime {
		t.Error("Expected LocalTime=true")
	}
}

// TestTimeRotationWriter 测试按时间轮换的 writer
func TestTimeRotationWriter(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "time-test.log")

	tests := []struct {
		name    string
		config  *RotationConfig
		wantErr bool
	}{
		{
			name: "valid time rotation",
			config: &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "24h",
				MaxAgeTime:      "168h",
				RotationPattern: ".%Y%m%d",
			},
			wantErr: false,
		},
		{
			name: "hourly rotation",
			config: &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "1h",
				MaxAgeTime:      "72h",
				RotationPattern: ".%Y%m%d%H",
			},
			wantErr: false,
		},
		{
			name: "invalid rotation time (uses default)",
			config: &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "invalid",
				MaxAgeTime:      "168h",
				RotationPattern: ".%Y%m%d",
			},
			wantErr: false,
		},
		{
			name: "invalid max age (uses default)",
			config: &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "24h",
				MaxAgeTime:      "invalid",
				RotationPattern: ".%Y%m%d",
			},
			wantErr: false,
		},
		{
			name: "empty pattern (uses default)",
			config: &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "24h",
				MaxAgeTime:      "168h",
				RotationPattern: "",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer, err := newTimeRotationWriter(tt.config, outputPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTimeRotationWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && writer == nil {
				t.Error("newTimeRotationWriter() returned nil writer without error")
			}
		})
	}
}

// TestRotationWriterWrite 测试 writer 写入功能
func TestRotationWriterWrite(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "write-test.log")

	cfg := &RotationConfig{
		Type:       RotationBySize,
		MaxSize:    1, // 1MB
		MaxBackups: 2,
		MaxAge:     1,
		Compress:   false,
	}

	writer, err := NewRotationWriter(cfg, outputPath)
	if err != nil {
		t.Fatalf("NewRotationWriter() error = %v", err)
	}

	// 写入测试数据
	testData := []byte("test log message\n")
	n, err := writer.Write(testData)
	if err != nil {
		t.Errorf("Write() error = %v", err)
	}

	if n != len(testData) {
		t.Errorf("Write() wrote %d bytes, expected %d", n, len(testData))
	}

	// 验证文件已创建
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("Log file was not created")
	}
}

// TestRotationConfigDefaults 测试轮换配置默认值
func TestRotationConfigDefaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Rotation.Type != RotationBySize {
		t.Errorf("Expected default RotationType=size, got %v", cfg.Rotation.Type)
	}

	if cfg.Rotation.MaxSize != 100 {
		t.Errorf("Expected default MaxSize=100, got %d", cfg.Rotation.MaxSize)
	}

	if cfg.Rotation.MaxBackups != 5 {
		t.Errorf("Expected default MaxBackups=5, got %d", cfg.Rotation.MaxBackups)
	}

	if cfg.Rotation.MaxAge != 7 {
		t.Errorf("Expected default MaxAge=7, got %d", cfg.Rotation.MaxAge)
	}

	if !cfg.Rotation.Compress {
		t.Error("Expected default Compress=true")
	}

	if cfg.Rotation.RotationTime != "24h" {
		t.Errorf("Expected default RotationTime=24h, got %s", cfg.Rotation.RotationTime)
	}

	if cfg.Rotation.MaxAgeTime != "168h" {
		t.Errorf("Expected default MaxAgeTime=168h, got %s", cfg.Rotation.MaxAgeTime)
	}

	if cfg.Rotation.RotationPattern != ".%Y%m%d" {
		t.Errorf("Expected default RotationPattern=.%%Y%%m%%d, got %s", cfg.Rotation.RotationPattern)
	}
}

// TestRotationType 测试轮换类型常量
func TestRotationType(t *testing.T) {
	if string(RotationBySize) != "size" {
		t.Errorf("Expected RotationBySize='size', got %s", string(RotationBySize))
	}

	if string(RotationByTime) != "time" {
		t.Errorf("Expected RotationByTime='time', got %s", string(RotationByTime))
	}
}

// TestRotationWriterCleanup 测试 writer 清理
func TestRotationWriterCleanup(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "cleanup-test.log")

	cfg := &RotationConfig{
		Type:       RotationBySize,
		MaxSize:    1,
		MaxBackups: 1,
		MaxAge:     1,
		Compress:   false,
	}

	writer, err := NewRotationWriter(cfg, outputPath)
	if err != nil {
		t.Fatalf("NewRotationWriter() error = %v", err)
	}

	// 写入一些数据
	writer.Write([]byte("test message\n"))

	// 对于 lumberjack，可以调用 Close 清理
	if lumberLogger, ok := writer.(*lumberjack.Logger); ok {
		if err := lumberLogger.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	}
}

// TestRotationWithDifferentFormats 测试不同格式的轮换模式
func TestRotationWithDifferentFormats(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "format-test.log")

	tests := []struct {
		name    string
		pattern string
	}{
		{
			name:    "daily pattern",
			pattern: ".%Y%m%d",
		},
		{
			name:    "hourly pattern",
			pattern: ".%Y%m%d%H",
		},
		{
			name:    "minute pattern",
			pattern: ".%Y%m%d%H%M",
		},
		{
			name:    "custom pattern",
			pattern: "-%Y-%m-%d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    "1h",
				MaxAgeTime:      "24h",
				RotationPattern: tt.pattern,
			}

			writer, err := newTimeRotationWriter(cfg, outputPath)
			if err != nil {
				t.Errorf("newTimeRotationWriter() error = %v", err)
				return
			}
			if writer == nil {
				t.Error("newTimeRotationWriter() returned nil")
			}
		})
	}
}

// TestRotationTimeIntervals 测试不同时间间隔
func TestRotationTimeIntervals(t *testing.T) {
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "interval-test.log")

	tests := []struct {
		name         string
		rotationTime string
		maxAgeTime   string
		wantErr      bool
	}{
		{
			name:         "1 hour rotation",
			rotationTime: "1h",
			maxAgeTime:   "24h",
			wantErr:      false,
		},
		{
			name:         "daily rotation",
			rotationTime: "24h",
			maxAgeTime:   "168h",
			wantErr:      false,
		},
		{
			name:         "weekly rotation",
			rotationTime: "168h",
			maxAgeTime:   "720h",
			wantErr:      false,
		},
		{
			name:         "minute rotation",
			rotationTime: "1m",
			maxAgeTime:   "1h",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RotationConfig{
				Type:            RotationByTime,
				RotationTime:    tt.rotationTime,
				MaxAgeTime:      tt.maxAgeTime,
				RotationPattern: ".%Y%m%d",
			}

			writer, err := newTimeRotationWriter(cfg, outputPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("newTimeRotationWriter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && writer == nil {
				t.Error("newTimeRotationWriter() returned nil")
			}
		})
	}
}
