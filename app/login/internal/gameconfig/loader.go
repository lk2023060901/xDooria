package gameconfig

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

var (
	// T 是全局配置表实例，加载后可在任何地方直接引用
	T *Tables
)

// Load 初始化全局配置表实例
func Load(dataDir string, l logger.Logger) error {
	loader, err := NewFileJsonLoader(dataDir, l)
	if err != nil {
		return err
	}

	tables, err := NewTables(loader)
	if err != nil {
		return err
	}

	T = tables
	return nil
}

// NewFileJsonLoader 创建一个简单的本地文件 JSON 加载器
func NewFileJsonLoader(dataDir string, l logger.Logger) (JsonLoader, error) {
	if l == nil {
		return nil, fmt.Errorf("logger is required for NewFileJsonLoader")
	}

	return func(tableName string) ([]map[string]interface{}, error) {
		tableNameLower := strings.ToLower(tableName)
		filePath := filepath.Join(dataDir, fmt.Sprintf("%s.json", tableNameLower))

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read required config file %s: %w", filePath, err)
		}

		var res []map[string]interface{}
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config file %s: %w", filePath, err)
		}

		l.Debug("config file loaded successfully", "table", tableName, "path", filePath, "records", len(res))
		return res, nil
	}, nil
}
