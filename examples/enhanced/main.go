package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	v2 "goQuant/internal/dataManager/v2"
)

func main() {
	baseDir := "/home/maeda/Documents/projects/goQuant/data/enhanced_wsdata"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		log.Fatalf("failed to create data directory: %v", err)
	}

	proxyURL := "http://127.0.0.1:7897"
	processor, err := v2.NewEnhancedMultiKlineProcessor(baseDir, proxyURL)
	if err != nil {
		log.Fatalf("failed to create enhanced processor: %v", err)
	}
	defer processor.Close()

	log.Printf("✓ Enhanced Multi-Processor created\n")

	subscriptions := []struct {
		symbol   string
		interval string
	}{
		{"BTCUSDT", "1m"},
		{"ETHUSDT", "1m"},
		{"BTCUSDT", "3m"},
		{"ETHUSDT", "3m"},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		log.Printf("received signal: %v\n", sig)
		cancel()
	}()

	// Start subscriptions
	for _, sub := range subscriptions {
		if err := processor.StartSubscription(ctx, sub.symbol, sub.interval); err != nil {
			log.Printf("failed to start subscription for %s %s: %v", sub.symbol, sub.interval, err)
		}
	}

	// Print stats every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			select {
			case <-ctx.Done():
				return
			default:
				processor.PrintAllStats()
			}
		}
	}()

	<-ctx.Done()
	log.Println("✓ Shutting down...")
}
