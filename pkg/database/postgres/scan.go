package postgres

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5"
)

// 字段映射缓存
var (
	structCache   = make(map[reflect.Type]*structInfo)
	structCacheMu sync.RWMutex
)

// structInfo 结构体信息
type structInfo struct {
	fields []*fieldInfo
}

// fieldInfo 字段信息
type fieldInfo struct {
	name  string // 数据库列名
	index int    // 结构体字段索引
}

// getStructInfo 获取结构体信息（带缓存）
func getStructInfo(t reflect.Type) *structInfo {
	structCacheMu.RLock()
	info, ok := structCache[t]
	structCacheMu.RUnlock()

	if ok {
		return info
	}

	structCacheMu.Lock()
	defer structCacheMu.Unlock()

	// 双重检查
	if info, ok := structCache[t]; ok {
		return info
	}

	info = &structInfo{
		fields: make([]*fieldInfo, 0, t.NumField()),
	}

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		// 跳过未导出的字段
		if !field.IsExported() {
			continue
		}

		// 获取数据库列名
		dbTag := field.Tag.Get("db")
		if dbTag == "-" {
			continue
		}

		columnName := dbTag
		if columnName == "" {
			// 如果没有 db tag，使用 snake_case 转换
			columnName = toSnakeCase(field.Name)
		}

		info.fields = append(info.fields, &fieldInfo{
			name:  columnName,
			index: i,
		})
	}

	structCache[t] = info
	return info
}

// scanOne 扫描单条记录到结构体
func scanOne[T any](rows pgx.Rows) (*T, error) {
	if !rows.Next() {
		return nil, ErrNoRows
	}

	var result T
	if err := scanStruct(rows, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// scanAll 扫描多条记录到结构体切片
func scanAll[T any](rows pgx.Rows) ([]*T, error) {
	results := make([]*T, 0)

	for rows.Next() {
		var item T
		if err := scanStruct(rows, &item); err != nil {
			return nil, err
		}
		results = append(results, &item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

// scanStruct 扫描当前行到结构体
func scanStruct(rows pgx.Rows, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return fmt.Errorf("dest must be a pointer to struct")
	}

	t := v.Type()
	info := getStructInfo(t)

	// 获取列名
	fieldDescriptions := rows.FieldDescriptions()
	columnCount := len(fieldDescriptions)

	// 创建扫描目标
	values := make([]any, columnCount)
	columnMap := make(map[string]int) // 列名 -> 索引

	for i, fd := range fieldDescriptions {
		columnMap[string(fd.Name)] = i
	}

	// 映射字段到列
	for _, field := range info.fields {
		if colIdx, ok := columnMap[field.name]; ok {
			fieldValue := v.Field(field.index)
			values[colIdx] = fieldValue.Addr().Interface()
		}
	}

	// 填充未映射的列（使用占位符）
	for i := range values {
		if values[i] == nil {
			var placeholder any
			values[i] = &placeholder
		}
	}

	return rows.Scan(values...)
}

// scanRowToStruct 扫描当前行到结构体（用于 Tx.QueryOne）
func scanRowToStruct(rows pgx.Rows, dest any) error {
	return scanStruct(rows, dest)
}

// scanRowsToSlice 扫描所有行到切片（用于 Tx.QueryAll）
func scanRowsToSlice(rows pgx.Rows, dest any) error {
	v := reflect.ValueOf(dest)
	if v.Kind() != reflect.Ptr {
		return fmt.Errorf("dest must be a pointer")
	}

	v = v.Elem()
	if v.Kind() != reflect.Slice {
		return fmt.Errorf("dest must be a pointer to slice")
	}

	// 获取切片元素类型
	elemType := v.Type().Elem()
	if elemType.Kind() != reflect.Ptr {
		return fmt.Errorf("slice element must be pointer type")
	}
	structType := elemType.Elem()

	for rows.Next() {
		// 创建新的结构体实例
		item := reflect.New(structType)
		if err := scanStruct(rows, item.Interface()); err != nil {
			return err
		}
		v.Set(reflect.Append(v, item))
	}

	return rows.Err()
}

// toSnakeCase 将驼峰命名转换为蛇形命名
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('_')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}
