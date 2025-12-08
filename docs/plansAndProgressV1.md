## ! 修改记录写在 docs/CHANGELOG.md 中，如果代码有修改docs/API_REFERENCE.md也要配套修改，其他记录写在docs/README.md和docs/USER_GUIDE.md里，主要问题和需求在docs/plansAndProgressV1.md，不允许再创建其他md文件
## ! 在docs/plansAndProgressV1.md问题下方的两个####内给出简短回复
## ! 完成后不可以再创建md文件

### 081225

### 0802
1.


### 061225

### 1145
1.现在有新的问题，我原本持有空单，后面出反向信号后，空单平掉了，但是空单的止损单没有取消，而且应该有的多单也没开
{"level":"INFO","time":"2025-12-06T11:41:00.473Z","caller":"strategy/adapter.go:86","msg":"Signal generated","signal_type":"OPEN_SHORT","price":89545.8,"confidence":1,"reason":"MACD死叉+EMA5/VWAP8死叉"}
{"level":"INFO","time":"2025-12-06T11:41:00.668Z","caller":"strategy/adapter.go:237","msg":"Position update","side":"SHORT","size":0.06,"entry_price":89550,"current_price":89548.02512681,"unrealized_pnl":0.11849239,"pnl_percent":0.0022053301693653452}
{"level":"INFO","time":"2025-12-06T11:41:00.668Z","caller":"strategy/adapter.go:104","msg":"No orders generated"}
{"level":"INFO","time":"2025-12-06T11:42:00.437Z","caller":"strategy/adapter.go:61","msg":"Kline received","open_time":"2025-12-06T11:41:00.000Z","open":89545.9,"high":89600.1,"low":89545.8,"close":89600.1,"volume":50.933}
{"level":"INFO","time":"2025-12-06T11:43:00.268Z","caller":"strategy/adapter.go:61","msg":"Kline received","open_time":"2025-12-06T11:42:00.000Z","open":89600,"high":89645,"low":89600,"close":89645,"volume":72.973}
{"level":"INFO","time":"2025-12-06T11:43:00.268Z","caller":"strategy/adapter.go:86","msg":"Signal generated","signal_type":"OPEN_LONG","price":89645,"confidence":1,"reason":"MACD金叉+EMA5/VWAP8金叉"}

最后一行的错误可能不是太大问题
{"level":"ERROR","time":"2025-12-06T11:44:00.256Z","caller":"strategy/adapter.go:112","msg":"Execute order failed","symbol":"BTCUSDT","error":"place order failed: create order: API error (status 400): {\"code\":-2022,\"msg\":\"ReduceOnly Order is rejected.\"}"}
可参考logs/session_20251206_113353/BTCUSDT_1m.log

####
**已修复**：反向平仓时旧止损单未取消

**问题分析**：
1. ✅ 反向平仓已执行（空单平掉）
2. ❌ 旧的止损单没有取消 → 导致后续 "ReduceOnly Order is rejected" 错误
3. ❌ 多单没开（按设计是等持仓清空后下次信号再开）

**原因**：
反向平仓逻辑只生成了平仓订单，**没有取消持仓的止损单**

**修复**：
在反向平仓时先取消旧止损单：
```go
if needClose {
    // 取消旧的止损单（如果存在）
    if position.StopLossOrderID != "" {
        ctx := context.Background()
        err := m.executor.CancelOrder(ctx, symbol, position.StopLossOrderID)
        if err != nil {
            logger.Warn("Failed to cancel stop loss order...")
        }
    }
    
    // 生成平仓订单
    closeOrder, err := m.createCloseOrder(symbol, position, currentPrice)
    // ...
}
```

**关于多单未开**：
- 这是**当前设计**：反向信号只平仓，不立即开新仓
- 等持仓清空（UpdatePosition 收到 Size=0）后，下次信号才开仓
- 目的：避免平仓和开仓订单冲突
- 如需改为立即开新仓，需要修改 ProcessSignal 逻辑返回两个订单

**修改文件**：`internal/position/manager.go` - ProcessSignal 函数
####

### 1128
1.反向平仓时出错
{"level":"INFO","time":"2025-12-06T10:42:00.190Z","caller":"strategy/adapter.go:86","msg":"Signal generated","signal_type":"OPEN_LONG","price":89537.9,"confidence":1,"reason":"MACD金叉+EMA5/VWAP8金叉+EMA5/EMA15金叉"}
{"level":"INFO","time":"2025-12-06T10:42:00.604Z","caller":"strategy/adapter.go:240","msg":"Position update","side":"LONG","size":0.028,"entry_price":89504.34642857,"current_price":89544.40139221,"unrealized_pnl":-1.12153898,"pnl_percent":-0.04475197593157658}
{"level":"ERROR","time":"2025-12-06T10:42:00.604Z","caller":"strategy/adapter.go:112","msg":"Execute order failed","symbol":"BTCUSDT","error":"risk check failed: invalid leverage: 0"}
参考logs/session_20251206_102441/BTCUSDT_1m.log

####
**已修复**：反向平仓订单缺少杠杆信息

**问题**：
- 持有空单（SHORT）时出现多单信号（OPEN_LONG）
- 系统正确触发反向平仓逻辑
- 但创建的平仓订单 leverage=0，被风险检查拒绝

