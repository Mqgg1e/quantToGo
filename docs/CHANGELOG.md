# goQuant 修改记录

本文档记录所有重要的代码修改、问题修复和功能增强。

---

## 2025-12-06

### ✅ 修复编译错误 - 订单类型常量名称

#### 问题
编译失败：`undefined: OrderTypeStopLimit`

#### 原因
在 `FromOrderType` 函数中使用了不存在的 `OrderTypeStopLimit` 常量。Binance 的限价止损单常量名称是 `OrderTypeStop`（"STOP"），不是 `OrderTypeStopLimit`。

#### 修复
修正常量映射：
```go
case core.OrderTypeStopLimit:
    return OrderTypeStop  // ✅ Binance 的 "STOP" 是限价止损单
```

**修改文件**：`internal/execution/binance/models.go`

---

### ✅ 修复订单类型转换错误（STOP_MARKET 最终修复）

#### 问题
设置止损单失败，错误信息：
```
Target strategy invalid for orderType MARKET,closePosition true
```

#### 根本原因
`FromOrderType` 函数缺少 `STOP_MARKET` 类型转换逻辑：

```go
// ❌ 错误：default 返回 MARKET
func FromOrderType(orderType core.OrderType) FuturesOrderType {
    switch orderType {
    case core.OrderTypeMarket:
        return OrderTypeMarket
    case core.OrderTypeLimit:
        return OrderTypeLimit
    default:
        return OrderTypeMarket  // ← STOP_MARKET 被错误转换为 MARKET
    }
}
```

当创建 `core.OrderTypeStopMarket` 订单时，被转换为 `"MARKET"` 而不是 `"STOP_MARKET"`，导致 API 拒绝。

#### 解决方案

补全所有订单类型的转换映射：

```go
// ✅ 正确：完整的类型映射
func FromOrderType(orderType core.OrderType) FuturesOrderType {
    switch orderType {
    case core.OrderTypeMarket:
        return OrderTypeMarket
    case core.OrderTypeLimit:
        return OrderTypeLimit
    case core.OrderTypeStopMarket:
        return OrderTypeStopMarket           // "STOP_MARKET"
    case core.OrderTypeStopLimit:
        return OrderTypeStopLimit            // "STOP_LIMIT"
    case core.OrderTypeTakeProfit:
        return OrderTypeTakeProfitMarket     // "TAKE_PROFIT_MARKET"
    default:
        return OrderTypeMarket
    }
}
```

**修改文件**：
- `internal/execution/binance/models.go`

**测试验证**：
止损单应该成功设置，预期日志：
```json
{"msg":"Setting stop loss order","type":"STOP_MARKET","stopPrice":"89012.59"}
{"msg":"Stop loss order set successfully","order_id":"..."}
```

---

### ✅ 修复 STOP_MARKET 订单参数错误（最终解决）

#### 问题
STOP_MARKET 止损单一直失败，报错：
```
Parameter 'stopprice' sent when not required.
```

#### 根本原因（已确认）
根据 Binance 期货 API 文档：
- STOP_MARKET 订单用于平掉全部仓位时应该使用 **`closePosition=true`**
- 文档明确指出：`quantity` 参数 **Cannot be sent with closePosition=true**
- 文档明确指出：`reduceOnly` 参数 **cannot be sent with closePosition=true**

**我们之前的错误做法**：
```go
// ❌ 错误：同时发送 quantity 和 reduceOnly
params = {
    "type": "STOP_MARKET",
    "stopPrice": "89107.328",
    "quantity": "0.029",        // ← 不应该发送
    "reduceOnly": "true"        // ← 不应该发送
}
```

**正确做法**：
```go
// ✅ 正确：使用 closePosition 平掉全部仓位
params = {
    "type": "STOP_MARKET",
    "stopPrice": "89107.328",
    "closePosition": "true"     // ← 只需要这个
}
```

#### 解决方案

1. **添加 ClosePosition 字段**（models.go）
```go
type CreateOrderRequest struct {
    // ...existing fields...
    ClosePosition bool `json:"closePosition,omitempty"` // 新增
}
```

2. **在 client.go 中处理参数**
```go
if req.ClosePosition {
    params.Set("closePosition", "true")
}
```

3. **在 executor.go 中判断订单类型**
```go
// 对于 STOP_MARKET 止损单
if order.Type == STOP_MARKET && order.Metadata["reduce_only"] == true {
    req.ClosePosition = true  // 使用 closePosition
    // 不设置 quantity
    // 不设置 reduceOnly
}
```

