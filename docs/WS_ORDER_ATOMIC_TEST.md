# WebSocket订单原子操作测试

## 测试场景

这个测试包含4个真实交易场景，测试币安期货WebSocket订单的原子操作：

### 场景1: 市价开仓
- 开一个市价单（BTC做多 0.001）
- 不平仓，保持持仓

### 场景2: 止损单操作
- 对当前持仓下一个止损单（STOP_MARKET）
- 止损价格：入场价 × 0.98（下方2%）
- 立即撤销止损单

### 场景3: 跟踪止损单操作
- 下一个跟踪止损单（TRAILING_STOP_MARKET）
- 激活价格：入场价 × 1.01（上方1%，有盈利时触发）
- 回调比例：0.5%
- 立即撤销跟踪止损单

### 场景4: 复杂操作流程
1. 下限价止盈单（入场价 × 1.02，上方2%）
2. 市价加仓 0.001 BTC
3. 撤销旧止盈单
4. 下新止盈单（数量匹配加仓后的仓位）
5. 市价全部平仓
6. 撤销止盈单（如果还存在）

## 测试参数

```
Symbol: BTCUSDT
当前价格: ~90,000 USDT
测试数量: 0.001 BTC (~90 USDT)
杠杆: 默认（配置文件中的设置）
保证金模式: ISOLATED（逐仓）
```

## 运行测试

### 方式1: 使用脚本

```bash
# 设置环境变量
export BINANCE_API_KEY='your_api_key'
export BINANCE_SECRET_KEY='your_secret_key'

# 运行测试脚本
./scripts/test-ws-order-atomic.sh
```

### 方式2: 直接运行测试

```bash
# 设置环境变量
export BINANCE_API_KEY='your_api_key'
export BINANCE_SECRET_KEY='your_secret_key'

# 运行测试
cd /home/maeda/Documents/projects/goQuant
go test -v -timeout 5m ./internal/execution/binance -run TestWSOrderAtomic
```

## 预期输出

```
📊 场景1: 开市价单（做多）
✅ 场景1成功 - 市价单已下
📍 当前持仓: side=LONG, size=0.001, entry_price=90123.45

📊 场景2: 下止损单，然后撤销
✅ 止损单已下: stop_price=88321.00
✅ 场景2成功 - 止损单已撤销

📊 场景3: 下跟踪止损单，然后撤销
✅ 跟踪止损单已下: activate_price=91024.68, callback_rate=0.5
✅ 场景3成功 - 跟踪止损单已撤销

📊 场景4: 限价止盈单 + 加仓 + 修改 + 平仓 + 撤单
✅ 4.1 限价止盈单已下: price=91925.92, quantity=0.001
✅ 4.2 市价加仓成功: add_quantity=0.001
📍 加仓后持仓: size=0.002, entry_price=90124.00
✅ 4.3 旧止盈单已撤销
✅ 4.3 新止盈单已下（匹配新仓位）: quantity=0.002
✅ 4.4 市价平仓成功: quantity=0.002
✅ 持仓已清空
✅ 4.5 止盈单已手动撤销

🎉 所有场景测试完成！
```

## ⚠️ 注意事项

### 1. 这是真实交易
- **会真实下单**，使用真实资金
- 测试前确保账户有足够余额（建议至少200 USDT）
- 建议先在测试网测试

### 2. 市场风险
- BTC价格波动较大
- 测试过程中价格可能变化
- 止损/止盈单可能被意外触发

### 3. 测试环境
- 确保网络稳定
- 确保API权限正确（需要交易权限）
- 确保账户未被限制交易

### 4. 清理
- 测试结束会自动平仓
- 如果测试中断，请手动检查并清理持仓
- 检查是否有未撤销的挂单

## 测试时间

整个测试大约需要 **15-20秒**：
- 场景1: ~3秒（市价开仓）
- 场景2: ~3秒（止损单操作）
- 场景3: ~3秒（跟踪止损单操作）
- 场景4: ~8秒（复杂操作）

## 故障排查

### 测试失败：订单被拒绝
```
可能原因：
1. 余额不足
2. 杠杆未设置
3. 保证金模式未设置
4. API权限不足
```

**解决方案**：
```bash
# 手动设置杠杆和保证金模式
curl -X POST 'https://fapi.binance.com/fapi/v1/leverage' \
  -H 'X-MBX-APIKEY: your_api_key' \
  -d 'symbol=BTCUSDT&leverage=5&timestamp=...'
```

### 测试失败：持仓未清空
```
可能原因：
1. 平仓订单未完全成交
2. 持仓数量精度问题
```

**解决方案**：
```bash
# 手动平仓
# 登录币安网页版，手动平掉剩余持仓
```

### 测试超时
```
可能原因：
1. 网络问题
2. WebSocket连接断开
3. 订单成交延迟
```

**解决方案**：
- 检查网络连接
- 重新运行测试
- 增加timeout时间：`-timeout 10m`

## 代码位置

- **测试文件**: `internal/execution/binance/ws_order_atomic_test.go`
- **WebSocket订单客户端**: `internal/execution/binance/ws_order.go`
- **测试脚本**: `scripts/test-ws-order-atomic.sh`

## 扩展测试

如果要测试ETH，修改测试代码：

```go
symbol := "ETHUSDT"
currentPrice := 3100.0  // ETH当前价格约3100
quantity := 0.01        // 0.01 ETH（约31 USDT）
```

## 相关文档

- [币安期货API文档](https://binance-docs.github.io/apidocs/futures/cn/)
- [WebSocket订单指南](../../docs/WEBSOCKET_ORDER_GUIDE.md)
- [测试指南](../../TEST_GUIDE.md)