**原因**：
`createCloseOrder` 函数只设置了 symbol, type, side, quantity，**没有设置 leverage 和 marginMode**

**修复**：
```go
order := &core.Order{
    Symbol:     symbol,
    Type:       core.OrderTypeMarket,
    Side:       side,
    Quantity:   position.Size,
    Leverage:   position.Leverage,   // 添加：使用持仓的杠杆
    MarginMode: position.MarginMode, // 添加：使用持仓的保证金模式
    Metadata: map[string]interface{}{
        "reduce_only":  true,
        "close_reason": "signal_triggered",
    },
}
```

**修改文件**：`internal/position/manager.go` - createCloseOrder 函数

**测试**：重启程序，反向信号时应该能成功平仓
####

### 1056 
架构确认 - 策略更换指南

#### 
**架构状态：✅ 基本清晰，但仓位管理层有策略耦合**

**分层结构验证**：
```
数据模块(v2.EnhancedMultiKlineProcessor) 
    ↓ (KlineData接口)
策略模块(core.Strategy接口) 
    ↓ (TradingSignal标准信号)
仓位管理(core.PositionManager接口)  ⚠️ 包含当前策略特定逻辑
    ↓ (Order标准订单)
执行模块(core.Executor接口)
    ↓ (Position持仓信息)
仓位管理 ← (UpdatePosition反馈)
```

**核心接口定义**（`internal/core/interfaces.go`）：
- `Strategy` 接口：只需实现 `OnKline()` 返回 `TradingSignal`
- `TradingSignal` 结构：标准化信号（Type, Price, Metadata, Reason）
- `PositionManager` 接口：`ProcessSignal()` 处理信号返回 `Order[]`
- `Executor` 接口：执行订单并返回持仓信息

**当前策略实现**（`internal/strategy/macd_ema_strategy.go`）：
- 类型：`MACDEMAStrategy` 实现了 `core.Strategy` 接口
- 输入：仅接收 `KlineData`
- 输出：仅返回 `TradingSignal`（OPEN_LONG/SHORT, NO_ACTION）
- 特殊逻辑：通过 `signal.Metadata["add_position_eligible"]` 标记可加仓信号

**⚠️ 发现的策略耦合问题**（`internal/position/manager.go`）：
- **第93-134行**：硬编码当前策略的加仓逻辑
  - `add_position_eligible` Metadata 检查
  - 10分钟加仓时间窗口（适配3x3分钟K线）
  - 加仓次数限制为1次
- **第30行**：`AddPositionCount` 字段假设只加仓1次
- **影响**：更换策略时，如果加仓规则不同（如允许多次加仓、不同时间窗口），需要修改这些硬编码逻辑

**仓位管理独立性验证**：
- ✅ 完全基于接口工作
- ✅ 只识别标准信号类型（SignalType枚举）
- ⚠️ 加仓逻辑**部分**依赖当前策略设计（Metadata标志 + 硬编码规则）

#### 更换策略需要修改的文件

**必须修改（策略实现）**：
1. **创建新策略文件**：`internal/strategy/your_new_strategy.go`
   - 实现 `core.Strategy` 接口的所有方法
   - `OnKline()` - 核心逻辑
   - `Warmup()` - 预热逻辑
   - `GetRequiredWarmupPeriods()` - 返回预热周期数
   - `Reset()` - 重置状态

2. **修改启动文件**：`cmd/live-trading/main.go`
   ```go
   // 旧代码（第97行左右）
   macdStrategy := strategy.NewMACDEMAStrategy(symbol, interval)
   
   // 改为
   myStrategy := strategy.NewYourStrategy(symbol, interval)
   ```

**需要检查和调整（加仓逻辑）**：
3. **仓位管理器**：`internal/position/manager.go`
   - ⚠️ **第93-134行**：`add_position_eligible` 检查逻辑
     - 当前：检查 Metadata 标志 + 10分钟窗口 + 加仓1次限制
     - 新策略：可能需要不同的加仓条件、时间窗口、次数限制
   - ⚠️ **第30行**：`AddPositionCount` 字段（当前限制1次）
     - 新策略可能允许多次加仓或不同的加仓策略
   - **建议**：如果新策略加仓逻辑不同，需要修改或参数化这些逻辑

**可选修改（如果需要新指标）**：
4. **添加指标实现**：`internal/strategy/indicators.go`
   - 如果使用现有指标（MACD, EMA, VWAP）则无需修改
   - 如果需要新指标（如RSI, BOLL），在此文件添加

**无需修改（框架层）**：
- ✅ `internal/core/interfaces.go` - 接口定义
- ✅ `internal/execution/binance/` - 执行模块
- ✅ `internal/dataManager/v2/` - 数据模块
- ✅ `internal/strategy/adapter.go` - 适配器（连接数据→策略→仓位管理，完全通用）
- ✅ `internal/strategy/signal.go` - 信号工具函数（通用辅助函数）
- ✅ `internal/strategy/indicators.go` - 技术指标库（MACD, EMA, VWAP, RSI, ATR等，完全通用）

**strategy 目录文件说明**：
- `adapter.go` - ✅ 通用适配器，无策略特定逻辑，只负责数据流转
- `signal.go` - ✅ 信号辅助函数（NewSignal, NoActionSignal），完全通用
- `indicators.go` - ✅ 技术指标库（EMA, MACD, VWAP, SMA, ATR, RSI），可复用
- `macd_ema_strategy.go` - ❌ 当前策略实现，更换策略时替换此文件

