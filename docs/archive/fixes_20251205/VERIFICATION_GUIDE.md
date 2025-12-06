# 修复验证指南

## 🎯 问题总结

**原问题**：策略总是生成 `ADD_SHORT`/`ADD_LONG` 加仓信号，导致无持仓时订单被拒绝。

**原因**：当满足所有三个交叉条件（MACD + EMA5/VWAP8 + EMA5/EMA15）时，策略直接返回加仓信号而不是开仓信号。

**修复**：
1. 策略层只生成开仓信号（`OPEN_LONG`/`OPEN_SHORT`）
2. 使用 `Metadata["strong_signal"]=1.0` 标记强信号
3. 仓位管理器根据持仓状态和信号强度决定是否加仓

---

## ✅ 修复内容

### 文件变更

1. **`/internal/strategy/macd_ema_strategy.go`**
   - 修改 `checkScenario1()` 方法
   - 不再直接修改信号类型为 `ADD_SHORT`/`ADD_LONG`
   - 添加 `strong_signal` 和 `add_position_eligible` 元数据标记

2. **`/internal/position/manager.go`**
   - 修改 `ProcessSignal()` 方法
   - 处理开仓信号时检查是否为强信号
   - 有持仓且为强信号时执行加仓

---

## 🧪 如何验证修复

### 方法 1: 运行实时交易并观察日志

```bash
# 启动实时交易
./scripts/start-live.sh

# 在另一个终端查看最新的日志
tail -f logs/session_*/BTCUSDT_1m.log
```

**预期看到的日志**：

#### ✅ 无持仓时（应该生成开仓信号）
```json
{"msg":"Signal generated","signal_type":"OPEN_SHORT","reason":"MACD死叉+EMA5/VWAP8死叉+EMA5/EMA15死叉(强信号)"}
{"msg":"Creating order","signal_type":"OPEN_SHORT","usdt_amount":XXX}
{"msg":"Order placed","side":"SELL","status":"FILLED"}
```

#### ✅ 有空单持仓时（应该执行加仓）
```json
{"msg":"Signal generated","signal_type":"OPEN_SHORT","reason":"MACD死叉+EMA5/VWAP8死叉+EMA5/EMA15死叉(强信号)"}
{"msg":"Strong signal detected, adding to position","position_side":"SHORT"}
{"msg":"Creating order","signal_type":"OPEN_SHORT","usdt_amount":XXX}
{"msg":"Order placed","side":"SELL","status":"FILLED"}
```

#### ❌ 之前的错误日志（已修复）
```json
// 这种情况不应该再出现
{"msg":"Signal generated","signal_type":"ADD_SHORT","reason":"..."}
{"msg":"No orders generated"}
```

---

### 方法 2: 代码审查检查点

#### ✅ 策略层 (`macd_ema_strategy.go`)

```go
// 检查点 1: checkScenario1 应该只生成 OPEN 信号
if s.hasRecentCross(s.macdCrosses, CrossTypeDeath, 3) &&
    s.hasRecentCross(s.emaVwapCrosses, CrossTypeDeath, 3) {
    
    signal := NewSignal(core.SignalTypeOpenShort, ...)  // ✅ OPEN_SHORT
    
    if s.hasRecentCross(s.emaEmaCrosses, CrossTypeDeath, 3) {
        signal.Metadata["strong_signal"] = 1.0  // ✅ 标记强信号
        // ❌ 不应该有: signal.Type = core.SignalTypeAddShort
    }
}
```

#### ✅ 仓位管理层 (`manager.go`)

```go
// 检查点 2: ProcessSignal 应该处理强信号
if signal.Type == core.SignalTypeOpenLong || signal.Type == core.SignalTypeOpenShort {
    if hasPosition {
        // ✅ 检查强信号
        strongSignalValue, hasStrongSignal := signal.Metadata["strong_signal"]
        isStrongSignal := hasStrongSignal && strongSignalValue == 1.0
        
        if isStrongSignal && 方向一致 {
            // ✅ 执行加仓
            return createAddOrder(...)
        }
    }
    // ✅ 无持仓时执行开仓
    return createOpenOrder(...)
}
```

---

## 📊 测试场景

### 场景 1: 首次信号（无持仓）
- **输入**: MACD死叉 + EMA5/VWAP8死叉 + EMA5/EMA15死叉
- **预期**: 生成 `OPEN_SHORT` + `strong_signal=1.0`
- **结果**: 执行开仓（20%资金）

### 场景 2: 二次信号（已有空单）
- **输入**: 再次出现相同的死叉组合
- **预期**: 生成 `OPEN_SHORT` + `strong_signal=1.0`
- **结果**: 检测到强信号且方向一致，执行加仓（40%资金）

### 场景 3: 普通信号（已有空单）
- **输入**: MACD死叉 + EMA5/VWAP8死叉（无 EMA5/EMA15死叉）
- **预期**: 生成 `OPEN_SHORT` + `strong_signal=0.0` 或无此字段
- **结果**: 不是强信号，忽略

### 场景 4: 反向信号（已有空单）
- **输入**: MACD金叉 + EMA5/VWAP8金叉
- **预期**: 生成 `OPEN_LONG`
- **结果**: 检测到反向信号，先平空单再开多单

---

## 🔍 故障排查

### 如果仍然看到 "No orders generated"

1. **检查是否真的是强信号被忽略**
   ```bash
   # 查看信号类型和原因
   cat logs/session_*/BTCUSDT_1m.log | grep "Signal generated" | tail -10
   ```

2. **检查是否有持仓状态**
   ```bash
   # 查看是否有仓位更新
   cat logs/session_*/BTCUSDT_1m.log | grep "Position update" | tail -5
   ```

3. **检查策略是否预热完成**
   ```bash
   # 查看预热状态
   cat logs/trading.log | grep "warmed up"
   ```

4. **检查是否有风险限制**
   ```bash
   # 查看是否有风险检查失败
   cat logs/trading.log | grep -i "risk\|error"
   ```

---

## 📝 验证清单

- [ ] 编译成功（无错误）
- [ ] 策略层不再生成 `ADD_SHORT`/`ADD_LONG` 信号
- [ ] 强信号带有 `strong_signal=1.0` 标记
- [ ] 无持仓时强信号能正常开仓
- [ ] 有持仓时强信号能正常加仓
- [ ] 有持仓时弱信号被忽略
- [ ] 反向信号能触发平仓+反向开仓

---

## 🚀 下一步

如果验证成功：
1. ✅ 提交代码更改
2. ✅ 更新文档
3. ✅ 进行更长时间的实盘测试

如果仍有问题：
1. 🔍 收集完整日志
2. 🔍 检查市场数据是否满足条件
3. 🔍 验证交叉检测逻辑是否正确
4. 🔍 检查 `hasRecentCross()` 方法的实现

---

**修复完成时间**: 2025-12-05  
**相关文件**: STRATEGY_FIX_SUMMARY.md

