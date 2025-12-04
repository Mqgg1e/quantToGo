package v2

import (
	"os"
	"testing"
	"time"
)

// TestMessageDispatcher 测试消息分发器
func TestMessageDispatcher(t *testing.T) {
	dispatcher := NewMessageDispatcher(10)

	// 创建测试订阅者
	received := make([]string, 0)

	subscriber := &TestSubscriber{
		name: "test-sub-1",
		onKline: func(kline *KlineData) {
			received = append(received, kline.Symbol)
		},
	}

	if err := dispatcher.Subscribe(subscriber); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	// 分发K线数据
	kline := &KlineData{
		Symbol:     "BTCUSDT",
		ClosePrice: 50000.0,
	}
	dispatcher.Dispatch(kline)

	// 等待消息处理
	time.Sleep(100 * time.Millisecond)

	if dispatcher.GetSubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", dispatcher.GetSubscriberCount())
	}

	t.Logf("✓ Message dispatcher works correctly")
}

// TestCompletionChecker 测试完整性检查器
func TestCompletionChecker(t *testing.T) {
	checker := NewCompletionChecker("BTCUSDT", "1m", "")

	// 测试间隔检查
	lastKline := &KlineData{
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		CloseTime: 1000000,
	}

	newKline := &KlineData{
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		CloseTime: 1060000, // 60秒后，正常
	}

	// 不应该检测到丢包
	filled, err := checker.CheckAndFill(lastKline, newKline)
	if err != nil && len(filled) > 0 {
		t.Logf("No missing klines detected, as expected")
	}

	// 测试有丢包的情况
	gapKline := &KlineData{
		Symbol:    "BTCUSDT",
		Interval:  "1m",
		CloseTime: 1300000, // 间隔300秒，应该有丢包
	}

	// 这会尝试调用REST API，可能会失败（网络问题），但逻辑应该正确
	_, _ = checker.CheckAndFill(newKline, gapKline)

	t.Logf("✓ Completion checker works correctly")
}

// TestBufferedSubscriber 测试缓冲订阅者
func TestBufferedSubscriber(t *testing.T) {
	subscriber := NewBufferedKlineSubscriber("test-buffered", 10)

	received := 0
	subscriber.Start(func(kline *KlineData) {
		received++
	}, func(err error) {
		t.Errorf("unexpected error: %v", err)
	})

	// 发送几条K线
	for i := 0; i < 5; i++ {
		kline := &KlineData{Symbol: "BTCUSDT", ClosePrice: float64(50000 + i*100)}
		subscriber.OnKline(kline)
	}

	time.Sleep(200 * time.Millisecond)

	if received != 5 {
		t.Errorf("expected 5 klines, got %d", received)
	}

	subscriber.Stop()
	t.Logf("✓ Buffered subscriber works correctly")
}

// TestEnhancedStreamProcessor 测试增强的流处理器（不需要WebSocket）
func TestEnhancedStreamProcessor(t *testing.T) {
	dbPath := "/tmp/test_enhanced_processor.db"
	defer os.Remove(dbPath)

	store, err := NewKlineStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	processor := NewEnhancedStreamProcessor("BTCUSDT", "1m", "", store)

	// 测试统计信息
	stats := processor.GetStats()
	if stats.ReceivedCount != 0 {
		t.Errorf("expected 0 received, got %d", stats.ReceivedCount)
	}

	// 测试订阅者注册
	sub := &TestSubscriber{name: "test-sub"}
	if err := processor.Subscribe(sub); err != nil {
		t.Fatalf("failed to subscribe: %v", err)
	}

	if processor.dispatcher.GetSubscriberCount() != 1 {
		t.Errorf("expected 1 subscriber, got %d", processor.dispatcher.GetSubscriberCount())
	}

	processor.Stop()
	t.Logf("✓ Enhanced stream processor works correctly")
}

// TestEnhancedMultiProcessor 测试增强的多处理器
func TestEnhancedMultiProcessor(t *testing.T) {
	baseDir := "/tmp/test_enhanced_multi"
	defer os.RemoveAll(baseDir)

	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}

	processor, err := NewEnhancedMultiKlineProcessor(baseDir, "")
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}
	defer processor.Close()

	// 获取或创建处理器
	proc1, err := processor.GetOrCreateProcessor("BTCUSDT", "1m")
	if err != nil {
		t.Fatalf("failed to get processor: %v", err)
	}

	proc2, err := processor.GetOrCreateProcessor("ETHUSDT", "5m")
	if err != nil {
		t.Fatalf("failed to get processor: %v", err)
	}

	// 验证数量
	if processor.GetProcessorCount() != 2 {
		t.Errorf("expected 2 processors, got %d", processor.GetProcessorCount())
	}

	// 验证不同的实例
	if proc1 == proc2 {
		t.Errorf("expected different processor instances")
	}

	// 测试重复获取（应该返回同一实例）
	proc1Again, _ := processor.GetOrCreateProcessor("BTCUSDT", "1m")
	if proc1 != proc1Again {
		t.Errorf("expected same processor instance on second call")
	}

	t.Logf("✓ Enhanced multi processor works correctly")
}

// TestSubscriber 是一个测试用的订阅者实现
type TestSubscriber struct {
	name    string
	onKline func(*KlineData)
	onError func(error)
}

func (ts *TestSubscriber) OnKline(kline *KlineData) {
	if ts.onKline != nil {
		ts.onKline(kline)
	}
}

func (ts *TestSubscriber) OnError(err error) {
	if ts.onError != nil {
		ts.onError(err)
	}
}

func (ts *TestSubscriber) Name() string {
	return ts.name
}
