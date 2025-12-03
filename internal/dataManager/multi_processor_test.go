package dataFromWS

import (
	"os"
	"testing"
)

// TestMultiKlineProcessor 测试多K线处理器
func TestMultiKlineProcessor(t *testing.T) {
	// 创建临时目录
	tmpDir := "/tmp/test_multi_klines"
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	// 创建多处理器
	processor, err := NewMultiKlineProcessor(tmpDir)
	if err != nil {
		t.Fatalf("failed to create multi processor: %v", err)
	}
	defer processor.Close()

	// 模拟多个Symbol+Interval的K线数据
	testKlines := []*KlineData{
		{
			EventType:   "kline",
			EventTime:   1638747660000,
			Symbol:      "BTCUSDT",
			StartTime:   1638747600000,
			CloseTime:   1638747660000,
			Interval:    "1m",
			OpenPrice:   43500.50,
			ClosePrice:  43520.75,
			HighPrice:   43550.25,
			LowPrice:    43480.00,
			BaseVolume:  100.5,
			QuoteVolume: 4365200.50,
			IsClosed:    true,
		},
		{
			EventType:   "kline",
			EventTime:   1638747720000,
			Symbol:      "BTCUSDT",
			StartTime:   1638747660000,
			CloseTime:   1638747720000,
			Interval:    "5m",
			OpenPrice:   43520.0,
			ClosePrice:  43540.0,
			HighPrice:   43560.0,
			LowPrice:    43510.0,
			BaseVolume:  505.5,
			QuoteVolume: 21835200.0,
			IsClosed:    true,
		},
		{
			EventType:   "kline",
			EventTime:   1638747780000,
			Symbol:      "ETHUSDT",
			StartTime:   1638747720000,
			CloseTime:   1638747780000,
			Interval:    "1m",
			OpenPrice:   3045.50,
			ClosePrice:  3055.75,
			HighPrice:   3060.25,
			LowPrice:    3045.00,
			BaseVolume:  200.0,
			QuoteVolume: 611000.0,
			IsClosed:    true,
		},
	}

	// 为每个K线获取或创建对应的store并保存
	for _, kline := range testKlines {
		store, err := processor.getOrCreateStore(kline.Symbol, kline.Interval)
		if err != nil {
			t.Fatalf("failed to get store: %v", err)
		}
		if err := store.SaveKline(kline); err != nil {
			t.Fatalf("failed to save kline: %v", err)
		}
	}

	// 验证创建了正确数量的数据库
	if processor.GetStoreCount() != 3 {
		t.Errorf("expected 3 stores, got %d", processor.GetStoreCount())
	}

	// 验证BTCUSDT 1m数据
	klines, err := processor.QueryKlines("BTCUSDT", "1m", 10)
	if err != nil {
		t.Fatalf("failed to query BTCUSDT 1m: %v", err)
	}
	if len(klines) != 1 || klines[0].ClosePrice != 43520.75 {
		t.Errorf("BTCUSDT 1m data mismatch")
	}

	// 验证BTCUSDT 5m数据
	klines, err = processor.QueryKlines("BTCUSDT", "5m", 10)
	if err != nil {
		t.Fatalf("failed to query BTCUSDT 5m: %v", err)
	}
	if len(klines) != 1 || klines[0].ClosePrice != 43540.0 {
		t.Errorf("BTCUSDT 5m data mismatch")
	}

	// 验证ETHUSDT 1m数据
	klines, err = processor.QueryKlines("ETHUSDT", "1m", 10)
	if err != nil {
		t.Fatalf("failed to query ETHUSDT 1m: %v", err)
	}
	if len(klines) != 1 || klines[0].ClosePrice != 3055.75 {
		t.Errorf("ETHUSDT 1m data mismatch")
	}

	// 验证已加载的stores
	stores := processor.GetLoadedStores()
	if len(stores) != 3 {
		t.Errorf("expected 3 loaded stores, got %d", len(stores))
	}

	t.Logf("✓ Successfully created %d separate databases for different symbols/intervals", processor.GetStoreCount())
}

// TestMultiProcessorDatabaseIsolation 测试数据库隔离
func TestMultiProcessorDatabaseIsolation(t *testing.T) {
	tmpDir := "/tmp/test_isolation"
	defer os.RemoveAll(tmpDir)
	os.MkdirAll(tmpDir, 0755)

	processor, err := NewMultiKlineProcessor(tmpDir)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}
	defer processor.Close()

	// 为BTCUSDT 1m保存两条数据
	btcKlines := []*KlineData{
		{
			EventType:   "kline",
			EventTime:   1638747660000,
			Symbol:      "BTCUSDT",
			StartTime:   1638747600000,
			CloseTime:   1638747660000,
			Interval:    "1m",
			OpenPrice:   43500.0,
			ClosePrice:  43520.0,
			HighPrice:   43550.0,
			LowPrice:    43480.0,
			BaseVolume:  100.0,
			QuoteVolume: 4350000.0,
			IsClosed:    true,
		},
		{
			EventType:   "kline",
			EventTime:   1638747720000,
			Symbol:      "BTCUSDT",
			StartTime:   1638747660000,
			CloseTime:   1638747720000,
			Interval:    "1m",
			OpenPrice:   43520.0,
			ClosePrice:  43540.0,
			HighPrice:   43560.0,
			LowPrice:    43510.0,
			BaseVolume:  105.0,
			QuoteVolume: 4370700.0,
			IsClosed:    true,
		},
	}

	store, _ := processor.getOrCreateStore("BTCUSDT", "1m")
	for _, kline := range btcKlines {
		store.SaveKline(kline)
	}

	// 为ETHUSDT 1m保存一条数据
	ethKline := &KlineData{
		EventType:   "kline",
		EventTime:   1638747780000,
		Symbol:      "ETHUSDT",
		StartTime:   1638747720000,
		CloseTime:   1638747780000,
		Interval:    "1m",
		OpenPrice:   3045.0,
		ClosePrice:  3055.0,
		HighPrice:   3060.0,
		LowPrice:    3045.0,
		BaseVolume:  200.0,
		QuoteVolume: 611000.0,
		IsClosed:    true,
	}

	ethStore, _ := processor.getOrCreateStore("ETHUSDT", "1m")
	ethStore.SaveKline(ethKline)

	// 验证BTCUSDT有2条
	count, _ := processor.GetKlineCount("BTCUSDT", "1m")
	if count != 2 {
		t.Errorf("BTCUSDT 1m: expected 2, got %d", count)
	}

	// 验证ETHUSDT有1条
	count, _ = processor.GetKlineCount("ETHUSDT", "1m")
	if count != 1 {
		t.Errorf("ETHUSDT 1m: expected 1, got %d", count)
	}

	t.Logf("✓ Database isolation verified: separate storage for each symbol+interval")
}