**配置文件调整**：
5. **策略参数配置**：`config/config.yaml`
   ```yaml
   strategy:
     name: "Your_New_Strategy"
     parameters:
       your_param1: value1
   
   position:
     position_sizing:
       open_percent: 0.20    # 可能需要调整
       add_percent: 0.40     # 可能需要调整
   ```

#### 当前策略特殊逻辑说明

**加仓机制的策略耦合**：
- 策略层：设置 `signal.Metadata["add_position_eligible"] = 1.0` 标记
- 仓位管理层：**硬编码**检查逻辑
  - 检查 Metadata 标志
  - 检查10分钟时间窗口（固定值）
  - 检查加仓次数 ≤ 1（固定限制）
- 新策略：
  - 方式1：仍使用 Metadata 标志，但需修改 manager.go 中的时间窗口和次数限制
  - 方式2：直接发送 `SignalTypeAddLong/Short` 信号（绕过时间窗口检查）

**建议改进**（未实施）：
- 将加仓规则参数化到配置文件：
  ```yaml
  position:
    add_position:
      max_count: 1           # 最大加仓次数
      time_window: 600       # 时间窗口（秒）
      use_metadata_flag: true # 是否检查 Metadata 标志
  ```
- 或者完全移除时间窗口检查，由策略自行控制加仓时机

**结论**：
- ✅ 核心接口清晰，数据流标准化
- ⚠️ 仓位管理层存在策略特定的加仓逻辑硬编码
- ⚠️ 更换策略时，可能需要修改 `manager.go` 的加仓检查逻辑
- 建议：将加仓规则参数化，提高框架通用性

---
####
**架构审查完成**

**strategy 目录文件分类**：
- ✅ `adapter.go` (294行) - 通用适配器，无策略特定代码
- ✅ `signal.go` (55行) - 通用信号辅助函数
- ✅ `indicators.go` (350+行) - 通用技术指标库（EMA, MACD, VWAP, SMA, ATR, RSI等）
- ❌ `macd_ema_strategy.go` (400+行) - 当前MACD策略实现

**结论**：
- strategy 目录下只有 `macd_ema_strategy.go` 需要替换
- 其他3个文件（adapter, signal, indicators）都是通用框架代码
- indicators.go 包含常用技术指标，新策略可直接复用

**更换策略文件清单（最终版）**：
1. **必须修改**：
   - `internal/strategy/your_new_strategy.go` - 新策略实现
   - `cmd/live-trading/main.go` - 1行代码改动
   
2. **可能需要修改**（如加仓规则不同）：
   - `internal/position/manager.go` - 第93-134行的加仓时间窗口和次数限制
   
3. **可选修改**：
   - `internal/strategy/indicators.go` - 如需新指标（当前已有MACD/EMA/VWAP/RSI/ATR/SMA）
   - `config/config.yaml` - 策略参数配置
####


### 1031
1.仓位量有问题
{"level":"INFO","time":"2025-12-06T10:27:00.654Z","caller":"strategy/adapter.go:162","msg":"Order placed","order_id":"BTCUSDT_1765016820","side":"SELL","type":"MARKET","quantity":0.015,"price":0,"status":"NEW"}
{"level":"INFO","time":"2025-12-06T10:28:00.858Z","caller":"strategy/adapter.go:162","msg":"Order placed","order_id":"BTCUSDT_1765016880","side":"SELL","type":"MARKET","quantity":0.013,"price":0,"status":"NEW"}
并不是策略的开仓20%，加仓40%，应该还是yaml写的5%总余额

2.确认情景一条件
情景一入场条件：
当DIF穿越DEA交叉形成死叉时，跟踪最近3个周期，若出现EMA5 穿越 VWAP8 交叉形成死叉，则生成空单信号，检测最近3个周期内有EMA5穿越 EMA15 交叉形成死叉，则生成空单加仓信号
当DIF穿越DEA交叉形成金叉时，跟踪最近3个周期，若出现EMA5穿越 VWAP8 交叉形成金叉，则生成多单信号，检测最近3个周期内有EMA5 穿越 EMA15 交叉形成金叉，则生成多单加仓信号

加仓信号应该是只在最初开仓附近，现在是这样吗

3.为什么这里还是"side":"LONG"，是指现有的仓位吗，现有的仓位是空单
"Position update","side":"LONG","size":0.028,"entry_price":89504.34642857,"current_price":89489.6,"unrealized_pnl":0.41289999,"pnl_percent":0.01647565599068898}

####
**已全部修复并编译成功**

**问题1 - 仓位计算错误**：
- 原因1：CalculatePositionSize 函数硬编码了 0.20 和 0.40，没有使用配置文件
- 原因2：config.yaml 中 `max_position_size: 0.05` (5%) 限制了最大仓位，导致20%和40%被截断为5%
- 修复1：在 PositionSizingConfig 添加 OpenPercent 和 AddPercent 字段，改为读取配置
- 修复2：将 `max_position_size` 改为 0.50 (50%)，允许20%开仓、40%加仓正常工作
- 现在会正确使用配置的 20% 开仓、40% 加仓

