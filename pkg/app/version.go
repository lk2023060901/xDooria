package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var (
	// 以下变量由编译时通过 -ldflags "-X 'pkg/app.Version=v1.0.0'" 注入
	Version   = "unknown"
	GitCommit = "unknown"
	BuildDate = "unknown"
	AppName   = "" // 默认为空，以便在运行时推断
)

func init() {
	if AppName == "" {
		if execPath, err := os.Executable(); err == nil {
			AppName = filepath.Base(execPath)
		} else {
			AppName = "xdooria-app"
		}
	}
}

// Info 返回完整的版本信息结构体
type Info struct {
	AppName   string `json:"app_name"`
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// GetInfo 获取当前应用信息
func GetInfo() Info {
	return Info{
		AppName:   AppName,
		Version:   Version,
		GitCommit: GitCommit,
		BuildDate: BuildDate,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String 返回格式化的版本字符串
func (i Info) String() string {
	return fmt.Sprintf("%s %s (commit: %s, build: %s, go: %s, plat: %s)",
		i.AppName, i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Platform)
}
