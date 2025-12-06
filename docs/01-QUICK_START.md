# 快速上手指南

## 🚀 5分钟启动实盘交易

### 前置要求

- ✅ Go 1.21+ 已安装
- ✅ 币安期货账户
- ✅ API密钥已创建（需要交易权限）

---

## 步骤1: 设置环境变量

```bash
export BINANCE_API_KEY="your_api_key_here"
export BINANCE_SECRET_KEY="your_secret_key_here"
```

**重要提示：**
- 不要将密钥提交到Git
- 可以添加到 `~/.bashrc` 或 `~/.zshrc` 永久保存
- 测试环境使用测试网密钥

---

## 步骤2: 配置文件

```bash
cd /home/maeda/Documents/projects/goQuant
cp config/config.example.yaml config/config.yaml
```

**修改 `config/config.yaml`：**

```yaml
app:
  mode: "live"  # ← 确认是live模式

data:
  subscriptions:
    - symbol: "ETHUSDT"  # ← 修改为你要交易的币种
      interval: "3m"

execution:
  binance:
    base_url: "https://fapi.binance.com"  # ← 实盘
    # base_url: "https://testnet.binancefuture.com"  # ← 测试网
    testnet: false  # ← 实盘设为false
```

---

## 步骤3: 启动交易机器人

### 方式1: 使用启动脚本（推荐）

```bash
./scripts/start-live.sh
```

### 方式2: 手动编译运行

```bash
go build -o bin/live-trading cmd/live-trading/main.go
./bin/live-trading
```

---

## 预期输出

```
✅ Config loaded: mode=live, leverage=5x
Setting up ETHUSDT: leverage=5x, margin=ISOLATED
✅ Position manager created
✅ Data processor created
Warming up strategy for ETHUSDT 3m...
✅ Strategy started: ETHUSDT 3m - MACD(16,26,9) + EMA(5,15) + VWAP(8)
📊 Account: Total=10000.00 USDT, Available=10000.00 USDT, UnrealizedPnL=0.00 USDT
📈 No open positions

🚀 Trading bot started successfully!
Strategy: MACD(16,26,9) + EMA(5,15) + VWAP(8)
Risk Control: StopLoss=0.6%, TrailingStop=3-levels
Position Sizing: Open=20%, Add=40%

Press Ctrl+C to stop...

[StrategyAdapter] 📊 Signal: [15:03:00] ETHUSDT OPEN_LONG @ 2250.50 - MACD金叉+EMA5/VWAP8金叉
[StrategyAdapter] ✅ Order placed: ETHUSDT BUY 0.4445 MARKET @ 0.00
[StrategyAdapter] ✅ Order filled: avg price 2250.80
[StrategyAdapter] 📈 Position updated: ETHUSDT 0.4445 @ 2250.80, PnL: 0.00%
```

---

## 🧪 测试网测试（强烈推荐先测试）

### 1. 获取测试网API密钥

访问: https://testnet.binancefuture.com/

### 2. 修改配置

```yaml
execution:
  binance:
    base_url: "https://testnet.binancefuture.com"
    testnet: true
```

### 3. 设置测试网密钥

```bash
export BINANCE_API_KEY="testnet_api_key"
export BINANCE_SECRET_KEY="testnet_secret_key"
```

### 4. 运行

```bash
./scripts/start-live.sh
```

---

## 📊 监控运行状态

### 查看实时日志

```bash
# 如果使用脚本启动，日志会输出到终端
tail -f logs/trading.log  # 如果配置了日志文件
```

### 检查持仓

策略会自动输出持仓更新：

```
[StrategyAdapter] 📈 Position updated: ETHUSDT 0.4445 @ 2250.80, PnL: 1.25%
```

### 检查账户余额

定期输出账户信息（可在代码中配置）

---

## 🛑 停止交易

按 `Ctrl+C` 优雅关闭：

```
^C
🛑 Shutting down gracefully...
✅ Bot stopped
```

**重要：** 停止机器人不会自动平仓，需要手动平仓或等待策略信号

---

## ⚠️ 风险提示

### 首次运行检查清单

- [ ] ✅ 已在测试网测试
- [ ] ✅ 确认杠杆倍数（默认5倍）
- [ ] ✅ 确认保证金模式（默认逐仓）
- [ ] ✅ 确认仓位大小（开仓20%，加仓40%）
- [ ] ✅ 确认止损设置（0.6%）
- [ ] ✅ 小额资金测试（建议<1000 USDT）
- [ ] ✅ 监控运行至少24小时
- [ ] ✅ 备用资金准备（应对极端行情）

### 风险控制

策略内置风险控制：

- **固定止损**: 0.6%
- **跟踪止盈**: 三段式（0.5%-0.8%）
- **最大持仓**: 3个（可配置）
- **单次开仓**: 20%资金
- **加仓**: 40%资金

### 紧急情况处理

**如需立即平仓：**

1. 停止机器人（Ctrl+C）
2. 登录币安网页/APP手动平仓
3. 或使用币安API平仓工具

---

## 🔧 常见问题

### Q1: 提示API密钥错误

**A:** 检查环境变量是否正确设置：
```bash
echo $BINANCE_API_KEY
echo $BINANCE_SECRET_KEY
```

### Q2: 订单被拒绝

**A:** 可能原因：
- 杠杆未设置
- 保证金模式未设置
- 余额不足
- API权限不足

### Q3: 无法连接WebSocket

**A:** 检查：
- 网络连接
- 代理设置（config.yaml中的proxy_url）
- 防火墙设置

### Q4: 策略不生成信号

**A:** 检查：
- 策略是否已预热（需要45个K线）
- 技术指标是否满足条件
- 查看日志中的指标值

---

## 📈 性能优化建议

### 1. 使用更快的网络

- 建议使用VPS（延迟<50ms）
- 配置代理加速

### 2. 数据库优化

- 定期清理旧数据
- 使用SSD存储

### 3. 日志管理

```yaml
observability:
  logging:
    level: "info"  # 生产环境使用info，调试使用debug
    output_path: "logs/trading.log"
```

---

## 📞 获取帮助

- 查看文档: `COMPLETE_IMPLEMENTATION.md`
- 查看API文档: `BINANCE_IMPLEMENTATION.md`
- 检查代码: `internal/strategy/macd_ema_strategy.go`

---

## ✅ 成功启动标志

如果看到以下输出，说明启动成功：

```
✅ Config loaded
✅ Position manager created
✅ Data processor created
✅ Strategy started: ETHUSDT 3m
🚀 Trading bot started successfully!
```

**现在你的量化交易机器人已经开始运行了！** 🎉

---

**最后提醒：** 
- 量化交易有风险，投资需谨慎
- 建议先小额测试
- 持续监控系统运行
- 定期检查策略表现

祝交易顺利！🚀

