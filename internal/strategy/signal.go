package strategy

import (
	"fmt"
	"time"

	"goQuant/internal/core"
)

// SignalType 信号类型别名（便于使用）
type SignalType = core.SignalType

const (
	SignalTypeOpenLong   = core.SignalTypeOpenLong
	SignalTypeOpenShort  = core.SignalTypeOpenShort
	SignalTypeAddLong    = core.SignalTypeAddLong
	SignalTypeAddShort   = core.SignalTypeAddShort
	SignalTypeCloseLong  = core.SignalTypeCloseLong
	SignalTypeCloseShort = core.SignalTypeCloseShort
	SignalTypeNoAction   = core.SignalTypeNoAction
)

// TradingSignal 交易信号别名
type TradingSignal = core.TradingSignal

// NewSignal 创建交易信号的辅助函数
func NewSignal(signalType SignalType, symbol string, price float64, reason string) *TradingSignal {
	return &TradingSignal{
		Type:        signalType,
		Symbol:      symbol,
		Timestamp:   time.Now(),
		Price:       price,
		Metadata:    make(map[string]float64),
		Reason:      reason,
		AddPosition: false,
	}
}

// NoActionSignal 创建无操作信号
func NoActionSignal(symbol string) *TradingSignal {
	return NewSignal(SignalTypeNoAction, symbol, 0, "no action")
}

// WithMetadata 为信号添加元数据（辅助函数）
func WithMetadata(s *TradingSignal, key string, value float64) *TradingSignal {
	if s.Metadata == nil {
		s.Metadata = make(map[string]float64)
	}
	s.Metadata[key] = value
	return s
}

// WithConfidence 设置信号置信度（辅助函数）
//func WithConfidence(s *TradingSignal, confidence float64) *TradingSignal {
//	s.Confidence = confidence
//	return s
//}

// IsEntry 判断是否为入场信号
func IsEntry(s *TradingSignal) bool {
	return s.Type == SignalTypeOpenLong || s.Type == SignalTypeOpenShort
}

// IsExit 判断是否为离场信号
func IsExit(s *TradingSignal) bool {
	return s.Type == SignalTypeCloseLong || s.Type == SignalTypeCloseShort
}

// IsAddPosition 判断是否为加仓信号
func IsAddPosition(s *TradingSignal) bool {
	return s.Type == SignalTypeAddLong || s.Type == SignalTypeAddShort
}

// IsLong 判断是否为多头信号
func IsLong(s *TradingSignal) bool {
	return s.Type == SignalTypeOpenLong || s.Type == SignalTypeAddLong
}

// IsShort 判断是否为空头信号
func IsShort(s *TradingSignal) bool {
	return s.Type == SignalTypeOpenShort || s.Type == SignalTypeAddShort
}

// SignalString 返回信号的字符串表示
func SignalString(s *TradingSignal) string {
	return fmt.Sprintf("[%s] %s %s @ %.4f - %s (conf: %.2f)",
		s.Timestamp.Format("15:04:05"),
		s.Symbol,
		s.Type,
		s.Price,
		s.Reason,
		s.AddPosition,
	)
}

// CrossType 交叉类型
type CrossType int

const (
	CrossTypeNone   CrossType = 0  // 无交叉
	CrossTypeGolden CrossType = 1  // 金叉（快线上穿慢线）
	CrossTypeDeath  CrossType = -1 // 死叉（快线下穿慢线）
)

// DetectCross 检测交叉
// prev1, prev2: 前一周期的快线和慢线值
// curr1, curr2: 当前周期的快线和慢线值
func DetectCross(prev1, prev2, curr1, curr2 float64) CrossType {
	// 前一周期：快线 < 慢线，当前周期：快线 > 慢线 => 金叉
	if prev1 < prev2 && curr1 > curr2 {
		return CrossTypeGolden
	}
	// 前一周期：快线 > 慢线，当前周期：快线 < 慢线 => 死叉
	if prev1 > prev2 && curr1 < curr2 {
		return CrossTypeDeath
	}
	return CrossTypeNone
}

// TrendDirection 趋势方向
type TrendDirection int

const (
	TrendNone TrendDirection = 0  // 无明显趋势
	TrendUp   TrendDirection = 1  // 上升趋势
	TrendDown TrendDirection = -1 // 下降趋势
)