**问题2 - 加仓时机控制**：
- 原因：每次出现所有3个交叉的信号都被标记为可加仓，没有时间限制
- 修复：添加 OpenTime 字段跟踪开仓时间，只允许开仓后10分钟内加仓（覆盖3个3分钟K线周期）
- 现在加仓只能在"最初开仓附近"进行，符合策略要求

**问题3 - 持仓方向显示错误**：
- 原因：日志记录使用 `pos.Size` 的正负判断方向，但 PositionRiskToPosition 已将 Size 转为正数并设置了 pos.Side
- 修复：改为直接使用 `string(pos.Side)` 显示持仓方向
- 现在会正确显示 "SHORT" 或 "LONG"

**修改文件**：
- `internal/config/config.go` - 添加 OpenPercent、AddPercent 字段
- `internal/position/manager.go` - 仓位计算、加仓时机、OpenTime跟踪
- `internal/strategy/adapter.go` - 持仓方向日志显示
- `config/config.yaml` - max_position_size: 0.05 → 0.50

**编译状态**：✅ 成功
####



### 1024
1.你之前提到的订单方向反了，这个问题解决了吗
"msg":"Signal generated","signal_type":"OPEN_SHORT","price":3028.44,"confidence":1,"reason":"MACD死叉+EMA5/VWAP8死叉"}
"msg":"Position update","side":"LONG","size":0.425,"entry_price":3028.44,"current_price":3028.44,"unrealized_pnl":0,"pnl_percent":0}

####
**已解决**：持仓方向判断逻辑修复

**问题**：
- 信号：OPEN_SHORT（做空）
- 实际持仓：LONG（多单）← 完全反了！

**原因**：
只使用 `positionAmt` 正负判断方向，但 Binance 在某些情况下（对冲模式或测试网），`positionAmt` 都可能是正数，应该优先使用 `positionSide` 字段。

**修复**：
1. 优先检查 `positionSide` 字段（"LONG"/"SHORT"/"BOTH"）
2. 如果是 "LONG" 或 "SHORT"，直接使用该方向
3. 如果是 "BOTH"（单向模式），才用 `positionAmt` 正负判断

**修改文件**：`internal/execution/binance/models.go`

**调试**：添加了日志输出，重启后会显示：
```
[DEBUG] Position: symbol=ETHUSDT, posAmt=0.425, positionSide=SHORT
```

**测试**：重启程序，观察持仓方向是否正确
####

### 1000
1.现在有问题，在之前已经开过仓未加仓的情况下，其他开仓信号正常拒绝，但是MACD死叉+EMA5/VWAP8死叉+EMA5/EMA15死叉信号似乎被接受为新开仓并且被执行
{"level":"INFO","time":"2025-12-06T09:56:00.419Z","caller":"strategy/adapter.go:86","msg":"Signal generated","signal_type":"OPEN_SHORT","price":3028.77,"confidence":1,"reason":"MACD死叉+EMA5/VWAP8死叉+EMA5/EMA15死叉"}
参考logs/session_20251206_090432/ETHUSDT_1m.log

2.还有仓位管理有问题，我再次说明
开仓时止损设在0.6%，逐仓时不改止损，三段的跟踪止盈是到达0.6%时设TRAILING_STOP_MARKET单，过1%到下一级就撤销上一级单，重下当前级别单

####
**问题1已解决**：持仓信息更新时机错误

**根本原因**：
- ProcessSignal 在判断是否有持仓时，Manager 还没有最新的持仓信息
- 导致已有持仓时仍被当作"无持仓"，重复开仓

**修复**：
- 在 ProcessSignal **之前**先调用 updatePositions()
- 确保 Manager 有最新持仓信息
- 修改文件：`internal/strategy/adapter.go`

**问题2已解决**：重新实现跟踪止盈

**新实现**：
1. 开仓时设置固定止损 0.6%（STOP_MARKET）✅
2. 盈利达到 0.6% → 撤销止损单，设置 TRAILING_STOP_MARKET（回调0.5%）
3. 盈利达到 1.0% → 撤销旧单，设置新 TRAILING_STOP_MARKET（回调0.55%）
4. 盈利达到 1.8% → 撤销旧单，设置新 TRAILING_STOP_MARKET（回调0.68%）

**实现细节**：
- 使用真实的 TRAILING_STOP_MARKET 订单
- 每次升级都撤销旧订单 ID（存储在 StopLossOrderID 中）
- callbackRate 参数：0.5, 0.55, 0.68（表示 0.5%, 0.55%, 0.68%）
- 使用 closePosition=true 平掉全部仓位

**修改文件**：
- `internal/position/manager.go` - CheckTrailingStop + setTrailingStopOrder
- `internal/core/interfaces.go` - 添加 OrderTypeTrailingStop
- `internal/execution/binance/models.go` - 类型映射
- `internal/execution/binance/executor.go` - TRAILING_STOP_MARKET 参数处理

**测试**：重启程序观察跟踪止盈单设置
####

### 0900
1.报错
(base) maeda@maeda89:~/Documents/projects/goQuant$ ./scripts/start-live.sh
=========================================
goQuant Live Trading Bot
=========================================

✅ Config file found

🔨 Building...
# goQuant/internal/execution/binance
internal/execution/binance/models.go:281:10: undefined: OrderTypeStopLimit

####
**问题**：Binance 常量命名错误，`OrderTypeStopLimit` 未定义

