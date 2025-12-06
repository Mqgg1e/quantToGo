package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HistoricalKlineRequest REST API获取历史K线的请求参数
type HistoricalKlineRequest struct {
	Symbol    string
	Interval  string
	Limit     int    // 获取的K线数量（最大1500）
	StartTime *int64 // 可选：开始时间戳（毫秒）
	EndTime   *int64 // 可选：结束时间戳（毫秒）
	ProxyURL  string // 可选：代理地址
}

// GetHistoricalKlines 从币安REST API获取历史K线数据
// 用于策略预热，避免等待WebSocket数据
func GetHistoricalKlines(ctx context.Context, req HistoricalKlineRequest) ([]*KlineData, error) {
	// 构建请求URL
	baseURL := "https://fapi.binance.com/fapi/v1/klines"
	params := url.Values{}
	params.Add("symbol", req.Symbol)
	params.Add("interval", req.Interval)

	if req.Limit > 0 {
		if req.Limit > 1500 {
			req.Limit = 1500 // 币安限制
		}
		params.Add("limit", strconv.Itoa(req.Limit))
	} else {
		params.Add("limit", "500") // 默认500根
	}

	if req.StartTime != nil {
		params.Add("startTime", strconv.FormatInt(*req.StartTime, 10))
	}

	if req.EndTime != nil {
		params.Add("endTime", strconv.FormatInt(*req.EndTime, 10))
	}

	fullURL := baseURL + "?" + params.Encode()

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 如果有代理，设置代理
	if req.ProxyURL != "" {
		proxyURL, err := url.Parse(req.ProxyURL)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL: %w", err)
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	// 创建请求
	httpReq, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 发送请求
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// 解析响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 币安返回的是数组的数组
	// [[openTime, open, high, low, close, volume, closeTime, ...], ...]
	var rawKlines [][]interface{}
	if err := json.Unmarshal(body, &rawKlines); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// 转换为KlineData
	klines := make([]*KlineData, 0, len(rawKlines))
	for _, raw := range rawKlines {
		if len(raw) < 12 {
			continue
		}

		kline, err := parseRawKline(raw, req.Symbol, req.Interval)
		if err != nil {
			return nil, fmt.Errorf("parse kline: %w", err)
		}

		klines = append(klines, kline)
	}

	return klines, nil
}

// parseRawKline 解析币安API返回的原始K线数据
func parseRawKline(raw []interface{}, symbol, interval string) (*KlineData, error) {
	// 辅助函数：安全转换
	toInt64 := func(v interface{}) (int64, error) {
		switch val := v.(type) {
		case float64:
			return int64(val), nil
		case int64:
			return val, nil
		case string:
			return strconv.ParseInt(val, 10, 64)
		default:
			return 0, fmt.Errorf("cannot convert %T to int64", v)
		}
	}

	toFloat64 := func(v interface{}) (float64, error) {
		switch val := v.(type) {
		case float64:
			return val, nil
		case string:
			return strconv.ParseFloat(val, 64)
		default:
			return 0, fmt.Errorf("cannot convert %T to float64", v)
		}
	}

	// 解析字段
	openTime, err := toInt64(raw[0])
	if err != nil {
		return nil, fmt.Errorf("parse openTime: %w", err)
	}

	open, err := toFloat64(raw[1])
	if err != nil {
		return nil, fmt.Errorf("parse open: %w", err)
	}

	high, err := toFloat64(raw[2])
	if err != nil {
		return nil, fmt.Errorf("parse high: %w", err)
	}

	low, err := toFloat64(raw[3])
	if err != nil {
		return nil, fmt.Errorf("parse low: %w", err)
	}

	closePrice, err := toFloat64(raw[4])
	if err != nil {
		return nil, fmt.Errorf("parse close: %w", err)
	}

	volume, err := toFloat64(raw[5])
	if err != nil {
		return nil, fmt.Errorf("parse volume: %w", err)
	}

	closeTime, err := toInt64(raw[6])
	if err != nil {
		return nil, fmt.Errorf("parse closeTime: %w", err)
	}

	quoteVolume, err := toFloat64(raw[7])
	if err != nil {
		return nil, fmt.Errorf("parse quoteVolume: %w", err)
	}

	trades, err := toInt64(raw[8])
	if err != nil {
		return nil, fmt.Errorf("parse trades: %w", err)
	}

	takerBuyBaseVolume, err := toFloat64(raw[9])
	if err != nil {
		return nil, fmt.Errorf("parse takerBuyBaseVolume: %w", err)
	}

	takerBuyQuoteVolume, err := toFloat64(raw[10])
	if err != nil {
		return nil, fmt.Errorf("parse takerBuyQuoteVolume: %w", err)
	}

	// 创建KlineData
	kline := &KlineData{
		Symbol:              symbol,
		Interval:            interval,
		StartTime:           openTime,
		CloseTime:           closeTime,
		OpenPrice:           open,
		HighPrice:           high,
		LowPrice:            low,
		ClosePrice:          closePrice,
		BaseVolume:          volume,
		QuoteVolume:         quoteVolume,
		TradeNum:            int(trades),
		TakerBuyBaseVolume:  takerBuyBaseVolume,
		TakerBuyQuoteVolume: takerBuyQuoteVolume,
		isClosedField:       true, // REST API返回的都是已关闭的K线
	}

	return kline, nil
}

// WarmupStrategy 使用历史K线预热策略
// 返回预热使用的K线数量
func WarmupStrategy(ctx context.Context, symbol, interval string, requiredKlines int, proxyURL string) ([]*KlineData, error) {
	// 获取历史K线
	req := HistoricalKlineRequest{
		Symbol:   symbol,
		Interval: interval,
		Limit:    requiredKlines,
		ProxyURL: proxyURL,
	}

	klines, err := GetHistoricalKlines(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get historical klines: %w", err)
	}

	if len(klines) < requiredKlines {
		return nil, fmt.Errorf("insufficient klines: got %d, need %d", len(klines), requiredKlines)
	}

	return klines, nil
}
