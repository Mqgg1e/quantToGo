# 日志系统使用指南

## 📋 概述

系统已集成 **zap** 结构化日志库，提供高性能、易于分析的JSON格式日志。

## 🗂️ 日志文件位置

- **主日志文件**: `logs/trading.log`
- **备份文件**: `logs/trading.log.YYYY-MM-DD.gz` (压缩的旧日志)

**配置**:
- 单个文件最大: 100MB
- 保留备份数: 10个
- 保留时间: 30天
- 自动压缩: 是

## 📊 日志级别

- **DEBUG**: 调试信息（技术指标值等）
- **INFO**: 一般信息（K线、信号、订单）
- **WARN**: 警告（风险控制、数据补全）
- **ERROR**: 错误（API失败、连接断开）
- **FATAL**: 致命错误（程序退出）

## 🎯 记录的关键信息

### 1. K线数据 (INFO级别)

每根K线收盘时记录：
- 品种和周期
- 开盘时间
- OHLC价格
- 成交量

**示例**:
```json
{
  "time": "2025-12-05T12:00:00Z",
  "level": "INFO",
  "msg": "Kline received",
  "symbol": "BTCUSDT",
  "interval": "3m",
  "open_time": "2025-12-05T11:57:00Z",
  "open": 91000.00,
  "high": 91200.00,
  "low": 90950.00,
  "close": 91150.00,
  "volume": 234.56
}
```

### 2. 交易信号 (INFO级别)

策略生成信号时记录：
- 信号类型（OPEN_LONG/SHORT, ADD, CLOSE）
- 当前价格
- 置信度
- 生成原因

**示例**:
```json
{
  "time": "2025-12-05T12:00:00Z",
  "level": "INFO",
  "msg": "Signal generated",
  "symbol": "BTCUSDT",
  "signal_type": "OPEN_LONG",
  "price": 91150.00,
  "confidence": 0.85,
  "reason": "MACD金叉+EMA5/VWAP8金叉"
}
```

### 3. 订单记录 (INFO级别)

订单提交和成交时记录：
- 订单ID
- 方向（BUY/SELL）
- 类型（MARKET/LIMIT）
- 数量和价格
- 状态

**示例**:
```json
{
  "time": "2025-12-05T12:00:01Z",
  "level": "INFO",
  "msg": "Order event",
  "order_id": "BTCUSDT_1733400001",
  "symbol": "BTCUSDT",
  "side": "BUY",
  "type": "MARKET",
  "quantity": 0.1,
  "price": 91150.00,
  "status": "FILLED"
}
```

### 4. 仓位更新 (INFO级别)

仓位变化时记录：
- 持仓方向（LONG/SHORT）
- 持仓数量
- 入场价格
- 未实现盈亏

**示例**:
```json
{
  "time": "2025-12-05T12:03:00Z",
  "level": "INFO",
  "msg": "Position update",
  "symbol": "BTCUSDT",
  "side": "LONG",
  "size": 0.1,
  "entry_price": 91150.00,
  "current_price": 91300.00,
  "unrealized_pnl": 15.00,
  "pnl_percent": 0.16
}
```

### 5. 策略预热 (INFO级别)

策略启动时记录预热进度：

**示例**:
```json
{
  "time": "2025-12-05T11:30:00Z",
  "level": "INFO",
  "msg": "Strategy warming up",
  "symbol": "BTCUSDT",
  "current_klines": 30,
  "required_klines": 45,
  "progress_percent": 66.67
}
```

### 6. 技术指标 (DEBUG级别)

每根K线后记录指标值（需要debug级别）：

**示例**:
```json
{
  "time": "2025-12-05T12:00:00Z",
  "level": "DEBUG",
  "msg": "Indicator values",
  "symbol": "BTCUSDT",
  "close": 91150.00,
  "dif": 123.45,
  "dea": 115.20,
  "ema5": 91100.00,
  "ema15": 91050.00,
  "vwap8": 91080.00
}
```

### 7. 风险控制 (WARN级别)

触发止损/止盈时记录：

**示例**:
```json
{
  "time": "2025-12-05T12:05:00Z",
  "level": "WARN",
  "msg": "Risk control triggered",
  "symbol": "BTCUSDT",
  "event_type": "stop_loss",
  "trigger_value": 90500.00,
  "threshold": 90550.00,
  "action": "close_position"
}
```

### 8. 数据补全 (WARN级别)

WebSocket缺失数据时记录：

**示例**:
```json
{
  "time": "2025-12-05T12:00:00Z",
  "level": "WARN",
  "msg": "Data completion",
  "symbol": "BTCUSDT",
  "interval": "3m",
  "missing_count": 2,
  "start_time": "2025-12-05T11:45:00Z",
  "end_time": "2025-12-05T11:51:00Z"
}
```

### 9. 账户快照 (INFO级别)

