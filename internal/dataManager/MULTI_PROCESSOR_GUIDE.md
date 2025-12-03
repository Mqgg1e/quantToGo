# 多交易对多周期独立数据库存储

## 功能说明

`MultiKlineProcessor` 提供了为每个**交易对+周期组合**创建独立SQLite数据库的功能。这样做的好处是：

✅ **数据隔离** - 不同交易对/周期的K线数据完全分离  
✅ **性能优化** - 查询时只需访问特定的小型数据库，速度更快  
✅ **便于管理** - 可以独立地删除、备份、导入导出特定交易对的数据  
✅ **并发友好** - 多个goroutine可以同时访问不同的数据库  
✅ **扩展灵活** - 可以轻松添加新的交易对和周期  

## 文件命名规则

```
baseDir/
├── BTCUSDT_1m.db    # BTC 1分钟K线数据
├── BTCUSDT_5m.db    # BTC 5分钟K线数据
├── ETHUSDT_1m.db    # ETH 1分钟K线数据
├── ETHUSDT_5m.db    # ETH 5分钟K线数据
├── BNBUSDT_1m.db    # BNB 1分钟K线数据
└── ...
```

## 快速开始

### 基础使用

```go
package main

import (
    "context"
    "goQuant/internal/dataManager"
)

func main() {
    // 创建多处理器，指定基础目录
    processor, err := dataManager.NewMultiKlineProcessor("./data/klines")
    if err != nil {
        panic(err)
    }
    defer processor.Close()

    // 订阅K线
    ctx := context.Background()
    msgCh, errCh, closeFn := dataManager.SubscribeKlines(ctx, "BTCUSDT", "1m", "")
    defer closeFn()

    // 处理消息流（自动为不同Symbol+Interval创建独立DB）
    processor.ProcessStream(ctx, msgCh, errCh)
}
```

### 多交易对多周期订阅

```go
// 配置多个订阅
subscriptions := []struct {
    symbol   string
    interval string
}{
    {"BTCUSDT", "1m"},
    {"BTCUSDT", "5m"},
    {"ETHUSDT", "1m"},
    {"ETHUSDT", "5m"},
    {"BNBUSDT", "1m"},
}

// 为每个订阅启动goroutine
for _, sub := range subscriptions {
    go func(symbol, interval string) {
        msgCh, errCh, closeFn := SubscribeKlines(ctx, symbol, interval, "")
        defer closeFn()
        processor.ProcessStream(ctx, msgCh, errCh)
    }(sub.symbol, sub.interval)
}
```

## API 参考

### 创建处理器

```go
// NewMultiKlineProcessor 创建多K线处理器
// baseDir: 数据库基础目录，每个Symbol+Interval自动创建独立db文件
func NewMultiKlineProcessor(baseDir string) (*MultiKlineProcessor, error)

// 例如:
processor, _ := NewMultiKlineProcessor("./data/klines")
// 会自动为每个Symbol+Interval创建数据库
```

### 数据库操作

```go
// 获取或创建指定Symbol+Interval的存储
store, err := processor.getOrCreateStore("BTCUSDT", "1m")

// 处理消息流（自动识别Symbol+Interval）
processor.ProcessStream(ctx, msgCh, errCh)

// 查询K线数据
klines, err := processor.QueryKlines("BTCUSDT", "1m", 50)  // 获取最新50条

// 获取K线数量
count, err := processor.GetKlineCount("BTCUSDT", "1m")

// 获取已加载的所有Symbol+Interval
stores := processor.GetLoadedStores()  // []string{"BTCUSDT_1m", "BTCUSDT_5m", ...}

// 获取已创建的数据库数量
numDbs := processor.GetStoreCount()

// 关闭处理器
processor.Close()
```

## 存储结构

每个数据库具有相同的表结构：

```sql
CREATE TABLE klines (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    event_type TEXT NOT NULL,           -- "kline"
    event_time INTEGER NOT NULL,        -- 事件时间(毫秒)
    symbol TEXT NOT NULL,               -- 交易对
    start_time INTEGER NOT NULL,        -- K线开始时间(毫秒)
    close_time INTEGER NOT NULL,        -- K线收盘时间(毫秒)
    interval TEXT NOT NULL,             -- 周期
    open_price REAL NOT NULL,           -- 开盘价(float64)
    close_price REAL NOT NULL,          -- 收盘价(float64)
    high_price REAL NOT NULL,           -- 最高价(float64)
    low_price REAL NOT NULL,            -- 最低价(float64)
    base_volume REAL NOT NULL,          -- 基础资产成交量(float64)
    quote_volume REAL NOT NULL,         -- 计价资产成交量(float64)
    is_closed INTEGER NOT NULL,         -- 是否已收盘
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(symbol, interval, close_time)
);

-- 自动创建的索引:
CREATE INDEX idx_klines_symbol_interval ON klines(symbol, interval);
CREATE INDEX idx_klines_close_time ON klines(close_time);
CREATE INDEX idx_klines_symbol_interval_close_time ON klines(symbol, interval, close_time);
```

