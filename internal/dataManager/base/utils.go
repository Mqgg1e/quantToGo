package base

import (
	"encoding/json"
	"strconv"
)

// parseJSON 封装JSON解析
func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// stringToFloat64 将字符串转换为float64
func stringToFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// toFloat64 将interface{}转换为float64，支持数字和字符串
func toFloat64(v interface{}) float64 {
	if v == nil {
		return 0
	}

	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		return stringToFloat64(val)
	default:
		return 0
	}
}
