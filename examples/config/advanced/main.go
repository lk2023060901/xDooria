package main

import (
	"fmt"
	"log"

	"github.com/lk2023060901/xdooria/pkg/config"
)

func main() {
	fmt.Println("=== 示例 7：高级用法 ===\n")

	// 1. 使用默认值
	fmt.Println("【1. 使用默认值】")
	testWithDefaults()

	// 2. 多路径搜索
	fmt.Println("\n【2. 配置文件搜索路径】")
	testConfigPaths()

	// 3. 获取所有配置
	fmt.Println("\n【3. 获取所有配置】")
	testAllSettings()

	// 4. 组合使用多个选项
	fmt.Println("\n【4. 组合使用多个选项】")
	testCombinedOptions()

	fmt.Println("\n✅ 示例完成")
}

func testWithDefaults() {
	// 设置默认值
	defaults := map[string]any{
		"server.port":         8080,
		"server.host":         "localhost",
		"database.pool.max":   100,
		"database.pool.min":   10,
		"logger.level":        "info",
		"feature.cache":       true,
		"feature.compression": false,
	}

	mgr := config.NewManager(config.WithDefaults(defaults))

	// 即使没有加载配置文件，也能获取默认值
	fmt.Printf("  server.port (默认值): %d\n", mgr.GetInt("server.port"))
	fmt.Printf("  server.host (默认值): %s\n", mgr.GetString("server.host"))
	fmt.Printf("  logger.level (默认值): %s\n", mgr.GetString("logger.level"))
	fmt.Printf("  feature.cache (默认值): %v\n", mgr.GetBool("feature.cache"))

	// 加载配置文件后，文件值会覆盖默认值
	if err := mgr.LoadFile("config.yaml"); err == nil {
		fmt.Printf("\n  加载配置文件后:\n")
		fmt.Printf("  server.port (配置文件): %d\n", mgr.GetInt("server.port"))
		fmt.Printf("  logger.level (配置文件): %s\n", mgr.GetString("logger.level"))
		fmt.Printf("  database.pool.max (默认值，文件中未配置): %d\n", mgr.GetInt("database.pool.max"))
	}
}

func testConfigPaths() {
	mgr := config.NewManager(
		config.WithConfigName("app"),      // 配置文件名（不含扩展名）
		config.WithConfigType("yaml"),     // 配置文件类型
		config.WithConfigPaths(            // 搜索路径（按顺序）
			".",                    // 当前目录
			"./config",            // config 子目录
			"/etc/myapp",          // 系统配置目录
			"$HOME/.myapp",        // 用户主目录
		),
	)

	fmt.Println("  配置文件搜索路径：")
	fmt.Println("    1. ./app.yaml")
	fmt.Println("    2. ./config/app.yaml")
	fmt.Println("    3. /etc/myapp/app.yaml")
	fmt.Println("    4. $HOME/.myapp/app.yaml")
	fmt.Println("\n  说明: Manager 会按顺序搜索，找到第一个存在的文件")

	// 注意：在实际使用中，不需要调用 LoadFile
	// Manager 会自动搜索配置文件
	_ = mgr
}

func testAllSettings() {
	mgr := config.NewManager()
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v", err)
		return
	}

	// 获取所有配置（以 map 形式）
	allSettings := mgr.AllSettings()

	fmt.Println("  所有配置项：")
	for key, value := range allSettings {
		fmt.Printf("    %s: %v\n", key, value)
	}

	fmt.Println("\n  说明: AllSettings 返回整个配置的 map 表示")
	fmt.Println("       可用于配置导出、备份或调试")
}

func testCombinedOptions() {
	// 组合使用多个选项
	mgr := config.NewManager(
		// 设置默认值
		config.WithDefaults(map[string]any{
			"server.port": 8080,
			"server.host": "localhost",
		}),
		// 设置环境变量前缀
		config.WithEnvPrefix("MYAPP"),
		// 设置配置文件类型
		config.WithConfigType("yaml"),
	)

	// 加载配置文件
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Printf("加载配置文件失败: %v", err)
		return
	}

	fmt.Println("  配置优先级（从高到低）：")
	fmt.Println("    1. 环境变量 (MYAPP_SERVER_PORT)")
	fmt.Println("    2. 配置文件 (config.yaml)")
	fmt.Println("    3. 默认值 (WithDefaults)")
	fmt.Println()

	port := mgr.GetInt("server.port")
	host := mgr.GetString("server.host")

	fmt.Printf("  实际生效值:\n")
	fmt.Printf("    server.port: %d\n", port)
	fmt.Printf("    server.host: %s\n", host)
	fmt.Println("\n  说明: 多个配置源会按优先级自动合并")
}
