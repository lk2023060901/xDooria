package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lk2023060901/xdooria/pkg/config"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	configPath string
	logPath    string
)

// LoadConfig 集成 pkg/config 提供统一加载能力
// 严格遵守优先级：1. 命令行显式参数 > 2. 环境变量 > 3. 配置文件 > 4. 默认值
func LoadConfig(target any, opts ...config.Option) error {
	// 1. 获取执行目录，用于计算默认值
	execDir, err := GetExecDir()
	if err != nil {
		return fmt.Errorf("failed to get executable directory: %w", err)
	}

	// 2. 预计算默认物理路径
	defaultConfig := filepath.Join(execDir, "config.yaml")
	defaultLog := filepath.Join(execDir, "logs", "app.log")

	// 3. 注册命令行参数
	if pflag.Lookup("config") == nil {
		pflag.StringVarP(&configPath, "config", "c", defaultConfig, "path to config file")
	}
	if pflag.Lookup("log.path") == nil {
		pflag.StringVar(&logPath, "log.path", defaultLog, "output path for logs")
	}

	// 4. 解析命令行参数
	if !pflag.Parsed() {
		pflag.Parse()
	}

	// 5. 创建 Viper 实例并配置环境变量映射
	v := viper.New()
	v.SetEnvPrefix("XDOORIA")
	v.AutomaticEnv()
	// 将环境变量中的 "_" 替换为配置中的 "."，例如 XDOORIA_LOG_LEVEL -> log.level
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	// 6. 确定配置文件路径
	// 优先级：Flag 显式指定 > 环境变量 XDOORIA_CONFIG > 默认物理路径
	finalConfigPath := configPath
	if !pflag.CommandLine.Changed("config") {
		if envConfig := os.Getenv("XDOORIA_CONFIG"); envConfig != "" {
			finalConfigPath = envConfig
		}
	}

	// 检查配置文件是否存在（必须存在）
	if _, err := os.Stat(finalConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("config file not found at %s", finalConfigPath)
	}
	configPath = finalConfigPath

	// 7. 设置配置项优先级
	// A. 设置最低优先级的默认值（会被配置文件和环境变量覆盖）
	v.SetDefault("log.output_path", defaultLog)
	v.SetDefault("log.enable_file", true)

	// B. 如果命令行显式使用了 --log.path，则强制覆盖所有来源（最高优先级）
	if pflag.CommandLine.Changed("log.path") {
		v.Set("log.output_path", logPath)
	}

	// 8. 初始化配置管理器
	mgr := config.NewManager(append(opts, config.WithViper(v))...)

	// 9. 加载配置文件
	if err := mgr.LoadFile(configPath); err != nil {
		return err
	}

	// 10. 解析到目标结构体
	if err := mgr.Unmarshal(target); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// 11. 获取最终生效的日志路径（用于自动创建目录）
	logPath = v.GetString("log.output_path")
	logDir := filepath.Dir(logPath)
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		_ = os.MkdirAll(logDir, 0755)
	}

	return nil
}

// getExecDir 获取可执行文件所在目录（处理符号链接）
func GetExecDir() (string, error) {
	execPath, err := os.Executable()
	if err != nil {
		return "", err
	}
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		return filepath.Dir(execPath), nil
	}
	return filepath.Dir(realPath), nil
}

// GetConfigPath 返回最终使用的配置文件路径
func GetConfigPath() string {
	return configPath
}

func GetLogPath() string {
	return logPath
}
