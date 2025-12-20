package logger

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

// TestDefaultContextExtractor 测试默认的 context 提取器
func TestDefaultContextExtractor(t *testing.T) {
	ctx := context.Background()
	fields := DefaultContextExtractor(ctx)

	if fields != nil {
		t.Error("DefaultContextExtractor should return nil")
	}
}

// TestContextFieldExtractorType 测试 ContextFieldExtractor 类型
func TestContextFieldExtractorType(t *testing.T) {
	// 验证可以创建自定义提取器
	customExtractor := func(ctx context.Context) []zap.Field {
		return []zap.Field{
			zap.String("request_id", "test-123"),
			zap.String("user_id", "user-456"),
		}
	}

	// 验证类型兼容性
	var extractor ContextFieldExtractor = customExtractor
	if extractor == nil {
		t.Error("Failed to assign custom extractor")
	}

	// 测试提取器
	ctx := context.Background()
	fields := extractor(ctx)

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(fields))
	}
}

// TestContextExtractorWithValues 测试从 context 中提取值
func TestContextExtractorWithValues(t *testing.T) {
	type contextKey string

	// 定义 context key
	const (
		requestIDKey contextKey = "request_id"
		userIDKey    contextKey = "user_id"
		traceIDKey   contextKey = "trace_id"
	)

	// 创建自定义提取器
	customExtractor := func(ctx context.Context) []zap.Field {
		var fields []zap.Field

		if requestID, ok := ctx.Value(requestIDKey).(string); ok {
			fields = append(fields, zap.String("request_id", requestID))
		}

		if userID, ok := ctx.Value(userIDKey).(string); ok {
			fields = append(fields, zap.String("user_id", userID))
		}

		if traceID, ok := ctx.Value(traceIDKey).(string); ok {
			fields = append(fields, zap.String("trace_id", traceID))
		}

		return fields
	}

	// 测试空 context
	t.Run("empty context", func(t *testing.T) {
		ctx := context.Background()
		fields := customExtractor(ctx)

		if len(fields) != 0 {
			t.Errorf("Expected 0 fields for empty context, got %d", len(fields))
		}
	})

	// 测试包含单个值的 context
	t.Run("context with single value", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), requestIDKey, "req-123")
		fields := customExtractor(ctx)

		if len(fields) != 1 {
			t.Errorf("Expected 1 field, got %d", len(fields))
		}
	})

	// 测试包含多个值的 context
	t.Run("context with multiple values", func(t *testing.T) {
		ctx := context.Background()
		ctx = context.WithValue(ctx, requestIDKey, "req-123")
		ctx = context.WithValue(ctx, userIDKey, "user-456")
		ctx = context.WithValue(ctx, traceIDKey, "trace-789")

		fields := customExtractor(ctx)

		if len(fields) != 3 {
			t.Errorf("Expected 3 fields, got %d", len(fields))
		}
	})

	// 测试类型不匹配的值
	t.Run("context with wrong type", func(t *testing.T) {
		ctx := context.WithValue(context.Background(), requestIDKey, 12345) // int instead of string
		fields := customExtractor(ctx)

		if len(fields) != 0 {
			t.Errorf("Expected 0 fields for wrong type, got %d", len(fields))
		}
	})
}

// TestContextExtractorWithConfig 测试在配置中使用提取器
func TestContextExtractorWithConfig(t *testing.T) {
	type contextKey string
	const requestIDKey contextKey = "request_id"

	// 创建自定义提取器
	customExtractor := func(ctx context.Context) []zap.Field {
		if requestID, ok := ctx.Value(requestIDKey).(string); ok {
			return []zap.Field{zap.String("request_id", requestID)}
		}
		return nil
	}

	config := &Config{
		Level:            InfoLevel,
		EnableConsole:    true,
		ContextExtractor: customExtractor,
	}

	// 验证配置中的提取器
	if config.ContextExtractor == nil {
		t.Error("ContextExtractor should not be nil in config")
	}

	// 测试提取器
	ctx := context.WithValue(context.Background(), requestIDKey, "test-request")
	fields := config.ContextExtractor(ctx)

	if len(fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(fields))
	}
}

