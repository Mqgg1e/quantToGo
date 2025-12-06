package main

import (
	"goQuant/internal/logger"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 初始化按品种分文件的日志系统
	symbolLogger := logger.InitSymbolLogger("info")
	defer logger.CloseSymbolLogger()

	println("✅ 日志系统初始化完成")
	println("📁 会话目录:", symbolLogger.GetSessionDir())
	println("")

	// 模拟多个品种的K线
	symbols := []struct {
		symbol   string
		interval string
		price    float64
	}{
		{"BTCUSDT", "3m", 91150.00},
		{"ETHUSDT", "3m", 3117.58},
		{"BTCUSDT", "1m", 91160.00},
		{"ETHUSDT", "1m", 3118.20},
	}

	for i, s := range symbols {
		// 获取对应品种的logger
		log := symbolLogger.GetLogger(s.symbol, s.interval)

		// 记录K线
		log.Info("Kline received",
			zap.Int("sequence", i+1),
			zap.Time("time", time.Now()),
			zap.Float64("close", s.price),
			zap.Float64("volume", 123.45),
		)

		// 如果是第一根K线，记录信号
		if i%2 == 0 {
			log.Info("Signal generated",
				zap.String("signal_type", "OPEN_LONG"),
				zap.Float64("price", s.price),
				zap.String("reason", "测试信号"),
			)
		}

		time.Sleep(100 * time.Millisecond)
	}

	println("")
	println("✅ 测试完成！")
	println("📝 日志文件已生成在:", symbolLogger.GetSessionDir())
	println("")
	println("文件列表：")
	println("  - BTCUSDT_3m.log")
	println("  - ETHUSDT_3m.log")
	println("  - BTCUSDT_1m.log")
	println("  - ETHUSDT_1m.log")
}
