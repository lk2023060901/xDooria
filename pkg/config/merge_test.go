package config

import (
	"testing"
)

// 测试用的配置结构
type TestConfig struct {
	Server   ServerConfig            `json:"server"`
	Database DatabaseConfig          `json:"database"`
	Features map[string]bool         `json:"features"`
	Tags     []string                `json:"tags"`
	Metadata map[string]interface{}  `json:"metadata"`
	Extra    *ExtraConfig            `json:"extra"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
	TLS  bool   `json:"tls"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type ExtraConfig struct {
	Timeout int    `json:"timeout"`
	Retry   int    `json:"retry"`
	Name    string `json:"name"`
}

// TestMergeConfig_BasicTypes 测试基本类型合并
func TestMergeConfig_BasicTypes(t *testing.T) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			TLS:  false,
		},
	}

	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
			TLS:  true,
		},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	// Port 和 TLS 应该被覆盖
	if result.Server.Port != 9090 {
		t.Errorf("Expected Port=9090, got %d", result.Server.Port)
	}
	if result.Server.TLS != true {
		t.Errorf("Expected TLS=true, got %v", result.Server.TLS)
	}

	// Host 是零值，应该保留 dst 的值
	if result.Server.Host != "localhost" {
		t.Errorf("Expected Host=localhost, got %s", result.Server.Host)
	}
}

// TestMergeConfig_NestedStruct 测试嵌套结构体合并
func TestMergeConfig_NestedStruct(t *testing.T) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host:     "db.local",
			Port:     5432,
			Username: "admin",
			Password: "secret",
		},
	}

	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
		},
		Database: DatabaseConfig{
			Port:     3306,
			Password: "newpass",
		},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	// 验证 Server
	if result.Server.Host != "localhost" {
		t.Errorf("Expected Server.Host=localhost, got %s", result.Server.Host)
	}
	if result.Server.Port != 9090 {
		t.Errorf("Expected Server.Port=9090, got %d", result.Server.Port)
	}

	// 验证 Database
	if result.Database.Host != "db.local" {
		t.Errorf("Expected Database.Host=db.local, got %s", result.Database.Host)
	}
	if result.Database.Port != 3306 {
		t.Errorf("Expected Database.Port=3306, got %d", result.Database.Port)
	}
	if result.Database.Username != "admin" {
		t.Errorf("Expected Database.Username=admin, got %s", result.Database.Username)
	}
	if result.Database.Password != "newpass" {
		t.Errorf("Expected Database.Password=newpass, got %s", result.Database.Password)
	}
}

