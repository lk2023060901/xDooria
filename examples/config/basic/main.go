package main

import (
	"fmt"
	"log"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
)

// AppConfig 应用配置结构
type AppConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Database struct {
		Host     string        `yaml:"host"`
		Port     int           `yaml:"port"`
		User     string        `yaml:"user"`
		Password string        `yaml:"password"`
		DBName   string        `yaml:"dbname"`
		Timeout  time.Duration `yaml:"timeout"`
	} `yaml:"database"`

	Logger struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logger"`
}

func main() {
	fmt.Println("=== 示例 1：基本使用 ===")

	// 创建配置管理器
	mgr := config.NewManager()

	// 加载配置文件
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 解析整个配置到结构体
	var cfg AppConfig
	if err := mgr.Unmarshal(&cfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	// 打印配置信息
	fmt.Printf("【服务器配置】\n")
	fmt.Printf("  地址: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Println()

	fmt.Printf("【数据库配置】\n")
	fmt.Printf("  连接: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	fmt.Printf("  超时: %v\n", cfg.Database.Timeout)
	fmt.Println()

	fmt.Printf("【日志配置】\n")
	fmt.Printf("  级别: %s\n", cfg.Logger.Level)
	fmt.Printf("  格式: %s\n", cfg.Logger.Format)
	fmt.Println()

	// 使用 Get 方法直接获取配置值
	fmt.Printf("【直接获取配置值】\n")
	fmt.Printf("  server.port: %d\n", mgr.GetInt("server.port"))
	fmt.Printf("  server.host: %s\n", mgr.GetString("server.host"))
	fmt.Printf("  logger.level: %s\n", mgr.GetString("logger.level"))
	fmt.Println()

	// 检查配置项是否存在
	fmt.Printf("【配置项检查】\n")
	if mgr.IsSet("server.port") {
		fmt.Println("  ✓ server.port 已配置")
	}
	if !mgr.IsSet("nonexistent.key") {
		fmt.Println("  ✗ nonexistent.key 未配置")
	}
	fmt.Println()

	fmt.Println("✅ 示例完成")
}
