package v2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

// CompletionChecker 检查K线数据的完整性并调用REST API补全
type CompletionChecker struct {
	symbol   string
	interval string
	proxyURL string
	client   *http.Client
}

// NewCompletionChecker 创建完整性检查器
func NewCompletionChecker(symbol, interval, proxyURL string) *CompletionChecker {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 如果提供了代理，设置HTTP代理
	if proxyURL != "" {
		if proxyURLParsed, err := url.Parse(proxyURL); err == nil {
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURLParsed),
			}
		}
	}

	return &CompletionChecker{
		symbol:   symbol,
		interval: interval,
		proxyURL: proxyURL,
		client:   client,
	}
}

// CheckAndFill 检查K线完整性，如果有丢包则调用REST API补全
// lastKline: 已保存的最新K线
// newKline: 新接收到的K线
// 返回: 补全的K线列表（不包括lastKline和newKline）
func (cc *CompletionChecker) CheckAndFill(lastKline, newKline *KlineData) ([]*KlineData, error) {
	if lastKline == nil {
		return nil, nil // 第一条K线，无需检查
	}

	// 计算应该的时间间隔（毫秒）
	expectedInterval := cc.getIntervalMillis()
	actualGap := newKline.CloseTime - lastKline.CloseTime

	// 检查是否有丢包
	if actualGap > expectedInterval {
		fmt.Printf("[%s %s] Detected missing klines! Gap: %dms, Expected: %dms\n",
			cc.symbol, cc.interval, actualGap, expectedInterval)

		// 计算丢失的K线数量
		missingCount := (actualGap / expectedInterval) - 1

		if missingCount > 100 {
			fmt.Printf("[%s %s] Missing %d klines, too many to fill. Skipping REST fill.\n",
				cc.symbol, cc.interval, missingCount)
			return nil, nil
		}

		// 调用REST API获取缺失的K线
		return cc.fillMissingKlines(lastKline.CloseTime, newKline.StartTime)
	}

	return nil, nil
}

// fillMissingKlines 通过REST API获取缺失的K线
func (cc *CompletionChecker) fillMissingKlines(lastCloseTime, nextStartTime int64) ([]*KlineData, error) {
	// Binance REST API: https://api.binance.com/api/v3/klines
	// 参数: symbol, interval, startTime, endTime, limit

	startTime := lastCloseTime + 1 // 从最后一条K线之后开始
	endTime := nextStartTime - 1   // 到下一条K线之前结束

	fmt.Printf("[%s %s] Fetching missing klines via REST API: %d - %d\n",
		cc.symbol, cc.interval, startTime, endTime)

	endpoint := "https://fapi.binance.com/fapi/v1/klines"

	// 构建查询参数
	query := url.Values{}
	query.Set("symbol", cc.symbol)
	query.Set("interval", cc.interval)
	query.Set("startTime", fmt.Sprintf("%d", startTime))
	query.Set("endTime", fmt.Sprintf("%d", endTime))
	query.Set("limit", "1000")

	fullURL := endpoint + "?" + query.Encode()

	// 发送请求
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := cc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch klines: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析REST API返回的K线数据
	var restKlines [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&restKlines); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 转换为KlineData结构
	var klines []*KlineData
	for _, rawKline := range restKlines {
		kline, err := cc.parseRESTKline(rawKline)
		if err != nil {
			fmt.Printf("[%s %s] Failed to parse REST kline: %v\n", cc.symbol, cc.interval, err)
			continue
		}
		klines = append(klines, kline)
	}

	fmt.Printf("[%s %s] ✓ Filled %d missing klines via REST API\n", cc.symbol, cc.interval, len(klines))
	return klines, nil
}

// parseRESTKline 解析REST API返回的单条K线数据
// REST API返回格式: [openTime, open, high, low, close, volume, closeTime, quoteVolume, ...]
func (cc *CompletionChecker) parseRESTKline(rawKline []interface{}) (*KlineData, error) {
	if len(rawKline) < 8 {
		return nil, fmt.Errorf("invalid kline format")
	}

	// 类型转换
	openTime := int64(rawKline[0].(float64))
	closeTime := int64(rawKline[6].(float64))
	open := toFloat64(rawKline[1])
	high := toFloat64(rawKline[2])
	low := toFloat64(rawKline[3])
	close := toFloat64(rawKline[4])
	volume := toFloat64(rawKline[5])
	quoteVolume := toFloat64(rawKline[7])

	return &KlineData{
		EventType:     "kline",
		EventTime:     closeTime,
		Symbol:        cc.symbol,
		StartTime:     openTime,
		CloseTime:     closeTime,
		Interval:      cc.interval,
		OpenPrice:     open,
		ClosePrice:    close,
		HighPrice:     high,
		LowPrice:      low,
		BaseVolume:    volume,
		QuoteVolume:   quoteVolume,
		isClosedField: true,
	}, nil
}

// getIntervalMillis 获取时间间隔对应的毫秒数
func (cc *CompletionChecker) getIntervalMillis() int64 {
	intervalMap := map[string]int64{
		"1s":  1000,
		"1m":  60 * 1000,
		"3m":  3 * 60 * 1000,
		"5m":  5 * 60 * 1000,
		"15m": 15 * 60 * 1000,
		"30m": 30 * 60 * 1000,
		"1h":  60 * 60 * 1000,
		"2h":  2 * 60 * 60 * 1000,
		"4h":  4 * 60 * 60 * 1000,
		"6h":  6 * 60 * 60 * 1000,
		"8h":  8 * 60 * 60 * 1000,
		"12h": 12 * 60 * 60 * 1000,
		"1d":  24 * 60 * 60 * 1000,
		"3d":  3 * 24 * 60 * 60 * 1000,
		"1w":  7 * 24 * 60 * 60 * 1000,
		"1M":  30 * 24 * 60 * 60 * 1000,
	}

	if interval, exists := intervalMap[cc.interval]; exists {
		return interval
	}

	// 默认返回1分钟
	return 60 * 1000
}
