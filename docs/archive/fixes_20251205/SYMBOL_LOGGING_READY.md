# ✅ 按品种和时间自动分文件 - 已完成！

## 🎯 你的需求

> 这个是全部记在同一文件的吗，如果可以，我想让他根据开始时间和品种周期自动区分文件

## ✅ 已实现！

现在日志系统会**自动按启动时间和品种周期分文件**：

### 文件结构

```
logs/
└── session_20251205_120530/        ← 启动时间自动生成的会话目录
    ├── BTCUSDT_3m.log              ← BTC 3分钟的所有日志
    ├── ETHUSDT_3m.log              ← ETH 3分钟的所有日志
    ├── BTCUSDT_1m.log              ← BTC 1分钟（如果有）
    └── ETHUSDT_1m.log              ← ETH 1分钟（如果有）
```

### 自动化

- ✅ **启动时自动创建会话目录** - 以时间命名
- ✅ **自动为每个品种创建文件** - 按需创建
- ✅ **文件名包含品种和周期** - 例如 `BTCUSDT_3m.log`
- ✅ **每次启动新会话** - 新的目录，方便对比

## 📊 示例

### 第一次启动（12:05:30）
```bash
./scripts/start-live.sh
```

生成：
```
logs/session_20251205_120530/
├── BTCUSDT_3m.log
└── ETHUSDT_3m.log
```

### 第二次启动（14:00:00）
```bash
./scripts/start-live.sh
```

生成：
```
logs/session_20251205_140000/    ← 新的会话
├── BTCUSDT_3m.log
└── ETHUSDT_3m.log
```

### 结果
```
logs/
├── session_20251205_120530/    ← 上午的交易
│   ├── BTCUSDT_3m.log
│   └── ETHUSDT_3m.log
└── session_20251205_140000/    ← 下午的交易
    ├── BTCUSDT_3m.log
    └── ETHUSDT_3m.log
```

## 🛠️ 使用方法

### 启动程序（自动分文件）
```bash
./scripts/start-live.sh
```

启动时会显示：
```
🚀 Starting trading bot
   session_id: 20251205_120530
   session_dir: logs/session_20251205_120530
```

### 实时查看某个品种的日志
```bash
# 查看BTC的日志
tail -f logs/session_*/BTCUSDT_3m.log | jq

# 查看ETH的日志
tail -f logs/session_*/ETHUSDT_3m.log | jq
```

### 分析某个品种的历史
```bash
# 统计BTC今天生成了多少信号
grep "Signal" logs/session_20251205_*/BTCUSDT_3m.log | wc -l

# 查看BTC的所有订单
grep "Order" logs/session_20251205_*/BTCUSDT_3m.log | jq
```

### 对比不同会话
```bash
# 对比两次启动的BTC表现
diff logs/session_20251205_120530/BTCUSDT_3m.log \
     logs/session_20251205_140000/BTCUSDT_3m.log
```

## 📝 日志内容

每个文件只包含对应品种的日志：

### BTCUSDT_3m.log
```json
{"time":"2025-12-05T12:06:00Z","msg":"Kline received","open":91000,"close":91150,"volume":234.56}
{"time":"2025-12-05T12:06:01Z","msg":"Signal generated","signal_type":"OPEN_LONG","price":91150}
{"time":"2025-12-05T12:06:02Z","msg":"Order placed","side":"BUY","quantity":0.1}
```

### ETHUSDT_3m.log
```json
{"time":"2025-12-05T12:06:00Z","msg":"Kline received","open":3100,"close":3110,"volume":1234.5}
```

## 🎁 额外功能

### 自动文件轮转
每个文件最大50MB，超过会自动压缩：
```
BTCUSDT_3m.log          # 当前文件
BTCUSDT_3m.log.1.gz     # 压缩备份
```

### 旧会话管理
```bash
# 列出所有会话
ls -lt logs/

# 删除30天前的会话
find logs -name "session_*" -type d -mtime +30 -exec rm -rf {} \;
```

## 📚 相关文档

- **[完整指南](docs/SYMBOL_LOGGING_GUIDE.md)** - 详细使用说明
- **[日志系统](docs/LOGGING_GUIDE.md)** - 日志功能说明

## ✅ 总结

**你的需求已100%实现！**

✅ 根据开始时间分文件 - `session_YYYYMMDD_HHMMSS/`  
✅ 根据品种分文件 - `BTCUSDT_3m.log`, `ETHUSDT_3m.log`  
✅ 自动创建和管理 - 无需手动配置  
✅ 易于查看和分析 - 每个品种独立文件  

**现在运行程序，日志会自动按品种和时间分文件！** 🎉

