package v2

import "time"

// KlineData 表示一条K线数据
type KlineData struct {
	EventType     string  `json:"e"`        // Event type: "kline"
	EventTime     int64   `json:"E"`        // Event time (milliseconds)
	Symbol        string  `json:"s"`        // Symbol: "BTCUSDT"
	StartTime     int64   `json:"t"`        // Kline start time (milliseconds)
	CloseTime     int64   `json:"T"`        // Kline close time (milliseconds)
	Interval      string  `json:"i"`        // Interval: "1m", "5m", etc.
	OpenPrice     float64 `json:"o,string"` // Open price
	ClosePrice    float64 `json:"c,string"` // Close price
	HighPrice     float64 `json:"h,string"` // High price
	LowPrice      float64 `json:"l,string"` // Low price
	BaseVolume    float64 `json:"v,string"` // Base asset volume
	QuoteVolume   float64 `json:"q,string"` // Quote asset volume
	isClosedField bool    `json:"x"`        // Is kline closed (internal field)

	// REST API额外字段
	TradeNum            int     // Number of trades
	TakerBuyBaseVolume  float64 // Taker buy base asset volume
	TakerBuyQuoteVolume float64 // Taker buy quote asset volume
}

// ParseKlineEvent 从WebSocket消息中解析K线数据
// 支持数字和字符串格式的价格和成交量字段
func ParseKlineEvent(data []byte) (*KlineData, error) {
	var rawMsg map[string]interface{}
	if err := parseJSON(data, &rawMsg); err != nil {
		return nil, err
	}

	k := rawMsg["k"].(map[string]interface{})

	kline := &KlineData{
		EventType:     rawMsg["e"].(string),
		EventTime:     int64(rawMsg["E"].(float64)),
		Symbol:        rawMsg["s"].(string),
		StartTime:     int64(k["t"].(float64)),
		CloseTime:     int64(k["T"].(float64)),
		Interval:      k["i"].(string),
		OpenPrice:     toFloat64(k["o"]),
		ClosePrice:    toFloat64(k["c"]),
		HighPrice:     toFloat64(k["h"]),
		LowPrice:      toFloat64(k["l"]),
		BaseVolume:    toFloat64(k["v"]),
		QuoteVolume:   toFloat64(k["q"]),
		isClosedField: k["x"].(bool),
	}

	return kline, nil
}

// ClosedAtTime 返回K线收盘时间
func (k *KlineData) ClosedAtTime() time.Time {
	return time.UnixMilli(k.CloseTime)
}

// StartAtTime 返回K线开盘时间
func (k *KlineData) StartAtTime() time.Time {
	return time.UnixMilli(k.StartTime)
}

// ========== 实现 core.KlineData 接口 ==========

// GetSymbol 实现 core.KlineData 接口
func (k *KlineData) GetSymbol() string {
	return k.Symbol
}

// GetInterval 实现 core.KlineData 接口
func (k *KlineData) GetInterval() string {
	return k.Interval
}

// GetStartTime 实现 core.KlineData 接口
func (k *KlineData) GetStartTime() time.Time {
	return k.StartAtTime()
}

// GetCloseTime 实现 core.KlineData 接口
func (k *KlineData) GetCloseTime() time.Time {
	return k.ClosedAtTime()
}

// GetOpenPrice 实现 core.KlineData 接口
func (k *KlineData) GetOpenPrice() float64 {
	return k.OpenPrice
}

// GetClosePrice 实现 core.KlineData 接口
func (k *KlineData) GetClosePrice() float64 {
	return k.ClosePrice
}

// GetHighPrice 实现 core.KlineData 接口
func (k *KlineData) GetHighPrice() float64 {
	return k.HighPrice
}

// GetLowPrice 实现 core.KlineData 接口
func (k *KlineData) GetLowPrice() float64 {
	return k.LowPrice
}

// GetVolume 实现 core.KlineData 接口
func (k *KlineData) GetVolume() float64 {
	return k.BaseVolume
}

// IsClosed 实现 core.KlineData 接口
func (k *KlineData) IsClosed() bool {
	return k.isClosedField
}