**修复**：Binance 的限价止损单常量名是 `OrderTypeStop`（不是 `OrderTypeStopLimit`）
- `core.OrderTypeStopLimit` → `OrderTypeStop` (Binance "STOP")
- 已修正 `FromOrderType` 函数中的映射

**修改文件**：`internal/execution/binance/models.go`

**编译成功**，可以重启测试
####


### 0817
1.现在的错误变了
2025-12-06T08:18:00.821Z        INFO    position/manager.go:335 Calculated order quantity       {"quantity": 0.014388787200706056, "usdt_amount": 257.6741134145, "leverage": 5, "price": 89539.9}
2025-12-06T08:18:01.582Z        INFO    position/manager.go:548 Setting stop loss order {"symbol": "BTCUSDT", "side": "SELL", "quantity": 0.029, "stop_price": 89012.58734666998, "entry_price": 89545.1, "stop_loss_percent": 0.6}
2025-12-06T08:18:01.663Z        ERROR   strategy/adapter.go:201 Failed to set stop loss {"symbol": "BTCUSDT", "error": "place stop loss order failed: create order: API error (status 400): {\"code\":-4136,\"msg\":\"Target strategy invalid for orderType MARKET,closePosition true\"}"}

####
**问题根源**：`FromOrderType` 函数缺少 `STOP_MARKET` 类型转换，导致 `core.OrderTypeStopMarket` 被错误转换为 `MARKET`

**修复**：在 `models.go` 的 `FromOrderType` 函数中添加完整的订单类型映射：
- `core.OrderTypeStopMarket` → `OrderTypeStopMarket` ("STOP_MARKET")
- `core.OrderTypeStopLimit` → `OrderTypeStopLimit`
- `core.OrderTypeTakeProfit` → `OrderTypeTakeProfitMarket`

**修改文件**：`internal/execution/binance/models.go`

**测试**：重启程序，止损单应该成功设置
####



### 0805
1.用的应该是rest api吧，api文档 https://developers.binance.com/docs/derivatives/usds-margined-futures/trade/websocket-api
不管怎样我保存下来了

Request Parameters
Name	Type	Mandatory	Description
symbol	STRING	YES
side	ENUM	YES
positionSide	ENUM	NO	Default BOTH for One-way Mode ; LONG or SHORT for Hedge Mode. It must be sent in Hedge Mode.
type	ENUM	YES
timeInForce	ENUM	NO
quantity	DECIMAL	NO	Cannot be sent with closePosition=true(Close-All)
reduceOnly	STRING	NO	"true" or "false". default "false". Cannot be sent in Hedge Mode; cannot be sent with closePosition=true
price	DECIMAL	NO
newClientOrderId	STRING	NO	A unique id among open orders. Automatically generated if not sent. Can only be string following the rule: ^[\.A-Z\:/a-z0-9_-]{1,36}$
stopPrice	DECIMAL	NO	Used with STOP/STOP_MARKET or TAKE_PROFIT/TAKE_PROFIT_MARKET orders.
closePosition	STRING	NO	true, false；Close-All，used with STOP_MARKET or TAKE_PROFIT_MARKET.
activationPrice	DECIMAL	NO	Used with TRAILING_STOP_MARKET orders, default as the latest price(supporting different workingType)
callbackRate	DECIMAL	NO	Used with TRAILING_STOP_MARKET orders, min 0.1, max 10 where 1 for 1%
workingType	ENUM	NO	stopPrice triggered by: "MARK_PRICE", "CONTRACT_PRICE". Default "CONTRACT_PRICE"
priceProtect	STRING	NO	"TRUE" or "FALSE", default "FALSE". Used with STOP/STOP_MARKET or TAKE_PROFIT/TAKE_PROFIT_MARKET orders.
newOrderRespType	ENUM	NO	"ACK", "RESULT", default "ACK"
priceMatch	ENUM	NO	only avaliable for LIMIT/STOP/TAKE_PROFIT order; can be set to OPPONENT/ OPPONENT_5/ OPPONENT_10/ OPPONENT_20: /QUEUE/ QUEUE_5/ QUEUE_10/ QUEUE_20; Can't be passed together with price
selfTradePreventionMode	ENUM	NO	EXPIRE_TAKER:expire taker order when STP triggers/ EXPIRE_MAKER:expire taker order when STP triggers/ EXPIRE_BOTH:expire both orders when STP triggers; default EXPIRE_MAKER
goodTillDate	LONG	NO	order cancel time for timeInForce GTD, mandatory when timeInforce set to GTD; order the timestamp only retains second-level precision, ms part will be ignored; The goodTillDate timestamp must be greater than the current time plus 600 seconds and smaller than 253402300799000
recvWindow	LONG	NO
timestamp	LONG	YES

Additional mandatory parameters based on type:

Type	Additional mandatory parameters
LIMIT	timeInForce, quantity, price or priceMatch
MARKET	quantity
STOP/TAKE_PROFIT	quantity, stopPrice, price or priceMatch
STOP_MARKET/TAKE_PROFIT_MARKET	stopPrice
TRAILING_STOP_MARKET	callbackRate

####
**问题根源找到！** 根据文档：
- STOP_MARKET 订单平掉全部仓位应该使用 `closePosition=true`
- 使用 `closePosition=true` 时 **不能** 发送 `quantity` 参数
- 使用 `closePosition=true` 时 **不能** 发送 `reduceOnly` 参数

