package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

var (
	// 以下变量通过 -ldflags 注入
	Version   = "unknown"
	GitCommit = "unknown"
	BuildDate = "unknown"
	AppName   = ""
)

func init() {
	// 如果编译时没注入 AppName，则根据文件名推断
	if AppName == "" {
		if execPath, err := os.Executable(); err == nil {
			AppName = filepath.Base(execPath)
		} else {
			AppName = "xdooria-app"
		}
	}
}

// Info 返回完整的版本信息
type Info struct {
	AppName   string
	Version   string
	GitCommit string
	BuildDate string
	GoVersion string
	Platform  string
}

// GetInfo 每次调用都返回当前最新的变量值
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

func (i Info) String() string {
	return fmt.Sprintf("%s %s (commit: %s, build: %s, go: %s, plat: %s)",
		i.AppName, i.Version, i.GitCommit, i.BuildDate, i.GoVersion, i.Platform)
}