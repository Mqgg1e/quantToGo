package v2

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EnhancedStreamProcessor 增强的流处理器，包含重连、完整性检查和消息分发
type EnhancedStreamProcessor struct {
	symbol            string
	interval          string
	proxyURL          string
	store             *KlineStore
	dispatcher        *MessageDispatcher
	connManager       *ConnectionManager
	completionChecker *CompletionChecker
	mu                sync.RWMutex
	running           bool
	lastKline         *KlineData
	statsLock         sync.RWMutex
	stats             *StreamStats
}

// StreamStats 流处理统计信息
type StreamStats struct {
	ReceivedCount    int64     // 接收到的K线总数
	SavedCount       int64     // 保存到数据库的K线总数
	FilledCount      int64     // REST API填补的K线总数
	ErrorCount       int64     // 错误总数
	LastKlineTime    time.Time // 最后接收到的K线时间
	ConnectionErrors int64     // 连接错误次数
	LastError        error     // 最后的错误
}

// NewEnhancedStreamProcessor 创建增强的流处理器
func NewEnhancedStreamProcessor(symbol, interval, proxyURL string, store *KlineStore) *EnhancedStreamProcessor {
	return &EnhancedStreamProcessor{
		symbol:            symbol,
		interval:          interval,
		proxyURL:          proxyURL,
		store:             store,
		dispatcher:        NewMessageDispatcher(100),
		connManager:       NewConnectionManager(symbol, interval, proxyURL),
		completionChecker: NewCompletionChecker(symbol, interval, proxyURL),
		stats: &StreamStats{
			LastKlineTime: time.Now(),
		},
	}
}

// Subscribe 订阅K线数据
func (esp *EnhancedStreamProcessor) Subscribe(subscriber KlineSubscriber) error {
	return esp.dispatcher.Subscribe(subscriber)
}

// Unsubscribe 取消订阅
func (esp *EnhancedStreamProcessor) Unsubscribe(subscriberName string) error {
	return esp.dispatcher.Unsubscribe(subscriberName)
}

// Start 启动流处理器（包含自动重连）
func (esp *EnhancedStreamProcessor) Start(ctx context.Context) error {
	esp.mu.Lock()
	if esp.running {
		esp.mu.Unlock()
		return fmt.Errorf("processor already running")
	}
	esp.running = true
	esp.mu.Unlock()

	go esp.runWithReconnect(ctx)
	fmt.Printf("[%s %s] ✓ Stream processor started\n", esp.symbol, esp.interval)
	return nil
}

// Stop 停止流处理器
func (esp *EnhancedStreamProcessor) Stop() {
	esp.mu.Lock()
	defer esp.mu.Unlock()
	esp.running = false
	fmt.Printf("[%s %s] ✓ Stream processor stopped\n", esp.symbol, esp.interval)
}

// runWithReconnect 运行处理器，包含自动重连逻辑
func (esp *EnhancedStreamProcessor) runWithReconnect(ctx context.Context) {
	for {
		esp.mu.RLock()
		if !esp.running {
			esp.mu.RUnlock()
			return
		}
		esp.mu.RUnlock()

		select {
		case <-ctx.Done():
			return
		default:
		}

		// 连接并处理消息
		msgCh, errCh, closeFn, success := esp.connManager.ConnectWithRetry(ctx)
		if !success {
			fmt.Printf("[%s %s] ✗ Failed to connect, stopping processor\n", esp.symbol, esp.interval)
			return
		}

		// 处理这次连接的消息
		disconnected := esp.processMessages(ctx, msgCh, errCh)
		closeFn()

		if !disconnected {
			// ctx被取消
			return
		}

		// 连接断开，更新错误统计
		esp.statsLock.Lock()
		esp.stats.ConnectionErrors++
		esp.statsLock.Unlock()

		// 等待后重新连接
		fmt.Printf("[%s %s] Waiting before reconnecting...\n", esp.symbol, esp.interval)
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			// 继续重连
		}
	}
}

