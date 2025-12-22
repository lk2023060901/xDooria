package main

import (
	"fmt"

	"github.com/lk2023060901/xdooria/pkg/config"
)

type ServerConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
	TLS  bool   `json:"tls"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
}

type AppConfig struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
}

func main() {
	fmt.Println("=== 示例 6：配置合并 ===")

	// 1. 基本合并：src 覆盖 dst
	fmt.Println("【1. 基本合并】")
	testBasicMerge()

	// 2. 部分覆盖
	fmt.Println("\n【2. 部分覆盖】")
	testPartialMerge()

	// 3. 嵌套结构合并
	fmt.Println("\n【3. 嵌套结构合并】")
	testNestedMerge()

	// 4. Nil 处理
	fmt.Println("\n【4. Nil 处理】")
	testNilHandling()

	fmt.Println("\n✅ 示例完成")
}

func testBasicMerge() {
	dst := &ServerConfig{
		Port: 8080,
		Host: "localhost",
		TLS:  false,
	}

	src := &ServerConfig{
		Port: 9090,
		TLS:  true,
	}

	fmt.Printf("  合并前 (dst): %+v\n", dst)
	fmt.Printf("  合并源 (src): %+v\n", src)

	result, err := config.MergeConfig(dst, src)
	if err != nil {
		fmt.Printf("  ❌ 合并失败: %v\n", err)
		return
	}

	fmt.Printf("  合并后: %+v\n", result)
	fmt.Println("  说明: src.Port 覆盖了 dst.Port, src.TLS 覆盖了 dst.TLS")
	fmt.Println("       dst.Host 保留（src.Host 为零值）")
}

func testPartialMerge() {
	dst := &DatabaseConfig{
		Host:     "localhost",
		Port:     5432,
		User:     "admin",
		Password: "oldpass",
	}

	src := &DatabaseConfig{
		Host:     "prod-db.example.com",
		Password: "newpass",
	}

	fmt.Printf("  合并前 (dst): %+v\n", dst)
	fmt.Printf("  合并源 (src): %+v\n", src)

	result, err := config.MergeConfig(dst, src)
	if err != nil {
		fmt.Printf("  ❌ 合并失败: %v\n", err)
		return
	}

	fmt.Printf("  合并后: %+v\n", result)
	fmt.Println("  说明: Host 和 Password 被覆盖，Port 和 User 保留")
}

func testNestedMerge() {
	dst := &AppConfig{
		Server: ServerConfig{
			Port: 8080,
			Host: "localhost",
			TLS:  false,
		},
		Database: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "admin",
			Password: "pass",
		},
	}

	src := &AppConfig{
		Server: ServerConfig{
			Port: 9090,
			TLS:  true,
		},
		Database: DatabaseConfig{
			Host: "prod-db.example.com",
		},
	}

	fmt.Printf("  合并前 (dst):\n")
	fmt.Printf("    Server: %+v\n", dst.Server)
	fmt.Printf("    Database: %+v\n", dst.Database)

	fmt.Printf("  合并源 (src):\n")
	fmt.Printf("    Server: %+v\n", src.Server)
	fmt.Printf("    Database: %+v\n", src.Database)

	result, err := config.MergeConfig(dst, src)
	if err != nil {
		fmt.Printf("  ❌ 合并失败: %v\n", err)
		return
	}

	fmt.Printf("  合并后:\n")
	fmt.Printf("    Server: %+v\n", result.Server)
	fmt.Printf("    Database: %+v\n", result.Database)
	fmt.Println("  说明: 嵌套结构递归合并，每个字段独立处理")
}

func testNilHandling() {
	cfg := &ServerConfig{
		Port: 8080,
		Host: "localhost",
		TLS:  true,
	}

	fmt.Printf("  原配置: %+v\n", cfg)

	// src 为 nil，应该返回 dst
	result1, err := config.MergeConfig(cfg, nil)
	if err != nil {
		fmt.Printf("  ❌ 合并失败: %v\n", err)
	} else {
		fmt.Printf("  MergeConfig(cfg, nil) = %+v\n", result1)
	}

	// dst 为 nil，应该返回 src
	result2, err := config.MergeConfig(nil, cfg)
	if err != nil {
		fmt.Printf("  ❌ 合并失败: %v\n", err)
	} else {
		fmt.Printf("  MergeConfig(nil, cfg) = %+v\n", result2)
	}

	// 两者都为 nil，应该报错
	result3, err := config.MergeConfig[ServerConfig](nil, nil)
	if err != nil {
		fmt.Printf("  MergeConfig(nil, nil) = ❌ 错误: %v (符合预期)\n", err)
	} else {
		fmt.Printf("  MergeConfig(nil, nil) = %+v (不应该成功)\n", result3)
	}
}