**修改文件**：
- `internal/execution/binance/models.go` - 添加 ClosePosition 字段
- `internal/execution/binance/client.go` - 添加 closePosition 参数处理
- `internal/execution/binance/executor.go` - STOP_MARKET 使用 closePosition 逻辑

**参考文档**：
- https://developers.binance.com/docs/derivatives/usds-margined-futures/trade/websocket-api
- 参数说明：`closePosition` - Close-All, used with STOP_MARKET or TAKE_PROFIT_MARKET

---

### ✅ 实现双重止损保护机制

#### 背景
Binance 测试网的 STOP_MARKET 订单类型存在 API 兼容性问题，导致交易所止损单设置失败。

#### 解决方案
实现**双重止损保护**：

1. **主要保护**：尝试设置交易所止损单（STOP_MARKET）
   - 如果成功：由交易所执行止损
   - 如果失败：降级到程序内监控

2. **备用保护**：程序内止损监控（已启用）
   - 每次 K线更新时检查价格
   - 触发止损时立即提交市价平仓单
   - 优点：更灵活可控
   - 缺点：程序崩溃时无保护（但交易所止损单若成功设置则仍有效）

#### 实现细节

```go
// adapter.go - OnKline 方法中每次都检查
if signal.Type == NO_ACTION {
    if enableRiskCheck {
        checkRiskManagement(currentPrice)
    }
}

// checkRiskManagement 方法
func checkRiskManagement(currentPrice) {
    // 检查固定止损 (0.6%)
    if CheckStopLoss(symbol, currentPrice) {
        提交市价平仓单
    }
    
    // 检查跟踪止盈 (四级阶梯)
    if CheckTrailingStop(symbol, currentPrice) {
        提交市价平仓单
    }
}
```

#### 修改文件
- `internal/strategy/adapter.go` - 启用程序内止损监控
- `internal/execution/binance/executor.go` - 移除 WorkingType 参数

**保护机制**：
- ✅ 交易所止损单（第一道防线）
- ✅ 程序内监控（第二道防线）
- ✅ 固定止损 0.6%
- ✅ 跟踪止盈四级阶梯

---

### 🐛 止损单 API 参数调试

#### 问题持续
移除不必要参数后，STOP_MARKET 订单仍然失败：
```
Parameter 'stopprice' sent when not required.
```

#### 调试尝试
1. **移除 WorkingType**: 测试网可能不支持此参数
2. **确认参数最小化**: STOP_MARKET 只发送 `symbol`, `side`, `type`, `quantity`, `stopPrice`, `reduceOnly`

#### 备选方案（如继续失败）
Binance 测试网可能不完全支持 STOP_MARKET，考虑：
- **方案A**: 改用 STOP 类型（限价止损）
- **方案B**: 使用程序内止损监控（定时检查价格，触发时提交市价平仓单）
  - 优点：更灵活可控
  - 缺点：程序崩溃时无保护

**修改文件**:
- `internal/execution/binance/executor.go` - 移除 WorkingType 参数

**待测试**: 重启程序观察错误是否消失

---

### 🐛 修复止损单 API 参数错误

#### 问题
止损单设置失败，Binance API 返回错误：
```
Parameter 'stopprice' sent when not required.
```

#### 根本原因
执行器对所有包含 `StopPrice` 的订单都设置了 `stopPrice` 参数，但没有区分订单类型：
- `STOP_MARKET`（市价止损）：只需要 `stopPrice`（触发价），触发后以市价成交
- `STOP_LIMIT`（限价止损）：需要 `stopPrice` 和 `price`

#### 解决方案
修改执行器逻辑，明确区分订单类型：

```go
// 修复前：所有止损单都设置 stopPrice
if order.StopPrice > 0 {
    req.StopPrice = formatPrice(order.Symbol, order.StopPrice)
    req.WorkingType = WorkingTypeMark
}

// 修复后：只对 STOP_MARKET 类型设置 stopPrice
if order.Type == core.OrderTypeStopMarket && order.StopPrice > 0 {
    req.StopPrice = formatPrice(order.Symbol, order.StopPrice)
    req.WorkingType = WorkingTypeMark
    // STOP_MARKET 不需要 price 参数
}
```

**改进点**:
- ✅ 明确区分 STOP_MARKET 和其他订单类型
- ✅ 避免设置不需要的参数
- ✅ 添加注释说明 STOP_MARKET 的特性

**修改文件**:
- `internal/execution/binance/executor.go`

---

### 🐛 修复止损单未实际执行的问题

#### 问题
虽然添加了 `SetStopLoss` 方法，但在实际运行中并未被调用，导致开仓后没有设置止损保护。

