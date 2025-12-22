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

// AppConfig 应用配置
type AppConfig struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
}

type ServerConfig struct {
	Name  string `mapstructure:"name"`
	Port  int    `mapstructure:"port"`
	Debug bool   `mapstructure:"debug"`
}

type DatabaseConfig struct {
	Host           string `mapstructure:"host"`
	Port           int    `mapstructure:"port"`
	MaxConnections int    `mapstructure:"max_connections"`
}

func main() {
	fmt.Println("=== Config Watcher 示例 ===")

	// 创建配置监听器（支持 yaml、json、toml）
	watcher, err := config.NewWatcher[AppConfig]("config.yaml")
	if err != nil {
		log.Fatalf("创建监听器失败: %v", err)
	}

	// 获取初始配置
	cfg := watcher.GetConfig()
	printConfig(cfg)

	// 注册配置变化回调
	watcher.OnChange(func(newCfg *AppConfig) {
		fmt.Println("\n🔄 配置文件已变化！")
		printConfig(newCfg)
		fmt.Println("\n💡 提示：继续修改 config.yaml 文件来测试热更新")
	})

	fmt.Println("\n💡 说明：")
	fmt.Println("1. 程序正在监听 config.yaml 文件的变化")
	fmt.Println("2. 请尝试修改 config.yaml 文件（例如修改端口号）")
	fmt.Println("3. 保存文件后，程序会自动检测并重新加载配置")
	fmt.Println("4. 按 Ctrl+C 退出程序")

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 模拟应用运行
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 定期使用最新配置
			currentCfg := watcher.GetConfig()
			fmt.Printf("⏰ [%s] 应用运行中... 当前端口: %d\n",
				time.Now().Format("15:04:05"),
				currentCfg.Server.Port,
			)
		case <-quit:
			fmt.Println("\n👋 程序退出")
			return
		}
	}
}

// printConfig 打印配置信息
func printConfig(cfg *AppConfig) {
	fmt.Println("📋 当前配置:")
	fmt.Printf("  服务名称: %s\n", cfg.Server.Name)
	fmt.Printf("  服务端口: %d\n", cfg.Server.Port)
	fmt.Printf("  调试模式: %v\n", cfg.Server.Debug)
	fmt.Printf("  数据库: %s:%d (最大连接数: %d)\n",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.MaxConnections,
	)
}
