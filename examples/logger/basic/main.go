package main

import (
	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 示例 1: 使用默认配置（仅控制台输出）
	log1, _ := logger.New(nil)
	log1.Info("使用默认配置")

	// 示例 2: 仅设置部分字段，其他使用默认值
	partialConfig := &logger.Config{
		Level:       logger.DebugLevel,
		Development: true, // 启用彩色输出
	}
	log2, _ := logger.New(partialConfig)
	log2.Debug("部分配置 - Debug 级别")
	log2.Info("部分配置 - Info 级别")

	// 示例 3: 使用全局 logger
	logger.Info("使用默认全局 logger")

	// 示例 4: 带字段的日志
	logger.Info("用户登录", zap.String("user_id", "12345"), zap.Int("age", 25))

	// 示例 5: 使用 WithFields 添加字段
	userLogger := logger.WithFields("service", "auth", "version", "1.0.0")
	userLogger.Info("服务启动")

	// 示例 6: 具名 logger
	dbLogger := logger.Named("database")
	dbLogger.Info("数据库连接成功")
}
