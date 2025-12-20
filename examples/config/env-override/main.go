package main

import (
	"fmt"
	"log"
	"os"

	"github.com/lk2023060901/xdooria/pkg/config"
)

func main() {
	fmt.Println("=== 示例 3：环境变量覆盖配置 ===\n")

	// 设置环境变量进行演示
	fmt.Println("【设置环境变量】")
	os.Setenv("MYAPP_SERVER_PORT", "9000")
	os.Setenv("MYAPP_SERVER_HOST", "127.0.0.1")
	os.Setenv("MYAPP_DATABASE_HOST", "prod-db.example.com")
	fmt.Println("  export MYAPP_SERVER_PORT=9000")
	fmt.Println("  export MYAPP_SERVER_HOST=127.0.0.1")
	fmt.Println("  export MYAPP_DATABASE_HOST=prod-db.example.com")
	fmt.Println()

	// 创建配置管理器并绑定环境变量
	mgr := config.NewManager()

	// 绑定环境变量（前缀 MYAPP_）
	// 环境变量 MYAPP_SERVER_PORT 会映射到 server.port
	// 环境变量 MYAPP_DATABASE_HOST 会映射到 database.host
	mgr.BindEnv("MYAPP")

	// 加载配置文件
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 获取配置值（环境变量会覆盖配置文件）
	fmt.Println("【配置文件中的值】")
	fmt.Println("  server.port: 8080")
	fmt.Println("  server.host: 0.0.0.0")
	fmt.Println("  database.host: localhost")
	fmt.Println()

	fmt.Println("【实际生效的值（环境变量已覆盖）】")
	serverPort := mgr.GetInt("server.port")
	serverHost := mgr.GetString("server.host")
	dbHost := mgr.GetString("database.host")
	dbPort := mgr.GetInt("database.port") // 未设置环境变量，使用配置文件

	fmt.Printf("  server.port: %d (来自环境变量 MYAPP_SERVER_PORT)\n", serverPort)
	fmt.Printf("  server.host: %s (来自环境变量 MYAPP_SERVER_HOST)\n", serverHost)
	fmt.Printf("  database.host: %s (来自环境变量 MYAPP_DATABASE_HOST)\n", dbHost)
	fmt.Printf("  database.port: %d (来自配置文件)\n", dbPort)
	fmt.Println()

	// 验证优先级
	fmt.Println("【配置优先级】")
	if serverPort == 9000 {
		fmt.Println("  ✓ 环境变量优先级高于配置文件")
	}
	if dbPort == 5432 {
		fmt.Println("  ✓ 未设置环境变量时使用配置文件值")
	}
	fmt.Println()

	// 清理环境变量
	os.Unsetenv("MYAPP_SERVER_PORT")
	os.Unsetenv("MYAPP_SERVER_HOST")
	os.Unsetenv("MYAPP_DATABASE_HOST")

	fmt.Println("✅ 示例完成")
	fmt.Println("\n💡 环境变量映射规则：")
	fmt.Println("  - 前缀: MYAPP_")
	fmt.Println("  - 分隔符: . → _")
	fmt.Println("  - 示例: server.port → MYAPP_SERVER_PORT")
	fmt.Println("  - 示例: database.postgres.host → MYAPP_DATABASE_POSTGRES_HOST")
	fmt.Println("\n💡 配置优先级（从高到低）：")
	fmt.Println("  1. 命令行参数")
	fmt.Println("  2. 环境变量")
	fmt.Println("  3. 配置文件")
	fmt.Println("  4. 默认值")
}