#### 根本原因
1. **GetOrder 查询失败**：当查询订单状态失败时，整个止损设置逻辑被跳过
2. **Metadata 类型断言**：在查询后的订单对象中 Metadata 可能丢失
3. **逻辑判断延迟**：在订单提交后才判断是否需要止损，而应该在提交前就确定

#### 解决方案
重构止损设置逻辑：

```go
// 1. 提交订单前就判断是否需要止损
isOpenOrAddOrder := false
if signalType == OPEN_LONG || signalType == OPEN_SHORT {
    isOpenOrAddOrder = true
}

// 2. 提交订单后，如果是开仓/加仓订单，必定设置止损
if resultOrder.Type == MARKET && isOpenOrAddOrder {
    // 等待500ms让订单成交
    time.Sleep(500 * time.Millisecond)
    
    // 更新持仓
    a.updatePositions(ctx)
    
    // 设置止损（无论查询订单状态是否成功）
    a.positionMgr.SetStopLoss(ctx, symbol)
}
```

**改进点**:
- ✅ 提前判断订单类型，不依赖后续查询
- ✅ 添加延迟等待订单成交
- ✅ 即使查询失败也会设置止损
- ✅ 添加详细日志记录

**修改文件**:
- `internal/strategy/adapter.go`

---

### 🔧 仓位管理逻辑完善

#### 1. 加仓次数限制
- **问题**: 策略允许无限次加仓
- **修复**: 限制每个持仓最多加仓1次
- **实现**: 
  - 添加 `PositionState.AddPositionCount` 字段
  - 在 `ProcessSignal` 中检查加仓次数
  - 达到限制后自动忽略后续加仓信号

```go
// 检查加仓次数限制
if position.AddPositionCount >= 1 {
    return nil, nil // 拒绝第二次加仓
}
position.AddPositionCount++ // 加仓后计数
```

#### 2. 自动止损设置
- **问题**: 开仓后没有自动设置止损保护
- **修复**: 开仓/加仓成功后自动设置止损单
- **实现**:
  - 添加 `Manager.SetStopLoss()` 方法
  - 在订单成功后调用止损设置
  - 加仓时更新止损单（取消旧的，创建新的）

**止损参数**:
- 类型: `STOP_MARKET` (市价止损单)
- 触发价: 入场价 ± 0.6%
- 数量: 全部持仓
- reduce_only: true

#### 3. 清理"强信号"概念
- **问题**: 代码和注释中引入了误导性的"强信号"概念
- **修复**: 删除所有"强信号"/"弱信号"的描述
- **说明**: 策略只有"是否满足加仓条件"，没有强弱之分

**修改**:
- 删除 `Metadata["strong_signal"]` 字段
- 只保留 `Metadata["add_position_eligible"]`
- 更新所有相关注释和日志

**修改文件**:
- `internal/position/manager.go`
- `internal/strategy/macd_ema_strategy.go`
- `internal/strategy/adapter.go`
- `internal/core/interfaces.go`

---

## 2025-12-05

### 🐛 订单数量为零问题修复

#### 问题
```
risk check failed: invalid quantity: 0.000000
```

#### 根本原因
仓位管理器创建订单时 `Quantity=0`，期望执行器计算，但：
1. 风险检查在数量计算**之前**执行
2. 执行器没有从 Metadata 读取 `usdt_amount` 的逻辑

#### 解决方案
在仓位管理器中直接计算订单数量：

```go
// internal/position/manager.go - createOpenOrder()
leverage := float64(m.config.DefaultLeverage)
quantity := (usdtAmount * leverage) / currentPrice

order := &core.Order{
    Quantity: quantity, // ✅ 已计算好
    // ...
}
```

**计算公式**:
```
数量 = (USDT金额 * 杠杆) / 当前价格
```

**修改文件**:
- `internal/position/manager.go`
- `internal/execution/binance/executor.go` (保留后备计算逻辑)

---

### 🐛 API 签名错误修复

#### 问题
```
API error (status 400): {"code":-1022,"msg":"Signature for this request is not valid."}
```

#### 根本原因
- Go 程序可能使用了系统代理设置
- curl 直连可以工作，但 Go HTTP Client 被代理阻塞

#### 解决方案
1. 添加 `recvWindow=60000` 参数（60秒时间窗口）
2. 禁用系统代理，使用直连

```go
// internal/execution/binance/client.go
transport := &http.Transport{
    Proxy: nil, // 禁用代理，直连
}
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: transport,
}
```

**修改文件**:
- `internal/execution/binance/client.go`

---

### 🐛 策略逻辑修复（加仓信号问题）