// DetectTrend 检测连续趋势
// prices: 价格序列（从旧到新）
// periods: 需要连续的周期数
// returns: 趋势方向和涨跌幅
func DetectTrend(prices []float64, periods int) (TrendDirection, float64) {
	if len(prices) < periods {
		return TrendNone, 0
	}

	// 取最近N个周期
	recent := prices[len(prices)-periods:]

	// 检查是否连续上涨
	isUpTrend := true
	for i := 1; i < len(recent); i++ {
		if recent[i] <= recent[i-1] {
			isUpTrend = false
			break
		}
	}

	// 检查是否连续下跌
	isDownTrend := true
	for i := 1; i < len(recent); i++ {
		if recent[i] >= recent[i-1] {
			isDownTrend = false
			break
		}
	}

	// 计算涨跌幅
	changePercent := (recent[len(recent)-1] - recent[0]) / recent[0]

	if isUpTrend {
		return TrendUp, changePercent
	}
	if isDownTrend {
		return TrendDown, changePercent
	}

	return TrendNone, changePercent
}

// KlineBuffer K线缓冲区（环形缓冲区，用于高效存储历史K线）
type KlineBuffer struct {
	data     []core.KlineData
	capacity int
	size     int
	head     int // 最新数据位置
}

// NewKlineBuffer 创建K线缓冲区
func NewKlineBuffer(capacity int) *KlineBuffer {
	return &KlineBuffer{
		data:     make([]core.KlineData, capacity),
		capacity: capacity,
		size:     0,
		head:     -1,
	}
}

// Add 添加K线数据
func (kb *KlineBuffer) Add(kline core.KlineData) {
	kb.head = (kb.head + 1) % kb.capacity
	kb.data[kb.head] = kline
	if kb.size < kb.capacity {
		kb.size++
	}
}

// Get 获取倒数第n个K线（0表示最新）
func (kb *KlineBuffer) Get(n int) (core.KlineData, bool) {
	if n >= kb.size {
		return nil, false
	}
	idx := (kb.head - n + kb.capacity) % kb.capacity
	return kb.data[idx], true
}

// GetAll 获取所有K线（从旧到新）
func (kb *KlineBuffer) GetAll() []core.KlineData {
	if kb.size == 0 {
		return nil
	}

	result := make([]core.KlineData, kb.size)
	for i := 0; i < kb.size; i++ {
		idx := (kb.head - kb.size + 1 + i + kb.capacity) % kb.capacity
		result[i] = kb.data[idx]
	}
	return result
}

// Size 返回当前缓冲区大小
func (kb *KlineBuffer) Size() int {
	return kb.size
}

// IsFull 判断缓冲区是否已满
func (kb *KlineBuffer) IsFull() bool {
	return kb.size == kb.capacity
}

// GetClosePrices 获取所有收盘价（从旧到新）
func (kb *KlineBuffer) GetClosePrices() []float64 {
	if kb.size == 0 {
		return nil
	}

	prices := make([]float64, kb.size)
	for i := 0; i < kb.size; i++ {
		idx := (kb.head - kb.size + 1 + i + kb.capacity) % kb.capacity
		prices[i] = kb.data[idx].GetClosePrice()
	}
	return prices
}

// GetHighPrices 获取所有最高价（从旧到新）
func (kb *KlineBuffer) GetHighPrices() []float64 {
	if kb.size == 0 {
		return nil
	}

	prices := make([]float64, kb.size)
	for i := 0; i < kb.size; i++ {
		idx := (kb.head - kb.size + 1 + i + kb.capacity) % kb.capacity
		prices[i] = kb.data[idx].GetHighPrice()
	}
	return prices
}

// GetLowPrices 获取所有最低价（从旧到新）
func (kb *KlineBuffer) GetLowPrices() []float64 {
	if kb.size == 0 {
		return nil
	}

	prices := make([]float64, kb.size)
	for i := 0; i < kb.size; i++ {
		idx := (kb.head - kb.size + 1 + i + kb.capacity) % kb.capacity
		prices[i] = kb.data[idx].GetLowPrice()
	}
	return prices
}

// GetVolumes 获取所有成交量（从旧到新）
func (kb *KlineBuffer) GetVolumes() []float64 {
	if kb.size == 0 {
		return nil
	}

	volumes := make([]float64, kb.size)
	for i := 0; i < kb.size; i++ {
		idx := (kb.head - kb.size + 1 + i + kb.capacity) % kb.capacity
		volumes[i] = kb.data[idx].GetVolume()
	}
	return volumes
}
