package binance

import (
	"strconv"
)

// parseFloat 安全地将字符串转换为float64
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.ParseFloat(s, 64)
}

// parseInt 安全地将字符串转换为int
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}

// formatFloat 将float64格式化为字符串（去除尾随零）
func formatFloat(f float64, precision int) string {
	return strconv.FormatFloat(f, 'f', precision, 64)
}

// formatQuantity 格式化数量（根据交易对精度）
func formatQuantity(symbol string, quantity float64) string {
	// 简化版本：使用固定精度
	// 实际应该根据交易规则动态确定精度
	switch symbol {
	case "BTCUSDT", "ETHUSDT":
		return formatFloat(quantity, 3)
	default:
		return formatFloat(quantity, 3)
	}
}

// formatPrice 格式化价格（根据交易对精度）
func formatPrice(symbol string, price float64) string {
	// 简化版本：使用固定精度
	// 实际应该根据交易规则动态确定精度
	switch symbol {
	case "BTCUSDT":
		return formatFloat(price, 2)
	case "ETHUSDT":
		return formatFloat(price, 2)
	default:
		return formatFloat(price, 2)
	}
}
