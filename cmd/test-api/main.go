package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"goQuant/internal/execution/binance"
)

func main() {
	fmt.Println("🧪 Testing Binance API Connection...")
	fmt.Println("")

	// 创建客户端
	apiKey := "UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
	secretKey := "IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"
	baseURL := "https://testnet.binancefuture.com"

	client := binance.NewClient(apiKey, secretKey, baseURL)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 测试获取账户信息
	fmt.Println("1️⃣  Testing GetAccount()...")
	account, err := client.GetAccount(ctx)
	if err != nil {
		log.Fatalf("❌ GetAccount failed: %v", err)
	}

	fmt.Printf("✅ Account retrieved successfully!\n")
	fmt.Printf("   Total Wallet Balance: %s USDT\n", account.TotalWalletBalance)
	fmt.Printf("   Available Balance: %s USDT\n", account.AvailableBalance)
	fmt.Printf("   Total Cross UnPnL: %s USDT\n", account.TotalCrossUnPnl)
	fmt.Printf("   Can Trade: %v\n", account.CanTrade)
	fmt.Println("")

	// 测试获取持仓信息
	fmt.Println("2️⃣  Testing GetPositionRisk(BTCUSDT)...")
	positions, err := client.GetPositionRisk(ctx, "BTCUSDT")
	if err != nil {
		log.Fatalf("❌ GetPositionRisk failed: %v", err)
	}

	fmt.Printf("✅ Position retrieved successfully!\n")
	for _, pos := range positions {
		if pos.PositionAmt != "0" {
			fmt.Printf("   Symbol: %s\n", pos.Symbol)
			fmt.Printf("   Position: %s\n", pos.PositionAmt)
			fmt.Printf("   Entry Price: %s\n", pos.EntryPrice)
			fmt.Printf("   Unrealized PnL: %s\n", pos.UnRealizedProfit)
		}
	}
	fmt.Println("")

	fmt.Println("🎉 All tests passed!")
}
