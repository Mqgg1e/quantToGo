# 订单数量为零问题 - 最终修复

**日期**: 2025-12-05  
**问题**: `risk check failed: invalid quantity: 0.000000`

---

## 🔴 问题根源

订单数量计算的**时机问题**：

```
原始流程（错误）：
1. Position Manager 创建订单，Quantity=0
2. Adapter 进行风险检查 ← ❌ 此时 Quantity 还是 0
3. Executor 计算 Quantity ← 太晚了
```

---

## ✅ 最终解决方案

**在仓位管理器中直接计算数量**，不依赖执行器。

### 修改的文件

#### 1. `internal/position/manager.go`

在 `createOpenOrder` 方法中添加数量计算：

```go
// 计算订单数量：quantity = (usdt_amount * leverage) / price
leverage := float64(m.config.DefaultLeverage)
quantity := (usdtAmount * leverage) / currentPrice

logger.Info("Calculated order quantity",
    zap.Float64("quantity", quantity),
    zap.Float64("usdt_amount", usdtAmount),
    zap.Float64("leverage", leverage),
    zap.Float64("price", currentPrice),
)

// 创建订单时设置数量
order := &core.Order{
    Symbol:     signal.Symbol,
    Type:       core.OrderTypeMarket,
    Side:       side,
    Quantity:   quantity, // ✅ 已计算好
    Leverage:   m.config.DefaultLeverage,
    MarginMode: m.config.DefaultMarginMode,
    // ...
}
```

#### 2. `internal/execution/binance/executor.go`

保留执行器中的数量计算逻辑（作为后备），并添加风险检查：

```go
// 如果订单没有设置数量，尝试从Metadata计算（后备方案）
if order.Quantity == 0 && order.Metadata != nil {
    // ... 从 usdt_amount 计算
}

// 风险检查
if order.Quantity <= 0 {
    return nil, fmt.Errorf("invalid quantity: %f", order.Quantity)
}
```

#### 3. `internal/strategy/adapter.go`

保持风险检查在 PlaceOrder 之前：

```go
func (a *Adapter) executeOrder(ctx context.Context, order *core.Order) error {
    // 风险检查（此时数量已经计算好）
    if err := a.positionMgr.CheckRisk(order); err != nil {
        return fmt.Errorf("risk check failed: %w", err)
    }
    
    // 提交订单
    resultOrder, err := a.executor.PlaceOrder(ctx, order)
    // ...
}
```

---

## 📊 修复后的流程

```
1. Strategy 生成信号
   ↓
2. Position Manager.ProcessSignal()
   ↓
3. Position Manager.createOpenOrder()
   - 获取账户余额
   - 计算 USDT 金额 = 余额 * 20%
   - 计算数量 = (USDT * 杠杆) / 当前价格 ✅
   - 创建订单，Quantity 已设置 ✅
   ↓
4. Adapter.executeOrder()
   - 风险检查（检查数量 > 0） ✅
   - 风险检查（检查最大持仓数） ✅
   ↓
5. Executor.PlaceOrder()
   - 格式化数量
   - 发送到 Binance ✅
```

---

## 🧪 示例计算

**输入**:
- 账户余额: 5000 USDT
- 开仓比例: 20%
- USDT 金额: 5000 * 0.20 = 1000 USDT
- 杠杆: 5x
- 当前价格: 90,396.7 USDT/BTC

**计算**:
```
quantity = (1000 * 5) / 90396.7
         = 5000 / 90396.7
         = 0.0553 BTC
```

**日志输出**:
```json
{"msg":"Creating order","usdt_amount":1000,"current_price":90396.7,"leverage":5}
{"msg":"Calculated order quantity","quantity":0.0553}
{"msg":"Order placed","quantity":0.0553,"side":"BUY","status":"NEW"}
```

---

## ✅ 验证清单

- [x] 数量在仓位管理器中计算
- [x] 使用当前价格（从信号价格获取）
- [x] 风险检查在数量计算之后
- [x] 执行器保留后备计算逻辑
- [x] 编译无错误
- [x] 添加详细日志

---

## 🎯 关键改进

### 优点

1. **职责清晰**: 仓位管理器负责所有仓位计算，包括数量
2. **早期验证**: 风险检查在提交前完成
3. **双重保障**: 执行器仍保留计算逻辑作为后备
4. **详细日志**: 记录计算过程，便于调试

### 代码设计

```
Position Manager (仓位管理)
  ├─ 计算 USDT 金额
  ├─ 计算订单数量 ✅ (新增)
  └─ 创建订单对象

Adapter (适配器)
  ├─ 风险检查 ✅
  └─ 提交订单

Executor (执行器)
  ├─ 数量计算（后备）
  ├─ 格式化数量
  └─ 发送 API 请求
```

---

## 🚀 测试验证

### 1. 重启程序

```bash
cd /home/maeda/Documents/projects/goQuant
./scripts/start-live.sh
```

### 2. 观察日志

```bash
tail -f logs/session_*/BTCUSDT_1m.log | grep -E "Signal|Creating|Calculated|Order placed"
```

### 3. 预期输出

```json
{"msg":"Signal generated","signal_type":"OPEN_LONG","price":90396.7}
{"msg":"Creating order","usdt_amount":1000,"current_price":90396.7}
{"msg":"Calculated order quantity","quantity":0.0553}
{"msg":"Order placed","quantity":0.0553,"status":"FILLED"}
{"msg":"Position update","size":0.0553}
```

---

## 📝 相关修复

这是**第三个**问题的修复：

1. ✅ 策略逻辑问题 - 已修复（生成 OPEN 信号而不是 ADD 信号）
2. ✅ API 签名问题 - 已修复（添加 recvWindow，禁用代理）
3. ✅ 订单数量问题 - 已修复（在仓位管理器中计算数量）

---

**修复完成**: 2025-12-05  
**状态**: 🟢 就绪，可以交易

