# 📁 按品种和时间自动分文件的日志系统

## ✅ 已实现的功能

### 自动分文件结构

现在日志会按**启动时间**和**品种周期**自动分文件：

```
logs/
└── session_20251205_120530/        # 启动时间命名的会话目录
    ├── BTCUSDT_3m.log              # BTC 3分钟K线日志
    ├── ETHUSDT_3m.log              # ETH 3分钟K线日志
    ├── BTCUSDT_1m.log              # BTC 1分钟K线日志（如果监听的话）
    └── ETHUSDT_1m.log              # ETH 1分钟K线日志
```

### 文件命名规则

- **会话目录**: `session_YYYYMMDD_HHMMSS`
  - 例如: `session_20251205_120530` 表示 2025年12月5日 12:05:30 启动
  
- **日志文件**: `{SYMBOL}_{INTERVAL}.log`
  - 例如: `BTCUSDT_3m.log` 表示BTCUSDT的3分钟K线日志

## 🎯 优势

### 1. 清晰的会话管理
每次启动都有独立的会话目录，方便：
- 回测不同时段的策略表现
- 对比不同启动的运行情况
- 独立分析每个交易会话

### 2. 按品种隔离
每个品种有独立的日志文件，便于：
- 单独分析某个品种的表现
- 减少文件大小，提高查询速度
- 并行分析多个品种

### 3. 自动化管理
- 启动时自动创建新会话
- 自动创建所需的日志文件
- 文件轮转和压缩（单文件50MB，保留5个备份）

## 📝 日志示例

### BTCUSDT_3m.log
```json
{"level":"INFO","time":"2025-12-05T12:05:30Z","msg":"Log file created","symbol":"BTCUSDT","interval":"3m","file":"logs/session_20251205_120530/BTCUSDT_3m.log"}
{"level":"INFO","time":"2025-12-05T12:06:00Z","msg":"Kline received","open_time":"2025-12-05T12:03:00Z","open":91000,"high":91200,"low":90900,"close":91150,"volume":234.56}
{"level":"INFO","time":"2025-12-05T12:06:01Z","msg":"Signal generated","signal_type":"OPEN_LONG","price":91150,"confidence":0.85,"reason":"MACD金叉+EMA5/VWAP8金叉"}
{"level":"INFO","time":"2025-12-05T12:06:02Z","msg":"Order placed","order_id":"BTCUSDT_1733400362","side":"BUY","type":"MARKET","quantity":0.1,"price":91150,"status":"FILLED"}
```

### ETHUSDT_3m.log
```json
{"level":"INFO","time":"2025-12-05T12:05:30Z","msg":"Log file created","symbol":"ETHUSDT","interval":"3m","file":"logs/session_20251205_120530/ETHUSDT_3m.log"}
{"level":"INFO","time":"2025-12-05T12:06:00Z","msg":"Kline received","open_time":"2025-12-05T12:03:00Z","open":3100,"high":3120,"low":3090,"close":3110,"volume":1234.5}
```

## 🛠️ 使用方法

### 启动程序（自动分文件）

```bash
./scripts/start-live.sh
```

程序启动时会自动：
1. 创建会话目录（`session_YYYYMMDD_HHMMSS`）
2. 为每个监听的品种创建独立日志文件
3. 打印会话信息

### 查看日志

#### 查看当前会话的所有文件
```bash
# 查找最新的会话目录
ls -lt logs/ | head -2

# 进入会话目录
cd logs/session_20251205_120530/

# 查看BTCUSDT的日志
tail -f BTCUSDT_3m.log | jq
```

#### 实时查看特定品种
```bash
# 查看BTC的日志
tail -f logs/session_*/BTCUSDT_3m.log | jq

# 查看ETH的日志
tail -f logs/session_*/ETHUSDT_3m.log | jq
```

#### 对比不同会话
```bash
# 对比两个会话的BTC表现
diff <(jq -r '.msg' logs/session_20251205_100000/BTCUSDT_3m.log) \
     <(jq -r '.msg' logs/session_20251205_120000/BTCUSDT_3m.log)
```

### 分析历史数据

#### 统计某个品种的信号数
```bash
grep "Signal generated" logs/session_20251205_120530/BTCUSDT_3m.log | wc -l
```