## 性能特点

### 优势

| 特性 | 优势 |
|------|------|
| 数据隔离 | 每个Symbol+Interval独立DB，查询时I/O量小 |
| 并发访问 | 多个goroutine可同时访问不同DB，无锁竞争 |
| 内存占用 | 数据库缓存较小，总体内存占用低 |
| 删除清理 | 只需删除对应.db文件，不影响其他数据 |
| 备份恢复 | 可以选择性地备份某些Symbol的数据 |

### 应用场景

✅ **多交易对策略** - 同时监控多个币种的多个周期  
✅ **数据分析** - 对单个交易对的数据进行深度分析，响应快  
✅ **分布式部署** - 不同机器监控不同交易对，轻松集成  
✅ **增量更新** - 定期清理旧数据时可以按交易对操作  
✅ **实时告警** - 快速查询最新数据，决策延迟低  

## 实例代码

### 完整示例（cmd/bot/main.go）

```go
package main

func main() {
    baseDir := "./data/klines"
    
    // 创建多处理器
    processor, _ := NewMultiKlineProcessor(baseDir)
    defer processor.Close()

    // 配置订阅列表
    subscriptions := []struct {
        symbol   string
        interval string
    }{
        {"BTCUSDT", "1m"},
        {"BTCUSDT", "5m"},
        {"ETHUSDT", "1m"},
        {"ETHUSDT", "5m"},
    }

    // 并发订阅
    var wg sync.WaitGroup
    for _, sub := range subscriptions {
        wg.Add(1)
        go func(symbol, interval string) {
            defer wg.Done()
            msgCh, errCh, closeFn := SubscribeKlines(ctx, symbol, interval, proxyURL)
            defer closeFn()
            processor.ProcessStream(ctx, msgCh, errCh)
        }(sub.symbol, sub.interval)
    }

    // 定期输出统计
    ticker := time.NewTicker(30 * time.Second)
    for {
        select {
        case <-ticker.C:
            printStatistics(processor, subscriptions)
        }
    }
}
```

## 与单数据库模式对比

### 单数据库模式（KlineProcessor）
```
all_klines.db
├── BTCUSDT 1m 数据
├── BTCUSDT 5m 数据
├── ETHUSDT 1m 数据
└── ETHUSDT 5m 数据
```

**缺点：**
- 所有数据在一个大库中，查询时可能需要扫描大量数据
- 删除某个交易对数据会影响其他数据
- 索引可能不够优化

### 多数据库模式（MultiKlineProcessor） ✅ 推荐
```
data/klines/
├── BTCUSDT_1m.db
├── BTCUSDT_5m.db
├── ETHUSDT_1m.db
└── ETHUSDT_5m.db
```

**优点：**
- 数据完全隔离，查询快速
- 可独立管理每个Symbol的数据
- 支持更好的并发访问
- 更容易扩展和维护

## 使用建议

### 推荐的目录结构

```
project/
├── data/
│   └── klines/              # 多处理器基础目录
│       ├── BTCUSDT_1m.db
│       ├── BTCUSDT_5m.db
│       ├── ETHUSDT_1m.db
│       └── ...
├── cmd/
│   └── bot/
│       └── main.go          # 使用MultiKlineProcessor
└── internal/
    └── dataManager/
        ├── models.go
        ├── multi_processor.go
        └── ...
```

### 生产环境部署

```go
// 使用环境变量配置基础目录
baseDir := os.Getenv("KLINE_DATA_DIR")
if baseDir == "" {
    baseDir = "/var/data/klines"
}

processor, err := NewMultiKlineProcessor(baseDir)
if err != nil {
    log.Fatalf("Failed to create processor: %v", err)
}
defer processor.Close()
```

## 故障排除

### Q: 如何查看已创建的所有数据库？
```bash
ls -lh ./data/klines/
# BTCUSDT_1m.db
# BTCUSDT_5m.db
# ETHUSDT_1m.db
```

### Q: 如何查询某个特定数据库的数据？
```go
klines, err := processor.QueryKlines("BTCUSDT", "1m", 100)
for _, kline := range klines {
    fmt.Printf("%s: %.2f\n", time.UnixMilli(kline.CloseTime), kline.ClosePrice)
}
```

### Q: 如何清理旧数据？
```bash
# 删除某个交易对的所有周期数据
rm ./data/klines/BTCUSDT_*.db

# 删除某个周期的所有交易对数据
rm ./data/klines/*_1m.db

# 清理所有数据
rm -rf ./data/klines/
```

## 总结

`MultiKlineProcessor` 是生产环境的最佳选择，它提供：

- 🚀 **高性能** - 独立数据库确保查询速度
- 🔒 **数据安全** - 各Symbol数据独立存储
- 📊 **易于分析** - 单个交易对的数据更容易深度分析
- 🔄 **高并发** - 支持多goroutine同时操作不同DB
- 💾 **灵活管理** - 独立控制每个交易对的生命周期
