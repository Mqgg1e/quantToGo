# K线数据存储模块

该模块提供了WebSocket K线数据的解析和SQLite数据库存储功能。

## 功能特性

- ✅ 从Binance WebSocket订阅K线数据
- ✅ 自动解析JSON消息为结构化数据
- ✅ 将收盘K线自动保存到SQLite数据库
- ✅ 支持批量保存和事务处理
- ✅ 数字类型正确存储（浮点数而非字符串）
- ✅ 提供查询接口获取已保存的K线数据

## 数据模型

### KlineData 结构

```go
type KlineData struct {
    EventType    string    // 事件类型: "kline"
    EventTime    int64     // 事件时间 (毫秒)
    Symbol       string    // 交易对: "BTCUSDT"
    StartTime    int64     // K线开始时间 (毫秒)
    CloseTime    int64     // K线收盘时间 (毫秒)
    Interval     string    // 时间间隔: "1m", "5m", "1h"
    OpenPrice    float64   // 开盘价格
    ClosePrice   float64   // 收盘价格
    HighPrice    float64   // 最高价格
    LowPrice     float64   // 最低价格
    BaseVolume   float64   // 基础资产成交量
    QuoteVolume  float64   // 计价资产成交量
    IsClosed     bool      // 是否已收盘
}
```

## 数据库架构

### 表结构

```sql
CREATE TABLE klines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT,
    event_time INTEGER,
    symbol TEXT,
    start_time INTEGER,
    close_time INTEGER,
    interval TEXT,
    open_price REAL,          -- 浮点数存储
    close_price REAL,         -- 浮点数存储
    high_price REAL,          -- 浮点数存储
    low_price REAL,           -- 浮点数存储
    base_volume REAL,         -- 浮点数存储
    quote_volume REAL,        -- 浮点数存储
    is_closed INTEGER,
    created_at DATETIME,
    UNIQUE(symbol, interval, close_time)
);
```

### 索引

- `symbol, interval` - 快速查询特定交易对和时间间隔
- `close_time` - 快速按时间排序
- `symbol, interval, close_time` - 复合索引用于精确查询

## 使用示例

### 基础使用

```go
package main

import (
    "context"
    "time"
    "goQuant/internal/dataManager"
)

func main() {
    // 1. 创建处理器（指定数据库路径）
    processor, err := dataManager.NewKlineProcessor("./data/klines.db")
    if err != nil {
        panic(err)
    }
    defer processor.Close()

    // 2. 连接到WebSocket
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
    defer cancel()

    msgCh, errCh, closeFn := dataManager.SubscribeKlines(ctx, "BTCUSDT", "1m", "")
    defer closeFn()

    // 3. 处理K线流（自动保存收盘K线到数据库）
    processor.ProcessStream(ctx, msgCh, errCh)
}
```

### 查询已保存的K线

```go
// 查询最新的10条K线
klines, err := processor.QueryKlines("BTCUSDT", "1m", 10)
if err != nil {
    log.Fatal(err)
}

for _, kline := range klines {
    fmt.Printf("Symbol: %s, Interval: %s, Close: %.2f, Time: %d\n",
        kline.Symbol, kline.Interval, kline.ClosePrice, kline.CloseTime)
}

// 获取K线总数
count, err := processor.GetKlineCount("BTCUSDT", "1m")
fmt.Printf("Total klines: %d\n", count)
```

### 手动保存单条K线

```go
kline := &KlineData{
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
}

err := processor.store.SaveKline(kline)
if err != nil {
    log.Fatal(err)
}
```

### 直接使用KlineStore

```go
// 创建存储实例（跳过处理器）
store, err := dataManager.NewKlineStore("./data/klines.db")
if err != nil {
    panic(err)
}
defer store.Close()

// 保存单条K线
store.SaveKline(kline)

// 批量保存（事务处理）
klines := []*KlineData{...}
store.SaveKlines(klines)

// 查询
results, _ := store.GetKlines("BTCUSDT", "1m", 50)
```

## 工作流程

```
WebSocket订阅
    ↓
接收JSON消息
    ↓
ParseKlineEvent() - 解析为KlineData
    ↓
检查IsClosed是否为true
    ↓
是 → SaveKline() - 保存到数据库
    ↓
否 → 继续监听下一条消息
```

## 文件说明

- `models.go` - 数据结构和解析逻辑
- `klinestore.go` - SQLite数据库操作
- `processor.go` - 高级处理器（自动保存收盘K线）
- `utils.go` - 辅助函数
- `processor_test.go` - 单元测试和示例

## 依赖

- `github.com/gorilla/websocket` - WebSocket连接
- `github.com/mattn/go-sqlite3` - SQLite驱动

## 测试

```bash
go test ./internal/dataManager/ -v
```

## 注意事项

1. **数字类型**: 所有价格和成交量都以 `float64` 类型存储，确保精度
2. **收盘判断**: 只有当 `x` 字段为 `true` 时才会保存K线
3. **唯一性**: 每条K线使用 `symbol + interval + close_time` 作为唯一键，重复插入会被替换
4. **代理设置**: 如需代理，传入代理URL如 `"http://127.0.0.1:7890"`
5. **线程安全**: 所有数据库操作都是线程安全的

## 示例消息格式

来自Binance WebSocket的原始K线消息：

```json
{
  "e": "kline",
  "E": 1764752110147,
  "s": "ETHUSDT",
  "k": {
    "t": 1764752100000,
    "T": 1764752159999,
    "s": "ETHUSDT",
    "i": "1m",
    "o": "3046.22",
    "c": "3045.67",
    "h": "3046.48",
    "l": "3045.67",
    "v": "320.564",
    "q": "976494.31500",
    "x": true
  }
}
```

记录的项目（你指定的）:
- `e`: 事件类型 ("kline")
- `E`: 事件时间 (毫秒，整数)
- `s`: 交易对符号 ("ETHUSDT")
- `t`: K线开始时间 (毫秒，整数)
- `T`: K线收盘时间 (毫秒，整数)
- `i`: 时间间隔 ("1m", "5m", etc.)
- `o`: 开盘价 (浮点数)
- `c`: 收盘价 (浮点数)
- `h`: 最高价 (浮点数)
- `l`: 最低价 (浮点数)
- `v`: 基础资产成交量 (浮点数)
- `q`: 计价资产成交量 (浮点数)
