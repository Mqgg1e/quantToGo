package logger

import (
	"time"

	"go.uber.org/zap"
)

// TradingLogger 交易专用日志记录器
type TradingLogger struct {
	logger *zap.Logger
}

// NewTradingLogger 创建交易日志记录器
func NewTradingLogger() *TradingLogger {
	return &TradingLogger{
		logger: Logger.With(zap.String("component", "trading")),
	}
}

// LogKline 记录K线接收
func (t *TradingLogger) LogKline(symbol, interval string, openTime int64, open, high, low, close, volume float64) {
	t.logger.Info("Kline received",
		zap.String("symbol", symbol),
		zap.String("interval", interval),
		zap.Time("open_time", time.UnixMilli(openTime)),
		zap.Float64("open", open),
		zap.Float64("high", high),
		zap.Float64("low", low),
		zap.Float64("close", close),
		zap.Float64("volume", volume),
	)
}

// LogSignal 记录交易信号
func (t *TradingLogger) LogSignal(symbol, signalType string, price float64, confidence float64, reason string) {
	t.logger.Info("Signal generated",
		zap.String("symbol", symbol),
		zap.String("signal_type", signalType),
		zap.Float64("price", price),
		zap.Float64("confidence", confidence),
		zap.String("reason", reason),
	)
}

// LogPosition 记录仓位变化
func (t *TradingLogger) LogPosition(symbol string, side string, size, entryPrice, currentPrice, unrealizedPnL, realizedPnL float64) {
	t.logger.Info("Position update",
		zap.String("symbol", symbol),
		zap.String("side", side),
		zap.Float64("size", size),
		zap.Float64("entry_price", entryPrice),
		zap.Float64("current_price", currentPrice),
		zap.Float64("unrealized_pnl", unrealizedPnL),
		zap.Float64("realized_pnl", realizedPnL),
		zap.Float64("pnl_percent", (currentPrice-entryPrice)/entryPrice*100),
	)
}

// LogOrder 记录订单
func (t *TradingLogger) LogOrder(orderID, symbol, side, orderType string, quantity, price float64, status string) {
	t.logger.Info("Order event",
		zap.String("order_id", orderID),
		zap.String("symbol", symbol),
		zap.String("side", side),
		zap.String("type", orderType),
		zap.Float64("quantity", quantity),
		zap.Float64("price", price),
		zap.String("status", status),
	)
}

// LogStrategyWarmup 记录策略预热状态
func (t *TradingLogger) LogStrategyWarmup(symbol string, currentKlines, requiredKlines int, progress float64) {
	t.logger.Info("Strategy warming up",
		zap.String("symbol", symbol),
		zap.Int("current_klines", currentKlines),
		zap.Int("required_klines", requiredKlines),
		zap.Float64("progress_percent", progress*100),
	)
}

// LogIndicator 记录技术指标值
func (t *TradingLogger) LogIndicator(symbol string, indicators map[string]float64) {
	fields := []zap.Field{zap.String("symbol", symbol)}
	for name, value := range indicators {
		fields = append(fields, zap.Float64(name, value))
	}
	t.logger.Debug("Indicator values", fields...)
}

// LogRiskControl 记录风险控制事件
func (t *TradingLogger) LogRiskControl(symbol, eventType string, trigger, threshold float64, action string) {
	t.logger.Warn("Risk control triggered",
		zap.String("symbol", symbol),
		zap.String("event_type", eventType),
		zap.Float64("trigger_value", trigger),
		zap.Float64("threshold", threshold),
		zap.String("action", action),
	)
}

// LogAccountInfo 记录账户信息
func (t *TradingLogger) LogAccountInfo(totalBalance, availableBalance, margin, unrealizedPnL float64) {
	t.logger.Info("Account snapshot",
		zap.Float64("total_balance", totalBalance),
		zap.Float64("available_balance", availableBalance),
		zap.Float64("margin", margin),
		zap.Float64("unrealized_pnl", unrealizedPnL),
		zap.Float64("margin_ratio", margin/totalBalance*100),
	)
}

// LogDataCompletion 记录数据补全
func (t *TradingLogger) LogDataCompletion(symbol, interval string, missingCount int, startTime, endTime int64) {
	t.logger.Warn("Data completion",
		zap.String("symbol", symbol),
		zap.String("interval", interval),
		zap.Int("missing_count", missingCount),
		zap.Time("start_time", time.UnixMilli(startTime)),
		zap.Time("end_time", time.UnixMilli(endTime)),
	)
}

// LogConnection 记录连接状态
func (t *TradingLogger) LogConnection(event, endpoint string, attempt int, success bool, err error) {
	if success {
		t.logger.Info("Connection event",
			zap.String("event", event),
			zap.String("endpoint", endpoint),
			zap.Int("attempt", attempt),
			zap.Bool("success", success),
		)
	} else {
		t.logger.Error("Connection event",
			zap.String("event", event),
			zap.String("endpoint", endpoint),
			zap.Int("attempt", attempt),
			zap.Bool("success", success),
			zap.Error(err),
		)
	}
}

// LogStrategyDecision 记录策略决策详情
func (t *TradingLogger) LogStrategyDecision(symbol string, decision string, conditions map[string]bool, values map[string]float64) {
	fields := []zap.Field{
		zap.String("symbol", symbol),
		zap.String("decision", decision),
	}

	// 添加条件判断
	for name, met := range conditions {
		fields = append(fields, zap.Bool("condition_"+name, met))
	}

	// 添加相关值
	for name, value := range values {
		fields = append(fields, zap.Float64("value_"+name, value))
	}

	t.logger.Info("Strategy decision", fields...)
}
