# 🧪 UserDataStream 测试指南（修复后）

## 问题已修复 ✅

根据您的测试输出，我已经修复了以下问题：

1. ✅ **nil pointer panic** - 重连时因 executor 为 nil 导致崩溃
2. ✅ **WebSocket 超时** - 1分钟后连接断开（"i/o timeout"）
3. ✅ **代理配置** - WebSocket 没有使用代理导致连接失败
4. ✅ **连接保活** - 没有发送 ping 消息保持连接活跃

---

## 快速测试

### 1. 重新编译（已完成）
```bash
go build -o bin/test-userdata-stream cmd/test-userdata-stream/main.go
```
✅ 编译成功

### 2. 运行测试
```bash
./scripts/test-userdata-stream.sh
```

或直接运行：
```bash
./bin/test-userdata-stream
```

---

## 预期改进

### 启动输出应该显示：
```
🧪 Testing Binance UserDataStream...

1️⃣  Creating AccountCache...
✅ AccountCache created

✅ Proxy configured: http://127.0.0.1:7897  ← 新增

2️⃣  Initializing cache from REST API...
✅ Cache initialized successfully

3️⃣  Starting UserDataStream...
INFO ListenKey created {"listen_key": "JH5A2umj..."}
INFO WebSocket connected successfully
✅ UserDataStream started successfully
```

### 运行中不应该再出现：
- ❌ `ERROR Failed to read message {"error": "i/o timeout"}`
- ❌ `ERROR Panic in readLoop {"panic": "nil pointer dereference"}`
- ❌ 频繁的重连日志

### 应该能看到（debug 模式）：
```
DEBUG WebSocket ping sent              ← 每 54 秒
DEBUG ListenKey kept alive successfully ← 每 30 分钟
```

---

## 手动测试步骤

### 1. 启动测试程序
```bash
./bin/test-userdata-stream
```

### 2. 打开币安测试网
访问: https://testnet.binancefuture.com

### 3. 下市价单测试
在测试网下一个 BTCUSDT 或 ETHUSDT 的市价单（任意方向）

### 4. 观察程序输出
应该立即看到：

```
INFO binance/userdata_stream.go:xxx Received UserDataStream event
     {"event_type": "ORDER_TRADE_UPDATE", "event_time": ...}

INFO binance/userdata_stream.go:xxx Order update received
     {"symbol": "BTCUSDT", "side": "BUY", "order_id": "...", "status": "NEW"}

INFO binance/userdata_stream.go:xxx Order update received
     {"symbol": "BTCUSDT", "side": "BUY", "order_id": "...", "status": "FILLED"}

INFO binance/userdata_stream.go:xxx Account update received
     {"reason": "ORDER", "balance_count": 1, "position_count": 1}

INFO cache/account_cache.go:xxx Position updated
     {"symbol": "BTCUSDT", "side": "LONG", "size": 0.001, "entry_price": 42000}
```

### 5. 查看缓存状态
下一次状态更新（最多等待 30 秒）应该显示：

```
📊 Cache Status Update:
   Balance: 5310.50 USDT  ← 余额变化
   Positions: 1           ← 持仓数量增加
   Orders: 1              ← 订单数量增加
   Last Update: 2025-12-12 10:00:00
   Version: 5             ← 版本号增加
   
   Position Details:
     - BTCUSDT LONG: 0.0010 @ 42000.00 (PnL: 5.00, 0.12%)
   
   Order Details:
     - BTCUSDT BUY MARKET: 0.0010 @ 42000.00 [FILLED]
```

### 6. 平仓测试
在测试网平仓，应该看到：

```
INFO Order update received {"status": "FILLED"}
INFO Account update received {"reason": "ORDER"}
INFO Position closed {"symbol": "BTCUSDT"}
```

缓存状态更新：
```
📊 Cache Status Update:
   Positions: 0  ← 持仓已清空
   Orders: 0     ← 订单已清空
```

---