// processMessages 处理消息流
// 返回true表示连接断开，返回false表示ctx取消
func (esp *EnhancedStreamProcessor) processMessages(ctx context.Context, msgCh <-chan []byte, errCh <-chan error) bool {
	for {
		select {
		case <-ctx.Done():
			return false

		case err := <-errCh:
			if err != nil {
				fmt.Printf("[%s %s] Stream error: %v\n", esp.symbol, esp.interval, err)
				esp.statsLock.Lock()
				esp.stats.ErrorCount++
				esp.stats.LastError = err
				esp.statsLock.Unlock()
				esp.dispatcher.DispatchError(err)
				return true
			}
			return true

		case msg := <-msgCh:
			if msg == nil {
				return true
			}

			// 解析K线数据
			kline, err := ParseKlineEvent(msg)
			if err != nil {
				fmt.Printf("[%s %s] Failed to parse kline: %v\n", esp.symbol, esp.interval, err)
				esp.statsLock.Lock()
				esp.stats.ErrorCount++
				esp.statsLock.Unlock()
				continue
			}

			// 更新统计
			esp.statsLock.Lock()
			esp.stats.ReceivedCount++
			esp.stats.LastKlineTime = time.Now()
			esp.statsLock.Unlock()

			// 只处理收盘的K线
			if !kline.IsClosed {
				continue
			}

			// 检查完整性并填补缺失的K线
			filledKlines, err := esp.completionChecker.CheckAndFill(esp.lastKline, kline)
			if err != nil {
				fmt.Printf("[%s %s] Completion check error: %v\n", esp.symbol, esp.interval, err)
				// 不中断处理，继续处理当前K线
			}

			// 保存填补的K线
			for _, filledKline := range filledKlines {
				if err := esp.store.SaveKline(filledKline); err != nil {
					fmt.Printf("[%s %s] Failed to save filled kline: %v\n", esp.symbol, esp.interval, err)
				} else {
					esp.statsLock.Lock()
					esp.stats.SavedCount++
					esp.stats.FilledCount++
					esp.statsLock.Unlock()
					esp.dispatcher.Dispatch(filledKline)
				}
			}

			// 保存当前K线
			if err := esp.store.SaveKline(kline); err != nil {
				fmt.Printf("[%s %s] Failed to save kline: %v\n", esp.symbol, esp.interval, err)
				esp.statsLock.Lock()
				esp.stats.ErrorCount++
				esp.statsLock.Unlock()
			} else {
				fmt.Printf("[%s %s] ✓ Saved kline: close=%.2f vol=%.2f\n",
					esp.symbol, esp.interval, kline.ClosePrice, kline.BaseVolume)
				esp.statsLock.Lock()
				esp.stats.SavedCount++
				esp.statsLock.Unlock()
				esp.lastKline = kline
				esp.dispatcher.Dispatch(kline)
			}
		}
	}
}

// GetStats 获取统计信息
func (esp *EnhancedStreamProcessor) GetStats() *StreamStats {
	esp.statsLock.RLock()
	defer esp.statsLock.RUnlock()

	// 返回拷贝
	stats := *esp.stats
	return &stats
}

// ResetStats 重置统计信息
func (esp *EnhancedStreamProcessor) ResetStats() {
	esp.statsLock.Lock()
	defer esp.statsLock.Unlock()

	esp.stats = &StreamStats{
		LastKlineTime: time.Now(),
	}
}

// PrintStats 打印统计信息
func (esp *EnhancedStreamProcessor) PrintStats() {
	stats := esp.GetStats()
	fmt.Printf("\n[%s %s] Statistics:\n", esp.symbol, esp.interval)
	fmt.Printf("  Received:      %d\n", stats.ReceivedCount)
	fmt.Printf("  Saved:         %d\n", stats.SavedCount)
	fmt.Printf("  Filled (REST): %d\n", stats.FilledCount)
	fmt.Printf("  Errors:        %d\n", stats.ErrorCount)
	fmt.Printf("  Conn Errors:   %d\n", stats.ConnectionErrors)
	fmt.Printf("  Last Kline:    %s\n", stats.LastKlineTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Subscribers:   %d\n", esp.dispatcher.GetSubscriberCount())
	if stats.LastError != nil {
		fmt.Printf("  Last Error:    %v\n", stats.LastError)
	}
}
