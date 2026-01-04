package dao

import (
	"fmt"
	"path/filepath"

	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/app/game/internal/gameconfig"
)

// GameConfigConfig 游戏配置表加载配置
type GameConfigConfig struct {
	DataDir string `mapstructure:"data_dir"`
}

// ConfigDAO 负责初始化全局配置表
type ConfigDAO struct{}

// NewConfigDAO 创建配置 DAO
func NewConfigDAO(gameCfg *GameConfigConfig, a *app.BaseApp) (*ConfigDAO, error) {
	// 1. 获取应用主日志对象
	l := a.AppLogger()

	// 2. 确定数据目录
	dataDir := gameCfg.DataDir
	if dataDir == "" {
		execDir, err := app.GetExecDir()
		if err != nil {
			return nil, err
		}
		dataDir = filepath.Join(execDir, "configs", "data")
	}

	// 3. 加载到全局变量 gameconfig.T
	if err := gameconfig.Load(dataDir, l); err != nil {
		return nil, fmt.Errorf("failed to load global gameconfig: %w", err)
	}

	return &ConfigDAO{}, nil
}
