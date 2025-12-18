package logger

// Level 日志等级
type Level string

const (
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
	PanicLevel Level = "panic"
	FatalLevel Level = "fatal"
)

// Format 日志格式
type Format string

const (
	JSONFormat    Format = "json"
	ConsoleFormat Format = "console"
)

// RotationType 轮换类型
type RotationType string

const (
	RotationBySize RotationType = "size"
	RotationByTime RotationType = "time"
)

// Config 日志配置
type Config struct {
	// 基础配置
	Level  Level  `mapstructure:"level"`  // 日志等级
	Format Format `mapstructure:"format"` // 输出格式 (json/console)

	// 输出配置
	EnableConsole bool   `mapstructure:"enable_console"` // 启用控制台输出
	EnableFile    bool   `mapstructure:"enable_file"`    // 启用文件输出
	OutputPath    string `mapstructure:"output_path"`    // 日志文件路径

	// 时间格式
	TimeFormat string `mapstructure:"time_format"` // 时间格式 (默认: 2006-01-02 15:04:05)

	// 轮换配置
	Rotation RotationConfig `mapstructure:"rotation"`

	// 堆栈跟踪
	EnableStacktrace bool  `mapstructure:"enable_stacktrace"` // 启用堆栈跟踪
	StacktraceLevel  Level `mapstructure:"stacktrace_level"`  // 堆栈跟踪等级

	// 采样配置 (防止高频日志)
	EnableSampling     bool `mapstructure:"enable_sampling"`      // 启用采样
	SamplingInitial    int  `mapstructure:"sampling_initial"`     // 每秒前 N 条日志
	SamplingThereafter int  `mapstructure:"sampling_thereafter"`  // 之后每 N 条记录 1 条

	// 开发模式
	Development bool `mapstructure:"development"` // 开发模式 (启用彩色输出、可读时间)

	// 全局字段
	GlobalFields map[string]interface{} `mapstructure:"global_fields"` // 全局字段

	// 性能优化
	EnableAsync bool `mapstructure:"enable_async"` // 启用异步写入
	BufferSize  int  `mapstructure:"buffer_size"`  // 缓冲区大小 (字节)
}

// RotationConfig 轮换配置
type RotationConfig struct {
	Type RotationType `mapstructure:"type"` // 轮换类型: size 或 time

	// 按大小轮换 (lumberjack)
	MaxSize    int  `mapstructure:"max_size"`    // 单文件最大大小 (MB)
	MaxBackups int  `mapstructure:"max_backups"` // 保留的旧文件数量
	MaxAge     int  `mapstructure:"max_age"`     // 保留天数
	Compress   bool `mapstructure:"compress"`    // 是否压缩旧文件

	// 按时间轮换 (file-rotatelogs)
	RotationTime    string `mapstructure:"rotation_time"`    // 轮换间隔: 1h, 24h
	MaxAgeTime      string `mapstructure:"max_age_time"`     // 保留时长: 168h (7天)
	RotationPattern string `mapstructure:"rotation_pattern"` // 文件名时间格式: .%Y%m%d%H
}

// DefaultConfig 默认配置
func DefaultConfig() *Config {
	return &Config{
		Level:         InfoLevel,
		Format:        ConsoleFormat, // 默认控制台格式
		EnableConsole: true,          // 默认仅控制台输出
		EnableFile:    false,         // 默认不输出到文件
		TimeFormat:    "2006-01-02 15:04:05",
		Rotation: RotationConfig{
			Type:            RotationBySize,
			MaxSize:         100,
			MaxBackups:      5,
			MaxAge:          7,
			Compress:        true,
			RotationTime:    "24h",
			MaxAgeTime:      "168h",
			RotationPattern: ".%Y%m%d",
		},
		EnableStacktrace:   true,
		StacktraceLevel:    ErrorLevel,
		EnableSampling:     false,
		SamplingInitial:    100,
		SamplingThereafter: 100,
		Development:        false,
		GlobalFields:       make(map[string]interface{}),
		EnableAsync:        false,
		BufferSize:         256 * 1024, // 256KB
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.EnableFile && c.OutputPath == "" {
		return ErrInvalidOutputPath
	}
	if !c.EnableConsole && !c.EnableFile {
		return ErrNoOutputEnabled
	}
	return nil
}
