package main

import (
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("=== Logger File Rotation 示例 ===")

	// 示例 1: 按大小轮换 (Size Rotation)
	fmt.Println("【1. 按大小轮换】")
	testSizeRotation()

	// 示例 2: 按时间轮换 (Time Rotation)
	fmt.Println("\n【2. 按时间轮换】")
	testTimeRotation()

	// 示例 3: 自定义轮换配置
	fmt.Println("\n【3. 自定义轮换配置】")
	testCustomRotation()

	// 示例 4: 同时输出到控制台和文件
	fmt.Println("\n【4. 控制台 + 文件输出】")
	testConsoleAndFile()

	fmt.Println("\n✅ 示例完成")
	fmt.Println("检查日志文件:")
	fmt.Println("  - ./logs/size-rotation.log")
	fmt.Println("  - ./logs/time-rotation.log.*")
	fmt.Println("  - ./logs/custom.log*")
	fmt.Println("  - ./logs/combined.log")
}

func testSizeRotation() {
	cfg := &logger.Config{
		Level:         logger.InfoLevel,
		Format:        logger.JSONFormat,
		EnableConsole: false,
		EnableFile:    true,
		OutputPath:    "./logs/size-rotation.log",
		Rotation: logger.RotationConfig{
			Type:       logger.RotationBySize,
			MaxSize:    1,   // 1MB
			MaxBackups: 3,   // 保留 3 个备份
			MaxAge:     7,   // 保留 7 天
			Compress:   true, // 压缩旧日志
		},
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}
	defer log.Sync()

	fmt.Println("  配置:")
	fmt.Println("    - 最大文件大小: 1MB")
	fmt.Println("    - 最大备份数: 3")
	fmt.Println("    - 保留时间: 7 天")
	fmt.Println("    - 压缩旧文件: 是")

	// 写入一些日志
	for i := 0; i < 10; i++ {
		log.Info("按大小轮换测试",
			zap.Int("iteration", i),
			zap.String("message", "这是一条测试日志，用于演示按大小轮换功能"),
		)
	}

	fmt.Println("  ✓ 已写入 10 条日志")
}

func testTimeRotation() {
	cfg := &logger.Config{
		Level:         logger.InfoLevel,
		Format:        logger.ConsoleFormat,
		EnableConsole: false,
		EnableFile:    true,
		OutputPath:    "./logs/time-rotation.log",
		Rotation: logger.RotationConfig{
			Type:            logger.RotationByTime,
			RotationTime:    "1m",      // 每分钟轮换一次（演示用）
			MaxAgeTime:      "1h",      // 保留 1 小时
			RotationPattern: ".%Y%m%d%H%M", // 文件名模式
		},
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}
	defer log.Sync()

	fmt.Println("  配置:")
	fmt.Println("    - 轮换间隔: 1 分钟")
	fmt.Println("    - 保留时间: 1 小时")
	fmt.Println("    - 文件名模式: .YYYYMMDDHHmm")

	// 写入一些日志
	for i := 0; i < 5; i++ {
		log.Info("按时间轮换测试",
			zap.Int("iteration", i),
			zap.Time("timestamp", time.Now()),
		)
		time.Sleep(200 * time.Millisecond)
	}

	fmt.Println("  ✓ 已写入 5 条日志")
}

func testCustomRotation() {
	cfg := &logger.Config{
		Level:         logger.DebugLevel,
		Format:        logger.JSONFormat,
		EnableConsole: false,
		EnableFile:    true,
		OutputPath:    "./logs/custom.log",
		Rotation: logger.RotationConfig{
			Type:            logger.RotationByTime,
			RotationTime:    "24h",     // 每天轮换
			MaxAgeTime:      "168h",    // 保留 7 天
			RotationPattern: "-%Y-%m-%d", // 自定义文件名模式
		},
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}
	defer log.Sync()

	fmt.Println("  配置:")
	fmt.Println("    - 轮换间隔: 每天")
	fmt.Println("    - 保留时间: 7 天")
	fmt.Println("    - 文件名模式: -YYYY-MM-DD")

	log.Debug("自定义轮换配置 - Debug")
	log.Info("自定义轮换配置 - Info")
	log.Warn("自定义轮换配置 - Warn")

	fmt.Println("  ✓ 已写入 3 条不同级别的日志")
}

func testConsoleAndFile() {
	cfg := &logger.Config{
		Level:         logger.InfoLevel,
		Format:        logger.ConsoleFormat,
		EnableConsole: true,  // 同时输出到控制台
		EnableFile:    true,
		OutputPath:    "./logs/combined.log",
		Development:   true, // 彩色输出
		Rotation: logger.RotationConfig{
			Type:       logger.RotationBySize,
			MaxSize:    10,
			MaxBackups: 2,
			Compress:   false,
		},
	}

	log, err := logger.New(cfg)
	if err != nil {
		fmt.Printf("创建 logger 失败: %v\n", err)
		return
	}
	defer log.Sync()

	fmt.Println("  配置:")
	fmt.Println("    - 控制台输出: 开启")
	fmt.Println("    - 文件输出: 开启")
	fmt.Println("    - 开发模式: 开启（彩色）")
	fmt.Println()

	log.Info("同时输出到控制台和文件",
		zap.String("output", "console+file"),
		zap.Bool("colored", true),
	)

	log.Warn("这是一条警告日志", zap.String("type", "warning"))
}
