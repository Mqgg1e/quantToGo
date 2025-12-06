# 币安API实现和策略集成完成

## ✅ 已完成的模块

### 1. 币安期货API客户端 (`internal/execution/binance/`)

#### 文件结构：
- `models.go` - 币安API数据结构定义
- `client.go` - HTTP API客户端
- `executor.go` - 实盘执行器（实现core.Executor接口）
- `utils.go` - 工具函数

#### 核心功能：

**交易接口：**
- ✅ `CreateOrder()` - 创建订单（支持市价/限价/止损/止盈）
- ✅ `CancelOrder()` - 撤销订单
- ✅ `GetOrder()` - 查询订单
- ✅ `GetOpenOrders()` - 查询未成交订单

**账户接口：**
- ✅ `GetAccount()` - 查询账户信息
- ✅ `GetPositionRisk()` - 查询持仓信息
- ✅ `SetLeverage()` - 设置杠杆
- ✅ `SetMarginType()` - 设置保证金模式

**市场数据接口：**
- ✅ `GetOrderBook()` - 获取订单簿
- ✅ `GetMarkPrice()` - 获取标记价格
- ✅ `GetNthPriceFromOrderBook()` - 获取订单簿第N档价格

**辅助方法：**
- ✅ `PlaceMarketOrder()` - 快捷下市价单
- ✅ `PlaceLimitOrder()` - 快捷下限价单
- ✅ `ClosePosition()` - 快捷平仓
- ✅ `CalculateQuantity()` - 根据USDT金额计算数量

---

### 2. 仓位管理模块 (`internal/position/`)

#### 文件：`manager.go`

#### 核心功能：

**信号处理：**
- ✅ `ProcessSignal()` - 将策略信号转换为订单
- ✅ 反向信号自动平仓
- ✅ 20%/40%资金分配规则
- ✅ 5倍杠杆逐仓模式

**风险控制：**
- ✅ `CheckStopLoss()` - 检查固定止损（0.6%）
- ✅ `CheckTrailingStop()` - 检查跟踪止盈（三段式）
  - Level 1: 0.6%-1.0% 盈利，回撤0.5%止盈
  - Level 2: 1.0%-1.8% 盈利，回撤0.55%止盈
  - Level 3: 1.8%-4.8% 盈利，回撤0.68%止盈
  - Level 4: >4.8% 盈利，回撤0.8%止盈
- ✅ `CheckRisk()` - 订单风险检查

**持仓管理：**
- ✅ `UpdatePosition()` - 更新持仓状态
- ✅ `GetPosition()` - 获取持仓
- ✅ `CalculatePositionSize()` - 计算仓位大小

---

### 3. MACD+EMA策略 (`internal/strategy/`)

#### 文件：`macd_ema_strategy.go`

#### 策略规则（基于testStrategy.md拆分后版本）：

**技术指标：**
- MACD (16, 26, 9)
- EMA (5, 15)
- VWAP (8)

**情景一：组合交叉信号**
```
空单开仓：MACD死叉 + 最近3周期内EMA5/VWAP8死叉
空单加仓：MACD死叉 + EMA5/VWAP8死叉 + EMA5/EMA15死叉

多单开仓：MACD金叉 + 最近3周期内EMA5/VWAP8金叉
多单加仓：MACD金叉 + EMA5/VWAP8金叉 + EMA5/EMA15金叉
```

**情景二：连续趋势信号**
```
多单开仓：连续4周期上涨 && 涨幅 > 0.55%
空单开仓：连续4周期下跌 && 跌幅 > 0.55%
```

**特性：**
- ✅ 交叉检测（金叉/死叉）
- ✅ 趋势检测（连续涨跌）
- ✅ 交叉历史记录（最近3个周期）
- ✅ K线缓冲区（环形队列）
- ✅ 自动预热（需要45个K线）

---

## 📖 使用示例

### 示例1：实盘执行器使用

