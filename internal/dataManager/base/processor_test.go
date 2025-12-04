package base

import (
	"os"
	"testing"
)

// TestKlineProcessor 测试K线处理器和数据库存储
func TestKlineProcessor(t *testing.T) {
	// 创建临时数据库
	dbPath := "/tmp/test_klines.db"
	defer os.Remove(dbPath)

	// 创建处理器
	processor, err := NewKlineProcessor(dbPath)
	if err != nil {
		t.Fatalf("failed to create processor: %v", err)
	}
	defer processor.Close()

	// 创建模拟的K线数据
	testKline := &KlineData{
		EventType:   "kline",
		EventTime:   1638747660000,
		Symbol:      "BTCUSDT",
		StartTime:   1638747660000,
		CloseTime:   1638747719999,
		Interval:    "1m",
		OpenPrice:   43500.50,
		ClosePrice:  43520.75,
		HighPrice:   43550.25,
		LowPrice:    43480.00,
		BaseVolume:  100.5,
		QuoteVolume: 4365200.50,
		IsClosed:    true,
	}

	// 保存K线
	if err := processor.store.SaveKline(testKline); err != nil {
		t.Fatalf("failed to save kline: %v", err)
	}

	// 查询K线
	klines, err := processor.QueryKlines("BTCUSDT", "1m", 10)
	if err != nil {
		t.Fatalf("failed to query klines: %v", err)
	}

	if len(klines) != 1 {
		t.Fatalf("expected 1 kline, got %d", len(klines))
	}

	kline := klines[0]
	if kline.Symbol != "BTCUSDT" {
		t.Errorf("expected symbol BTCUSDT, got %s", kline.Symbol)
	}
	if kline.ClosePrice != 43520.75 {
		t.Errorf("expected close price 43520.75, got %f", kline.ClosePrice)
	}

	// 验证计数
	count, err := processor.GetKlineCount("BTCUSDT", "1m")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected count 1, got %d", count)
	}

	t.Logf("✓ Successfully saved and retrieved kline: %s %s close=%.2f",
		kline.Symbol, kline.Interval, kline.ClosePrice)
}

// TestParseKlineEvent 测试K线事件解析
func TestParseKlineEvent(t *testing.T) {
	// 模拟Binance WebSocket返回的JSON消息
	jsonMsg := []byte(`{
		"e":"kline",
		"E":1764752110147,
		"s":"ETHUSDT",
		"k":{
			"t":1764752100000,
			"T":1764752159999,
			"s":"ETHUSDT",
			"i":"1m",
			"f":6966507461,
			"L":6966507969,
			"o":"3046.22",
			"c":"3045.67",
			"h":"3046.48",
			"l":"3045.67",
			"v":"320.564",
			"n":507,
			"x":true,
			"q":"976494.31500",
			"V":"49.871",
			"Q":"151914.82953",
			"B":"0"
		}
	}`)

	kline, err := ParseKlineEvent(jsonMsg)
	if err != nil {
		t.Fatalf("failed to parse kline: %v", err)
	}

	if kline.Symbol != "ETHUSDT" {
		t.Errorf("expected symbol ETHUSDT, got %s", kline.Symbol)
	}
	if kline.Interval != "1m" {
		t.Errorf("expected interval 1m, got %s", kline.Interval)
	}
	if kline.ClosePrice != 3045.67 {
		t.Errorf("expected close price 3045.67, got %f", kline.ClosePrice)
	}
	if !kline.IsClosed {
		t.Errorf("expected kline to be closed")
	}

	t.Logf("✓ Successfully parsed kline: %s %s close=%.2f (closed=%v)",
		kline.Symbol, kline.Interval, kline.ClosePrice, kline.IsClosed)
}

// TestKlineStoreWithMultipleRecords 测试批量保存
func TestKlineStoreWithMultipleRecords(t *testing.T) {
	dbPath := "/tmp/test_klines_batch.db"
	defer os.Remove(dbPath)

	store, err := NewKlineStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// 创建多条K线数据
	klines := []*KlineData{
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

	// 批量保存
	if err := store.SaveKlines(klines); err != nil {
		t.Fatalf("failed to save klines: %v", err)
	}

	// 验证数量
	count, err := store.GetKlineCount("BTCUSDT", "1m")
	if err != nil {
		t.Fatalf("failed to get count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected 2 klines, got %d", count)
	}

	t.Logf("✓ Successfully saved and verified %d klines", count)
}
