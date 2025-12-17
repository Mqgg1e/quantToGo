# WebSocket 下单功能使用指南

## 概述

Week 6 实现了完整的 WebSocket 订单接口，相比 REST API 具有更低的延迟和更好的性能。

## 主要特性

### 性能优势
- ⚡ **低延迟**: < 50ms（REST API 通常 100-200ms）
- 🔄 **避免限流**: 保持长连接，减少 REST API 频率限制
- 💪 **稳定性**: 自动重连、心跳保活机制

### 支持的订单类型

1. **市价单开仓** (`PlaceMarketOrder`)
   - 立即以市场价格执行
   - 适用于快速建仓

2. **限价单开仓** (`PlaceLimitOrder`)
   - 指定价格挂单
   - 支持 GTC/IOC/FOK 时效类型

3. **市价平仓** (`ClosePositionMarket`)
   - 以市价立即平掉指定数量
   - 自动设置 reduceOnly=true

4. **止损单** (`PlaceStopLossOrder`)
   - STOP_MARKET 类型
   - 触发后市价平掉全部仓位
   - closePosition=true

5. **跟踪止损单** (`PlaceTrailingStopOrder`)
   - TRAILING_STOP_MARKET 类型
   - 跟随价格移动止损
   - 支持回调比例 0.1%-10%

6. **撤销订单** (`CancelOrder`)
   - 通过订单 ID 撤销挂单

## 使用方法

### 1. 配置启用

在 `config/config.yaml` 中启用 WebSocket 下单:

```yaml
execution:
  binance:
    api_key: "${BINANCE_API_KEY}"
    secret_key: "${BINANCE_SECRET_KEY}"
    base_url: "https://testnet.binancefuture.com"  # 或生产环境
    use_ws_order: true  # 启用 WebSocket 下单
```

### 2. 程序中使用

#### 自动启用（推荐）

程序会根据配置自动启用 WebSocket 下单:

```go
// main.go 中自动处理
if cfg.Execution.Binance.UseWSOrder {
    if err := executor.EnableWebSocketOrder(ctx); err != nil {
        logger.Error("Failed to enable WebSocket ordering, falling back to REST API", zap.Error(err))
    } else {
        logger.Info("✅ WebSocket ordering enabled")
    }
}
```

#### 手动控制

也可以在程序中手动启用/禁用:

```go
// 启用 WebSocket 下单
if err := executor.EnableWebSocketOrder(ctx); err != nil {
    log.Printf("启用失败: %v", err)
}

// 禁用 WebSocket 下单（回退到 REST API）
executor.DisableWebSocketOrder()
```

### 3. 透明使用

一旦启用，`PlaceOrder()` 会自动使用 WebSocket，无需修改现有代码:

```go
// 下单代码保持不变
order := &core.Order{
    Symbol:   "BTCUSDT",
    Type:     core.OrderTypeMarket,
    Side:     core.OrderSideBuy,
    Quantity: 0.001,
}

result, err := executor.PlaceOrder(ctx, order)
// 自动通过 WebSocket 发送（如果已启用）
```

## 测试

### 运行测试程序

```bash
# 编译并运行交互式测试
./scripts/test-ws-order.sh

# 或直接运行
go run cmd/test-ws-order/main.go
```

### 测试选项

测试程序提供 6 种测试场景:

1. 测试市价开多单 (0.001 BTC)
2. 测试限价开多单 (0.001 BTC @ 90000)
3. 测试市价平仓
4. 测试止损单 (STOP_MARKET)
5. 测试跟踪止损单 (TRAILING_STOP_MARKET)
6. 查看当前持仓

## 技术细节

### 连接管理

- **连接地址**:
  - 生产环境: `wss://fstream.binance.com/ws-fapi/v1`
  - 测试网: `wss://stream.binancefuture.com/ws-fapi/v1`

- **心跳保活**: 每 54 秒发送一次 ping
- **自动重连**: 最多 5 次重试，指数退避

### 请求/响应机制

- 每个请求生成唯一 UUID
- 响应通过通道匹配返回
- 超时时间: 10 秒

### 签名机制

- 使用 HMAC SHA256 签名
- 自动添加时间戳和 recvWindow
- 接收窗口: 60 秒

## 降级机制

如果 WebSocket 连接失败或下单失败，系统会自动降级到 REST API:

```go
// executor 内部自动处理
if useWS {
    return e.placeOrderViaWebSocket(ctx, order, wsOrder)
}
// 失败时自动使用 REST API
return e.placeOrderViaREST(ctx, order)
```

## 最佳实践

### 1. 生产环境建议

```yaml
execution:
  binance:
    use_ws_order: true  # 推荐启用以获得更好性能
```

### 2. 测试环境

```yaml
execution:
  binance:
    use_ws_order: false  # 测试时可以先使用 REST API
```

### 3. 监控

观察日志中的 WebSocket 状态:

```
✅ WebSocket ordering enabled
WebSocket 订单连接已建立: wss://...
```

## 故障排查

### 连接失败

如果 WebSocket 连接失败:

1. 检查网络连接
2. 检查代理配置（如需要）
3. 验证 API Key 权限
4. 查看错误日志

### 订单失败

如果订单失败:

1. 检查订单参数（数量、价格等）
2. 验证账户余额
3. 检查杠杆和保证金设置
4. 查看币安 API 错误代码

### 降级到 REST API

如果需要临时禁用 WebSocket:

```go
executor.DisableWebSocketOrder()
```

或在配置文件中设置:

```yaml
execution:
  binance:
    use_ws_order: false
```

## 相关文档

- [CHANGELOG.md](CHANGELOG.md) - 详细变更记录
- [API_REFERENCE.md](API_REFERENCE.md) - API 接口文档
- [version2.md](version2.md) - 完整实施计划
- [plansAndProgressV2.md](plansAndProgressV2.md) - 进度记录

## 性能对比

| 指标 | REST API | WebSocket |
|------|----------|-----------|
| 延迟 | 100-200ms | < 50ms |
| 连接开销 | 每次请求建立 | 保持长连接 |
| 频率限制 | 容易触发 | 共享连接 |
| 可靠性 | 无自动重连 | 自动重连 |

## 总结

WebSocket 下单功能提供了更快、更可靠的订单执行方式，建议在生产环境中启用以获得最佳性能。同时保留了 REST API 作为降级方案，确保系统的稳定性和可用性。

