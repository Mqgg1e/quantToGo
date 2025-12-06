# 交易模式配置指南

## 当前状态

你现在运行的是：**模拟盘期货合约（币安测试网）**

K线数据保存位置：
- BTCUSDT: `data/wsdata/BTCUSDT_3m.db`
- ETHUSDT: `data/wsdata/ETHUSDT_3m.db`

---

## 🔧 如何切换交易模式

### 方式1: 模拟盘期货（当前模式）✅

**适用场景**: 测试策略，无风险

**配置文件**: `config/config.yaml`

```yaml
app:
  mode: "live"

execution:
  mode: "live"
  exchange: "binance"
  binance:
    api_key: "测试网API密钥"
    secret_key: "测试网SECRET"
    base_url: "https://testnet.binancefuture.com"  # ← 测试网
    ws_base_url: "wss://stream.binancefuture.com"
    testnet: true  # ← 测试网标志
```

**获取测试网密钥**: https://testnet.binancefuture.com/

---

### 方式2: 实盘期货合约 ⚠️

**适用场景**: 真实资金交易

**配置修改**:

```yaml
app:
  mode: "live"

execution:
  mode: "live"
  exchange: "binance"
  binance:
    api_key: "你的实盘API密钥"      # ← 从币安主站获取
    secret_key: "你的实盘SECRET"
    base_url: "https://fapi.binance.com"  # ← 实盘期货
    ws_base_url: "wss://fstream.binance.com"
    testnet: false  # ← 实盘标志
```

**获取实盘密钥**: https://www.binance.com/zh-CN/my/settings/api-management

**⚠️ 重要**: 
- 确保API权限只开启交易，禁止提现
- 建议设置IP白名单
- 小额资金测试

---

### 方式3: 模拟现货交易 🚧

**当前不支持** - 代码实现的是期货合约API

如需支持现货，需要修改代码：

#### 需要修改的文件：

1. **`internal/execution/binance/executor.go`**
   - 将期货API改为现货API
   - `/fapi/v2/account` → `/api/v3/account`
   - `/fapi/v1/order` → `/api/v3/order`

2. **`config/config.yaml`**
   ```yaml
   binance:
     base_url: "https://api.binance.com"      # 现货实盘
     # 或
     base_url: "https://testnet.binance.vision"  # 现货测试网
   ```

3. **订单参数差异**：
   - 现货没有杠杆、保证金模式
   - 订单结构不同
   - 需要移除`positionSide`等期货专用参数

---

### 方式4: 纯回测模式（不下单）

**适用场景**: 历史数据测试策略

**配置修改**:

```yaml
app:
  mode: "backtest"  # ← 回测模式

execution:
  mode: "backtest"  # ← 回测模式
  backtest:
    data_source: "database"
    database_path: "data/backtestData/ETHUSDT_3m_2021-01-01_2025-11-30_futures_kline.db"
    start_date: "2024-01-01T00:00:00Z"
    end_date: "2024-12-31T23:59:59Z"
    initial_balance: 10000
```

**注意**: 回测模式下不会连接API，只使用本地数据

---

## 📋 快速切换配置

### 切换到测试网期货（推荐新手）

```bash
# 编辑配置文件
vim config/config.yaml

# 修改以下部分
base_url: "https://testnet.binancefuture.com"
testnet: true

# 重新启动
./scripts/start-live.sh
```

### 切换到实盘期货（有经验用户）

```bash
# 编辑配置文件
vim config/config.yaml

# 修改以下部分
base_url: "https://fapi.binance.com"
testnet: false
api_key: "你的实盘密钥"
secret_key: "你的实盘SECRET"

# 重新启动
./scripts/start-live.sh
```

---

## 🗂️ 数据存储位置

### K线数据（实时采集）
```
data/wsdata/
├── BTCUSDT_3m.db    # BTC 3分钟K线
├── ETHUSDT_3m.db    # ETH 3分钟K线
└── ...
```

### 回测数据（历史数据）
```
data/backtestData/
└── ETHUSDT_3m_2021-01-01_2025-11-30_futures_kline.db
```

### 查看数据库内容

```bash
# 安装sqlite3
sudo apt-get install sqlite3

# 查看K线数量
sqlite3 data/wsdata/BTCUSDT_3m.db "SELECT COUNT(*) FROM klines;"

# 查看最新10条K线
sqlite3 data/wsdata/BTCUSDT_3m.db "SELECT * FROM klines ORDER BY start_time DESC LIMIT 10;"

# 查看数据范围
sqlite3 data/wsdata/BTCUSDT_3m.db "SELECT 
  datetime(MIN(start_time)/1000, 'unixepoch') as earliest,
  datetime(MAX(start_time)/1000, 'unixepoch') as latest,
  COUNT(*) as total
FROM klines;"
```

---

## 🔍 常见问题

### Q1: 签名错误怎么办？

```
⚠️ Failed to get account info: Signature for this request is not valid
```

**原因**: 
- 系统时间不准确
- API密钥错误
- URL和密钥类型不匹配（测试网密钥用了实盘URL）

**解决**:
```bash
# 1. 同步系统时间
sudo ntpdate -u time.nist.gov

# 2. 检查配置文件中URL和testnet标志是否匹配
# 测试网：base_url包含"testnet" 且 testnet: true
# 实盘：base_url是"fapi.binance.com" 且 testnet: false
```

### Q2: 如何知道订单是否真的执行了？

**测试网**: 
- 登录 https://testnet.binancefuture.com/
- 查看持仓和订单历史

**实盘**:
- 登录 https://www.binance.com/
- 期货账户 → 订单历史

### Q3: 数据库会一直增长吗？

是的。建议定期清理旧数据：

```bash
# 删除30天前的数据
sqlite3 data/wsdata/BTCUSDT_3m.db "DELETE FROM klines WHERE start_time < strftime('%s', 'now', '-30 days') * 1000;"

# 清理数据库
sqlite3 data/wsdata/BTCUSDT_3m.db "VACUUM;"
```

---

## ✅ 推荐配置方案

### 方案A: 新手学习（当前配置）✅

```yaml
# 测试网期货 - 无风险
base_url: "https://testnet.binancefuture.com"
testnet: true
```

**优点**: 免费测试资金，无风险
**缺点**: 数据延迟可能较大

### 方案B: 实盘小额测试

```yaml
# 实盘期货 - 小额资金
base_url: "https://fapi.binance.com"
testnet: false
initial_balance: 100  # 只投入100 USDT测试
```

**优点**: 真实市场数据，验证策略
**缺点**: 有亏损风险

### 方案C: 纯回测验证

```yaml
# 回测模式 - 历史数据
app:
  mode: "backtest"
execution:
  mode: "backtest"
```

**优点**: 快速验证策略，无需等待
**缺点**: 历史表现不代表未来

---

## 🎯 总结

| 模式 | 配置位置 | base_url | testnet | 真实交易 |
|------|---------|----------|---------|---------|
| **测试网期货** ✅ | config.yaml | testnet.binancefuture.com | true | 否 |
| **实盘期货** ⚠️ | config.yaml | fapi.binance.com | false | 是 |
| **现货** 🚧 | 需要修改代码 | api.binance.com | - | 是 |
| **回测** 📊 | config.yaml | mode: backtest | - | 否 |

**你当前运行的是**: 测试网期货（安全，推荐） ✅

如需切换，编辑 `config/config.yaml` 然后重启程序即可。

