package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 服务端配置结构
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Etcd     EtcdConfig     `mapstructure:"etcd"`
	NSQ      NSQConfig      `mapstructure:"nsq"`
	Log      LogConfig      `mapstructure:"log"`
}

type ServerConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
	Host string `mapstructure:"host"`
}

type DatabaseConfig struct {
	Host        string `mapstructure:"host"`
	Port        int    `mapstructure:"port"`
	User        string `mapstructure:"user"`
	Password    string `mapstructure:"password"`
	Database    string `mapstructure:"database"`
	MaxConns    int    `mapstructure:"max_conns"`
	MinConns    int    `mapstructure:"min_conns"`
	MaxIdleTime int    `mapstructure:"max_idle_time"`
	SSLMode     string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Addrs       []string `mapstructure:"addrs"`
	Password    string   `mapstructure:"password"`
	DB          int      `mapstructure:"db"`
	PoolSize    int      `mapstructure:"pool_size"`
	MaxRetries  int      `mapstructure:"max_retries"`
	DialTimeout int      `mapstructure:"dial_timeout"`
}

type EtcdConfig struct {
	Endpoints   []string `mapstructure:"endpoints"`
	DialTimeout int      `mapstructure:"dial_timeout"`
	Username    string   `mapstructure:"username"`
	Password    string   `mapstructure:"password"`
}

type NSQConfig struct {
	NSQDAddr    string `mapstructure:"nsqd_addr"`
	LookupdAddr string `mapstructure:"lookupd_addr"`
	MaxInFlight int    `mapstructure:"max_in_flight"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	OutputPath string `mapstructure:"output_path"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

func main() {
	fmt.Println("=== Viper YAML 示例 ===\n")

	// 示例 1: 基本用法 - 加载整个配置
	example1()

	// 示例 2: UnmarshalKey - 只加载配置的某个部分
	example2()

	// 示例 3: 环境变量覆盖
	example3()
}

// 示例 1: 加载整个配置
func example1() {
	fmt.Println("--- 示例 1: 加载整个配置 ---")

	v := viper.New()
	v.SetConfigFile("config.yaml")
	v.SetConfigType("yaml")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// 解析到结构体
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	// 输出配置
	fmt.Printf("服务名称: %s\n", cfg.Server.Name)
	fmt.Printf("服务端口: %d\n", cfg.Server.Port)
	fmt.Printf("环境: %s\n", cfg.Server.Env)
	fmt.Printf("数据库地址: %s@%s:%d/%s\n",
		cfg.Database.User,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.Database,
	)
	fmt.Printf("Redis 地址: %v\n", cfg.Redis.Addrs)
	fmt.Printf("日志级别: %s\n", cfg.Log.Level)
	fmt.Println()
}

// 示例 2: UnmarshalKey - 只加载某个 key
func example2() {
	fmt.Println("--- 示例 2: UnmarshalKey - 只加载某个 key ---")

	v := viper.New()
	v.SetConfigFile("config.yaml")
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	// 只解析 server 部分
	var serverCfg ServerConfig
	if err := v.UnmarshalKey("server", &serverCfg); err != nil {
		log.Fatalf("解析 server 配置失败: %v", err)
	}

	fmt.Printf("服务名称: %s\n", serverCfg.Name)
	fmt.Printf("服务端口: %d\n", serverCfg.Port)
	fmt.Println()

	// 只解析 database 部分
	var dbCfg DatabaseConfig
	if err := v.UnmarshalKey("database", &dbCfg); err != nil {
		log.Fatalf("解析 database 配置失败: %v", err)
	}

	fmt.Printf("数据库主机: %s\n", dbCfg.Host)
	fmt.Printf("数据库端口: %d\n", dbCfg.Port)
	fmt.Printf("最大连接数: %d\n", dbCfg.MaxConns)
	fmt.Println()
}

// 示例 3: 环境变量覆盖
func example3() {
	fmt.Println("--- 示例 3: 环境变量覆盖 ---")

	// 设置环境变量
	os.Setenv("XDOORIA_SERVER_PORT", "8080")
	os.Setenv("XDOORIA_DATABASE_HOST", "prod-db.example.com")
	defer func() {
		os.Unsetenv("XDOORIA_SERVER_PORT")
		os.Unsetenv("XDOORIA_DATABASE_HOST")
	}()

	v := viper.New()
	v.SetConfigFile("config.yaml")
	v.SetConfigType("yaml")

	// 启用环境变量支持
	v.SetEnvPrefix("XDOORIA")
	v.AutomaticEnv()
	// 将配置中的 . 替换为 _ 用于环境变量
	// 例如 server.port -> XDOORIA_SERVER_PORT
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("解析配置失败: %v", err)
	}

	fmt.Printf("服务端口 (环境变量覆盖): %d\n", cfg.Server.Port)
	fmt.Printf("数据库主机 (环境变量覆盖): %s\n", cfg.Database.Host)
	fmt.Println()
}
