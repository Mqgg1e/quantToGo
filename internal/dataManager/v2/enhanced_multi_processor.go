package v2

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
)

// EnhancedMultiKlineProcessor 增强的多交易对、多周期K线处理器
// 包含：自动重连、完整性检查、REST API补全、消息分发
type EnhancedMultiKlineProcessor struct {
	baseDir    string                              // 数据库基础目录
	stores     map[string]*KlineStore              // key: "SYMBOL_INTERVAL"
	processors map[string]*EnhancedStreamProcessor // key: "SYMBOL_INTERVAL"
	mu         sync.RWMutex
	closed     bool
	proxyURL   string
}

// NewEnhancedMultiKlineProcessor 创建增强的多K线处理器
func NewEnhancedMultiKlineProcessor(baseDir, proxyURL string) (*EnhancedMultiKlineProcessor, error) {
	if err := ensureDir(baseDir); err != nil {
		return nil, fmt.Errorf("ensure directory: %w", err)
	}

	return &EnhancedMultiKlineProcessor{
		baseDir:    baseDir,
		stores:     make(map[string]*KlineStore),
		processors: make(map[string]*EnhancedStreamProcessor),
		proxyURL:   proxyURL,
	}, nil
}

// GetOrCreateProcessor 获取或创建指定Symbol+Interval的流处理器
func (emkp *EnhancedMultiKlineProcessor) GetOrCreateProcessor(symbol, interval string) (*EnhancedStreamProcessor, error) {
	key := getStoreKey(symbol, interval)

	emkp.mu.RLock()
	if proc, exists := emkp.processors[key]; exists {
		emkp.mu.RUnlock()
		return proc, nil
	}
	emkp.mu.RUnlock()

	// 创建新的processor（需要写锁）
	emkp.mu.Lock()
	defer emkp.mu.Unlock()

	// 双重检查
	if proc, exists := emkp.processors[key]; exists {
		return proc, nil
	}

	// 获取或创建store
	store, err := emkp.getOrCreateStore(symbol, interval)
	if err != nil {
		return nil, fmt.Errorf("get store: %w", err)
	}

	// 创建增强的流处理器
	processor := NewEnhancedStreamProcessor(symbol, interval, emkp.proxyURL, store)
	emkp.processors[key] = processor

	fmt.Printf("[%s %s] ✓ Created enhanced stream processor\n", symbol, interval)
	return processor, nil
}

// getOrCreateStore 获取或创建store
func (emkp *EnhancedMultiKlineProcessor) getOrCreateStore(symbol, interval string) (*KlineStore, error) {
	key := getStoreKey(symbol, interval)

	if store, exists := emkp.stores[key]; exists {
		return store, nil
	}

	dbPath := filepath.Join(emkp.baseDir, fmt.Sprintf("%s_%s.db", symbol, interval))
	store, err := NewKlineStore(dbPath)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	emkp.stores[key] = store
	fmt.Printf("✓ Created database: %s\n", dbPath)
	return store, nil
}

// StartSubscription 启动某个交易对+周期的订阅（包含自动重连等）
func (emkp *EnhancedMultiKlineProcessor) StartSubscription(ctx context.Context, symbol, interval string) error {
	processor, err := emkp.GetOrCreateProcessor(symbol, interval)
	if err != nil {
		return fmt.Errorf("get processor: %w", err)
	}

	return processor.Start(ctx)
}

// Subscribe 在某个交易对+周期的处理器上订阅数据
func (emkp *EnhancedMultiKlineProcessor) Subscribe(symbol, interval string, subscriber KlineSubscriber) error {
	processor, err := emkp.GetOrCreateProcessor(symbol, interval)
	if err != nil {
		return fmt.Errorf("get processor: %w", err)
	}

	return processor.Subscribe(subscriber)
}

// Unsubscribe 取消订阅
func (emkp *EnhancedMultiKlineProcessor) Unsubscribe(symbol, interval, subscriberName string) error {
	emkp.mu.RLock()
	key := getStoreKey(symbol, interval)
	proc, exists := emkp.processors[key]
	emkp.mu.RUnlock()

	if !exists {
		return fmt.Errorf("processor not found for %s %s", symbol, interval)
	}

	return proc.Unsubscribe(subscriberName)
}

// GetStats 获取某个处理器的统计信息
func (emkp *EnhancedMultiKlineProcessor) GetStats(symbol, interval string) *StreamStats {
	emkp.mu.RLock()
	key := getStoreKey(symbol, interval)
	proc, exists := emkp.processors[key]
	emkp.mu.RUnlock()

	if !exists {
		return nil
	}

	return proc.GetStats()
}

// PrintAllStats 打印所有处理器的统计信息
func (emkp *EnhancedMultiKlineProcessor) PrintAllStats() {
	emkp.mu.RLock()
	defer emkp.mu.RUnlock()

	fmt.Println("\n===== Stream Processors Statistics =====")
	for _, proc := range emkp.processors {
		proc.PrintStats()
	}
	fmt.Println("=========================================")
}

// QueryKlines 查询K线数据
func (emkp *EnhancedMultiKlineProcessor) QueryKlines(symbol, interval string, limit int) ([]*KlineData, error) {
	emkp.mu.RLock()
	key := getStoreKey(symbol, interval)
	store, exists := emkp.stores[key]
	emkp.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no database for %s %s", symbol, interval)
	}

	return store.GetKlines(symbol, interval, limit)
}

// GetKlineCount 获取K线总数
func (emkp *EnhancedMultiKlineProcessor) GetKlineCount(symbol, interval string) (int, error) {
	emkp.mu.RLock()
	key := getStoreKey(symbol, interval)
	store, exists := emkp.stores[key]
	emkp.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("no database for %s %s", symbol, interval)
	}

	return store.GetKlineCount(symbol, interval)
}

// GetProcessorCount 获取处理器总数
func (emkp *EnhancedMultiKlineProcessor) GetProcessorCount() int {
	emkp.mu.RLock()
	defer emkp.mu.RUnlock()
	return len(emkp.processors)
}

// ListProcessors 列出所有处理器（Symbol_Interval）
func (emkp *EnhancedMultiKlineProcessor) ListProcessors() []string {
	emkp.mu.RLock()
	defer emkp.mu.RUnlock()

	var keys []string
	for key := range emkp.processors {
		keys = append(keys, key)
	}
	return keys
}

// Close 关闭所有处理器和数据库
func (emkp *EnhancedMultiKlineProcessor) Close() error {
	emkp.mu.Lock()
	defer emkp.mu.Unlock()

	if emkp.closed {
		return nil
	}

	// 停止所有处理器
	for _, proc := range emkp.processors {
		proc.Stop()
	}

	// 关闭所有数据库
	var lastErr error
	for key, store := range emkp.stores {
		if err := store.Close(); err != nil {
			fmt.Printf("Failed to close store %s: %v\n", key, err)
			lastErr = err
		}
	}

	emkp.closed = true
	emkp.processors = make(map[string]*EnhancedStreamProcessor)
	emkp.stores = make(map[string]*KlineStore)

	fmt.Println("✓ All processors and stores closed")
	return lastErr
}