// TestMergeConfig_Map 测试 map 合并
func TestMergeConfig_Map(t *testing.T) {
	dst := &TestConfig{
		Features: map[string]bool{
			"auth":  true,
			"cache": false,
		},
		Metadata: map[string]interface{}{
			"version": "1.0",
			"env":     "dev",
		},
	}

	src := &TestConfig{
		Features: map[string]bool{
			"cache": true,
			"log":   true,
		},
		Metadata: map[string]interface{}{
			"env":    "prod",
			"region": "us-west",
		},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	// 验证 Features map
	if result.Features["auth"] != true {
		t.Errorf("Expected Features[auth]=true, got %v", result.Features["auth"])
	}
	if result.Features["cache"] != true {
		t.Errorf("Expected Features[cache]=true, got %v", result.Features["cache"])
	}
	if result.Features["log"] != true {
		t.Errorf("Expected Features[log]=true, got %v", result.Features["log"])
	}

	// 验证 Metadata map
	if result.Metadata["version"] != "1.0" {
		t.Errorf("Expected Metadata[version]=1.0, got %v", result.Metadata["version"])
	}
	if result.Metadata["env"] != "prod" {
		t.Errorf("Expected Metadata[env]=prod, got %v", result.Metadata["env"])
	}
	if result.Metadata["region"] != "us-west" {
		t.Errorf("Expected Metadata[region]=us-west, got %v", result.Metadata["region"])
	}
}

// TestMergeConfig_Slice 测试切片合并（整体覆盖）
func TestMergeConfig_Slice(t *testing.T) {
	dst := &TestConfig{
		Tags: []string{"v1", "stable"},
	}

	src := &TestConfig{
		Tags: []string{"v2", "beta"},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	// 切片应该被整体覆盖
	if len(result.Tags) != 2 {
		t.Errorf("Expected Tags length=2, got %d", len(result.Tags))
	}
	if result.Tags[0] != "v2" || result.Tags[1] != "beta" {
		t.Errorf("Expected Tags=[v2, beta], got %v", result.Tags)
	}
}

// TestMergeConfig_Pointer 测试指针合并
func TestMergeConfig_Pointer(t *testing.T) {
	dst := &TestConfig{
		Extra: &ExtraConfig{
			Timeout: 30,
			Retry:   3,
			Name:    "service-a",
		},
	}

	src := &TestConfig{
		Extra: &ExtraConfig{
			Timeout: 60,
			Name:    "service-b",
		},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if result.Extra == nil {
		t.Fatal("Expected Extra to be non-nil")
	}
	if result.Extra.Timeout != 60 {
		t.Errorf("Expected Extra.Timeout=60, got %d", result.Extra.Timeout)
	}
	if result.Extra.Retry != 3 {
		t.Errorf("Expected Extra.Retry=3, got %d", result.Extra.Retry)
	}
	if result.Extra.Name != "service-b" {
		t.Errorf("Expected Extra.Name=service-b, got %s", result.Extra.Name)
	}
}

// TestMergeConfig_NilPointer 测试 nil 指针合并
func TestMergeConfig_NilPointer(t *testing.T) {
	dst := &TestConfig{
		Extra: nil,
	}

	src := &TestConfig{
		Extra: &ExtraConfig{
			Timeout: 60,
			Retry:   5,
		},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if result.Extra == nil {
		t.Fatal("Expected Extra to be non-nil")
	}
	if result.Extra.Timeout != 60 {
		t.Errorf("Expected Extra.Timeout=60, got %d", result.Extra.Timeout)
	}
}

// TestMergeConfig_NilSrc 测试 src 为 nil
func TestMergeConfig_NilSrc(t *testing.T) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	result, err := MergeConfig(dst, nil)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if result.Server.Host != "localhost" || result.Server.Port != 8080 {
		t.Errorf("Expected dst to be unchanged")
	}
}

// TestMergeConfig_NilDst 测试 dst 为 nil（应该返回 src）
func TestMergeConfig_NilDst(t *testing.T) {
	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
		},
	}

	result, err := MergeConfig[TestConfig](nil, src)
	if err != nil {
		t.Fatalf("MergeConfig should not fail when dst is nil: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}

	if result.Server.Port != 9090 {
		t.Errorf("Expected Port=9090, got %d", result.Server.Port)
	}
}

// TestMergeConfig_BothNil 测试 dst 和 src 都为 nil（应该报错）
func TestMergeConfig_BothNil(t *testing.T) {
	_, err := MergeConfig[TestConfig](nil, nil)
	if err == nil {
		t.Fatal("Expected error when both dst and src are nil")
	}
}

// TestMergeConfig_EmptyConfig 测试空配置合并
func TestMergeConfig_EmptyConfig(t *testing.T) {
	dst := &TestConfig{}
	src := &TestConfig{}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	if result == nil {
		t.Fatal("Expected result to be non-nil")
	}
}

// TestMergeConfig_ComplexScenario 测试复杂场景
func TestMergeConfig_ComplexScenario(t *testing.T) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			TLS:  false,
		},
		Database: DatabaseConfig{
			Host:     "db.local",
			Port:     5432,
			Username: "admin",
		},
		Features: map[string]bool{
			"auth":  true,
			"cache": false,
		},
		Tags: []string{"v1"},
		Extra: &ExtraConfig{
			Timeout: 30,
			Retry:   3,
		},
	}

	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
			TLS:  true,
		},
		Database: DatabaseConfig{
			Password: "secret",
		},
		Features: map[string]bool{
			"cache": true,
			"log":   true,
		},
		Tags: []string{"v2", "beta"},
		Extra: &ExtraConfig{
			Name: "service",
		},
	}

	result, err := MergeConfig(dst, src)
	if err != nil {
		t.Fatalf("MergeConfig failed: %v", err)
	}

	// 验证所有字段
	if result.Server.Host != "localhost" {
		t.Errorf("Server.Host should remain localhost")
	}
	if result.Server.Port != 9090 {
		t.Errorf("Server.Port should be 9090")
	}
	if result.Server.TLS != true {
		t.Errorf("Server.TLS should be true")
	}

	if result.Database.Host != "db.local" {
		t.Errorf("Database.Host should remain db.local")
	}
	if result.Database.Username != "admin" {
		t.Errorf("Database.Username should remain admin")
	}
	if result.Database.Password != "secret" {
		t.Errorf("Database.Password should be secret")
	}

	if len(result.Features) != 3 {
		t.Errorf("Features should have 3 entries")
	}

	if len(result.Tags) != 2 {
		t.Errorf("Tags should be replaced with [v2, beta]")
	}

	if result.Extra.Timeout != 30 {
		t.Errorf("Extra.Timeout should remain 30")
	}
	if result.Extra.Name != "service" {
		t.Errorf("Extra.Name should be service")
	}
}

