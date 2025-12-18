package logger

import (
	"io"
	"time"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"gopkg.in/natefinch/lumberjack.v2"
)

// NewRotationWriter 创建轮换 writer
// 注意: 该函数仅在 EnableFile=true 时调用
func NewRotationWriter(cfg *RotationConfig, outputPath string) (io.Writer, error) {
	switch cfg.Type {
	case RotationBySize:
		return newSizeRotationWriter(cfg, outputPath), nil
	case RotationByTime:
		return newTimeRotationWriter(cfg, outputPath)
	default:
		return newSizeRotationWriter(cfg, outputPath), nil
	}
}

// newSizeRotationWriter 创建按大小轮换的 writer
func newSizeRotationWriter(cfg *RotationConfig, outputPath string) io.Writer {
	return &lumberjack.Logger{
		Filename:   outputPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
		LocalTime:  true,
	}
}

// newTimeRotationWriter 创建按时间轮换的 writer
func newTimeRotationWriter(cfg *RotationConfig, outputPath string) (io.Writer, error) {
	// 解析轮换间隔
	rotationTime, err := time.ParseDuration(cfg.RotationTime)
	if err != nil {
		rotationTime = 24 * time.Hour // 默认每天轮换
	}

	// 解析保留时长
	maxAge, err := time.ParseDuration(cfg.MaxAgeTime)
	if err != nil {
		maxAge = 7 * 24 * time.Hour // 默认保留 7 天
	}

	// 文件名模式
	pattern := outputPath
	if cfg.RotationPattern != "" {
		pattern = outputPath + cfg.RotationPattern
	} else {
		pattern = outputPath + ".%Y%m%d%H"
	}

	writer, err := rotatelogs.New(
		pattern,
		rotatelogs.WithLinkName(outputPath),       // 软链接到当前日志
		rotatelogs.WithRotationTime(rotationTime), // 轮换间隔
		rotatelogs.WithMaxAge(maxAge),             // 保留时长
	)

	return writer, err
}
