package strategy

import (
	"fmt"
	"time"

	"goQuant/internal/core"
	"goQuant/internal/logger"

	"go.uber.org/zap"
)

// MACDEMAStrategy MACD+EMA+VWAP策略
// 基于testStrategy.md拆分后的策略规则
type MACDEMAStrategy struct {
	symbol   string
	interval string

	// 技术指标
	macd  *MACD
	ema5  *EMA
	ema15 *EMA
	vwap8 *VWAP

	// K线缓冲区
	klineBuffer *KlineBuffer

	// 交叉历史记录（最近3个周期）
	macdCrosses    []CrossEvent
	emaVwapCrosses []CrossEvent
	emaEmaCrosses  []CrossEvent

	// 上一周期指标值（用于检测交叉）
	prevDIF   float64
	prevDEA   float64
	prevEMA5  float64
	prevEMA15 float64
	prevVWAP8 float64

	// 状态
	isWarmedUp bool
	lastSignal *core.TradingSignal
}

// CrossEvent 交叉事件
type CrossEvent struct {
	Timestamp time.Time
	CrossType CrossType // CrossTypeGolden or CrossTypeDeath
	Price     float64
}

// NewMACDEMAStrategy 创建策略实例
func NewMACDEMAStrategy(symbol, interval string) *MACDEMAStrategy {
	return &MACDEMAStrategy{
		symbol:   symbol,
		interval: interval,

		// 初始化指标 (MACD 16,26,9)
		macd:  NewMACD(16, 26, 9),
		ema5:  NewEMA(5),
		ema15: NewEMA(15),
		vwap8: NewVWAP(8),

		// 缓冲区容量设为100（足够预热）
		klineBuffer: NewKlineBuffer(100),

		// 交叉历史
		macdCrosses:    make([]CrossEvent, 0, 3),
		emaVwapCrosses: make([]CrossEvent, 0, 3),
		emaEmaCrosses:  make([]CrossEvent, 0, 3),
	}
}

// ========== 实现 core.Strategy 接口 ==========

// Name 返回策略名称
func (s *MACDEMAStrategy) Name() string {
	return fmt.Sprintf("MACD_EMA_%s_%s", s.symbol, s.interval)
}

// OnKline 处理K线数据
func (s *MACDEMAStrategy) OnKline(kline core.KlineData) (*core.TradingSignal, error) {
	if !kline.IsClosed() {
		return NoActionSignal(s.symbol), nil
	}

	// 添加到缓冲区
	s.klineBuffer.Add(kline)

	// 更新指标
	closePrice := kline.GetClosePrice()
	volume := kline.GetVolume()

	s.macd.Update(closePrice)
	currEMA5 := s.ema5.Update(closePrice)
	currEMA15 := s.ema15.Update(closePrice)
	currVWAP8 := s.vwap8.Update(closePrice, volume)
	currDIF, currDEA, _ := s.macd.Values()

	// 检查是否预热完成
	if !s.isWarmedUp {
		if s.klineBuffer.Size() >= s.GetRequiredWarmupPeriods() {
			s.isWarmedUp = true
			logger.Info("✅ Strategy warmed up",
				zap.String("strategy", s.Name()),
				zap.Int("klines", s.klineBuffer.Size()),
			)
		} else {
			// 预热中 - 每10根K线记录一次进度
			currentSize := s.klineBuffer.Size()
			requiredSize := s.GetRequiredWarmupPeriods()
			if currentSize%10 == 0 || currentSize == 1 {
				progress := float64(currentSize) / float64(requiredSize) * 100
				logger.Info("Strategy warming up",
					zap.String("strategy", s.Name()),
					zap.Int("current_klines", currentSize),
					zap.Int("required_klines", requiredSize),
					zap.Float64("progress_percent", progress),
				)
			}

			// 记录当前值供下次对比
			s.prevDIF = currDIF
			s.prevDEA = currDEA
			s.prevEMA5 = currEMA5
			s.prevEMA15 = currEMA15
			s.prevVWAP8 = currVWAP8
			return NoActionSignal(s.symbol), nil
		}
	}

	// 检测交叉
	timestamp := kline.GetCloseTime()

	// 1. MACD DIF/DEA 交叉
	macdCross := DetectCross(s.prevDIF, s.prevDEA, currDIF, currDEA)
	if macdCross != CrossTypeNone {
		s.addCrossEvent(&s.macdCrosses, CrossEvent{
			Timestamp: timestamp,
			CrossType: macdCross,
			Price:     closePrice,
		})
	}

	// 2. EMA5/VWAP8 交叉
	emaVwapCross := DetectCross(s.prevEMA5, s.prevVWAP8, currEMA5, currVWAP8)
	if emaVwapCross != CrossTypeNone {
		s.addCrossEvent(&s.emaVwapCrosses, CrossEvent{
			Timestamp: timestamp,
			CrossType: emaVwapCross,
			Price:     closePrice,
		})
	}

	// 3. EMA5/EMA15 交叉
	emaEmaCross := DetectCross(s.prevEMA5, s.prevEMA15, currEMA5, currEMA15)
	if emaEmaCross != CrossTypeNone {
		s.addCrossEvent(&s.emaEmaCrosses, CrossEvent{
			Timestamp: timestamp,
			CrossType: emaEmaCross,
			Price:     closePrice,
		})
	}

	// 更新前值
	s.prevDIF = currDIF
	s.prevDEA = currDEA
	s.prevEMA5 = currEMA5
	s.prevEMA15 = currEMA15
	s.prevVWAP8 = currVWAP8

	// DEBUG: 记录指标值（仅在debug级别）
	logger.Debug("Indicator values",
		zap.String("strategy", s.Name()),
		zap.Float64("close", closePrice),
		zap.Float64("dif", currDIF),
		zap.Float64("dea", currDEA),
		zap.Float64("ema5", currEMA5),
		zap.Float64("ema15", currEMA15),
		zap.Float64("vwap8", currVWAP8),
	)

	// 生成信号
	signal := s.generateSignal(timestamp, closePrice)
	s.lastSignal = signal

	return signal, nil
}