// TestMergeConfigJSON_BasicTypes 测试 JSON 方式合并
func TestMergeConfigJSON_BasicTypes(t *testing.T) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
	}

	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
		},
	}

	result, err := MergeConfigJSON(dst, src)
	if err != nil {
		t.Fatalf("MergeConfigJSON failed: %v", err)
	}

	if result.Server.Port != 9090 {
		t.Errorf("Expected Port=9090, got %d", result.Server.Port)
	}

	// 注意：JSON 方式会将零值覆盖
	if result.Server.Host != "" {
		t.Logf("JSON merge: Host=%s (zero value from src)", result.Server.Host)
	}
}

// TestMergeConfigJSON_NestedMap 测试 JSON 方式合并嵌套 map
func TestMergeConfigJSON_NestedMap(t *testing.T) {
	dst := &TestConfig{
		Metadata: map[string]interface{}{
			"version": "1.0",
			"config": map[string]interface{}{
				"timeout": 30,
				"retry":   3,
			},
		},
	}

	src := &TestConfig{
		Metadata: map[string]interface{}{
			"env": "prod",
			"config": map[string]interface{}{
				"timeout": 60,
			},
		},
	}

	result, err := MergeConfigJSON(dst, src)
	if err != nil {
		t.Fatalf("MergeConfigJSON failed: %v", err)
	}

	if result.Metadata["version"] != "1.0" {
		t.Errorf("Expected version=1.0")
	}
	if result.Metadata["env"] != "prod" {
		t.Errorf("Expected env=prod")
	}

	config := result.Metadata["config"].(map[string]interface{})
	if config["timeout"].(float64) != 60 {
		t.Errorf("Expected timeout=60")
	}
	if config["retry"].(float64) != 3 {
		t.Errorf("Expected retry=3")
	}
}

// BenchmarkMergeConfig 性能测试：反射方式
func BenchmarkMergeConfig(b *testing.B) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host: "db.local",
			Port: 5432,
		},
		Features: map[string]bool{
			"auth": true,
		},
	}

	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
		},
		Features: map[string]bool{
			"cache": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MergeConfig(dst, src)
	}
}

// BenchmarkMergeConfigJSON 性能测试：JSON 方式
func BenchmarkMergeConfigJSON(b *testing.B) {
	dst := &TestConfig{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Host: "db.local",
			Port: 5432,
		},
		Features: map[string]bool{
			"auth": true,
		},
	}

	src := &TestConfig{
		Server: ServerConfig{
			Port: 9090,
		},
		Features: map[string]bool{
			"cache": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = MergeConfigJSON(dst, src)
	}
}
