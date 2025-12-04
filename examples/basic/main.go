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

base "goQuant/internal/dataManager/base"
)

func main() {
baseDir := "/home/maeda/Documents/projects/goQuant/data/basic_wsdata"
if err := os.MkdirAll(baseDir, 0755); err != nil {
log.Fatalf("failed to create data directory: %v", err)
}

processor, err := base.NewMultiKlineProcessor(baseDir)
if err != nil {
log.Fatalf("failed to create multi processor: %v", err)
}
defer processor.Close()

log.Printf("✓ Multi-Processor created\n")

subscriptions := []struct {
symbol   string
interval string
}{
{"BTCUSDT", "1m"},
{"ETHUSDT", "1m"},
}

proxyURL := "http://127.0.0.1:7897"

ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
go func() {
sig := <-sigChan
log.Printf("received signal: %v\n", sig)
cancel()
}()

var wg sync.WaitGroup
for _, sub := range subscriptions {
wg.Add(1)
go func(symbol, interval string) {
defer wg.Done()
msgCh, errCh, closeFn := base.SubscribeKlines(ctx, symbol, interval, proxyURL)
defer closeFn()
processor.ProcessStream(ctx, msgCh, errCh)
}(sub.symbol, sub.interval)
}

ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

go func() {
for range ticker.C {
select {
case <-ctx.Done():
return
default:
fmt.Println("Basic version running...")
}
}
}()

wg.Wait()
log.Println("✓ Shutting down...")
}
