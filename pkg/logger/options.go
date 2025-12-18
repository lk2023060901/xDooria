package logger

// Option 配置选项
type Option func(*Logger)

// WithName 设置 logger 名称
func WithName(name string) Option {
	return func(l *Logger) {
		l.name = name
	}
}

// WithGlobalFields 添加全局字段（作为 Option）
func WithGlobalFields(fields ...interface{}) Option {
	return func(l *Logger) {
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
	return func(l *Logger) {
		l.hooks = append(l.hooks, hooks...)
	}
}

// WithLevel 设置日志等级
func WithLevel(level Level) Option {
	return func(l *Logger) {
		l.config.Level = level
	}
}

// WithDevelopment 启用开发模式
func WithDevelopment(dev bool) Option {
	return func(l *Logger) {
		l.config.Development = dev
	}
}
