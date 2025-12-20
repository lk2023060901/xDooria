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
		Port int    `yaml:"port" validate:"required,min=1,max=65535"`
		Host string `yaml:"host" validate:"required"`
	} `yaml:"server"`

	Database struct {
		Host     string        `yaml:"host" validate:"required"`
		Port     int           `yaml:"port" validate:"required,min=1,max=65535"`
		User     string        `yaml:"user" validate:"required"`
		Password string        `yaml:"password" validate:"required"`
		DBName   string        `yaml:"dbname" validate:"required"`
		Timeout  time.Duration `yaml:"timeout"`
	} `yaml:"database"`

	Logger struct {
		Level  string `yaml:"level" validate:"required,oneof=debug info warn error"`
		Format string `yaml:"format" validate:"oneof=json text"`
	} `yaml:"logger"`

	Feature struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"feature"`
}

func main() {
	fmt.Println("=== Config Manager 使用示例 ===\n")

	// 示例 1: 基本使用
	example1()

	// 示例 2: UnmarshalKey
	example2()

	// 示例 3: 环境变量
	example3()

	// 示例 4: 配置验证
	example4()

	// 示例 5: 配置热重载
	example5()
}

// example1 基本使用
func example1() {
	fmt.Println("【示例 1】基本使用 - 加载和解析配置")

	mgr := config.NewManager()

	// 加载配置文件
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}

	// 解析整个配置
	var cfg AppConfig
	if err := mgr.Unmarshal(&cfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	fmt.Printf("服务器配置: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("数据库配置: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)
	fmt.Printf("日志级别: %s, 格式: %s\n", cfg.Logger.Level, cfg.Logger.Format)
	fmt.Println()
}

// example2 UnmarshalKey 解析部分配置
func example2() {
	fmt.Println("【示例 2】UnmarshalKey - 解析部分配置")

	mgr := config.NewManager()
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}

	// 解析 Server 配置到结构体
	var serverCfg struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	}
	if err := mgr.UnmarshalKey("server", &serverCfg); err != nil {
		log.Fatalf("解析 server 配置失败: %v", err)
	}
	fmt.Printf("Server: %+v\n", serverCfg)

	// 解析单个字段到基本类型
	var port int
	mgr.UnmarshalKey("server.port", &port)
	fmt.Printf("Port: %d\n", port)

	var host string
	mgr.UnmarshalKey("server.host", &host)
	fmt.Printf("Host: %s\n", host)

	var enabled bool
	mgr.UnmarshalKey("feature.enabled", &enabled)
	fmt.Printf("Feature enabled: %v\n", enabled)
	fmt.Println()
}

// example3 环境变量
func example3() {
	fmt.Println("【示例 3】环境变量 - 覆盖配置文件")

	mgr := config.NewManager()

	// 绑定环境变量（前缀 EXAMPLE_）
	mgr.BindEnv("EXAMPLE")

	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}

	// 环境变量 EXAMPLE_SERVER_PORT 会覆盖配置文件中的 server.port
	port := mgr.GetInt("server.port")
	host := mgr.GetString("server.host")

	fmt.Printf("Server Port (可能被环境变量覆盖): %d\n", port)
	fmt.Printf("Server Host: %s\n", host)

	// 检查配置项是否存在
	if mgr.IsSet("server.port") {
		fmt.Println("✓ server.port 已配置")
	}
	if !mgr.IsSet("nonexistent.key") {
		fmt.Println("✗ nonexistent.key 未配置")
	}
	fmt.Println()
}

// example4 配置验证
func example4() {
	fmt.Println("【示例 4】配置验证")

	mgr := config.NewManager()
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}

	var cfg AppConfig
	if err := mgr.Unmarshal(&cfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	// 验证配置
	validator := config.NewValidator()
	if err := validator.Validate(cfg); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
	} else {
		fmt.Println("✅ 配置验证通过")
	}

	// 单字段验证
	if err := validator.ValidateField(cfg.Server.Port, "min=1,max=65535"); err != nil {
		fmt.Printf("❌ Port 验证失败: %v\n", err)
	} else {
		fmt.Println("✅ Port 验证通过")
	}
	fmt.Println()
}

// example5 配置热重载
func example5() {
	fmt.Println("【示例 5】配置热重载")

	mgr := config.NewManager()
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v\n", err)
		return
	}

	// 注册回调函数
	reloadCount := 0
	mgr.Watch(func() {
		reloadCount++
		fmt.Printf("🔄 配置已重载（第 %d 次）\n", reloadCount)

		// 重新读取配置
		var cfg AppConfig
		if err := mgr.Unmarshal(&cfg); err != nil {
			log.Printf("重载配置失败: %v", err)
			return
		}

		fmt.Printf("新的服务器端口: %d\n", cfg.Server.Port)
	})

	fmt.Println("✅ 热重载监听已启动")
	fmt.Println("💡 修改 config.yaml 文件后将自动触发重载")
	fmt.Println("⚠️  本示例不会持续运行，生产环境需要保持进程运行")
	fmt.Println()
}
