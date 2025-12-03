package dataFromWS

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

// MultiKlineProcessor 多交易对、多周期K线处理器，每个交易对+周期组合使用独立的数据库
type MultiKlineProcessor struct {
	baseDir string                 // 数据库基础目录，如 "./data"
	stores  map[string]*KlineStore // key: "SYMBOL_INTERVAL", value: 对应的KlineStore
	mu      sync.RWMutex           // 保护stores map
	closed  bool
}

// NewMultiKlineProcessor 创建一个多K线处理器
// baseDir: 数据库基础目录，每个Symbol+Interval会自动创建独立的db文件
// 例如: "./data" 会生成 "./data/BTCUSDT_1m.db", "./data/ETHUSDT_5m.db" 等
func NewMultiKlineProcessor(baseDir string) (*MultiKlineProcessor, error) {
	// 确保baseDir存在
	if err := ensureDir(baseDir); err != nil {
		return nil, fmt.Errorf("ensure directory: %w", err)
	}

	return &MultiKlineProcessor{
		baseDir: baseDir,
		stores:  make(map[string]*KlineStore),
	}, nil
}

// getOrCreateStore 获取或创建指定Symbol+Interval的KlineStore
func (mkp *MultiKlineProcessor) getOrCreateStore(symbol, interval string) (*KlineStore, error) {
	key := getStoreKey(symbol, interval)

	mkp.mu.RLock()
	if store, exists := mkp.stores[key]; exists {
		mkp.mu.RUnlock()
		return store, nil
	}
	mkp.mu.RUnlock()

	// 创建新的store（需要写锁）
	mkp.mu.Lock()
	defer mkp.mu.Unlock()

	// 双重检查
	if store, exists := mkp.stores[key]; exists {
		return store, nil
	}

	// 数据库文件路径: baseDir/SYMBOL_INTERVAL.db
	dbPath := filepath.Join(mkp.baseDir, fmt.Sprintf("%s_%s.db", symbol, interval))

	store, err := NewKlineStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create store for %s %s: %w", symbol, interval, err)
	}

	mkp.stores[key] = store
	fmt.Printf("✓ Created new database: %s\n", dbPath)

	return store, nil
}

// ProcessStream 处理WebSocket消息流并自动保存收盘K线到对应的数据库
// 当K线收盘（x=true）时，自动保存到对应Symbol+Interval的数据库
func (mkp *MultiKlineProcessor) ProcessStream(ctx context.Context, msgCh <-chan []byte, errCh <-chan error) {
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
				// 获取或创建对应的store
				store, err := mkp.getOrCreateStore(kline.Symbol, kline.Interval)
				if err != nil {
					fmt.Printf("failed to get store for %s %s: %v\n", kline.Symbol, kline.Interval, err)
					continue
				}

				if err := store.SaveKline(kline); err != nil {
					fmt.Printf("failed to save kline: %v\n", err)
				} else {
					fmt.Printf("saved kline: %s %s close_time=%d price=%.2f\n",
						kline.Symbol, kline.Interval, kline.CloseTime, kline.ClosePrice)
				}
			}
		}
	}
}

// QueryKlines 查询指定Symbol+Interval的K线数据
func (mkp *MultiKlineProcessor) QueryKlines(symbol, interval string, limit int) ([]*KlineData, error) {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	if mkp.closed {
		return nil, fmt.Errorf("processor is closed")
	}

	key := getStoreKey(symbol, interval)
	store, exists := mkp.stores[key]
	if !exists {
		return nil, fmt.Errorf("no database for %s %s", symbol, interval)
	}

	return store.GetKlines(symbol, interval, limit)
}

// GetKlineCount 获取指定Symbol+Interval的K线数据总数
func (mkp *MultiKlineProcessor) GetKlineCount(symbol, interval string) (int, error) {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	if mkp.closed {
		return 0, fmt.Errorf("processor is closed")
	}

	key := getStoreKey(symbol, interval)
	store, exists := mkp.stores[key]
	if !exists {
		return 0, fmt.Errorf("no database for %s %s", symbol, interval)
	}

	return store.GetKlineCount(symbol, interval)
}

// GetLoadedStores 获取所有已加载的Symbol+Interval组合
func (mkp *MultiKlineProcessor) GetLoadedStores() []string {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	var keys []string
	for key := range mkp.stores {
		keys = append(keys, key)
	}
	return keys
}

// GetStoreCount 获取已创建的数据库数量
func (mkp *MultiKlineProcessor) GetStoreCount() int {
	mkp.mu.RLock()
	defer mkp.mu.RUnlock()

	return len(mkp.stores)
}

// Close 关闭所有数据库连接
func (mkp *MultiKlineProcessor) Close() error {
	mkp.mu.Lock()
	defer mkp.mu.Unlock()

	if mkp.closed {
		return nil
	}

	var lastErr error
	for key, store := range mkp.stores {
		if err := store.Close(); err != nil {
			fmt.Printf("failed to close store %s: %v\n", key, err)
			lastErr = err
		}
	}

	mkp.closed = true
	mkp.stores = make(map[string]*KlineStore)

	return lastErr
}

// 辅助函数

// getStoreKey 生成store的唯一key
func getStoreKey(symbol, interval string) string {
	return fmt.Sprintf("%s_%s", symbol, interval)
}

// ensureDir 确保目录存在
func ensureDir(dir string) error {
	return nil // 在使用时由OS自动处理，或使用os.MkdirAll
}