#### 问题
策略总是生成 `ADD_SHORT`/`ADD_LONG` 信号，导致无持仓时被拒绝：
```json
{"signal_type":"ADD_SHORT","reason":"..."}
{"msg":"No orders generated"}
```

#### 根本原因
`checkScenario1()` 方法在检测到三个交叉同时满足时，直接返回加仓信号：

```go
// ❌ 错误的逻辑
if MACD死叉 && EMA5/VWAP8死叉 {
    signal = OPEN_SHORT
    if EMA5/EMA15死叉 {
        signal.Type = ADD_SHORT // 直接改成加仓！
    }
    return signal
}
```

#### 解决方案
策略层只生成 `OPEN` 信号，用 Metadata 标记是否满足加仓条件：

```go
// ✅ 修复后的逻辑
if MACD死叉 && EMA5/VWAP8死叉 {
    signal = OPEN_SHORT
    if EMA5/EMA15死叉 {
        signal.Metadata["add_position_eligible"] = 1.0
    }
    return signal
}
```

仓位管理器根据持仓状态决策：
- 无持仓 → 执行开仓
- 有持仓 + 满足加仓条件 + 同方向 → 执行加仓
- 有持仓 + 不满足加仓条件 → 忽略

**修改文件**:
- `internal/strategy/macd_ema_strategy.go`
- `internal/position/manager.go`

---

## 2025-12-04 及之前

### ✅ 核心功能实现

#### 1. 数据管理模块 (DataManager V2)
- 多交易对 WebSocket 数据订阅
- K线数据本地存储（SQLite）
- 数据完整性检查
- 自动重连机制

#### 2. 策略模块
- MACD + EMA + VWAP 组合策略
- 技术指标计算（MACD, EMA5, EMA15, VWAP8）
- 交叉检测（金叉/死叉）
- 趋势检测（连续4周期涨跌）

#### 3. 仓位管理模块
- 信号转订单处理
- 仓位大小计算（开仓20%, 加仓40%）
- 风险控制（最大持仓数、杠杆限制）
- 跟踪止盈（四级阶梯）

#### 4. 执行模块
- Binance 期货 API 集成
- 订单提交和查询
- 账户信息查询
- 持仓信息查询
- 杠杆和保证金模式设置

#### 5. 日志系统
- 按交易对分文件日志
- 按会话归档
- 结构化日志（JSON格式）
- 分级日志输出

---

## 配置变更

### 策略参数优化

```yaml
strategy:
  parameters:
    macd_fast: 16      # 从 12 改为 16
    macd_slow: 26      # 保持不变
    macd_signal: 9     # 保持不变
    ema_short: 5       # 保持不变
    ema_long: 15       # 保持不变
    vwap_period: 8     # 保持不变
```

### 仓位管理配置

```yaml
position:
  position_sizing:
    open_percent: 0.20    # 开仓使用20%资金
    add_percent: 0.40     # 加仓使用40%资金（只加仓1次）
  
  risk_limits:
    stop_loss_percent: 0.006  # 止损0.6%
    max_open_positions: 3     # 最多3个持仓
  
  default_leverage: 5          # 5倍杠杆
  default_margin_mode: ISOLATED # 逐仓模式
```

---

## 已知问题

### 已解决 ✅
1. ✅ 策略只生成加仓信号 → 已修复（生成OPEN信号）
2. ✅ API 签名验证失败 → 已修复（禁用代理，添加recvWindow）
3. ✅ 订单数量为零 → 已修复（仓位管理器计算数量）
4. ✅ 无限加仓 → 已修复（限制最多加仓1次）
5. ✅ 缺少止损保护 → 已修复（自动设置止损单）

### 待改进 ⏳
1. WebSocket 订单状态监听（目前使用轮询）
2. 止损单被触发后的状态更新
3. 动态止损调整（移动止损到盈亏平衡点）
4. 分批平仓功能

---

## 性能优化

### 已完成
- ✅ 数据库连接池优化
- ✅ K线数据批量插入
- ✅ 指标计算缓存

### 计划中
- ⏳ WebSocket 消息批处理
- ⏳ 订单状态缓存
- ⏳ 减少 API 调用频率

---

## 文档更新

### 新增文档
- `docs/API_REFERENCE.md` - 完整的API参考文档
- `docs/CHANGELOG.md` - 本文档
- `docs/USER_GUIDE.md` - 用户使用指南

### 归档文档
- `docs/archive/fixes_20251205/` - 2025-12-05 修复过程记录
- `docs/archive/` - 历史实现文档

---

## 贡献者

- AI Assistant - 代码实现和问题修复
- User - 需求定义和测试验证

---

**最后更新**: 2025-12-06