#### 查看某个品种的所有订单
```bash
grep "Order placed" logs/session_20251205_120530/BTCUSDT_3m.log | jq
```

#### 导出某个品种的K线数据
```bash
grep "Kline received" logs/session_20251205_120530/BTCUSDT_3m.log | \
  jq -r '[.time, .close, .volume] | @csv' > btc_klines.csv
```

## 📊 文件管理

### 自动轮转
每个品种的日志文件会自动轮转：
- 单文件最大50MB
- 保留5个备份
- 旧文件自动压缩

示例：
```
BTCUSDT_3m.log              # 当前文件
BTCUSDT_3m.log.1.gz         # 备份1（压缩）
BTCUSDT_3m.log.2.gz         # 备份2
...
BTCUSDT_3m.log.5.gz         # 备份5（最旧，之后删除）
```

### 清理旧会话
```bash
# 删除30天前的会话
find logs -name "session_*" -type d -mtime +30 -exec rm -rf {} \;

# 只保留最近10个会话
ls -t logs/session_* | tail -n +11 | xargs rm -rf
```

## 🔍 查看工具

### 方式1: 使用日志查看脚本
```bash
./scripts/view-logs.sh
```

### 方式2: 直接查看
```bash
# 查看最新会话
cd $(ls -td logs/session_* | head -1)

# 查看所有品种的最新日志
for log in *.log; do
    echo "=== $log ==="
    tail -3 $log | jq
done
```

### 方式3: 监控所有品种
```bash
# 同时监控多个品种（需要tmux）
tmux new-session -d -s trading
tmux split-window -h
tmux select-pane -t 0
tmux send-keys "tail -f logs/session_*/BTCUSDT_3m.log | jq" Enter
tmux select-pane -t 1
tmux send-keys "tail -f logs/session_*/ETHUSDT_3m.log | jq" Enter
tmux attach -t trading
```

## 📚 配置选项

### 修改日志级别
编辑 `cmd/live-trading/main.go`:
```go
symbolLogger := logger.InitSymbolLogger("info")  // debug, info, warn, error
```

### 修改文件大小和保留数
编辑 `internal/logger/symbol_logger.go`:
```go
maxSize:    50,  // 单文件大小(MB)
maxBackups: 5,   // 保留备份数
maxAge:     30,  // 保留天数
```

## 🎯 典型使用场景

### 场景1: 实时监控多个品种
```bash
# 终端1: 监控BTC
tail -f logs/session_$(date +%Y%m%d)_*/BTCUSDT_3m.log | grep "Signal"

# 终端2: 监控ETH
tail -f logs/session_$(date +%Y%m%d)_*/ETHUSDT_3m.log | grep "Signal"
```

### 场景2: 分析单个品种表现
```bash
# 查看BTC今天的所有信号
cat logs/session_$(date +%Y%m%d)_*/BTCUSDT_3m.log | grep "Signal" | jq
```

### 场景3: 对比不同时段策略
```bash
# 对比上午和下午的表现
grep "Signal" logs/session_20251205_090000/BTCUSDT_3m.log | wc -l
grep "Signal" logs/session_20251205_140000/BTCUSDT_3m.log | wc -l
```

## ✅ 总结

### 你的需求
> 根据开始时间和品种周期自动区分文件

### 实现的功能
✅ **按启动时间分目录** - `session_YYYYMMDD_HHMMSS`  
✅ **按品种周期分文件** - `BTCUSDT_3m.log`, `ETHUSDT_3m.log`  
✅ **自动创建和管理** - 无需手动配置  
✅ **独立分析** - 每个品种独立文件  
✅ **清晰追溯** - 每个会话独立目录  

### 下次启动时
程序会自动创建新的会话目录：
```
logs/
├── session_20251205_120530/    # 第一次启动
│   ├── BTCUSDT_3m.log
│   └── ETHUSDT_3m.log
├── session_20251205_140000/    # 第二次启动（新的会话）
│   ├── BTCUSDT_3m.log
│   └── ETHUSDT_3m.log
└── session_20251206_090000/    # 第三次启动
    ├── BTCUSDT_3m.log
    └── ETHUSDT_3m.log
```

**完美实现你的需求！** 🎉

