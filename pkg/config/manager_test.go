package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ManagerTestConfig 测试配置结构
type ManagerTestConfig struct {
	Server struct {
		Port int    `yaml:"port" validate:"required,min=1,max=65535"`
		Host string `yaml:"host" validate:"required"`
	} `yaml:"server"`
	Database struct {
		Host     string        `yaml:"host"`
		Port     int           `yaml:"port"`
		User     string        `yaml:"user"`
		Password string        `yaml:"password"`
		Timeout  time.Duration `yaml:"timeout"`
	} `yaml:"database"`
	Feature struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"feature"`
}

// createTestConfigFile 创建测试配置文件
func createTestConfigFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return configPath
}

// TestManagerLoadFile 测试加载配置文件
func TestManagerLoadFile(t *testing.T) {
	configContent := `
server:
  port: 8080
  host: "localhost"
database:
  host: "db.example.com"
  port: 5432
  user: "testuser"
  password: "testpass"
  timeout: 30s
feature:
  enabled: true
`

	configPath := createTestConfigFile(t, configContent)

	mgr := NewManager()
	if err := mgr.LoadFile(configPath); err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// 测试 Unmarshal
	var cfg ManagerTestConfig
	if err := mgr.Unmarshal(&cfg); err != nil {
		t.Fatalf("Failed to unmarshal config: %v", err)
	}

	// 验证配置值
	if cfg.Server.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", cfg.Server.Port)
	}
	if cfg.Server.Host != "localhost" {
		t.Errorf("Expected host localhost, got %s", cfg.Server.Host)
	}
	if cfg.Database.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", cfg.Database.Timeout)
	}
	if !cfg.Feature.Enabled {
		t.Error("Expected feature.enabled to be true")
	}
}

// TestManagerUnmarshalKey 测试解析指定 key
func TestManagerUnmarshalKey(t *testing.T) {
	configContent := `
server:
  port: 8080
  host: "localhost"
database:
  host: "db.example.com"
  port: 5432
`

	configPath := createTestConfigFile(t, configContent)

	mgr := NewManager()
	if err := mgr.LoadFile(configPath); err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// 测试解析 struct
	var serverCfg struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	}
	if err := mgr.UnmarshalKey("server", &serverCfg); err != nil {
		t.Fatalf("Failed to unmarshal server config: %v", err)
	}
	if serverCfg.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", serverCfg.Port)
	}

	// 测试解析基本类型
	var port int
	if err := mgr.UnmarshalKey("server.port", &port); err != nil {
		t.Fatalf("Failed to unmarshal server.port: %v", err)
	}
	if port != 8080 {
		t.Errorf("Expected port 8080, got %d", port)
	}

	var host string
	if err := mgr.UnmarshalKey("server.host", &host); err != nil {
		t.Fatalf("Failed to unmarshal server.host: %v", err)
	}
	if host != "localhost" {
		t.Errorf("Expected host localhost, got %s", host)
	}
}

// TestManagerGet 测试获取配置值
func TestManagerGet(t *testing.T) {
	configContent := `
server:
  port: 8080
  host: "localhost"
feature:
  enabled: true
`

	configPath := createTestConfigFile(t, configContent)

	mgr := NewManager()
	if err := mgr.LoadFile(configPath); err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// 测试 GetInt
	port := mgr.GetInt("server.port")
	if port != 8080 {
		t.Errorf("Expected port 8080, got %d", port)
	}

	// 测试 GetString
	host := mgr.GetString("server.host")
	if host != "localhost" {
		t.Errorf("Expected host localhost, got %s", host)
	}

	// 测试 GetBool
	enabled := mgr.GetBool("feature.enabled")
	if !enabled {
		t.Error("Expected feature.enabled to be true")
	}

	// 测试 IsSet
	if !mgr.IsSet("server.port") {
		t.Error("Expected server.port to be set")
	}
	if mgr.IsSet("nonexistent.key") {
		t.Error("Expected nonexistent.key to not be set")
	}
}

// TestManagerBindEnv 测试环境变量绑定
func TestManagerBindEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("TEST_SERVER_PORT", "9000")
	os.Setenv("TEST_SERVER_HOST", "0.0.0.0")
	defer os.Unsetenv("TEST_SERVER_PORT")
	defer os.Unsetenv("TEST_SERVER_HOST")

	configContent := `
server:
  port: 8080
  host: "localhost"
`

	configPath := createTestConfigFile(t, configContent)

	mgr := NewManager()
	mgr.BindEnv("TEST")
	if err := mgr.LoadFile(configPath); err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// 环境变量应该覆盖配置文件
	port := mgr.GetInt("server.port")
	if port != 9000 {
		t.Errorf("Expected port 9000 from env, got %d", port)
	}

	host := mgr.GetString("server.host")
	if host != "0.0.0.0" {
		t.Errorf("Expected host 0.0.0.0 from env, got %s", host)
	}
}

// TestManagerWithDefaults 测试默认值
func TestManagerWithDefaults(t *testing.T) {
	defaults := map[string]any{
		"server.port": 3000,
		"server.host": "127.0.0.1",
	}

	mgr := NewManager(WithDefaults(defaults))

	// 测试默认值
	port := mgr.GetInt("server.port")
	if port != 3000 {
		t.Errorf("Expected default port 3000, got %d", port)
	}

	host := mgr.GetString("server.host")
	if host != "127.0.0.1" {
		t.Errorf("Expected default host 127.0.0.1, got %s", host)
	}
}

// TestManagerAllSettings 测试获取所有配置
func TestManagerAllSettings(t *testing.T) {
	configContent := `
server:
  port: 8080
  host: "localhost"
`

	configPath := createTestConfigFile(t, configContent)

	mgr := NewManager()
	if err := mgr.LoadFile(configPath); err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	allSettings := mgr.AllSettings()
	if len(allSettings) == 0 {
		t.Error("Expected non-empty settings map")
	}

	serverSettings, ok := allSettings["server"].(map[string]any)
	if !ok {
		t.Fatal("Expected server settings to be a map")
	}

	if serverSettings["port"] != 8080 {
		t.Errorf("Expected port 8080, got %v", serverSettings["port"])
	}
}
