# 订单数量为零问题修复

**日期**: 2025-12-05  
**问题**: 订单执行失败 `invalid quantity: 0.000000`

---

## 🔴 问题现象

```json
{"msg":"Signal generated","signal_type":"OPEN_SHORT","price":90466.5}
{"msg":"Execute order failed","error":"risk check failed: invalid quantity: 0.000000"}
```

- ✅ 策略正确生成了 `OPEN_SHORT` 信号
- ✅ 仓位管理器成功处理信号并创建订单
- ❌ 订单的 `Quantity` 字段为 0，风险检查失败

---

## 🔍 问题分析

### 根本原因

仓位管理器和执行器之间的**接口不匹配**：

1. **仓位管理器** (`position/manager.go`)：
   ```go
   order := &core.Order{
       Quantity: 0,  // 设置为0，期望执行器计算
       Metadata: map[string]interface{}{
           "usdt_amount": usdtAmount,  // 将USDT金额放在Metadata中
       },
   }
   ```

2. **执行器** (`execution/binance/executor.go`)：
   ```go
   // ❌ 原代码只是简单检查Quantity > 0
   if order.Quantity > 0 {
       req.Quantity = formatQuantity(order.Symbol, order.Quantity)
   }
   // 没有从Metadata中读取usdt_amount并计算数量的逻辑
   ```

### 问题流程

```
仓位管理器
  ↓
创建订单: Quantity=0, Metadata["usdt_amount"]=1000
  ↓
执行器
  ↓
检查: Quantity > 0? → NO
  ↓
不设置req.Quantity
  ↓
发送到Binance: 缺少quantity参数
  ↓
风险检查失败: invalid quantity: 0
```

---

## ✅ 解决方案

在执行器的 `PlaceOrder` 方法中添加逻辑，从 Metadata 中读取 `usdt_amount` 并计算实际数量。

### 修改的文件

**文件**: `internal/execution/binance/executor.go`

**修改内容**:

```go
// 数量计算逻辑
if order.Quantity == 0 && order.Metadata != nil {
    if usdtAmount, ok := order.Metadata["usdt_amount"].(float64); ok && usdtAmount > 0 {
        // 获取当前价格
        currentPrice := order.Price
        if currentPrice == 0 {
            // 市价单：从持仓风险接口获取标记价格
            positions, err := e.client.GetPositionRisk(ctx, order.Symbol)
            if err == nil && len(positions) > 0 {
                if markPrice, parseErr := parseFloat(positions[0].MarkPrice); parseErr == nil && markPrice > 0 {
                    currentPrice = markPrice
                }
            }
        }
        
        if currentPrice == 0 {
            return nil, fmt.Errorf("cannot calculate quantity: no price available")
        }
        
        // 计算数量 = (USDT金额 * 杠杆) / 价格
        leverage := float64(order.Leverage)
        if leverage == 0 {
            leverage = 1
        }
        quantity := (usdtAmount * leverage) / currentPrice
        order.Quantity = quantity
    }
}

if order.Quantity > 0 {
    req.Quantity = formatQuantity(order.Symbol, order.Quantity)
} else {
    return nil, fmt.Errorf("order quantity is zero or not set")
}
```

---

## 📊 修复后的流程

```
仓位管理器
  ↓
创建订单: Quantity=0, Metadata["usdt_amount"]=1000, Leverage=5
  ↓
执行器
  ↓
检查: Quantity == 0? → YES
  ↓
从Metadata读取: usdt_amount=1000
  ↓
获取当前价格: currentPrice=90466.5
  ↓
计算数量: quantity = (1000 * 5) / 90466.5 = 0.0553 BTC
  ↓
设置: order.Quantity = 0.0553
  ↓
格式化并发送到Binance
  ↓
✅ 订单成功提交
```

---

## 🧪 测试验证

### 预期结果

重新运行程序后，应该看到：

