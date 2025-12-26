package config

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// MergeConfig 合并配置
// - 如果 dst 和 src 都为 nil，返回错误
// - 如果 dst 为 nil，返回 src
// - 如果 src 为 nil，返回 dst
// - 如果都不为 nil，src 的值覆盖 dst 的值，返回合并后的 dst
func MergeConfig[T any](dst, src *T) (*T, error) {
	// 两者都为 nil 才报错
	if dst == nil && src == nil {
		return nil, fmt.Errorf("both dst and src cannot be nil")
	}

	// dst 为 nil，返回 src
	if dst == nil {
		return src, nil
	}

	// src 为 nil，返回 dst
	if src == nil {
		return dst, nil
	}

	// 两者都不为 nil，执行深度合并
	dstValue := reflect.ValueOf(dst).Elem()
	srcValue := reflect.ValueOf(src).Elem()

	if err := mergeValues(dstValue, srcValue); err != nil {
		return nil, err
	}

	return dst, nil
}

// mergeValues 递归合并两个 reflect.Value
func mergeValues(dst, src reflect.Value) error {
	// 如果 src 是零值，不进行覆盖
	if !src.IsValid() || isZeroValue(src) {
		return nil
	}

	switch dst.Kind() {
	case reflect.Struct:
		return mergeStruct(dst, src)
	case reflect.Map:
		return mergeMap(dst, src)
	case reflect.Slice:
		return mergeSlice(dst, src)
	case reflect.Ptr:
		return mergePointer(dst, src)
	default:
		// 基本类型直接覆盖
		if dst.CanSet() {
			dst.Set(src)
		}
		return nil
	}
}

// mergeStruct 合并结构体
func mergeStruct(dst, src reflect.Value) error {
	if src.Kind() != reflect.Struct {
		return fmt.Errorf("src is not a struct")
	}

	srcType := src.Type()
	for i := 0; i < src.NumField(); i++ {
		srcField := src.Field(i)
		fieldType := srcType.Field(i)

		// 跳过未导出的字段
		if !fieldType.IsExported() {
			continue
		}

		dstField := dst.FieldByName(fieldType.Name)
		if !dstField.IsValid() || !dstField.CanSet() {
			continue
		}

		// 递归合并字段
		if err := mergeValues(dstField, srcField); err != nil {
			return fmt.Errorf("failed to merge field %s: %w", fieldType.Name, err)
		}
	}

	return nil
}

// mergeMap 合并 map
func mergeMap(dst, src reflect.Value) error {
	if src.Kind() != reflect.Map {
		return fmt.Errorf("src is not a map")
	}

	if dst.IsNil() {
		dst.Set(reflect.MakeMap(dst.Type()))
	}

	// 遍历 src 的所有键值对
	iter := src.MapRange()
	for iter.Next() {
		key := iter.Key()
		srcValue := iter.Value()

		// 获取 dst 中对应的值
		dstValue := dst.MapIndex(key)

		if dstValue.IsValid() {
			// 如果 dst 中已存在该 key，递归合并
			newValue := reflect.New(dst.Type().Elem()).Elem()
			newValue.Set(dstValue)

			if err := mergeValues(newValue, srcValue); err != nil {
				return err
			}

			dst.SetMapIndex(key, newValue)
		} else {
			// 如果 dst 中不存在该 key，直接设置
			dst.SetMapIndex(key, srcValue)
		}
	}

	return nil
}

// mergeSlice 合并切片（src 覆盖 dst）
func mergeSlice(dst, src reflect.Value) error {
	if src.Kind() != reflect.Slice {
		return fmt.Errorf("src is not a slice")
	}

	// 切片直接覆盖（不做元素级合并）
	if dst.CanSet() {
		dst.Set(src)
	}

	return nil
}

// mergePointer 合并指针
func mergePointer(dst, src reflect.Value) error {
	if src.Kind() != reflect.Ptr {
		return fmt.Errorf("src is not a pointer")
	}

	if src.IsNil() {
		return nil
	}

	if dst.IsNil() {
		// 创建新的实例
		dst.Set(reflect.New(dst.Type().Elem()))
	}

	// 递归合并指针指向的值
	return mergeValues(dst.Elem(), src.Elem())
}

// isZeroValue 检查是否为零值
func isZeroValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.String:
		return v.String() == ""
	case reflect.Ptr, reflect.Interface, reflect.Func:
		return v.IsNil()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	case reflect.Struct:
		// 结构体需要检查所有字段
		zero := reflect.Zero(v.Type()).Interface()
		return reflect.DeepEqual(v.Interface(), zero)
	default:
		return false
	}
}

// MergeConfigJSON 使用 JSON 序列化方式合并配置（备选方案，性能较低但更安全）
// - 如果 dst 和 src 都为 nil，返回错误
// - 如果 dst 为 nil，返回 src
// - 如果 src 为 nil，返回 dst
// - 如果都不为 nil，src 的值覆盖 dst 的值，返回合并后的 dst
func MergeConfigJSON[T any](dst, src *T) (*T, error) {
	// 两者都为 nil 才报错
	if dst == nil && src == nil {
		return nil, fmt.Errorf("both dst and src cannot be nil")
	}

	// dst 为 nil，返回 src
	if dst == nil {
		return src, nil
	}

	// src 为 nil，返回 dst
	if src == nil {
		return dst, nil
	}

	// 将 dst 序列化为 JSON
	dstBytes, err := json.Marshal(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal dst: %w", err)
	}

	// 将 src 序列化为 JSON
	srcBytes, err := json.Marshal(src)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal src: %w", err)
	}

	// 将两个 JSON 合并
	var dstMap map[string]interface{}
	var srcMap map[string]interface{}

	if err := json.Unmarshal(dstBytes, &dstMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dst: %w", err)
	}

	if err := json.Unmarshal(srcBytes, &srcMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal src: %w", err)
	}

	// 合并 map
	mergedMap := mergeMaps(dstMap, srcMap)

	// 序列化回结构体
	mergedBytes, err := json.Marshal(mergedMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal merged: %w", err)
	}

	var result T
	if err := json.Unmarshal(mergedBytes, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal result: %w", err)
	}

	*dst = result
	return dst, nil
}

// mergeMaps 合并两个 map[string]interface{}
func mergeMaps(dst, src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// 复制 dst
	for k, v := range dst {
		result[k] = v
	}

	// 用 src 覆盖
	for k, v := range src {
		if dstValue, exists := result[k]; exists {
			// 如果两者都是 map，递归合并
			if dstMap, ok := dstValue.(map[string]interface{}); ok {
				if srcMap, ok := v.(map[string]interface{}); ok {
					result[k] = mergeMaps(dstMap, srcMap)
					continue
				}
			}
		}
		// 否则直接覆盖
		result[k] = v
	}

	return result
}