## 长时间稳定性测试

### 测试目标
程序应该能够稳定运行超过：
- ✅ 1 分钟（之前会超时）
- ✅ 5 分钟（验证 ping 机制）
- ✅ 30 分钟（验证 ListenKey 保活）

### 观察要点
1. **连接稳定**：
   - 不应该频繁重连
   - 每 54 秒看到一次 `DEBUG WebSocket ping sent`（如果是 debug 模式）

2. **ListenKey 保活**：
   - 每 30 分钟看到 `DEBUG ListenKey kept alive successfully`
   - 不应该因为 ListenKey 过期而重连

3. **事件接收**：
   - 所有手动操作都能实时收到事件
   - 缓存数据准确更新

---

## 如果还有问题

### 问题 1: 代理连接失败
**日志**: `Failed to set proxy` 或 `dial websocket: ... proxy error`

**解决**:
1. 确认代理正在运行：
```bash
curl -x http://127.0.0.1:7897 https://www.google.com
```

2. 如果代理不可用，可以临时禁用代理测试：

编辑 `cmd/test-userdata-stream/main.go`:
```go
// 注释掉这两行
// proxyURL := "http://127.0.0.1:7897"
// if err := client.SetProxy(proxyURL); err != nil { ... }
```

重新编译：
```bash
go build -o bin/test-userdata-stream cmd/test-userdata-stream/main.go
```

### 问题 2: 仍然收不到事件
**检查 API Key**:
```bash
# 测试 REST API 是否正常
./scripts/test-api.sh
```

**检查 ListenKey**:
程序日志中应该有：
```
INFO ListenKey created {"listen_key": "..."}
```

如果 ListenKey 创建失败，检查 API Key 权限。

### 问题 3: 日志太少
**启用 debug 日志**:

编辑 `cmd/test-userdata-stream/main.go`:
```go
logConfig.Level = "debug"  // 改为 debug
```

重新编译运行，会看到更详细的日志。

---

## 验证清单

请测试以下功能并打勾：

### 基础功能
- [ ] 程序成功启动（显示代理配置）
- [ ] WebSocket 连接成功
- [ ] 缓存初始化成功（显示当前余额）
- [ ] 没有 panic 或 timeout 错误

### 实时更新
- [ ] 手动下单后收到 ORDER_TRADE_UPDATE 事件
- [ ] 订单状态变化实时更新（NEW → FILLED）
- [ ] 持仓变化后收到 ACCOUNT_UPDATE 事件
- [ ] 余额变化实时更新
- [ ] 缓存状态准确（Positions、Orders 数量正确）

### 稳定性
- [ ] 程序稳定运行超过 1 分钟（不超时）
- [ ] 程序稳定运行超过 5 分钟（不重连）
- [ ] 程序稳定运行超过 30 分钟（ListenKey 保活）
- [ ] 多次下单平仓都能正确接收事件

### 优雅关闭
- [ ] Ctrl+C 能正常停止程序
- [ ] 停止时正确关闭 WebSocket 和 ListenKey
- [ ] 没有 goroutine 泄漏

---

## 成功标准

✅ **测试通过**条件：
1. 程序启动无错误
2. WebSocket 连接稳定（不超时、不 panic）
3. 能实时接收交易事件
4. 缓存数据准确更新
5. 手动下单 3-5 次都能正确响应
6. 稳定运行超过 10 分钟

🎯 **达到以上标准后**，可以进入下一阶段：
- Week 3: 执行层重构
- Week 4: 仓位管理重构
- Week 5: 主程序集成

---

## 下一步行动

1. **现在**: 重新运行测试程序
```bash
./bin/test-userdata-stream
```

2. **手动测试**: 在币安测试网下单，验证实时更新

3. **报告结果**: 测试通过后告诉我，我们继续下一阶段

4. **如果有问题**: 复制完整的日志输出给我，我会进一步调查

---

**准备好了！请运行测试并告诉我结果。** 🚀

