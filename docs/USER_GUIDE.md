# goQuant 用户指南

本文档提供完整的使用指南，包括快速开始、日志使用、模式切换等。

---

## 目录

1. [快速开始](#快速开始)
2. [配置说明](#配置说明)
3. [日志系统](#日志系统)
4. [模式切换](#模式切换)
5. [策略说明](#策略说明)
6. [常见问题](#常见问题)

---

## 快速开始

### 1. 环境准备

**系统要求**:
- Go 1.21+
- SQLite3
- Linux/macOS (Windows 需要 WSL)

**安装依赖**:
```bash
cd goQuant
go mod download
```

### 2. 配置设置

复制配置模板：
```bash
cp config/config.example.yaml config/config.yaml
```

编辑 `config/config.yaml`：

```yaml
# 基本配置
app:
  mode: "live"  # backtest | paper | live
  
# Binance API 配置
execution:
  binance:
    api_key: "YOUR_API_KEY"
    secret_key: "YOUR_SECRET_KEY"
    base_url: "https://testnet.binancefuture.com"  # 测试网
    testnet: true
```

### 3. 运行程序

**测试日志系统**:
```bash
./scripts/test-logging.sh
```

**启动实时交易**:
```bash
./scripts/start-live.sh
```

**查看日志**:
```bash
# 实时查看
tail -f logs/session_*/BTCUSDT_1m.log

# 查看所有日志
./scripts/view-logs.sh
```

---

## 配置说明

### 应用配置

```yaml
app:
  name: "goQuant"
  mode: "live"           # 运行模式
  environment: "development"
```

**运行模式**:
- `backtest` - 回测模式
- `paper` - 模拟盘
- `live` - 实盘交易

### 数据配置

```yaml
data:
  provider: "binance"
  proxy_url: "http://127.0.0.1:7897"  # 代理（可选）
  database_dir: "./data/wsdata"
  
  subscriptions:
    - symbol: "BTCUSDT"
      interval: "1m"
      enable_orderbook: true
      orderbook_levels: 5
```

**交易对配置**:
- `symbol`: 交易对名称
- `interval`: K线周期 (1m, 3m, 5m, 15m, 1h等)
- `enable_orderbook`: 是否订阅订单簿
- `orderbook_levels`: 订单簿深度

### 策略配置

```yaml
strategy:
  name: "MACD_EMA_Strategy"
  warmup_periods: 45  # 预热K线数量
  
  parameters:
    macd_fast: 16
    macd_slow: 26
    macd_signal: 9
    ema_short: 5
    ema_long: 15
    vwap_period: 8
```

### 仓位管理配置

```yaml
position:
  default_leverage: 5
  default_margin_mode: "ISOLATED"
  max_position_size: 0.5  # 最大仓位占比50%
  
  position_sizing:
    open_percent: 0.20    # 开仓使用20%资金
    add_percent: 0.40     # 加仓使用40%资金
  
  risk_limits:
    max_drawdown: 0.20           # 最大回撤20%
    max_daily_loss: 0.05         # 单日最大亏损5%
    max_open_positions: 3        # 最多3个持仓
    stop_loss_percent: 0.006     # 固定止损0.6%
    take_profit_percent: 0.048   # 最高止盈4.8%
  
  trailing_stop:
    enabled: true
    level1_profit_min: 0.006     # 0.6%-1.0%盈利
    level1_callback: 0.005       # 回撤0.5%止盈
    level2_profit_min: 0.010     # 1.0%-1.8%盈利
    level2_callback: 0.0055      # 回撤0.55%止盈
    level3_profit_min: 0.018     # 1.8%-4.8%盈利
    level3_callback: 0.0068      # 回撤0.68%止盈
    level4_profit_min: 0.048     # >4.8%盈利
    level4_callback: 0.008       # 回撤0.8%止盈
```

### 执行配置

```yaml
execution:
  mode: "live"
  exchange: "binance"
  
  binance:
    api_key: "YOUR_API_KEY"
    secret_key: "YOUR_SECRET_KEY"
    base_url: "https://testnet.binancefuture.com"
    testnet: true
  
  fees:
    maker_fee: 0.0002  # 0.02%
    taker_fee: 0.0004  # 0.04%
```

---

## 日志系统

### 日志结构

```
logs/
├── trading.log                    # 主日志文件
└── session_20251206_143333/       # 会话日志目录
    ├── BTCUSDT_1m.log            # BTCUSDT 1分钟日志
    ├── BTCUSDT_3m.log            # BTCUSDT 3分钟日志
    ├── ETHUSDT_1m.log            # ETHUSDT 1分钟日志
    └── ETHUSDT_3m.log            # ETHUSDT 3分钟日志
```

### 日志内容

**K线接收**:
```json
{
  "level": "INFO",
  "msg": "Kline received",
  "open_time": "2025-12-06T14:00:00.000Z",
  "open": 90000,
  "close": 90500,
  "volume": 123.45
}
```

**信号生成**:
```json
{
  "level": "INFO",
  "msg": "Signal generated",
  "signal_type": "OPEN_SHORT",
  "price": 90500,
  "confidence": 1,
  "reason": "MACD死叉+EMA5/VWAP8死叉+EMA5/EMA15死叉"
}
```

**订单执行**:
```json
{
  "level": "INFO",
  "msg": "Order placed",
  "order_id": "ORDER_123",
  "side": "SELL",
  "quantity": 0.055,
  "status": "FILLED"
}
```

**持仓更新**:
```json
{
  "level": "INFO",
  "msg": "Position update",
  "symbol": "BTCUSDT",
  "side": "SHORT",
  "size": 0.055,
  "entry_price": 90500,
  "unrealized_pnl": -50.5
}
```

### 查看日志

**实时查看特定交易对**:
```bash
tail -f logs/session_*/BTCUSDT_1m.log
```

**查看所有信号**:
```bash
cat logs/session_*/BTCUSDT_1m.log | grep "Signal generated"
```

**查看所有订单**:
```bash
cat logs/session_*/BTCUSDT_1m.log | grep "Order placed"
```

**查看错误**:
```bash
cat logs/session_*/BTCUSDT_1m.log | grep "ERROR"
```

**使用 jq 格式化**:
```bash
tail -f logs/session_*/BTCUSDT_1m.log | jq .
```

---

## 模式切换

### Backtest 模式（回测）

**配置**:
```yaml
app:
  mode: "backtest"

execution:
  mode: "backtest"
  backtest:
    data_source: "database"
    database_path: "./data/backtestData/ETHUSDT_3m.db"
    start_date: "2024-01-01T00:00:00Z"
    end_date: "2024-12-31T23:59:59Z"
    initial_balance: 10000.0
```

**运行**:
```bash
go run cmd/bot/main.go
```

### Paper 模式（模拟盘）

**配置**:
```yaml
app:
  mode: "paper"

execution:
  mode: "paper"
```

**特点**:
- 使用真实市场数据
- 模拟订单执行
- 不提交真实订单
- 适合策略验证

### Live 模式（实盘）

**配置**:
```yaml
app:
  mode: "live"

execution:
  mode: "live"
  binance:
    api_key: "YOUR_API_KEY"
    secret_key: "YOUR_SECRET_KEY"
    base_url: "https://fapi.binance.com"  # 主网
    testnet: false
```

**运行**:
```bash
./scripts/start-live.sh
```

**⚠️ 注意事项**:
1. 确保 API Key 有正确权限
2. 先在测试网验证策略
3. 设置合理的风险限制
4. 监控账户余额

---

## 策略说明

### MACD + EMA + VWAP 策略

#### 技术指标

- **MACD(16, 26, 9)**: 趋势指标
- **EMA(5, 15)**: 短期和中期均线
- **VWAP(8)**: 成交量加权平均价

#### 开仓条件

**情景一：交叉信号**

**做空条件**:
1. MACD 死叉（DIF下穿DEA）
2. EMA5 下穿 VWAP8

**做多条件**:
1. MACD 金叉（DIF上穿DEA）
2. EMA5 上穿 VWAP8

**情景二：趋势信号**

**做空条件**:
- 连续4个周期下跌
- 跌幅超过0.55%

**做多条件**:
- 连续4个周期上涨
- 涨幅超过0.55%

#### 加仓条件

在满足开仓条件的同时，还需要：
- **做空**: EMA5 下穿 EMA15
- **做多**: EMA5 上穿 EMA15

**限制**: 每个持仓最多加仓1次

#### 平仓条件

1. **反向信号**: 持有多单时出现空单信号（或反之）
2. **固定止损**: 亏损达到入场价的0.6%
3. **跟踪止盈**: 四级阶梯止盈
   - Level 1: 盈利0.6%-1.0%，回撤0.5%平仓
   - Level 2: 盈利1.0%-1.8%，回撤0.55%平仓
   - Level 3: 盈利1.8%-4.8%，回撤0.68%平仓
   - Level 4: 盈利>4.8%，回撤0.8%平仓

#### 资金管理

- **开仓**: 使用20%资金，5倍杠杆
- **加仓**: 使用40%资金，5倍杠杆
- **保证金模式**: 逐仓（ISOLATED）

#### 示例

**开仓计算**:
```
账户余额: 5000 USDT
开仓比例: 20%
USDT金额: 5000 * 0.20 = 1000 USDT
杠杆: 5x
价格: 90000 USDT/BTC
数量: (1000 * 5) / 90000 = 0.0556 BTC
```

**加仓计算**:
```
账户余额: 5000 USDT
加仓比例: 40%
USDT金额: 5000 * 0.40 = 2000 USDT
杠杆: 5x
价格: 90000 USDT/BTC
数量: (2000 * 5) / 90000 = 0.1111 BTC
```

---

## 常见问题

### 1. API 签名错误

**错误信息**:
```
API error (status 400): {"code":-1022,"msg":"Signature for this request is not valid."}
```

**解决方案**:
1. 检查系统时间是否同步：`sudo ntpdate -u time.google.com`
2. 确认 API Key 和 Secret 正确
3. 禁用代理或配置正确的代理

### 2. 订单数量为零

**错误信息**:
```
risk check failed: invalid quantity: 0.000000
```

**原因**: 账户余额不足或计算错误

**检查**:
```bash
# 查看日志中的数量计算
cat logs/session_*/BTCUSDT_1m.log | grep "Calculated order quantity"
```

### 3. 无订单生成

**可能原因**:
1. 策略未预热完成
2. 市场条件不满足
3. 已达到最大持仓数
4. 已达到加仓次数限制

**检查**:
```bash
# 查看信号生成
cat logs/session_*/BTCUSDT_1m.log | grep "Signal generated"

# 查看是否有限制日志
cat logs/session_*/BTCUSDT_1m.log | grep "limit reached"
```

### 4. WebSocket 断连

**错误信息**:
```
WebSocket connection closed
```

**解决方案**:
- 程序会自动重连
- 检查网络连接
- 检查代理设置

### 5. 数据库锁定

**错误信息**:
```
database is locked
```

**解决方案**:
```bash
# 停止所有使用数据库的进程
pkill -f goQuant

# 清理锁文件
rm data/wsdata/*.db-shm
rm data/wsdata/*.db-wal
```

---

## 监控和维护

### 健康检查

**检查程序运行状态**:
```bash
ps aux | grep live-trading
```

**检查日志更新**:
```bash
ls -lh logs/session_*/
```

**检查数据库大小**:
```bash
du -sh data/wsdata/
```

### 日常维护

**清理旧日志** (保留最近7天):
```bash
find logs/session_* -mtime +7 -exec rm -rf {} \;
```

**备份数据库**:
```bash
cp -r data/wsdata data/backup_$(date +%Y%m%d)
```

**查看系统资源**:
```bash
top -p $(pgrep -f live-trading)
```

---

## 安全建议

### API Key 管理

1. ✅ 使用测试网 API Key 进行测试
2. ✅ 生产环境使用只读或受限 API Key
3. ✅ 不要在代码中硬编码 API Key
4. ✅ 定期轮换 API Key
5. ✅ 限制 API Key 的 IP 白名单

### 风险控制

1. ✅ 设置最大持仓数限制
2. ✅ 设置最大回撤限制
3. ✅ 设置单日最大亏损限制
4. ✅ 使用逐仓模式，避免全账户爆仓
5. ✅ 设置合理的杠杆倍数（建议≤5x）

### 资金管理

1. ✅ 不要投入超过承受能力的资金
2. ✅ 保持一定比例的现金储备
3. ✅ 分散投资，不要all-in单一策略
4. ✅ 定期提取盈利
5. ✅ 设置账户报警阈值

---

## 性能优化

### 减少 API 调用

```yaml
# 增加数据缓存时间
data:
  cache_ttl: 60s
```

### 优化日志级别

```yaml
observability:
  logging:
    level: "warn"  # 生产环境使用 warn 或 error
```

### 数据库优化

```bash
# 定期优化数据库
sqlite3 data/wsdata/BTCUSDT_1m.db "VACUUM;"
```

---

## 技术支持

### 文档

- 架构文档: `docs/02-ARCHITECTURE.md`
- API 参考: `docs/API_REFERENCE.md`
- 修改记录: `docs/CHANGELOG.md`

### 调试技巧

**开启调试日志**:
```yaml
observability:
  logging:
    level: "debug"
```

**查看详细错误**:
```bash
cat logs/trading.log | grep ERROR | jq .
```

**检查策略指标值**:
```bash
# 添加日志输出指标值
cat logs/session_*/BTCUSDT_1m.log | grep "macd_dif"
```

---

**最后更新**: 2025-12-06

