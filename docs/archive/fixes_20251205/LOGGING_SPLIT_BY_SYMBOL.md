# 🎉 按品种和时间分文件 - 功能说明

## 你的需求
> 我想让日志根据开始时间和品种周期自动区分文件

## ✅ 已实现！

### 实现方式

程序启动时会自动创建：
```
logs/
└── session_20251205_120530/        ← 根据启动时间自动创建
    ├── BTCUSDT_3m.log              ← BTC 3分钟的日志
    └── ETHUSDT_3m.log              ← ETH 3分钟的日志
```

### 工作流程

1. **程序启动** → 创建会话目录 `session_YYYYMMDD_HHMMSS`
2. **接收K线** → 自动创建对应品种的日志文件
3. **记录数据** → 写入对应文件

### 实现代码

#### 核心文件
- ✅ `internal/logger/symbol_logger.go` - 按品种分文件的日志系统
- ✅ `internal/strategy/adapter.go` - 集成到策略适配器
- ✅ `cmd/live-trading/main.go` - 主程序初始化

#### 使用示例

```go
// 初始化（在main.go中）
symbolLogger := logger.InitSymbolLogger("info")
defer logger.CloseSymbolLogger()

// 获取品种专用logger（在adapter中）
btcLogger := symbolLogger.GetLogger("BTCUSDT", "3m")
ethLogger := symbolLogger.GetLogger("ETHUSDT", "3m")

// 记录日志（自动写入对应文件）
btcLogger.Info("Kline received", zap.Float64("close", 91150.00))
ethLogger.Info("Kline received", zap.Float64("close", 3117.58))
```

## 📊 效果展示

### 第一次启动（上午12:05）
```bash
$ ./scripts/start-live.sh

🚀 Starting trading bot
   session_id: 20251205_120530
   session_dir: logs/session_20251205_120530
```

生成结构：
```
logs/session_20251205_120530/
├── BTCUSDT_3m.log   ← BTC的所有日志
└── ETHUSDT_3m.log   ← ETH的所有日志
```

### 第二次启动（下午14:00）
```bash
$ ./scripts/start-live.sh

🚀 Starting trading bot
   session_id: 20251205_140000
   session_dir: logs/session_20251205_140000
```

生成结构：
```
logs/
├── session_20251205_120530/  ← 上午的会话
│   ├── BTCUSDT_3m.log
│   └── ETHUSDT_3m.log
└── session_20251205_140000/  ← 下午的会话（新的）
    ├── BTCUSDT_3m.log
    └── ETHUSDT_3m.log
```

## 🛠️ 实际使用

### 查看特定品种的日志
```bash
# BTC的日志
tail -f logs/session_*/BTCUSDT_3m.log | jq

# ETH的日志
tail -f logs/session_*/ETHUSDT_3m.log | jq
```

### 分析特定品种
```bash
# 统计BTC的信号数
grep "Signal" logs/session_20251205_*/BTCUSDT_3m.log | wc -l

# 查看BTC的订单
grep "Order" logs/session_20251205_*/BTCUSDT_3m.log | jq
```

### 对比不同会话
```bash
# 对比上午和下午BTC的表现
wc -l logs/session_20251205_120530/BTCUSDT_3m.log
wc -l logs/session_20251205_140000/BTCUSDT_3m.log
```

## 📝 日志内容示例

### logs/session_20251205_120530/BTCUSDT_3m.log
```json
{"level":"INFO","time":"2025-12-05T12:06:00Z","msg":"Kline received","open_time":"2025-12-05T12:03:00Z","open":91000,"high":91200,"low":90900,"close":91150,"volume":234.56}
{"level":"INFO","time":"2025-12-05T12:06:01Z","msg":"Signal generated","signal_type":"OPEN_LONG","price":91150,"confidence":0.85,"reason":"MACD金叉+EMA5/VWAP8金叉"}
{"level":"INFO","time":"2025-12-05T12:06:02Z","msg":"Order placed","order_id":"BTCUSDT_1733400362","side":"BUY","type":"MARKET","quantity":0.1,"price":91150,"status":"FILLED"}
{"level":"INFO","time":"2025-12-05T12:09:00Z","msg":"Kline received","open_time":"2025-12-05T12:06:00Z","open":91150,"high":91180,"low":91140,"close":91160,"volume":189.23}
```

### logs/session_20251205_120530/ETHUSDT_3m.log
```json
{"level":"INFO","time":"2025-12-05T12:06:00Z","msg":"Kline received","open_time":"2025-12-05T12:03:00Z","open":3100,"high":3120,"low":3090,"close":3110,"volume":1234.5}
{"level":"INFO","time":"2025-12-05T12:09:00Z","msg":"Kline received","open_time":"2025-12-05T12:06:00Z","open":3110,"high":3115,"low":3105,"close":3112,"volume":987.6}
```

## ✅ 优势

### 1. 清晰隔离
- ✅ 每个品种独立文件
- ✅ 每次启动独立会话
- ✅ 易于查找和分析

### 2. 性能优化
- ✅ 文件更小，查询更快
- ✅ 可并行分析多个品种
- ✅ 自动轮转（50MB/文件）

### 3. 便于管理
- ✅ 自动创建目录和文件
- ✅ 命名规范统一
- ✅ 可轻松删除旧会话

## 📚 相关文档

- **[SYMBOL_LOGGING_READY.md](SYMBOL_LOGGING_READY.md)** - 快速开始
- **[SYMBOL_LOGGING_GUIDE.md](docs/SYMBOL_LOGGING_GUIDE.md)** - 完整指南

## 🎯 总结

**完全满足你的需求！**

✅ **根据开始时间** → `session_20251205_120530/`  
✅ **根据品种周期** → `BTCUSDT_3m.log`, `ETHUSDT_3m.log`  
✅ **自动区分文件** → 无需手动配置  
✅ **易于分析** → 每个品种独立  

**现在启动程序，日志会自动按品种和时间分文件！** 🎉

