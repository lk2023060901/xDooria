package main

import (
	"fmt"
	"strings"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	fmt.Println("=== Logger Hook 示例 ===")

	// 示例 1: 使用内置的敏感数据脱敏 Hook
	fmt.Println("【1. 敏感数据脱敏】")
	testSensitiveDataHook()

	// 示例 2: 自定义 Hook - 日志过滤
	fmt.Println("\n【2. 日志过滤】")
	testFilterHook()

	// 示例 3: 自定义 Hook - 日志监控
	fmt.Println("\n【3. 日志监控统计】")
	testMonitoringHook()

	// 示例 4: 多个 Hook 组合
	fmt.Println("\n【4. 多个 Hook 组合】")
	testMultipleHooks()

	// 示例 5: 使用 HookFunc 简化创建
	fmt.Println("\n【5. HookFunc 简化】")
	testHookFunc()

	fmt.Println("\n✅ 示例完成")
}

func testSensitiveDataHook() {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.JSONFormat,
	}

	// 使用内置的敏感数据脱敏 Hook
	log, _ := logger.New(cfg,
		logger.WithHooks(
			logger.SensitiveDataHook([]string{"password", "token", "secret"}),
		),
	)
	defer log.Sync()

	fmt.Println("  脱敏字段: password, token, secret")

	// 这些敏感字段会被脱敏
	log.Info("用户登录",
		zap.String("username", "zhangsan"),
		zap.String("password", "123456"), // 会被脱敏
		zap.String("email", "zhangsan@example.com"),
	)

	log.Info("API 请求",
		zap.String("endpoint", "/api/data"),
		zap.String("token", "eyJhbGciOiJIUzI1NiIs..."), // 会被脱敏
		zap.String("method", "POST"),
	)
}

func testFilterHook() {
	// 自定义 Hook: 过滤掉包含特定关键词的日志
	filterHook := &LogFilterHook{
		keywords: []string{"test", "debug"},
	}

	cfg := &logger.Config{
		Level:  logger.DebugLevel,
		Format: logger.ConsoleFormat,
		Development: true,
	}

	log, _ := logger.New(cfg, logger.WithHooks(filterHook))
	defer log.Sync()

	fmt.Println("  过滤关键词: test, debug")
	fmt.Println("  以下日志中包含关键词的将被过滤:")

	log.Info("正常日志 1")                    // 会输出
	log.Info("这是一条测试日志 test")           // 会被过滤
	log.Debug("调试信息 debug")              // 会被过滤
	log.Info("正常日志 2")                    // 会输出
	log.Warn("警告: 发现 test 环境配置")        // 会被过滤

	fmt.Println("  ✓ 已过滤包含关键词的日志")
}

func testMonitoringHook() {
	// 自定义 Hook: 统计各级别日志数量
	monitorHook := &MonitoringHook{
		stats: make(map[zapcore.Level]int),
	}

	cfg := &logger.Config{
		Level:       logger.DebugLevel,
		Format:      logger.ConsoleFormat,
		Development: true,
	}

	log, _ := logger.New(cfg, logger.WithHooks(monitorHook))
	defer log.Sync()

	fmt.Println("  日志统计 Hook:")

	// 写入不同级别的日志
	log.Debug("调试信息 1")
	log.Debug("调试信息 2")
	log.Info("普通信息 1")
	log.Info("普通信息 2")
	log.Info("普通信息 3")
	log.Warn("警告信息")
	log.Error("错误信息")

	// 输出统计信息
	fmt.Println("\n  统计结果:")
	for level, count := range monitorHook.stats {
		fmt.Printf("    %s: %d 条\n", level, count)
	}
}

func testMultipleHooks() {
	// 组合多个 Hook
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.JSONFormat,
	}

	log, _ := logger.New(cfg,
		logger.WithHooks(
			logger.SensitiveDataHook([]string{"password"}),
			&LevelLimiterHook{maxLevel: zapcore.WarnLevel},
			&PrefixHook{prefix: "[APP]"},
		),
	)
	defer log.Sync()

	fmt.Println("  应用的 Hook:")
	fmt.Println("    1. 敏感数据脱敏 (password)")
	fmt.Println("    2. 级别限制 (最高 Warn)")
	fmt.Println("    3. 消息前缀 ([APP])")
	fmt.Println()

	log.Info("普通信息", zap.String("user", "zhangsan"))
	log.Warn("警告信息", zap.String("password", "secret123"))
	log.Error("错误信息") // 会被 LevelLimiterHook 过滤
}

func testHookFunc() {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.ConsoleFormat,
		Development: true,
	}

	// 使用 HookFunc 简化 Hook 创建
	upperCaseHook := logger.HookFunc(func(entry zapcore.Entry, fields []zapcore.Field) bool {
		entry.Message = strings.ToUpper(entry.Message)
		return true
	})

	log, _ := logger.New(cfg, logger.WithHooks(upperCaseHook))
	defer log.Sync()

	fmt.Println("  使用 HookFunc 创建 Hook (消息转大写):")

	log.Info("this is a test message")
	log.Warn("warning message")
}

// --- 自定义 Hook 实现 ---

// LogFilterHook 日志过滤 Hook
type LogFilterHook struct {
	keywords []string
}

func (h *LogFilterHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
	// 检查消息是否包含关键词
	for _, keyword := range h.keywords {
		if strings.Contains(entry.Message, keyword) {
			return false // 跳过该日志
		}
	}
	return true
}

// MonitoringHook 监控统计 Hook
type MonitoringHook struct {
	stats map[zapcore.Level]int
}

func (h *MonitoringHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
	h.stats[entry.Level]++
	return true
}

// LevelLimiterHook 级别限制 Hook
type LevelLimiterHook struct {
	maxLevel zapcore.Level
}

func (h *LevelLimiterHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
	// 只允许不高于 maxLevel 的日志
	return entry.Level <= h.maxLevel
}

// PrefixHook 消息前缀 Hook
type PrefixHook struct {
	prefix string
}

func (h *PrefixHook) OnWrite(entry zapcore.Entry, fields []zapcore.Field) bool {
	entry.Message = h.prefix + " " + entry.Message
	return true
}