```go
package main

import (
    "context"
    "fmt"
    "goQuant/internal/execution/binance"
    "goQuant/internal/core"
)

func main() {
    // 1. 创建币安执行器
    executor := binance.NewLiveExecutor(
        "your_api_key",
        "your_secret_key",
        "https://fapi.binance.com", // 实盘
        // "https://testnet.binancefuture.com" // 测试网
    )
    
    ctx := context.Background()
    
    // 2. 设置杠杆和保证金模式
    executor.SetLeverage(ctx, "ETHUSDT", 5)
    executor.SetMarginMode(ctx, "ETHUSDT", core.MarginModeIsolated)
    
    // 3. 获取账户信息
    account, _ := executor.GetAccount(ctx)
    fmt.Printf("可用余额: %.2f USDT\n", account.AvailableBalance)
    
    // 4. 下市价单（使用500 USDT，5倍杠杆）
    quantity, _ := executor.CalculateQuantity(ctx, "ETHUSDT", 500, 5)
    order, err := executor.PlaceMarketOrder(ctx, "ETHUSDT", core.OrderSideBuy, quantity)
    if err != nil {
        panic(err)
    }
    fmt.Printf("订单已提交: %s\n", order.ID)
    
    // 5. 查询持仓
    position, _ := executor.GetPosition(ctx, "ETHUSDT")
    if position != nil {
        fmt.Printf("持仓: %.4f ETH, 入场价: %.2f, 未实现盈亏: %.2f%%\n",
            position.Size, position.EntryPrice, position.UnrealizedPnLPercent)
    }
    
    // 6. 平仓
    closeOrder, _ := executor.ClosePosition(ctx, "ETHUSDT")
    fmt.Printf("平仓订单: %s\n", closeOrder.ID)
}
```

### 示例2：策略+仓位管理集成

```go
package main

import (
    "context"
    "goQuant/internal/config"
    "goQuant/internal/strategy"
    "goQuant/internal/position"
    "goQuant/internal/execution/binance"
    "goQuant/internal/dataManager/v2"
)

func main() {
    // 1. 加载配置
    cfg, _ := config.Load("config/config.yaml")
    
    // 2. 创建执行器
    executor := binance.NewLiveExecutor(
        cfg.Execution.Binance.APIKey,
        cfg.Execution.Binance.SecretKey,
        cfg.Execution.Binance.BaseURL,
    )
    
    // 3. 创建仓位管理器
    posMgr := position.NewManager(&cfg.Position, executor)
    
    // 4. 创建策略
    strategy := strategy.NewMACDEMAStrategy("ETHUSDT", "3m")
    
    // 5. 创建数据处理器
    processor, _ := v2.NewEnhancedMultiKlineProcessor(cfg.Data.DatabaseDir, cfg.Data.ProxyURL)
    
    // 6. 订阅K线数据
    ctx := context.Background()
    processor.StartSubscription(ctx, "ETHUSDT", "3m")
    
    // 7. 策略适配器
    adapter := &StrategyAdapter{
        strategy: strategy,
        posMgr:   posMgr,
        executor: executor,
    }
    
    // 8. 注册适配器到数据流
    processor.Subscribe("ETHUSDT", "3m", adapter)
    
    // 等待运行...
    select {}
}

// StrategyAdapter 适配器：连接数据→策略→仓位→执行
type StrategyAdapter struct {
    strategy strategy.Strategy
    posMgr   *position.Manager
    executor core.Executor
}

func (a *StrategyAdapter) OnKline(kline *v2.KlineData) {
    // 1. 策略生成信号
    signal, _ := a.strategy.OnKline(kline)
    
    if signal.Type == core.SignalTypeNoAction {
        return
    }
    
    // 2. 仓位管理器生成订单
    orders, _ := a.posMgr.ProcessSignal(signal, kline.GetClosePrice())
    
    // 3. 执行订单
    for _, order := range orders {
        resultOrder, err := a.executor.PlaceOrder(context.Background(), order)
        if err != nil {
            fmt.Printf("❌ 下单失败: %v\n", err)
        } else {
            fmt.Printf("✅ 订单已提交: %s %s %.4f @ %.2f\n",
                resultOrder.Symbol, resultOrder.Side, resultOrder.Quantity, resultOrder.Price)
        }
    }
    
    // 4. 更新持仓状态
    positions, _ := a.executor.GetPositions(context.Background())
    for _, pos := range positions {
        a.posMgr.UpdatePosition(pos)
    }
}

func (a *StrategyAdapter) OnError(err error) {
    fmt.Printf("❌ 错误: %v\n", err)
}

func (a *StrategyAdapter) Name() string {
    return "StrategyAdapter"
}
```

### 示例3：订单簿第3档价格限价单

