package cfg

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/lk2023060901/xdooria/pkg/logger"
)

// NewFileJsonLoader 创建一个智能的本地文件 JSON 加载器
func NewFileJsonLoader(dataDir string, l logger.Logger) (JsonLoader, error) {
	if l == nil {
		return nil, fmt.Errorf("logger is required for NewFileJsonLoader")
	}

	// 提前通过反射分析 Tables 结构体，识别哪些表是单例（One 模式）
	singletonTables := identifySingletonTables()

	return func(tableName string) ([]map[string]interface{}, error) {
		// Luban 传入的 tableName 通常是小写的
		fileName := fmt.Sprintf("%s.json", tableName)
		filePath := filepath.Join(dataDir, fileName)

		if _, err := os.Stat(filePath); err != nil {
			if os.IsNotExist(err) {
				l.Warn("optional config file not found, initializing as empty", 
					"table", tableName, 
					"path", filePath)
				
				// 智能判断：如果是单例表，必须返回包含一个空对象的数组 [{}] 以满足 size == 1 的约束
				if singletonTables[strings.ToLower(tableName)] {
					return []map[string]interface{}{{}}, nil
				}
				// 否则（Map/List模式），返回空数组 []
				return []map[string]interface{}{}, nil
			}
			return nil, fmt.Errorf("failed to stat config file %s: %w", filePath, err)
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file %s: %w", filePath, err)
		}

		var res []map[string]interface{}
		if err := json.Unmarshal(data, &res); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config file %s: %w", filePath, err)
		}

		return res, nil
	}, nil
}

// identifySingletonTables 利用反射自动探测 Tables 中哪些表是单例模式
func identifySingletonTables() map[string]bool {
	singletons := make(map[string]bool)
	t := reflect.TypeOf(Tables{})
	
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// 所有的表字段名通常是 TbAccount, TbGlobalConfig 这种形式
		tableName := strings.ToLower(field.Name)
		
		// 检查该字段类型（如 *TbGlobalConfig）是否拥有一个无参数的 Get 方法
		// 这是 Luban 单例模式生成的唯一特征
		if method, ok := field.Type.MethodByName("Get"); ok {
			// Method.Type.NumIn() 包括了接收者(Receiver)，所以单例模式的参数个数应为 1
			if method.Type.NumIn() == 1 {
				singletons[tableName] = true
			}
		}
	}
	return singletons
}