package dataFromWS

import (
	"context"
	"fmt"
	"sync"
)

// KlineProcessor 处理并保存K线数据到数据库
type KlineProcessor struct {
	store  *KlineStore
	mu     sync.RWMutex
	closed bool
}

// NewKlineProcessor 创建一个新的K线处理器
func NewKlineProcessor(dbPath string) (*KlineProcessor, error) {
	store, err := NewKlineStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create kline store: %w", err)
	}

	return &KlineProcessor{
		store: store,
	}, nil
}

// ProcessStream 处理WebSocket消息流并自动保存收盘K线
// 当K线收盘（x=true）时，自动保存到数据库
func (kp *KlineProcessor) ProcessStream(ctx context.Context, msgCh <-chan []byte, errCh <-chan error) {
	for {
		select {
		case <-ctx.Done():
			return

		case err := <-errCh:
			if err != nil {
				fmt.Printf("websocket error: %v\n", err)
			}
			return

		case msg := <-msgCh:
			if msg == nil {
				return
			}

			// 解析K线数据
			kline, err := ParseKlineEvent(msg)
			if err != nil {
				fmt.Printf("failed to parse kline: %v\n", err)
				continue
			}

			// 只保存收盘的K线
			if kline.IsClosed {
				if err := kp.store.SaveKline(kline); err != nil {
					fmt.Printf("failed to save kline: %v\n", err)
				} else {
					fmt.Printf("saved kline: %s %s close_time=%d price=%.2f\n",
						kline.Symbol, kline.Interval, kline.CloseTime, kline.ClosePrice)
				}
			}
		}
	}
}

// QueryKlines 查询已保存的K线数据
func (kp *KlineProcessor) QueryKlines(symbol, interval string, limit int) ([]*KlineData, error) {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if kp.closed {
		return nil, fmt.Errorf("processor is closed")
	}

	return kp.store.GetKlines(symbol, interval, limit)
}

// GetKlineCount 获取K线数据总数
func (kp *KlineProcessor) GetKlineCount(symbol, interval string) (int, error) {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if kp.closed {
		return 0, fmt.Errorf("processor is closed")
	}

	return kp.store.GetKlineCount(symbol, interval)
}

// Close 关闭处理器和数据库
func (kp *KlineProcessor) Close() error {
	kp.mu.Lock()
	defer kp.mu.Unlock()

	if kp.closed {
		return nil
	}

	kp.closed = true
	return kp.store.Close()
}
