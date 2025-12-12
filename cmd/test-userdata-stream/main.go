package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"goQuant/internal/cache"
	"goQuant/internal/core"
	"goQuant/internal/execution/binance"
	"goQuant/internal/logger"
)

func main() {
	fmt.Println("🧪 Testing Binance UserDataStream...")
	fmt.Println("")

	// 初始化日志系统
	logConfig := logger.DefaultConfig()
	logConfig.Level = "debug"
	if err := logger.Init(logConfig); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}
	defer logger.Close()

	// 创建客户端
	apiKey := "UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
	secretKey := "IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"
	baseURL := "https://testnet.binancefuture.com"
	proxyURL := "http://127.0.0.1:7897" // 代理地址

	client := binance.NewClient(apiKey, secretKey, baseURL)

	// 设置代理
	if err := client.SetProxy(proxyURL); err != nil {
		log.Printf("⚠️  Failed to set proxy: %v (continuing without proxy)", err)
	} else {
		fmt.Printf("✅ Proxy configured: %s\n", proxyURL)
	}

	ctx := context.Background()

	// 1. 创建账户缓存
	fmt.Println("1️⃣  Creating AccountCache...")
	accountCache := cache.NewAccountCache()
	fmt.Println("✅ AccountCache created")
	fmt.Println("")

	// 2. 初始化缓存（从 REST API 获取当前状态）
	fmt.Println("2️⃣  Initializing cache from REST API...")

	// 创建执行器（注入账户缓存）
	executor := binance.NewLiveExecutor(apiKey, secretKey, baseURL, accountCache)

	// 使用 InitFromRestAPI 方法初始化缓存
	if err := accountCache.InitFromRestAPI(ctx, executor); err != nil {
		log.Fatalf("❌ InitFromRestAPI failed: %v", err)
	}

	fmt.Printf("✅ Cache initialized successfully\n")
	fmt.Println("")

	// 3. 创建并启动 UserDataStream
	fmt.Println("3️⃣  Starting UserDataStream...")
	userStream := binance.NewUserDataStream(client, accountCache, executor)

	// 启动 UserDataStream
	if err := userStream.Start(ctx); err != nil {
		log.Fatalf("❌ Failed to start UserDataStream: %v", err)
	}

	fmt.Println("✅ UserDataStream started successfully")
	fmt.Println("")

	// 4. 打印当前缓存状态
	printCacheStatus(accountCache)

	// 5. 开始监控缓存变化
	fmt.Println("🔍 Monitoring cache updates (press Ctrl+C to stop)...")
	fmt.Println("")

	// 设置定时打印缓存状态
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 设置信号处理
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	// 监控循环
	for {
		select {
		case <-ticker.C:
			fmt.Println("📊 Cache Status Update:")
			printCacheStatus(accountCache)

		case <-sigCh:
			fmt.Println("\n🛑 Shutting down...")
			userStream.Stop()
			fmt.Println("✅ UserDataStream stopped")
			fmt.Println("🎉 Test completed!")
			return
		}
	}
}

// printCacheStatus 打印缓存状态
func printCacheStatus(cache *cache.AccountCache) {
	stats := cache.GetStats()

	fmt.Printf("   Balance: %.2f USDT\n", cache.GetBalance())
	fmt.Printf("   Positions: %d\n", stats["position_count"])
	fmt.Printf("   Orders: %d\n", stats["order_count"])
	fmt.Printf("   Last Update: %v\n", stats["last_update"])
	fmt.Printf("   Version: %d\n", stats["update_version"])

	// 打印持仓详情
	positions := cache.GetAllPositions()
	if len(positions) > 0 {
		fmt.Println("   Position Details:")
		for _, pos := range positions {
			pnlPercent := 0.0
			if pos.EntryPrice > 0 {
				if pos.Side == core.PositionSideLong {
					pnlPercent = (pos.CurrentPrice - pos.EntryPrice) / pos.EntryPrice * 100
				} else {
					pnlPercent = (pos.EntryPrice - pos.CurrentPrice) / pos.EntryPrice * 100
				}
			}
			fmt.Printf("     - %s %s: %.4f @ %.2f (PnL: %.2f, %.2f%%)\n",
				pos.Symbol, pos.Side, pos.Size, pos.EntryPrice,
				pos.UnrealizedPnL, pnlPercent)
		}
	}

	// 打印订单详情
	orders := cache.GetAllOrders()
	if len(orders) > 0 {
		fmt.Println("   Order Details:")
		for _, order := range orders {
			fmt.Printf("     - %s %s %s: %.4f @ %.2f [%s]\n",
				order.Symbol, order.Side, order.Type,
				order.Quantity, order.Price, order.Status)
		}
	}

	fmt.Println("")
}
