package strategy

import "math"

// ========== EMA (指数移动平均) ==========

// EMA 计算指数移动平均
type EMA struct {
	period      int
	multiplier  float64
	value       float64
	initialized bool
}

// NewEMA 创建EMA指标
func NewEMA(period int) *EMA {
	return &EMA{
		period:     period,
		multiplier: 2.0 / float64(period+1),
	}
}

// Update 更新EMA值
func (e *EMA) Update(price float64) float64 {
	if !e.initialized {
		e.value = price
		e.initialized = true
		return e.value
	}

	e.value = (price-e.value)*e.multiplier + e.value
	return e.value
}

// Value 获取当前EMA值
func (e *EMA) Value() float64 {
	return e.value
}

// Reset 重置EMA
func (e *EMA) Reset() {
	e.value = 0
	e.initialized = false
}

// ========== MACD (异同移动平均线) ==========

// MACD MACD指标
type MACD struct {
	fastEMA   *EMA
	slowEMA   *EMA
	signalEMA *EMA
	dif       float64
	dea       float64
	macd      float64
}

// NewMACD 创建MACD指标
// fastPeriod: 快线周期（默认12）
// slowPeriod: 慢线周期（默认26）
// signalPeriod: 信号线周期（默认9）
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD {
	return &MACD{
		fastEMA:   NewEMA(fastPeriod),
		slowEMA:   NewEMA(slowPeriod),
		signalEMA: NewEMA(signalPeriod),
	}
}

// Update 更新MACD值
func (m *MACD) Update(price float64) {
	fast := m.fastEMA.Update(price)
	slow := m.slowEMA.Update(price)

	m.dif = fast - slow
	m.dea = m.signalEMA.Update(m.dif)
	m.macd = (m.dif - m.dea) * 2
}

// Values 获取MACD值 (DIF, DEA, MACD)
func (m *MACD) Values() (float64, float64, float64) {
	return m.dif, m.dea, m.macd
}

// DIF 获取DIF值（快线-慢线）
func (m *MACD) DIF() float64 {
	return m.dif
}

// DEA 获取DEA值（信号线）
func (m *MACD) DEA() float64 {
	return m.dea
}

// Histogram 获取柱状图值
func (m *MACD) Histogram() float64 {
	return m.macd
}

// Reset 重置MACD
func (m *MACD) Reset() {
	m.fastEMA.Reset()
	m.slowEMA.Reset()
	m.signalEMA.Reset()
	m.dif = 0
	m.dea = 0
	m.macd = 0
}

// ========== VWAP (成交量加权平均价) ==========

// VWAP 成交量加权平均价
type VWAP struct {
	period    int
	prices    []float64
	volumes   []float64
	sumPV     float64 // price * volume累计
	sumVolume float64 // volume累计
}

// NewVWAP 创建VWAP指标
func NewVWAP(period int) *VWAP {
	return &VWAP{
		period:  period,
		prices:  make([]float64, 0, period),
		volumes: make([]float64, 0, period),
	}
}

// Update 更新VWAP值
func (v *VWAP) Update(price, volume float64) float64 {
	v.prices = append(v.prices, price)
	v.volumes = append(v.volumes, volume)

	v.sumPV += price * volume
	v.sumVolume += volume

	// 超过周期时移除最旧数据
	if len(v.prices) > v.period {
		oldPrice := v.prices[0]
		oldVolume := v.volumes[0]
		v.sumPV -= oldPrice * oldVolume
		v.sumVolume -= oldVolume
		v.prices = v.prices[1:]
		v.volumes = v.volumes[1:]
	}

	if v.sumVolume == 0 {
		return price
	}

	return v.sumPV / v.sumVolume
}

// Value 获取当前VWAP值
func (v *VWAP) Value() float64 {
	if v.sumVolume == 0 {
		return 0
	}
	return v.sumPV / v.sumVolume
}

// Reset 重置VWAP
func (v *VWAP) Reset() {
	v.prices = v.prices[:0]
	v.volumes = v.volumes[:0]
	v.sumPV = 0
	v.sumVolume = 0
}

// ========== SMA (简单移动平均) ==========

// SMA 简单移动平均
type SMA struct {
	period int
	values []float64
	sum    float64
}

// NewSMA 创建SMA指标
func NewSMA(period int) *SMA {
	return &SMA{
		period: period,
		values: make([]float64, 0, period),
	}
}

// Update 更新SMA值
func (s *SMA) Update(price float64) float64 {
	s.values = append(s.values, price)
	s.sum += price

	if len(s.values) > s.period {
		s.sum -= s.values[0]
		s.values = s.values[1:]
	}

	return s.sum / float64(len(s.values))
}

