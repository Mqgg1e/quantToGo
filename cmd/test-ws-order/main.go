package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"goQuant/internal/cache"
	"goQuant/internal/config"
	"goQuant/internal/core"
	"goQuant/internal/execution/binance"
	"goQuant/internal/logger"
)

func main() {
	// 初始化日志
	logConfig := logger.DefaultConfig()
	logConfig.Level = "info"
	if err := logger.Init(logConfig); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	// 加载配置
	cfg, err := config.Load("config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Println("=== WebSocket 订单测试 ===")
	log.Printf("API Key: %s...", cfg.Execution.Binance.APIKey[:8])
	log.Printf("Base URL: %s", cfg.Execution.Binance.BaseURL)

	ctx := context.Background()

	// 创建账户缓存
	accountCache := cache.NewAccountCache()

	// 创建执行器
	executor := binance.NewLiveExecutor(
		cfg.Execution.Binance.APIKey,
		cfg.Execution.Binance.SecretKey,
		cfg.Execution.Binance.BaseURL,
		accountCache,
	)

	// 设置代理
	if cfg.Data.ProxyURL != "" {
		if err := executor.GetClient().SetProxy(cfg.Data.ProxyURL); err != nil {
			log.Fatalf("Failed to set proxy: %v", err)
		}
		log.Printf("✅ Proxy set: %s", cfg.Data.ProxyURL)
	}

	// 初始化缓存
	if err := accountCache.InitFromRestAPI(ctx, executor); err != nil {
		log.Fatalf("Failed to initialize cache: %v", err)
	}
	log.Println("✅ Account cache initialized")

	// 启用 WebSocket 订单
	if err := executor.EnableWebSocketOrder(ctx); err != nil {
		log.Fatalf("Failed to enable WebSocket ordering: %v", err)
	}
	log.Println("✅ WebSocket ordering enabled")

	// 等待连接稳定
	time.Sleep(2 * time.Second)

	// 获取账户信息
	account, err := executor.GetAccount(ctx)
	if err != nil {
		log.Fatalf("Failed to get account: %v", err)
	}
	log.Printf("📊 Account Balance: %.2f USDT", account.AvailableBalance)

	// 测试币种
	symbol := "BTCUSDT"

	// 显示菜单
	fmt.Println("\n=== 测试选项 ===")
	fmt.Println("1. 测试市价开多单 (0.001 BTC)")
	fmt.Println("2. 测试限价开多单 (0.001 BTC @ 90000)")
	fmt.Println("3. 测试市价平仓")
	fmt.Println("4. 测试止损单 (STOP_MARKET)")
	fmt.Println("5. 测试跟踪止损单 (TRAILING_STOP_MARKET)")
	fmt.Println("6. 查看当前持仓")
	fmt.Println("0. 退出")
	fmt.Print("\n请选择 (0-6): ")

	var choice int
	fmt.Scanln(&choice)

	switch choice {
	case 1:
		// 测试市价开多单
		log.Println("\n=== 测试 1: 市价开多单 ===")
		order := &core.Order{
			Symbol:   symbol,
			Type:     core.OrderTypeMarket,
			Side:     core.OrderSideBuy,
			Quantity: 0.001,
		}
		result, err := executor.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("❌ 下单失败: %v", err)
		} else {
			exchangeOrderID := ""
			if result.Metadata != nil {
				if eid, ok := result.Metadata["exchange_order_id"].(string); ok {
					exchangeOrderID = eid
				}
			}
			log.Printf("✅ 订单成功:")
			log.Printf("  订单ID: %s", exchangeOrderID)
			log.Printf("  状态: %v", result.Status)
			log.Printf("  数量: %.3f", result.Quantity)
			log.Printf("  成交数量: %.3f", result.FilledQty)
		}

	case 2:
		// 测试限价开多单
		log.Println("\n=== 测试 2: 限价开多单 ===")
		order := &core.Order{
			Symbol:   symbol,
			Type:     core.OrderTypeLimit,
			Side:     core.OrderSideBuy,
			Quantity: 0.001,
			Price:    90000.0,
		}
		result, err := executor.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("❌ 下单失败: %v", err)
		} else {
			exchangeOrderID := ""
			if result.Metadata != nil {
				if eid, ok := result.Metadata["exchange_order_id"].(string); ok {
					exchangeOrderID = eid
				}
			}
			log.Printf("✅ 订单成功:")
			log.Printf("  订单ID: %s", exchangeOrderID)
			log.Printf("  状态: %v", result.Status)
			log.Printf("  价格: %.2f", result.Price)
			log.Printf("  数量: %.3f", result.Quantity)
		}

	case 3:
		// 测试市价平仓
		log.Println("\n=== 测试 3: 市价平仓 ===")

		// 获取当前持仓
		positions, err := executor.GetPositions(ctx)
		if err != nil {
			log.Fatalf("Failed to get positions: %v", err)
		}

		var btcPosition *core.Position
		for _, pos := range positions {
			if pos.Symbol == symbol && pos.Size != 0 {
				btcPosition = pos
				break
			}
		}

		if btcPosition == nil {
			log.Println("❌ 没有 BTC 持仓")
			break
		}

		log.Printf("当前持仓: %.4f BTC", btcPosition.Size)

		side := core.OrderSideSell
		if btcPosition.Size < 0 {
			side = core.OrderSideBuy
		}

		order := &core.Order{
			Symbol:   symbol,
			Type:     core.OrderTypeMarket,
			Side:     side,
			Quantity: abs(btcPosition.Size),
			Metadata: map[string]interface{}{
				"reduce_only": true,
			},
		}
		result, err := executor.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("❌ 平仓失败: %v", err)
		} else {
			exchangeOrderID := ""
			if result.Metadata != nil {
				if eid, ok := result.Metadata["exchange_order_id"].(string); ok {
					exchangeOrderID = eid
				}
			}
			log.Printf("✅ 平仓成功:")
			log.Printf("  订单ID: %s", exchangeOrderID)
			log.Printf("  状态: %v", result.Status)
		}

	case 4:
		// 测试止损单
		log.Println("\n=== 测试 4: 止损单 (STOP_MARKET) ===")
		order := &core.Order{
			Symbol:    symbol,
			Type:      core.OrderTypeStopMarket,
			Side:      core.OrderSideSell,
			StopPrice: 85000.0, // 假设当前价格在 90000 附近
		}
		result, err := executor.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("❌ 下单失败: %v", err)
		} else {
			exchangeOrderID := ""
			if result.Metadata != nil {
				if eid, ok := result.Metadata["exchange_order_id"].(string); ok {
					exchangeOrderID = eid
				}
			}
			log.Printf("✅ 止损单成功:")
			log.Printf("  订单ID: %s", exchangeOrderID)
			log.Printf("  触发价: %.2f", result.StopPrice)
			log.Printf("  状态: %v", result.Status)
		}

	case 5:
		// 测试跟踪止损单
		log.Println("\n=== 测试 5: 跟踪止损单 (TRAILING_STOP_MARKET) ===")
		order := &core.Order{
			Symbol: symbol,
			Type:   core.OrderTypeTrailingStop,
			Side:   core.OrderSideSell,
			Metadata: map[string]interface{}{
				"callback_rate":  1.0, // 1% 回调
				"close_position": true,
			},
		}
		result, err := executor.PlaceOrder(ctx, order)
		if err != nil {
			log.Printf("❌ 下单失败: %v", err)
		} else {
			exchangeOrderID := ""
			if result.Metadata != nil {
				if eid, ok := result.Metadata["exchange_order_id"].(string); ok {
					exchangeOrderID = eid
				}
			}
			log.Printf("✅ 跟踪止损单成功:")
			log.Printf("  订单ID: %s", exchangeOrderID)
			log.Printf("  状态: %v", result.Status)
		}

	case 6:
		// 查看持仓
		log.Println("\n=== 当前持仓 ===")
		positions, err := executor.GetPositions(ctx)
		if err != nil {
			log.Printf("❌ 获取持仓失败: %v", err)
		} else if len(positions) == 0 {
			log.Println("无持仓")
		} else {
			for _, pos := range positions {
				if pos.Size != 0 {
					log.Printf("📈 %s: %.4f @ %.2f, 未实现盈亏: %.2f USDT (%.2f%%)",
						pos.Symbol, pos.Size, pos.EntryPrice,
						pos.UnrealizedPnL, pos.UnrealizedPnLPercent)
				}
			}
		}

	case 0:
		log.Println("退出...")
		os.Exit(0)

	default:
		log.Println("无效选择")
	}

	log.Println("\n测试完成，按 Enter 退出...")
	fmt.Scanln()
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