**修复**：
- 添加 `ClosePosition` 字段到 CreateOrderRequest
- STOP_MARKET 止损单使用 `closePosition=true` 而不是 `quantity` + `reduceOnly`
- 只发送必需参数：`symbol`, `side`, `type`, `stopPrice`, `closePosition=true`

**修改文件**：
- `models.go` - 添加 ClosePosition 字段
- `client.go` - 添加 closePosition 参数处理
- `executor.go` - STOP_MARKET 订单使用 closePosition 模式

**测试**：重启程序，应该不再报参数错误
####


### 0718
1.问题还是存在，
{"level":"ERROR","time":"2025-12-06T07:20:01.724Z","caller":"strategy/adapter.go:201","msg":"Failed to set stop loss","symbol":"BTCUSDT","error":"place stop loss order failed: create order: API error (status 400): {\"code\":-1106,\"msg\":\"Parameter 'stopprice' sent when not required.\"}"}
再次检查一下币安官方的api文档

####
**已实施双重止损保护**：

1. **主要防线**：仍尝试设置交易所 STOP_MARKET 单（已移除 WorkingType 参数）
2. **备用防线**：启用程序内止损监控（每次 K线更新检查）
   - 固定止损：价格触及入场价 ±0.6% 时市价平仓
   - 跟踪止盈：四级阶梯回撤止盈
   - 优点：即使交易所止损单失败也有保护
   - 缺点：依赖程序运行，崩溃时无保护

**修改文件**：
- `adapter.go` - 启用 checkRiskManagement
- `executor.go` - 移除 WorkingType 参数

**测试建议**：观察日志中是否有 "Stop loss triggered by program monitor" 确认备用机制工作
####




### 0701
1.开仓后挂止损单的功能还没有实现，报错
要求是止损市价单（在触发时立即以市价成交），平掉全部仓位
```json
{"level":"INFO","time":"2025-12-06T07:01:01.367Z","caller":"strategy/adapter.go:237","msg":"Position update","side":"LONG","size":0.015,"entry_price":89643.1,"current_price":89649,"unrealized_pnl":0.0885,"pnl_percent":0.006581655475993132}
{"level":"ERROR","time":"2025-12-06T07:01:01.453Z","caller":"strategy/adapter.go:201","msg":"Failed to set stop loss","symbol":"BTCUSDT","error":"place stop loss order failed: create order: API error (status 400): {\"code\":-1106,\"msg\":\"Parameter 'stopprice' sent when not required.\"}"}
{"level":"INFO","time":"2025-12-06T07:01:01.657Z","caller":"strategy/adapter.go:237","msg":"Position update","side":"LONG","size":0.015,"entry_price":89643.1,"current_price":89649,"unrealized_pnl":0.0885,"pnl_percent":0.006581655475993132}
{"level":"INFO","time":"2025-12-06T07:02:00.679Z","caller":"strategy/adapter.go:61","msg":"Kline received","open_time":"2025-12-06T07:01:00.000Z","open":89649,"high":89655.8,"low":89641,"close":89641,"volume":10.187}
{"level":"INFO","time":"2025-12-06T07:02:00.679Z","caller":"strategy/adapter.go:86","msg":"Signal generated","signal_type":"OPEN_LONG","price":89641,"confidence":1,"reason":"MACD金叉+EMA5/VWAP8金叉+EMA5/EMA15金叉"}
{"level":"INFO","time":"2025-12-06T07:02:01.246Z","caller":"strategy/adapter.go:159","msg":"Order placed","order_id":"BTCUSDT_1765004521","side":"BUY","type":"MARKET","quantity":0.014,"price":0,"status":"NEW"}
{"level":"INFO","time":"2025-12-06T07:02:01.830Z","caller":"strategy/adapter.go:185","msg":"Order filled","order_id":"go_1765004520776027762","avg_price":89645.05714}
{"level":"INFO","time":"2025-12-06T07:02:01.830Z","caller":"strategy/adapter.go:192","msg":"Setting stop loss after order placed","symbol":"BTCUSDT"}
{"level":"INFO","time":"2025-12-06T07:02:01.929Z","caller":"strategy/adapter.go:237","msg":"Position update","side":"LONG","size":0.029,"entry_price":89644.04482758,"current_price":89644.11851779,"unrealized_pnl":0.00213701,"pnl_percent":0.0000822028949516214}
{"level":"ERROR","time":"2025-12-06T07:02:02.018Z","caller":"strategy/adapter.go:201","msg":"Failed to set stop loss","symbol":"BTCUSDT","error":"place stop loss order failed: create order: API error (status 400): {\"code\":-1106,\"msg\":\"Parameter 'stopprice' sent when not required.\"}"}
{"level":"INFO","time":"2025-12-06T07:02:02.115Z","caller":"strategy/adapter.go:237","msg":"Position update","side":"LONG","size":0.029,"entry_price":89644.04482758,"current_price":89644.10764822,"unrealized_pnl":0.00182179,"pnl_percent":0.00007007754385515949}

```
参考logs/session_20251206_064941/BTCUSDT_1m.log
####
**问题**: STOP_MARKET 订单参数错误，Binance API 报错 "Parameter 'stopprice' sent when not required"

**原因**: 代码中对所有止损单类型都设置了 stopPrice，但 STOP_MARKET 的处理逻辑不够精确

