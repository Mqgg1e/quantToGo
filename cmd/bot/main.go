package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	dfs "goQuant/internal/dataManager"
)

func main() {
	// 确保数据目录存在
	baseDir := "/home/maeda/Documents/projects/goQuant/data/wsdata"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	// 创建多K线处理器（每个Symbol+Interval使用独立的数据库）
	processor, err := dfs.NewMultiKlineProcessor(baseDir)
	if err != nil {
		log.Fatalf("failed to create multi processor: %v", err)
	}
	defer processor.Close()

	log.Printf("✓ Multi-Processor created, base directory: %s\n", baseDir)

	// 配置要订阅的交易对和周期
	subscriptions := []struct {
		symbol   string
		interval string
	}{
		{"BTCUSDT", "1m"},
		{"BTCUSDT", "3m"},
		{"ETHUSDT", "1m"},
		{"ETHUSDT", "3m"},
	}

	proxyURL := "http://127.0.0.1:7897"

	// 创建上下文和信号处理
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 处理中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("received signal: %v\n", sig)
		cancel()
	}()

	// 为每个订阅启动一个goroutine处理K线流
	var wg sync.WaitGroup
	for _, sub := range subscriptions {
		wg.Add(1)
		go func(symbol, interval string) {
			defer wg.Done()
			processSymbolKlines(ctx, processor, symbol, interval, proxyURL)
		}(sub.symbol, sub.interval)
	}

	// 定期输出统计信息
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				printStatistics(processor, subscriptions)
			}
		}
	}()

	// 等待所有goroutine完成或Context取消
	wg.Wait()
	<-ctx.Done()
	log.Println("✓ Shutting down...")
}

// processSymbolKlines 处理单个交易对+周期的K线流
func processSymbolKlines(ctx context.Context, processor *dfs.MultiKlineProcessor, symbol, interval, proxyURL string) {
	log.Printf("subscribing to %s %s\n", symbol, interval)

	msgCh, errCh, closeFn := dfs.SubscribeKlines(ctx, symbol, interval, proxyURL)
	defer closeFn()

	processor.ProcessStream(ctx, msgCh, errCh)
}

// printStatistics 打印多数据库统计信息
func printStatistics(processor *dfs.MultiKlineProcessor, subscriptions []struct {
	symbol   string
	interval string
}) {
	fmt.Println("\n===== Statistics =====")

	totalKlines := 0
	for _, sub := range subscriptions {
		count, err := processor.GetKlineCount(sub.symbol, sub.interval)
		if err != nil {
			log.Printf("no data for %s %s (database not created yet)", sub.symbol, sub.interval)
			continue
		}

		// 获取最新的3条K线显示
		klines, err := processor.QueryKlines(sub.symbol, sub.interval, 3)
		if err != nil {
			log.Printf("failed to query klines for %s %s: %v", sub.symbol, sub.interval, err)
			continue
		}

		totalKlines += count
		fmt.Printf("\n%s [%s] - Total: %d (DB: %s_%s.db)\n", sub.symbol, sub.interval, count, sub.symbol, sub.interval)
		fmt.Println("  Latest klines:")

		for i, kline := range klines {
			closeTime := time.UnixMilli(kline.CloseTime).Format("2006-01-02 15:04:05")
			fmt.Printf("    %d. %s | O:%.2f C:%.2f H:%.2f L:%.2f | Vol:%.2f\n",
				i+1, closeTime, kline.OpenPrice, kline.ClosePrice, kline.HighPrice, kline.LowPrice, kline.BaseVolume)
		}
	}

	fmt.Printf("\nDatabase files created: %d\n", processor.GetStoreCount())
	fmt.Printf("Total klines stored: %d\n", totalKlines)
	fmt.Println("====================\n")
}
