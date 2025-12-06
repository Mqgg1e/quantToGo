package main

import (
	"goQuant/internal/logger"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 初始化日志
	cfg := logger.DefaultConfig()
	cfg.Level = "info"
	if err := logger.Init(cfg); err != nil {
		panic(err)
	}
	defer logger.Close()

	// 测试基本日志
	logger.Info("🚀 Testing logging system",
		zap.String("version", "1.0"),
		zap.Time("start_time", time.Now()),
	)

	// 创建交易日志记录器
	tradingLog := logger.NewTradingLogger()

	// 测试K线日志
	tradingLog.LogKline(
		"BTCUSDT",
		"3m",
		time.Now().UnixMilli(),
		91000.00,
		91200.00,
		90900.00,
		91150.00,
		234.56,
	)

	// 测试信号日志
	tradingLog.LogSignal(
		"BTCUSDT",
		"OPEN_LONG",
		91150.00,
		0.85,
		"MACD金叉+EMA5/VWAP8金叉",
	)

	// 测试订单日志
	tradingLog.LogOrder(
		"ORDER_123",
		"BTCUSDT",
		"BUY",
		"MARKET",
		0.1,
		91150.00,
		"FILLED",
	)

	// 测试仓位日志
	tradingLog.LogPosition(
		"BTCUSDT",
		"LONG",
		0.1,
		91150.00,
		91300.00,
		15.00,
		0.00,
	)

	logger.Info("✅ Logging test completed successfully")
}
