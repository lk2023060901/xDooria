package logger

// Option 配置选项
type Option func(*BaseLogger)

// WithName 设置 logger 名称
func WithName(name string) Option {
	return func(l *BaseLogger) {
		l.name = name
	}
}

// WithGlobalFields 添加全局字段（作为 Option）
func WithGlobalFields(fields ...interface{}) Option {
	return func(l *BaseLogger) {
		if len(fields)%2 != 0 {
			return
		}
		for i := 0; i < len(fields); i += 2 {
			key, ok := fields[i].(string)
			if !ok {
				continue
			}
			l.globalFields[key] = fields[i+1]
		}
	}
}

// WithHooks 添加钩子
func WithHooks(hooks ...Hook) Option {
	return func(l *BaseLogger) {
		l.hooks = append(l.hooks, hooks...)
	}
}

// WithLevel 设置日志等级
func WithLevel(level Level) Option {
	return func(l *BaseLogger) {
		l.config.Level = level
	}
}

// WithDevelopment 启用开发模式
func WithDevelopment(dev bool) Option {
	return func(l *BaseLogger) {
		l.config.Development = dev
	}
}

// WithContextExtractor 设置 context 字段提取器
func WithContextExtractor(extractor ContextFieldExtractor) Option {
	return func(l *BaseLogger) {
		l.contextExtractor = extractor
	}
}
