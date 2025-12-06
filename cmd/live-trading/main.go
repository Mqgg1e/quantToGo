package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"goQuant/internal/config"
	"goQuant/internal/core"
	v2 "goQuant/internal/dataManager/v2"
	"goQuant/internal/execution/binance"
	"goQuant/internal/logger"
	"goQuant/internal/position"
	"goQuant/internal/strategy"

	"go.uber.org/zap"
)

func main() {
	// 0. 初始化日志系统
	logConfig := logger.DefaultConfig()
	logConfig.Level = "info" // debug, info, warn, error
	if err := logger.Init(logConfig); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	// 初始化按品种分文件的日志系统
	symbolLogger := logger.InitSymbolLogger("info")
	defer logger.CloseSymbolLogger()

	logger.Info("🚀 Starting trading bot",
		zap.String("mode", "live"),
		zap.String("session_id", symbolLogger.GetSessionID()),
		zap.String("session_dir", symbolLogger.GetSessionDir()),
	)

	// 1. 加载配置
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	logger.Info("✅ Config loaded",
		zap.String("mode", cfg.App.Mode),
		zap.Int("leverage", cfg.Position.DefaultLeverage),
	)

	// 2. 创建币安执行器
	executor := binance.NewLiveExecutor(
		cfg.Execution.Binance.APIKey,
		cfg.Execution.Binance.SecretKey,
		cfg.Execution.Binance.BaseURL,
	)

	ctx := context.Background()

	// 3. 设置杠杆和保证金模式（针对要交易的每个交易对）
	for _, sub := range cfg.Data.Subscriptions {
		symbol := sub.Symbol

		log.Printf("Setting up %s: leverage=%dx, margin=%s",
			symbol, cfg.Position.DefaultLeverage, cfg.Position.DefaultMarginMode)

		if err := executor.SetLeverage(ctx, symbol, cfg.Position.DefaultLeverage); err != nil {
			log.Printf("⚠️  Failed to set leverage for %s: %v (may already be set)", symbol, err)
		}

		if err := executor.SetMarginMode(ctx, symbol, cfg.Position.DefaultMarginMode); err != nil {
			log.Printf("⚠️  Failed to set margin mode for %s: %v (may already be set)", symbol, err)
		}
	}

	// 4. 创建仓位管理器
	posMgr := position.NewManager(&cfg.Position, executor)
	log.Printf("✅ Position manager created")

	// 5. 创建数据处理器
	processor, err := v2.NewEnhancedMultiKlineProcessor(
		cfg.Data.DatabaseDir,
		cfg.Data.ProxyURL,
	)
	if err != nil {
		log.Fatalf("Failed to create data processor: %v", err)
	}
	defer processor.Close()

	log.Printf("✅ Data processor created")

	// 6. 为每个订阅创建策略和适配器

	for _, sub := range cfg.Data.Subscriptions {
		symbol := sub.Symbol
		interval := sub.Interval

		// 创建MACD+EMA策略
		macdStrategy := strategy.NewMACDEMAStrategy(symbol, interval)

		// 使用REST API预热策略
		logger.Info("Warming up strategy using REST API",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.Int("required_klines", macdStrategy.GetRequiredWarmupPeriods()),
		)

		historicalKlines, err := v2.WarmupStrategy(
			ctx,
			symbol,
			interval,
			macdStrategy.GetRequiredWarmupPeriods(),
			cfg.Data.ProxyURL,
		)

		if err != nil {
			logger.Error("Failed to warmup strategy",
				zap.String("symbol", symbol),
				zap.String("interval", interval),
				zap.Error(err),
			)
			// 不致命，继续运行，策略会通过WebSocket逐步预热
		} else {
			// 将[]*v2.KlineData转换为[]core.KlineData
			coreKlines := make([]core.KlineData, len(historicalKlines))
			for i, k := range historicalKlines {
				coreKlines[i] = k
			}

			// 将历史K线喂给策略
			err = macdStrategy.Warmup(coreKlines)
			if err != nil {
				logger.Error("Failed to feed historical klines",
					zap.String("symbol", symbol),
					zap.Error(err),
				)
			} else {
				logger.Info("✅ Strategy warmed up with REST API",
					zap.String("symbol", symbol),
					zap.String("interval", interval),
					zap.Int("klines", len(historicalKlines)),
				)
			}
		}

		// 创建策略适配器
		adapter := strategy.NewAdapter(macdStrategy, posMgr, executor, symbol, interval)

		// 启动订阅
		if err := processor.StartSubscription(ctx, symbol, interval); err != nil {
			log.Fatalf("Failed to start subscription %s %s: %v", symbol, interval, err)
		}

		// 注册适配器
		if err := processor.Subscribe(symbol, interval, adapter); err != nil {
			log.Fatalf("Failed to subscribe adapter %s %s: %v", symbol, interval, err)
		}

		logger.Info("✅ Strategy started",
			zap.String("symbol", symbol),
			zap.String("interval", interval),
			zap.String("strategy", "MACD(16,26,9) + EMA(5,15) + VWAP(8)"),
		)
	}

	// 7. 显示账户信息
	account, err := executor.GetAccount(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get account info: %v", err)
	} else {
		log.Printf("📊 Account: Total=%.2f USDT, Available=%.2f USDT, UnrealizedPnL=%.2f USDT",
			account.TotalBalance, account.AvailableBalance, account.UnrealizedPnL)
	}

	// 8. 显示当前持仓
	positions, err := executor.GetPositions(ctx)
	if err != nil {
		log.Printf("⚠️  Failed to get positions: %v", err)
	} else {
		if len(positions) > 0 {
			log.Printf("📈 Current positions:")
			for _, pos := range positions {
				log.Printf("  - %s: %.4f @ %.2f, PnL: %.2f%% (%.2f USDT)",
					pos.Symbol, pos.Size, pos.EntryPrice,
					pos.UnrealizedPnLPercent, pos.UnrealizedPnL)
			}
		} else {
			log.Printf("📈 No open positions")
		}
	}

	log.Printf("\n🚀 Trading bot started successfully!")
	log.Printf("Strategy: MACD(16,26,9) + EMA(5,15) + VWAP(8)")
	log.Printf("Risk Control: StopLoss=0.6%%, TrailingStop=3-levels")
	log.Printf("Position Sizing: Open=20%%, Add=40%%")
	log.Printf("\nPress Ctrl+C to stop...\n")

	// 9. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("\n🛑 Shutting down gracefully...")

	// 10. 清理资源
	processor.Close()

	log.Println("✅ Bot stopped")
}