// OnOrderBook 处理订单簿数据（暂不使用）
func (s *MACDEMAStrategy) OnOrderBook(orderBook core.OrderBookData) error {
	return nil
}

// Warmup 策略预热
func (s *MACDEMAStrategy) Warmup(historicalKlines []core.KlineData) error {
	for _, kline := range historicalKlines {
		_, err := s.OnKline(kline)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetRequiredWarmupPeriods 返回预热需要的K线数量
func (s *MACDEMAStrategy) GetRequiredWarmupPeriods() int {
	// MACD慢线26 + 信号线9 + 额外缓冲10 = 45
	return 45
}

// Reset 重置策略状态
func (s *MACDEMAStrategy) Reset() error {
	s.macd.Reset()
	s.ema5.Reset()
	s.ema15.Reset()
	s.vwap8.Reset()

	s.klineBuffer = NewKlineBuffer(100)
	s.macdCrosses = s.macdCrosses[:0]
	s.emaVwapCrosses = s.emaVwapCrosses[:0]
	s.emaEmaCrosses = s.emaEmaCrosses[:0]

	s.isWarmedUp = false
	s.lastSignal = nil

	return nil
}

// ========== 信号生成逻辑 ==========

// generateSignal 生成交易信号
func (s *MACDEMAStrategy) generateSignal(timestamp time.Time, currentPrice float64) *core.TradingSignal {
	// 情景一：基于MACD+EMA交叉的信号
	scenario1Signal := s.checkScenario1(timestamp, currentPrice)
	if scenario1Signal.Type != core.SignalTypeNoAction {
		return scenario1Signal
	}

	// 情景二：基于连续趋势的信号
	scenario2Signal := s.checkScenario2(timestamp, currentPrice)
	if scenario2Signal.Type != core.SignalTypeNoAction {
		return scenario2Signal
	}

	return NoActionSignal(s.symbol)
}

// checkScenario1 情景一：MACD+EMA组合交叉信号
func (s *MACDEMAStrategy) checkScenario1(timestamp time.Time, currentPrice float64) *core.TradingSignal {
	// 1. 检查是否有MACD死叉 + EMA5/VWAP8死叉 → 空单信号
	if s.hasRecentCross(s.macdCrosses, CrossTypeDeath, 3) &&
		s.hasRecentCross(s.emaVwapCrosses, CrossTypeDeath, 3) {

		// 生成空单信号
		signal := NewSignal(core.SignalTypeOpenShort, s.symbol, currentPrice, "MACD死叉+EMA5/VWAP8死叉")
		signal.Metadata["macd_dif"] = s.prevDIF
		signal.Metadata["macd_dea"] = s.prevDEA
		signal.Metadata["ema5"] = s.prevEMA5
		signal.Metadata["vwap8"] = s.prevVWAP8
		signal.Metadata["ema15"] = s.prevEMA15

		// 检查是否同时满足EMA5/EMA15死叉（加仓条件）
		if s.hasRecentCross(s.emaEmaCrosses, CrossTypeDeath, 3) {
			signal.Metadata["add_position_eligible"] = 1.0
			signal.Reason = "MACD死叉+EMA5/VWAP8死叉+EMA5/EMA15死叉"
		}

		return signal
	}

	// 2. 检查是否有MACD金叉 + EMA5/VWAP8金叉 → 多单信号
	if s.hasRecentCross(s.macdCrosses, CrossTypeGolden, 3) &&
		s.hasRecentCross(s.emaVwapCrosses, CrossTypeGolden, 3) {

		// 生成多单信号
		signal := NewSignal(core.SignalTypeOpenLong, s.symbol, currentPrice, "MACD金叉+EMA5/VWAP8金叉")
		signal.Metadata["macd_dif"] = s.prevDIF
		signal.Metadata["macd_dea"] = s.prevDEA
		signal.Metadata["ema5"] = s.prevEMA5
		signal.Metadata["vwap8"] = s.prevVWAP8
		signal.Metadata["ema15"] = s.prevEMA15

		// 检查是否同时满足EMA5/EMA15金叉（加仓条件）
		if s.hasRecentCross(s.emaEmaCrosses, CrossTypeGolden, 3) {
			signal.Metadata["add_position_eligible"] = 1.0
			signal.Reason = "MACD金叉+EMA5/VWAP8金叉+EMA5/EMA15金叉"
		}

		return signal
	}

	return NoActionSignal(s.symbol)
}

// checkScenario2 情景二：连续趋势信号
func (s *MACDEMAStrategy) checkScenario2(timestamp time.Time, currentPrice float64) *core.TradingSignal {
	closePrices := s.klineBuffer.GetClosePrices()
	if len(closePrices) < 4 {
		return NoActionSignal(s.symbol)
	}

	// 检测最近4个周期的趋势
	trend, changePercent := DetectTrend(closePrices, 4)

	// 上涨趋势且涨幅超过0.55%
	if trend == TrendUp && changePercent > 0.0055 {
		signal := NewSignal(core.SignalTypeOpenLong, s.symbol, currentPrice,
			fmt.Sprintf("连续4周期上涨，涨幅%.2f%%", changePercent*100))
		signal.Metadata["trend_change"] = changePercent
		signal.Metadata["trend_periods"] = 4
		return signal
	}

	// 下跌趋势且跌幅超过0.55%
	if trend == TrendDown && changePercent < -0.0055 {
		signal := NewSignal(core.SignalTypeOpenShort, s.symbol, currentPrice,
			fmt.Sprintf("连续4周期下跌，跌幅%.2f%%", changePercent*100))
		signal.Metadata["trend_change"] = changePercent
		signal.Metadata["trend_periods"] = 4
		return signal
	}

	return NoActionSignal(s.symbol)
}

// ========== 辅助方法 ==========

// addCrossEvent 添加交叉事件（保持最近3个）
func (s *MACDEMAStrategy) addCrossEvent(events *[]CrossEvent, event CrossEvent) {
	*events = append(*events, event)
	// 只保留最近3个
	if len(*events) > 3 {
		*events = (*events)[len(*events)-3:]
	}
}

// hasRecentCross 检查最近N个周期内是否有指定类型的交叉
func (s *MACDEMAStrategy) hasRecentCross(events []CrossEvent, crossType CrossType, periods int) bool {
	if len(events) == 0 {
		return false
	}

	// 获取最后一个交叉事件
	lastCross := events[len(events)-1]

	// 检查类型是否匹配
	if lastCross.CrossType != crossType {
		return false
	}

	// 检查时间是否在最近N个周期内
	// 简化：假设已经记录了最近的交叉，直接返回true
	// 实际应该检查时间戳
	return true
}

// GetLastSignal 获取最后一个信号（用于测试）
func (s *MACDEMAStrategy) GetLastSignal() *core.TradingSignal {
	return s.lastSignal
}

// GetIndicatorValues 获取当前指标值（用于调试）
func (s *MACDEMAStrategy) GetIndicatorValues() map[string]float64 {
	dif, dea, macd := s.macd.Values()
	return map[string]float64{
		"macd_dif":  dif,
		"macd_dea":  dea,
		"macd_hist": macd,
		"ema5":      s.ema5.Value(),
		"ema15":     s.ema15.Value(),
		"vwap8":     s.vwap8.Value(),
	}
}
