package main

import (
	"fmt"
	"log"

	"github.com/lk2023060901/xdooria/pkg/config"
)

func main() {
	fmt.Println("=== 示例 2：UnmarshalKey - 解析部分配置 ===")

	mgr := config.NewManager()
	if err := mgr.LoadFile("config.yaml"); err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 1. 解析整个 Server 配置块到结构体
	fmt.Println("【1. 解析 Server 配置到结构体】")
	var serverCfg struct {
		Port int    `yaml:"port"`
		Host string `yaml:"host"`
	}
	if err := mgr.UnmarshalKey("server", &serverCfg); err != nil {
		log.Fatalf("解析 server 配置失败: %v", err)
	}
	fmt.Printf("  Server: %+v\n", serverCfg)
	fmt.Println()

	// 2. 解析整个 Database 配置块到结构体
	fmt.Println("【2. 解析 Database 配置到结构体】")
	var dbCfg struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
	}
	if err := mgr.UnmarshalKey("database", &dbCfg); err != nil {
		log.Fatalf("解析 database 配置失败: %v", err)
	}
	fmt.Printf("  Database: %+v\n", dbCfg)
	fmt.Println()

	// 3. 解析单个字段到基本类型 (int)
	fmt.Println("【3. 解析单个字段到基本类型】")
	var port int
	mgr.UnmarshalKey("server.port", &port)
	fmt.Printf("  server.port (int): %d\n", port)

	// 4. 解析单个字段到基本类型 (string)
	var host string
	mgr.UnmarshalKey("server.host", &host)
	fmt.Printf("  server.host (string): %s\n", host)

	// 5. 解析单个字段到基本类型 (bool)
	var enabled bool
	mgr.UnmarshalKey("feature.enabled", &enabled)
	fmt.Printf("  feature.enabled (bool): %v\n", enabled)
	fmt.Println()

	// 6. 解析嵌套配置
	fmt.Println("【4. 解析嵌套字段】")
	var dbHost string
	mgr.UnmarshalKey("database.host", &dbHost)
	fmt.Printf("  database.host: %s\n", dbHost)

	var dbPort int
	mgr.UnmarshalKey("database.port", &dbPort)
	fmt.Printf("  database.port: %d\n", dbPort)
	fmt.Println()

	// 7. 使用 Get 方法获取任意类型
	fmt.Println("【5. 使用 Get 方法】")
	fmt.Printf("  Get('server.port'): %v (type: any)\n", mgr.Get("server.port"))
	fmt.Printf("  GetInt('server.port'): %d (type: int)\n", mgr.GetInt("server.port"))
	fmt.Printf("  GetString('server.host'): %s (type: string)\n", mgr.GetString("server.host"))
	fmt.Printf("  GetBool('feature.enabled'): %v (type: bool)\n", mgr.GetBool("feature.enabled"))
	fmt.Println()

	fmt.Println("✅ 示例完成")
	fmt.Println("\n💡 总结：")
	fmt.Println("  - UnmarshalKey(key, &struct) → 解析配置块到结构体")
	fmt.Println("  - UnmarshalKey(key, &int/string/bool) → 解析单个字段到基本类型")
	fmt.Println("  - Get/GetInt/GetString/GetBool → 直接获取配置值")
}