// TestMultipleExtractors 测试多个提取器的组合
func TestMultipleExtractors(t *testing.T) {
	type contextKey string

	const (
		requestIDKey contextKey = "request_id"
		userIDKey    contextKey = "user_id"
	)

	// 第一个提取器 - 提取 request_id
	extractor1 := func(ctx context.Context) []zap.Field {
		if requestID, ok := ctx.Value(requestIDKey).(string); ok {
			return []zap.Field{zap.String("request_id", requestID)}
		}
		return nil
	}

	// 第二个提取器 - 提取 user_id
	extractor2 := func(ctx context.Context) []zap.Field {
		if userID, ok := ctx.Value(userIDKey).(string); ok {
			return []zap.Field{zap.String("user_id", userID)}
		}
		return nil
	}

	// 组合提取器
	combinedExtractor := func(ctx context.Context) []zap.Field {
		var fields []zap.Field
		fields = append(fields, extractor1(ctx)...)
		fields = append(fields, extractor2(ctx)...)
		return fields
	}

	// 测试组合提取器
	ctx := context.Background()
	ctx = context.WithValue(ctx, requestIDKey, "req-123")
	ctx = context.WithValue(ctx, userIDKey, "user-456")

	fields := combinedExtractor(ctx)

	if len(fields) != 2 {
		t.Errorf("Expected 2 fields from combined extractor, got %d", len(fields))
	}
}

// TestContextExtractorNilSafety 测试 nil 安全性
func TestContextExtractorNilSafety(t *testing.T) {
	config := &Config{
		Level:         InfoLevel,
		EnableConsole: true,
	}

	// ContextExtractor 默认应该为 nil
	if config.ContextExtractor != nil {
		t.Error("ContextExtractor should be nil by default")
	}

	// 使用默认提取器
	config.ContextExtractor = DefaultContextExtractor

	ctx := context.Background()
	fields := config.ContextExtractor(ctx)

	if fields != nil {
		t.Error("DefaultContextExtractor should return nil")
	}
}

// TestContextExtractorWithDifferentTypes 测试不同类型的字段提取
func TestContextExtractorWithDifferentTypes(t *testing.T) {
	type contextKey string

	const (
		stringKey contextKey = "string_value"
		intKey    contextKey = "int_value"
		boolKey   contextKey = "bool_value"
		floatKey  contextKey = "float_value"
	)

	extractor := func(ctx context.Context) []zap.Field {
		var fields []zap.Field

		if val, ok := ctx.Value(stringKey).(string); ok {
			fields = append(fields, zap.String("string_field", val))
		}

		if val, ok := ctx.Value(intKey).(int); ok {
			fields = append(fields, zap.Int("int_field", val))
		}

		if val, ok := ctx.Value(boolKey).(bool); ok {
			fields = append(fields, zap.Bool("bool_field", val))
		}

		if val, ok := ctx.Value(floatKey).(float64); ok {
			fields = append(fields, zap.Float64("float_field", val))
		}

		return fields
	}

	// 创建包含不同类型值的 context
	ctx := context.Background()
	ctx = context.WithValue(ctx, stringKey, "test")
	ctx = context.WithValue(ctx, intKey, 42)
	ctx = context.WithValue(ctx, boolKey, true)
	ctx = context.WithValue(ctx, floatKey, 3.14)

	fields := extractor(ctx)

	if len(fields) != 4 {
		t.Errorf("Expected 4 fields with different types, got %d", len(fields))
	}
}

// TestContextExtractorChaining 测试提取器链式调用
func TestContextExtractorChaining(t *testing.T) {
	type contextKey string
	const keyA contextKey = "key_a"
	const keyB contextKey = "key_b"

	// 创建第一个提取器
	extractorA := func(ctx context.Context) []zap.Field {
		if val, ok := ctx.Value(keyA).(string); ok {
			return []zap.Field{zap.String("field_a", val)}
		}
		return nil
	}

	// 创建第二个提取器，依赖第一个
	extractorB := func(ctx context.Context, prevFields []zap.Field) []zap.Field {
		fields := prevFields
		if val, ok := ctx.Value(keyB).(string); ok {
			fields = append(fields, zap.String("field_b", val))
		}
		return fields
	}

	// 测试链式调用
	ctx := context.Background()
	ctx = context.WithValue(ctx, keyA, "value_a")
	ctx = context.WithValue(ctx, keyB, "value_b")

	fieldsA := extractorA(ctx)
	finalFields := extractorB(ctx, fieldsA)

	if len(finalFields) != 2 {
		t.Errorf("Expected 2 fields from chained extractors, got %d", len(finalFields))
	}
}
