package main

import (
	"fmt"
	"time"

	"github.com/lk2023060901/xdooria/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	fmt.Println("=== Logger Structured Logging 示例 ===")

	// 示例 1: JSON 格式的结构化日志
	fmt.Println("【1. JSON 格式】")
	testJSONFormat()

	// 示例 2: 使用不同类型的字段
	fmt.Println("\n【2. 多种字段类型】")
	testVariousFieldTypes()

	// 示例 3: WithFields 预设字段
	fmt.Println("\n【3. WithFields 预设字段】")
	testWithFields()

	// 示例 4: Named Logger 和层级结构
	fmt.Println("\n【4. Named Logger】")
	testNamedLogger()

	// 示例 5: 全局字段
	fmt.Println("\n【5. 全局字段】")
	testGlobalFields()

	// 示例 6: 复杂对象序列化
	fmt.Println("\n【6. 复杂对象】")
	testComplexObjects()

	fmt.Println("\n✅ 示例完成")
}

func testJSONFormat() {
	cfg := &logger.Config{
		Level:         logger.InfoLevel,
		Format:        logger.JSONFormat,
		EnableConsole: true,
		TimeFormat:    "2006-01-02 15:04:05",
	}

	log, _ := logger.New(cfg)
	defer log.Sync()

	fmt.Println("  JSON 格式输出:")
	log.Info("用户登录事件",
		zap.String("user_id", "user-12345"),
		zap.String("username", "zhangsan"),
		zap.String("ip", "192.168.1.100"),
		zap.Int("login_count", 42),
		zap.Bool("success", true),
	)
}

func testVariousFieldTypes() {
	log, _ := logger.New(&logger.Config{
		Level:  logger.DebugLevel,
		Format: logger.JSONFormat,
	})
	defer log.Sync()

	fmt.Println("  支持的字段类型:")

	// 基本类型
	log.Info("基本类型",
		zap.String("string", "文本"),
		zap.Int("int", 123),
		zap.Int64("int64", 9223372036854775807),
		zap.Float64("float64", 3.14159),
		zap.Bool("bool", true),
	)

	// 时间类型
	log.Info("时间类型",
		zap.Time("time", time.Now()),
		zap.Duration("duration", 5*time.Second),
	)

	// 数组和切片
	log.Info("数组类型",
		zap.Strings("tags", []string{"golang", "logging", "zap"}),
		zap.Ints("scores", []int{95, 88, 92}),
	)

	// 错误类型
	err := fmt.Errorf("示例错误")
	log.Error("错误类型", zap.Error(err))

	// Any 类型（自动推断）
	log.Info("Any 类型",
		zap.Any("map", map[string]int{"a": 1, "b": 2}),
		zap.Any("slice", []int{1, 2, 3}),
	)
}

func testWithFields() {
	log, _ := logger.New(&logger.Config{
		Level:       logger.InfoLevel,
		Format:      logger.JSONFormat,
		Development: false,
	})
	defer log.Sync()

	fmt.Println("  使用 WithFields 预设公共字段:")

	// 创建带预设字段的 logger
	requestLogger := log.WithFields(
		"request_id", "req-abc123",
		"service", "api-server",
		"version", "1.0.0",
	)

	// 后续所有日志都会包含这些字段
	requestLogger.Info("请求开始")
	requestLogger.Info("处理中", zap.String("handler", "CreateUser"))
	requestLogger.Info("请求完成", zap.Int("status_code", 200))
}

func testNamedLogger() {
	log, _ := logger.New(&logger.Config{
		Level:  logger.DebugLevel,
		Format: logger.JSONFormat,
	})
	defer log.Sync()

	fmt.Println("  命名 Logger 层级:")

	// 创建命名 logger
	dbLogger := log.Named("database")
	dbLogger.Info("数据库连接成功")

	// 子命名 logger
	pgLogger := dbLogger.Named("postgres")
	pgLogger.Info("PostgreSQL 初始化完成")

	redisLogger := dbLogger.Named("redis")
	redisLogger.Info("Redis 连接成功")

	// 业务 logger
	apiLogger := log.Named("api")
	apiLogger.Info("API 服务器启动")

	userHandler := apiLogger.Named("user")
	userHandler.Info("用户处理器注册")
}

func testGlobalFields() {
	cfg := &logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.JSONFormat,
		GlobalFields: map[string]interface{}{
			"app":     "xdooria",
			"env":     "production",
			"version": "1.0.0",
			"region":  "cn-north-1",
		},
	}

	log, _ := logger.New(cfg)
	defer log.Sync()

	fmt.Println("  全局字段会自动添加到所有日志:")
	log.Info("服务启动")
	log.Info("处理请求", zap.String("endpoint", "/api/users"))
	log.Warn("慢查询警告", zap.Duration("duration", 2*time.Second))
}

func testComplexObjects() {
	log, _ := logger.New(&logger.Config{
		Level:  logger.InfoLevel,
		Format: logger.JSONFormat,
	})
	defer log.Sync()

	fmt.Println("  复杂对象序列化:")

	// 定义复杂结构
	type User struct {
		ID       string
		Username string
		Email    string
		Roles    []string
		Metadata map[string]interface{}
	}

	user := User{
		ID:       "user-001",
		Username: "zhangsan",
		Email:    "zhangsan@example.com",
		Roles:    []string{"admin", "user"},
		Metadata: map[string]interface{}{
			"created_at": time.Now(),
			"active":     true,
			"login_count": 100,
		},
	}

	// 使用 zap.Any 序列化整个对象
	log.Info("用户信息",
		zap.Any("user", user),
		zap.String("action", "login"),
	)

	// 使用 zap.Object 自定义序列化（需要实现 ObjectMarshaler 接口）
	// 或者使用 zap.Inline 嵌入字段
	log.Info("订单创建",
		zap.String("order_id", "order-12345"),
		zap.Any("user", map[string]string{
			"id":   user.ID,
			"name": user.Username,
		}),
		zap.Float64("total", 199.99),
		zap.String("currency", "CNY"),
	)
}
