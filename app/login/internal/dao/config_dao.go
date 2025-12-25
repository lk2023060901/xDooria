package dao

import (
	"fmt"
	"path/filepath"

	"github.com/lk2023060901/xdooria/pkg/app"
	"github.com/lk2023060901/xdooria/pkg/gameconfig"
)

// ConfigDAO 负责加载和提供 Luban 配置表
type ConfigDAO struct {
	Tables *cfg.Tables
}

func NewConfigDAO(a *app.BaseApp) (*ConfigDAO, error) {
	// 1. 获取执行目录
	execDir, err := app.GetExecDir() 
	if err != nil {
		return nil, err
	}

	// 2. 确定 JSON 数据目录 (默认在 configs/data)
	dataDir := filepath.Join(execDir, "configs", "data")

	// 3. 获取应用主日志对象
	l := a.AppLogger()

	// 4. 创建加载器并初始化
	loader, err := cfg.NewFileJsonLoader(dataDir, l)
	if err != nil {
		return nil, err
	}
	tables, err := cfg.NewTables(loader)
	if err != nil {
		return nil, fmt.Errorf("failed to load luban tables: %w", err)
	}

	return &ConfigDAO{
		Tables: tables,
	}, nil
}
