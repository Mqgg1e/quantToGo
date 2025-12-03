# K线存储API 快速参考

## 核心API

### 创建处理器
```go
processor, err := dataManager.NewKlineProcessor("./data/klines.db")
defer processor.Close()
```

### 订阅K线
```go
msgCh, errCh, closeFn := dataManager.SubscribeKlines(
    ctx,           // context.Context
    "BTCUSDT",     // 交易对
    "1m",          // 时间间隔
    "",            // 代理URL（可选）
)
defer closeFn()
```

### 处理K线流（自动保存）
```go
// 在goroutine中运行，自动保存收盘K线到数据库
processor.ProcessStream(ctx, msgCh, errCh)
```

### 解析单条K线
```go
jsonData := []byte(`{"e":"kline",...}`)
kline, err := dataManager.ParseKlineEvent(jsonData)
```

### 查询K线
```go
// 获取最新的10条K线
klines, err := processor.QueryKlines("BTCUSDT", "1m", 10)

// 获取K线总数
count, err := processor.GetKlineCount("BTCUSDT", "1m")
```

### 手动保存
```go
kline := &dataManager.KlineData{
    Symbol:     "BTCUSDT",
    Interval:   "1m",
    ClosePrice: 43500.50,
    OpenPrice:  43400.00,
    HighPrice:  43550.00,
    LowPrice:   43380.00,
    BaseVolume: 100.5,
    QuoteVolume: 4365000.0,
    IsClosed:   true,
    // ... 其他字段
}
processor.store.SaveKline(kline)
```

## KlineData 结构

```go
type KlineData struct {
    EventType   string    // "kline"
    EventTime   int64     // 毫秒时间戳
    Symbol      string    // "BTCUSDT"
    StartTime   int64     // K线开始时间
    CloseTime   int64     // K线收盘时间
    Interval    string    // "1m", "5m", "1h"
    OpenPrice   float64   // 开盘价
    ClosePrice  float64   // 收盘价
    HighPrice   float64   // 最高价
    LowPrice    float64   // 最低价
    BaseVolume  float64   // 基础资产成交量
    QuoteVolume float64   // 计价资产成交量
    IsClosed    bool      // 是否收盘
}
```

## 时间转换

```go
// 毫秒时间戳 → time.Time
closeTime := time.UnixMilli(kline.CloseTime)

// time.Time → 格式化字符串
formatted := time.UnixMilli(kline.CloseTime).Format("2006-01-02 15:04:05")
```

## 完整示例

```go
package main

import (
	"context"
	"fmt"
	"time"
	dfs "goQuant/internal/dataManager"
)

func main() {
	// 创建处理器
	processor, _ := dfs.NewKlineProcessor("./data/klines.db")
	defer processor.Close()

	// 订阅K线
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	msgCh, errCh, closeFn := dfs.SubscribeKlines(ctx, "BTCUSDT", "1m", "")
	defer closeFn()

	// 处理流（自动保存收盘K线）
	go processor.ProcessStream(ctx, msgCh, errCh)

	// 等待
	<-ctx.Done()

	// 查询结果
	klines, _ := processor.QueryKlines("BTCUSDT", "1m", 5)
	for _, k := range klines {
		fmt.Printf("时间: %s, 收盘价: %.2f, 成交量: %.2f\n",
			time.UnixMilli(k.CloseTime).Format("15:04:05"),
			k.ClosePrice,
			k.BaseVolume,
		)
	}
}
```

## 常见问题

**Q: 如何指定数据库路径？**
A: 在 `NewKlineProcessor()` 时传入路径：`NewKlineProcessor("./my_db/klines.db")`

**Q: 如何同时订阅多个交易对？**
A: 为每个交易对创建一个 goroutine 调用 `ProcessStream()`

**Q: 数据是否会重复？**
A: 不会，数据库有 UNIQUE 约束防止重复：`(symbol, interval, close_time)`

**Q: 只有收盘K线才会保存吗？**
A: 是的，只有当 `IsClosed == true` 时才自动保存

**Q: 价格是如何存储的？**
A: 以 `float64` 类型存储，支持精确计算和比较

**Q: 可以离线使用吗？**
A: `KlineStore` 不需要网络，`SubscribeKlines` 需要网络