定期记录账户状态：

**示例**:
```json
{
  "time": "2025-12-05T12:00:00Z",
  "level": "INFO",
  "msg": "Account snapshot",
  "total_balance": 5000.00,
  "available_balance": 4800.00,
  "margin": 200.00,
  "unrealized_pnl": 15.00,
  "margin_ratio": 4.00
}
```

## 🛠️ 查看日志

### 方式1: 使用日志查看工具（推荐）

```bash
./scripts/view-logs.sh
```

交互式菜单可以选择：
1. 实时查看所有日志
2. 只看K线更新
3. 只看交易信号
4. 只看订单记录
5. 只看持仓更新
6. 只看错误和警告
7. 查看最后50条
8. 关键词搜索

### 方式2: 直接查看文件

```bash
# 实时查看（格式化输出）
tail -f logs/trading.log | jq

# 查看最后100行
tail -100 logs/trading.log | jq

# 只看INFO级别
grep '"level":"INFO"' logs/trading.log | jq

# 只看某个品种
grep 'BTCUSDT' logs/trading.log | jq
```

### 方式3: 查找特定事件

```bash
# 查找所有交易信号
grep "Signal generated" logs/trading.log | jq

# 查找所有订单
grep "Order event" logs/trading.log | jq

# 查找错误
grep '"level":"ERROR"' logs/trading.log | jq

# 按时间范围查找
grep "2025-12-05T12:" logs/trading.log | jq
```

## 📈 日志分析示例

### 统计今天的信号数量

```bash
grep "$(date +%Y-%m-%d)" logs/trading.log | grep "Signal generated" | wc -l
```

### 查看所有开仓信号

```bash
grep "Signal generated" logs/trading.log | grep -E "OPEN_LONG|OPEN_SHORT" | jq -r '. | "\(.time) \(.symbol) \(.signal_type) @ \(.price)"'
```

### 查看盈利情况

```bash
grep "Position update" logs/trading.log | jq -r '. | "\(.time) \(.symbol) PnL: \(.unrealized_pnl) (\(.pnl_percent)%)"'
```

### 检查错误和警告

```bash
grep -E '"level":"(WARN|ERROR)"' logs/trading.log | jq -r '. | "\(.time) [\(.level)] \(.msg)"'
```

## ⚙️ 修改日志级别

编辑 `cmd/live-trading/main.go`:

```go
logConfig := logger.DefaultConfig()
logConfig.Level = "debug"  // 改为 "debug" 查看指标值
```

**级别说明**:
- `debug`: 所有日志（包括指标值）- **会产生大量日志**
- `info`: 正常运行日志（推荐）
- `warn`: 只记录警告和错误
- `error`: 只记录错误

## 📊 日志结构

所有日志都是JSON格式，包含标准字段：

```json
{
  "time": "ISO8601时间戳",
  "level": "日志级别",
  "caller": "调用位置",
  "msg": "消息内容",
  "component": "组件名称",
  // ...其他业务字段
}
```

## 🔍 性能说明

- **JSON格式**: 便于程序化分析
- **文件轮转**: 自动管理磁盘空间
- **异步写入**: 不影响交易性能
- **压缩存储**: 节省磁盘空间

## 💡 最佳实践

### 1. 日常监控
```bash
# 实时查看信号和订单
tail -f logs/trading.log | grep -E "Signal|Order" | jq
```

### 2. 问题排查
```bash
# 查看最近的错误
tail -1000 logs/trading.log | grep ERROR | jq
```

### 3. 策略分析
```bash
# 导出今天的所有信号到CSV
grep "Signal generated" logs/trading.log | \
  grep "$(date +%Y-%m-%d)" | \
  jq -r '[.time, .symbol, .signal_type, .price, .reason] | @csv'
```

### 4. 性能分析
```bash
# 统计每小时的K线数量
grep "Kline received" logs/trading.log | \
  jq -r '.time[:13]' | \
  sort | uniq -c
```

## 🚨 注意事项

1. **DEBUG级别会产生大量日志** - 仅在需要时启用
2. **日志文件会自动轮转** - 旧文件会被压缩
3. **保留30天的历史** - 超过会自动删除
4. **定期检查磁盘空间** - 确保有足够空间

## 🔧 故障排查

### 日志文件不存在
```bash
mkdir -p logs
./scripts/start-live.sh
```

### 日志没有更新
- 检查程序是否运行: `ps aux | grep live-trading`
- 检查文件权限: `ls -l logs/`

### 日志太多
```bash
# 手动清理旧日志
rm logs/trading.log.*.gz

# 或减少保留天数（修改main.go中的MaxAge）
```

## 📚 相关文档

- [策略文档](03-STRATEGY.md)
- [架构文档](02-ARCHITECTURE.md)
- [快速开始](01-QUICK_START.md)