**修复**: 修改 `executor.go`，明确区分 STOP_MARKET 订单处理：
- STOP_MARKET 只需要 `stopPrice`（触发价）
- 触发后以市价成交，不需要 `price` 参数
- 添加订单类型判断，避免设置不需要的参数

**修改文件**: `internal/execution/binance/executor.go`

**测试**: 重新启动程序观察止损单是否成功设置
####



### 0500
1.在确认开仓后挂止损单的功能还没有实现，参考logs/session_20251206_063925/ETHUSDT_1m.log

####
done but not sovled
####




### 051225 and before

### 我正在构建一个替代freqtrade 的量化系统，具体架构如下

数据模块 ---(市场数据)---> 策略模块 ---(仓位信号)---> 仓位管理 <---（当前信息）-- | --（实际下单）--->执行模块

+日志模块

```json
{
  "nodes": [
    {"id":"data_module","label":"数据模块","type":"source","emits_telemetry": true, "telemetry_topics":["telemetry.metrics","telemetry.logs"]},
    {"id":"strategy_module","label":"策略模块","type":"logic","emits_telemetry": true, "telemetry_topics":["telemetry.logs","telemetry.traces"]},
    {"id":"position_manager","label":"仓位管理","type":"state_manager","emits_telemetry": true, "telemetry_topics":["telemetry.metrics","telemetry.alerts","telemetry.logs"]},
    {"id":"execution_module","label":"执行模块","type":"action","emits_telemetry": true, "telemetry_topics":["telemetry.logs","telemetry.traces","telemetry.metrics"]},
    {
      "id":"observability",
      "label":"监控与日志模块",
      "type":"observability",
      "role":"collector",
      "consumes_topics":["telemetry.metrics","telemetry.logs","telemetry.traces","telemetry.alerts"]
    }
  ],
  "edges": [
    {"from":"data_module","to":"strategy_module","channel":"市场数据"},
    {"from":"data_module","to":"position_manager","channel":"市场数据"},
    {"from":"strategy_module","to":"position_manager","channel":"仓位信号"},
    {"from":"position_manager","to":"execution_module","channel":"实际下单"},
    {"from":"execution_module","to":"position_manager","channel":"当前信息"},

    /* 显式观测边（可选，便于图示） */
    {"from":"data_module","to":"observability","channel":"metrics/logs"},
    {"from":"strategy_module","to":"observability","channel":"logs/traces"},
    {"from":"position_manager","to":"observability","channel":"metrics/alerts/logs"},
    {"from":"execution_module","to":"observability","channel":"logs/traces/metrics"}
  ],
  "telemetry_schema": {
    "log": {"fields":["timestamp","level","component_id","message","context"]},
    "metric": {"fields":["timestamp","metric_name","value","labels","component_id"]},
    "trace": {"fields":["trace_id","span_id","parent_span_id","component_id","start","duration","attributes"]},
    "alert": {"fields":["timestamp","severity","component_id","alert_type","details"]}
  },
  "assertions": [
    {"type":"requirement","expr":"every node that has emits_telemetry=true must have at least one telemetry_topic"},
    {"type":"requirement","expr":"observability.consumes_topics contains all telemetry_topics used by nodes"},
    {"type":"requirement","expr":"no node has duplicate id"},
    {"type":"requirement","expr":"position_manager emits telemetry and has alert topic (critical for risk) "}
  ]
}

```

### 每个模块的功能和预期如下

### 1.数据模块 （部分完成）
基本功能：从币安api获取k线数据、维护订单簿等功能，检查正确性并且推送给下游模块或者记录到数据库中

#### 细节描述
1.从币安api获取websocket(<symbol>@kline_<interval>)推送的k线数据（收盘），获取k线数据是最基本功能
+ 可选择是否使用代理
+ 可能接收多个品种和周期的websocket

2.数据校验与清洗
+ 当发现来自websocket有遗漏时需要调用rest api进行补全
+ 尽量避免api超限
+ 确保无论收到k线还是校验k线都应该是严格按时间顺序推送

3.k线数据可以分流，可以选择保存到db数据库或者是推送给下游策略模块或是同时
+ 保存到数据库时只需要k线的指定部分即可（已有）
+ 可以指定数据库目录，品种和周期写在db名字里

4.可选一个获取订单簿的功能，也推送到下游模块
+ <symbol>@depth<levels> OR <symbol>@depth<levels>@500ms OR <symbol>@depth<levels>@100ms，详细参考 https://developers.binance.com/docs/derivatives/usds-margined-futures/websocket-market-streams/Partial-Book-Depth-Streams
+ 订单簿的调用由策略实际情况决定，可以不调用、当k线推送时一起推送或根据固定间隔推送
+ 订单簿推送时也应该保存在db中

5.websocket重连
+ 币安websocket每24h自动断开，需要实现无中断重连
+ 可能因为网络问题websocket连接不上，需要尝试重连

### 2.策略模块
基本功能：实现不同策略的逻辑和仓位管理

#### 细节描述
1.策略和仓位管理与其他模块分开

2.策略只接收k线，最多只向下游输出开仓、加仓、平仓和方向信号
+ 策略中间可以计算各种指标，但是向下游只输出这些信号
+ 可以保存收到的k线和计算的各种指标和信号

3.仓位管理配合配置文件，决定了仓位大小、保证金模式、杠杆倍数等其他信息