```go
// 策略规则：采用订单簿第3个价格限价单
ctx := context.Background()
executor := binance.NewLiveExecutor(apiKey, secretKey, baseURL)

// 获取订单簿第3档价格
// 买入时看卖单（asks），卖出时看买单（bids）
buyPrice, _ := executor.GetOrderBookPrice(ctx, "ETHUSDT", "buy", 3)
sellPrice, _ := executor.GetOrderBookPrice(ctx, "ETHUSDT", "sell", 3)

// 下限价单
quantity, _ := executor.CalculateQuantity(ctx, "ETHUSDT", 500, 5)
order, _ := executor.PlaceLimitOrder(ctx, "ETHUSDT", core.OrderSideBuy, buyPrice, quantity)

fmt.Printf("限价单已提交: 价格=%.2f, 数量=%.4f\n", buyPrice, quantity)
```

---

## 🔧 配置文件更新

需要在 `config/config.yaml` 中添加币安API配置：

```yaml
execution:
  mode: "live"  # backtest | paper | live
  exchange: "binance"
  
  binance:
    api_key: "${BINANCE_API_KEY}"       # 从环境变量读取
    secret_key: "${BINANCE_SECRET_KEY}"
    base_url: "https://fapi.binance.com"
    ws_base_url: "wss://fstream.binance.com"
    testnet: false
```

**环境变量设置：**
```bash
export BINANCE_API_KEY="your_api_key_here"
export BINANCE_SECRET_KEY="your_secret_key_here"
```

---

## 🚀 数据流架构

```
WebSocket K线数据
      ↓
DataManager (v2)
      ↓
StrategyAdapter
      ↓
Strategy (MACD+EMA)
      ↓
TradingSignal
      ↓
PositionManager
   ├─→ 风险检查
   ├─→ 仓位计算 (20%/40%)
   ├─→ 止损/止盈检查
   └─→ Order
      ↓
Executor (Binance)
   ├─→ 设置杠杆/保证金模式
   ├─→ 提交订单到币安
   ├─→ 查询订单状态
   └─→ 更新持仓
      ↓
反馈到PositionManager
```

---

## 📊 关键数据结构映射

### 币安API → Core类型

| 币安类型 | Core类型 | 说明 |
|---------|---------|------|
| `OrderResponse` | `core.Order` | 订单信息 |
| `PositionRisk` | `core.Position` | 持仓信息 |
| `AccountInfo` | `core.Account` | 账户信息 |
| `FuturesOrderSide` | `core.OrderSide` | 订单方向 |
| `FuturesOrderType` | `core.OrderType` | 订单类型 |
| `MarginType` | `core.MarginMode` | 保证金模式 |

### 信号类型对应的订单操作

| 信号类型 | 操作 | 资金比例 | 杠杆 |
|---------|------|---------|------|
| `SignalTypeOpenLong` | 开多单 | 20% | 5x |
| `SignalTypeOpenShort` | 开空单 | 20% | 5x |
| `SignalTypeAddLong` | 加多仓 | 40% | 5x |
| `SignalTypeAddShort` | 加空仓 | 40% | 5x |
| `SignalTypeCloseLong` | 平多仓 | 全部 | - |
| `SignalTypeCloseShort` | 平空仓 | 全部 | - |

---

## ⚠️ 重要提示

### 1. API密钥安全
- ❌ 不要将API密钥硬编码到代码中
- ✅ 使用环境变量或加密配置文件
- ✅ 限制API权限（只开启交易权限，不开启提现权限）

### 2. 测试建议
- ✅ 先在Binance Testnet测试
- ✅ 实盘前用小额资金测试
- ✅ 验证杠杆和保证金模式设置正确

### 3. 风险控制
- ✅ 策略已实现0.6%固定止损
- ✅ 策略已实现三段跟踪止盈
- ✅ 建议设置最大日亏损限制

### 4. 网络问题
- ✅ 如需代理，在配置中设置proxy_url
- ✅ API请求有超时设置（10秒）
- ✅ 签名使用本地时间戳，确保系统时间准确

---

## 🧪 下一步测试计划

1. **单元测试** - 测试策略信号生成
2. **集成测试** - 测试完整数据流
3. **Testnet测试** - 币安测试网验证
4. **小额实盘** - 真实环境验证
5. **全量运行** - 正式上线

---

**创建时间：** 2024-12-05  
**状态：** ✅ 币安API和策略实现完成，准备测试

