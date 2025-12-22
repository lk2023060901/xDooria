package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lk2023060901/xdooria/pkg/config"
)

type AppConfig struct {
	Server struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	} `yaml:"server"`

	Logger struct {
		Level string `yaml:"level"`
	} `yaml:"logger"`
}

func main() {
	fmt.Println("=== 示例 5：配置热重载 ===")

	mgr := config.NewManager()

	// 加载初始配置
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	var cfg AppConfig
	mgr.Unmarshal(&cfg)

	fmt.Printf("【初始配置】\n")
	fmt.Printf("  服务器: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  日志级别: %s\n", cfg.Logger.Level)
	fmt.Println()

	// 注册热重载回调
	reloadCount := 0
	mgr.Watch(func() {
		reloadCount++
		fmt.Printf("\n🔄 配置文件已变更（第 %d 次重载）\n", reloadCount)
		fmt.Printf("   时间: %s\n", time.Now().Format("15:04:05"))

		// 重新读取配置
		var newCfg AppConfig
		if err := mgr.Unmarshal(&newCfg); err != nil {
			log.Printf("   ❌ 重载配置失败: %v", err)
			return
		}

		fmt.Printf("【新配置】\n")
		fmt.Printf("  服务器: %s:%d\n", newCfg.Server.Host, newCfg.Server.Port)
		fmt.Printf("  日志级别: %s\n", newCfg.Logger.Level)

		// 实际应用中，这里可以：
		// - 重新初始化日志级别
		// - 更新服务器配置（如果支持动态更新）
		// - 重新连接数据库（如果连接参数改变）
		// - 触发其他模块的配置更新

		cfg = newCfg
	})

	fmt.Println("✅ 配置热重载监听已启动")
	fmt.Println("💡 测试步骤：")
	fmt.Println("  1. 打开另一个终端")
	fmt.Println("  2. 修改 config.yaml 文件")
	fmt.Println("     例如: sed -i '' 's/port: 8080/port: 9090/' config.yaml")
	fmt.Println("     或者: echo 'server:\\n  port: 9090\\n  host: \"0.0.0.0\"\\nlogger:\\n  level: \"debug\"' > config.yaml")
	fmt.Println("  3. 观察此终端的输出")
	fmt.Println("  4. 按 Ctrl+C 退出程序")

	// 定期打印当前配置（方便观察）
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			fmt.Printf("\n📊 当前配置状态（%s）\n", time.Now().Format("15:04:05"))
			fmt.Printf("  服务器: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
			fmt.Printf("  日志级别: %s\n", cfg.Logger.Level)
			fmt.Printf("  重载次数: %d\n", reloadCount)
		}
	}()

	// 等待信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("⏳ 程序运行中，等待配置文件变更...")
	<-sigCh

	fmt.Println("\n\n👋 程序已退出")
	fmt.Printf("   总共重载次数: %d\n", reloadCount)
}