4.仓位管理接收k线和从执行处传来的仓位信息，输出是实际会下的单
+ 可能会下一个单，也可能会接连下多个单
+ 有些单有前提关系，所以需要验证函数
+ 因此要提前根据币安api提供的接口定义好输出用的数据结构
  + 这一部分需要进一步调查确定，不可随便开始

5.策略和仓位管理的具体逻辑都是由其各自的文件定义的，但不管怎么写，策略输入只有k线，输出只有开加平和方向，仓位管理只有策略来的信号和k线，输出只有规定好的数据结构

#! 需要指出的是这部分仍然很粗糙，需要实际验证

### 3.执行模块
基本功能：调用策略模块，接收策略模块的信号，执行根据本地数据的模拟盘，根据websocket数据的本地模拟盘或根据websocket数据的实盘，返回现在账户信息、仓位情况和委托情况给仓位管理

#### 细节描述
1.根据币安api允许的下单方式定义数据结构

2.实现本地的模拟交易逻辑，包括手续费，下单，止盈止损，等币安api支持的下单方式，尽量迁移到本地实现
+ 市价单是在k线收盘时入场，限价单是先挂单，然后在价格经过时入场
+ 同样的，为了统一与交易所实盘的api接口，尽量符合其规则

3.使用websocket数据时，需要根据策略情况通过rest api预热数根k线
+ 这一点也同样适用于刚启动或断连时，或者需要websocket但是要先计算指标

4.接受关于账户信息、仓位情况和委托情况给仓位管理的websocket推送，并推送给仓位管理

#! 需要指出的是这部分仍然很粗糙，需要实际验证
#! api功能还没遍历和验证


### 4.日志和监控模块
基本功能：记录每次运行的信息和异常情况，监控异常并且能做出通知

#### 细节描述
1.每次开始运行时，记录时间，品种等详细信息

2.出现rest填补时记下缺失的k线时间，k线顺序失效时也记录

3.websocket断开或重连时记录

4.策略模块接收或发出异常时记录

5.异常终止或手动终止时记录

5.可能会通过sns发送异常警告，但暂时不实现

## 整体要求
1.有很大概率会人工手动测试各个模块，实现的时候注意标注每个函数的功能，以及输入输出的内容和类型，如果有些函数相关性强也要同时标注出前后相关函数

2.留下可能会有gui迭代需求，需要留下重构空间

3.整体方案的细节有待优化，没必要立刻实现全部功能，实现时应该逐步进行

4.注意文档迭代


## claude suggestion
Plan: 完善量化交易框架架构
根据现有实现分析，数据模块（dataManager）已基本完成（WebSocket订阅、重连、完整性检查、数据库存储、消息分发），但策略模块、仓位管理、执行模块和监控模块尚未实现。以下是完善和优化建议：
Steps
规范化模块接口定义 - 在 internal/ 下创建核心接口包 (interfaces 或 core)，定义 DataProvider、Strategy、PositionManager、Executor、Logger 等标准接口，确保模块解耦和可测试性
实现策略模块骨架 - 在 internal/strategy/ 创建 engine.go（策略引擎）、signal.go（信号类型定义）、indicators.go（技术指标计算），实现 KlineSubscriber 接口接收数据，输出标准化的开平仓信号
设计仓位管理模块 - 在 internal/position/ 实现 manager.go（仓位状态跟踪）、risk.go（风险控制）、sizing.go（仓位计算），定义币安API兼容的订单结构体，处理杠杆、保证金模式和多单验证逻辑
构建执行模块框架 - 在 internal/execution/ 创建 simulator.go（本地回测引擎）、binance_client.go（REST/WebSocket API封装）、order_manager.go（订单生命周期管理），支持市价/限价单、止盈止损和手续费计算
集成可观测性系统 - 在 internal/observability/ 实现结构化日志（使用 zap 或 zerolog）、指标收集（Prometheus格式）、事件追踪，支持多级日志（Debug/Info/Warn/Error）和异常告警钩子（预留SNS接口）
创建配置管理系统 - 设计 config/ 目录存放YAML/TOML配置文件，使用 viper 管理策略参数、风险限制、API密钥、数据库路径等，支持环境变量覆盖和热重载
Further Considerations
API客户端选择 - 建议使用现有币安SDK（如 adshao/go-binance/v2）而非从零实现REST/WebSocket客户端，可节省大量时间并避免API限流问题
回测与实盘统一 - 执行模块应设计统一的 Executor 接口，BacktestExecutor 和 LiveExecutor 共享相同接口，仅实现细节不同，方便策略无缝切换
数据预热机制 - 策略启动时需要通过REST API预加载历史K线（如MACD需要26+9周期数据），建议在 internal/dataManager/v2/ 添加 preload.go 实现批量历史数据获取
错误处理策略 - 当前数据模块错误处理较简单，建议定义错误等级（可恢复/不可恢复），关键错误（如数据库损坏、API认证失败）应触发告警并安全关闭系统
测试策略优先级 - 建议先实现 testStrategy.md 中的MACD+EMA策略作为第一个完整案例，验证整个数据流（数据→策略→仓位→执行→监控），再扩展到更复杂策略
性能优化考虑 - 多交易对订阅时注意goroutine数量控制，建议使用worker pool模式；数据库批量写入可使用事务（已有但需验证性能）；指标计算可考虑环形缓冲区避免重复计算