// Value 获取当前SMA值
func (s *SMA) Value() float64 {
	if len(s.values) == 0 {
		return 0
	}
	return s.sum / float64(len(s.values))
}

// Reset 重置SMA
func (s *SMA) Reset() {
	s.values = s.values[:0]
	s.sum = 0
}

// ========== ATR (真实波动幅度) ==========

// ATR 真实波动幅度
type ATR struct {
	period      int
	atr         float64
	prevClose   float64
	initialized bool
}

// NewATR 创建ATR指标
func NewATR(period int) *ATR {
	return &ATR{
		period: period,
	}
}

// Update 更新ATR值
func (a *ATR) Update(high, low, close float64) float64 {
	if !a.initialized {
		a.prevClose = close
		a.atr = high - low
		a.initialized = true
		return a.atr
	}

	tr := math.Max(high-low, math.Max(
		math.Abs(high-a.prevClose),
		math.Abs(low-a.prevClose),
	))

	a.atr = ((a.atr * float64(a.period-1)) + tr) / float64(a.period)
	a.prevClose = close

	return a.atr
}

// Value 获取当前ATR值
func (a *ATR) Value() float64 {
	return a.atr
}

// Reset 重置ATR
func (a *ATR) Reset() {
	a.atr = 0
	a.prevClose = 0
	a.initialized = false
}

// ========== RSI (相对强弱指标) ==========

// RSI 相对强弱指标
type RSI struct {
	period      int
	gains       []float64
	losses      []float64
	avgGain     float64
	avgLoss     float64
	prevPrice   float64
	initialized bool
}

// NewRSI 创建RSI指标
func NewRSI(period int) *RSI {
	return &RSI{
		period: period,
		gains:  make([]float64, 0, period),
		losses: make([]float64, 0, period),
	}
}

// Update 更新RSI值
func (r *RSI) Update(price float64) float64 {
	if !r.initialized {
		r.prevPrice = price
		r.initialized = true
		return 50.0 // 初始值
	}

	change := price - r.prevPrice
	r.prevPrice = price

	var gain, loss float64
	if change > 0 {
		gain = change
		loss = 0
	} else {
		gain = 0
		loss = -change
	}

	r.gains = append(r.gains, gain)
	r.losses = append(r.losses, loss)

	if len(r.gains) > r.period {
		r.gains = r.gains[1:]
		r.losses = r.losses[1:]
	}

	// 计算平均涨跌
	sumGain, sumLoss := 0.0, 0.0
	for i := range r.gains {
		sumGain += r.gains[i]
		sumLoss += r.losses[i]
	}

	r.avgGain = sumGain / float64(len(r.gains))
	r.avgLoss = sumLoss / float64(len(r.losses))

	if r.avgLoss == 0 {
		return 100.0
	}

	rs := r.avgGain / r.avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}

// Value 获取当前RSI值
func (r *RSI) Value() float64 {
	if r.avgLoss == 0 {
		return 100.0
	}
	rs := r.avgGain / r.avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}

// Reset 重置RSI
func (r *RSI) Reset() {
	r.gains = r.gains[:0]
	r.losses = r.losses[:0]
	r.avgGain = 0
	r.avgLoss = 0
	r.prevPrice = 0
	r.initialized = false
}

// ========== Bollinger Bands (布林带) ==========

// BollingerBands 布林带
type BollingerBands struct {
	sma    *SMA
	period int
	stdDev float64
	values []float64
}

// NewBollingerBands 创建布林带指标
func NewBollingerBands(period int, stdDev float64) *BollingerBands {
	return &BollingerBands{
		sma:    NewSMA(period),
		period: period,
		stdDev: stdDev,
		values: make([]float64, 0, period),
	}
}

// Update 更新布林带值
// 返回：中轨、上轨、下轨
func (bb *BollingerBands) Update(price float64) (float64, float64, float64) {
	middle := bb.sma.Update(price)
	bb.values = append(bb.values, price)

	if len(bb.values) > bb.period {
		bb.values = bb.values[1:]
	}

	// 计算标准差
	variance := 0.0
	for _, v := range bb.values {
		variance += math.Pow(v-middle, 2)
	}
	variance /= float64(len(bb.values))
	std := math.Sqrt(variance)

	upper := middle + bb.stdDev*std
	lower := middle - bb.stdDev*std

	return middle, upper, lower
}

// Reset 重置布林带
func (bb *BollingerBands) Reset() {
	bb.sma.Reset()
	bb.values = bb.values[:0]
}