```json
// 1. 策略生成信号
{"msg":"Signal generated","signal_type":"OPEN_SHORT","price":90466.5}

// 2. 仓位管理器创建订单
{"msg":"Creating order","usdt_amount":1000,"leverage":5}

// 3. 执行器计算数量并下单
{"msg":"Order placed","quantity":0.0553,"side":"SELL","status":"FILLED"}

// 4. 持仓更新
{"msg":"Position update","side":"SHORT","size":0.0553}
```

### 验证步骤

1. 重新启动程序：
   ```bash
   ./scripts/start-live.sh
   ```

2. 观察日志：
   ```bash
   tail -f logs/session_*/BTCUSDT_1m.log
   ```

3. 检查关键信息：
   - ✅ Signal generated (OPEN_SHORT)
   - ✅ Creating order (显示 usdt_amount)
   - ✅ Order placed (显示 quantity)
   - ✅ Position update (显示持仓)

---

## 📝 技术细节

### 数量计算公式

对于Binance期货：

```
数量 = (USDT金额 * 杠杆) / 当前价格
```

**示例**:
- USDT金额: 1000
- 杠杆: 5x
- 当前价格: 90,000 USDT/BTC

```
数量 = (1000 * 5) / 90000 = 0.0556 BTC
```

### 价格获取策略

1. **优先使用订单价格** (`order.Price`)
   - 限价单会有明确的价格

2. **市价单获取标记价格**
   - 调用 `GetPositionRisk` 接口
   - 从返回的 `MarkPrice` 字段获取
   - 标记价格比最新成交价更稳定

3. **无价格时报错**
   - 避免使用 0 或无效价格计算

---

## 🔧 相关配置

### 仓位配置 (config.yaml)

```yaml
position:
  position_sizing:
    open_percent: 0.20    # 开仓使用20%资金
    add_percent: 0.40     # 加仓使用40%资金
  default_leverage: 5     # 5倍杠杆
```

**示例计算**:
- 账户余额: 5000 USDT
- 开仓比例: 20%
- USDT金额: 5000 * 0.20 = 1000 USDT
- 杠杆: 5x
- BTC价格: 90,000
- 下单数量: (1000 * 5) / 90000 = 0.0556 BTC

---

## 🎯 影响范围

### 受影响的订单类型

✅ **开仓订单** (OPEN_LONG / OPEN_SHORT)
- 仓位管理器使用 `usdt_amount` 传递金额
- 执行器需要计算数量

✅ **加仓订单** (ADD_LONG / ADD_SHORT) 
- 同样使用 `usdt_amount`
- 需要相同的计算逻辑

❌ **平仓订单** (CLOSE_LONG / CLOSE_SHORT)
- 直接使用持仓数量
- 不受影响

---

## 🚀 后续优化建议

### 1. 添加数量验证

在计算数量后，验证是否符合交易所限制：

```go
// 检查最小交易数量
minQty := 0.001  // Binance BTC最小数量
if order.Quantity < minQty {
    return nil, fmt.Errorf("quantity %f is below minimum %f", order.Quantity, minQty)
}
```

### 2. 添加详细日志

```go
logger.Info("Calculating order quantity",
    zap.Float64("usdt_amount", usdtAmount),
    zap.Float64("price", currentPrice),
    zap.Int("leverage", order.Leverage),
    zap.Float64("calculated_quantity", quantity),
)
```

### 3. 缓存价格

避免每次都调用 `GetPositionRisk`：
- 使用最近的K线价格
- 缓存标记价格（带过期时间）

---

## ✅ 修复完成

- [x] 识别问题：订单数量为0
- [x] 定位代码：执行器缺少数量计算逻辑
- [x] 实现修复：添加从Metadata读取usdt_amount并计算数量
- [x] 编译验证：无编译错误
- [x] 文档记录：创建修复文档

**状态**: 🟢 已修复，待测试验证

---

**修复日期**: 2025-12-05  
**相关问题**: 策略逻辑修复、API签名问题均已解决  
**下一步**: 重启程序，观察订单是否成功提交

