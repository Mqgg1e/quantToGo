## ! 修改记录写在 docs/CHANGELOG.md 中，如果代码有修改docs/API_REFERENCE.md也要配套修改，其他记录写在docs/README.md和docs/USER_GUIDE.md里，主要问题和需求在docs/plansAndProgressV1.md，不允许再创建其他md文件
## ! 在docs/plansAndProgressV1.md问题下方的两个####内给出简短回复
## ! 完成后不可以再创建md文件
## ! 之前的进度参考docs/plansAndProgressV1.md


### 091225

#### 0807
目前架构细节（execution和manager部分）

#### 当前架构详情

**一、缓存机制**
- **位置**: 内存缓存（无外部缓存系统如Redis）
- **实现**: 
  - `LiveExecutor` 内使用 `map` 缓存订单和持仓: `orderCache map[string]*core.Order`, `positionCache map[string]*core.Position`
  - `Manager` 内使用 `positions map[string]*PositionState` 缓存持仓状态
- **K线数据存储**: SQLite 数据库（位于 `data/wsdata/` 和 `data/enhanced_wsdata/`）
  - 每个 symbol+interval 组合独立一个 .db 文件
  - 表结构: klines 表，包含开高低收量等字段，有索引优化查询

**二、使用的API**
1. **币安期货API** (`internal/execution/binance/client.go`):
   - `POST /fapi/v1/order` - 创建订单
   - `DELETE /fapi/v1/order` - 撤销订单
   - `GET /fapi/v1/order` - 查询订单
   - `GET /fapi/v1/openOrders` - 查询未成交订单
   - `GET /fapi/v2/account` - 查询账户信息
   - `GET /fapi/v2/positionRisk` - 查询持仓风险
   - `POST /fapi/v1/leverage` - 设置杠杆
   - `POST /fapi/v1/marginType` - 设置保证金模式

2. **WebSocket 数据流** (`internal/dataManager/dataFromWS.go`):
   - `wss://fstream.binance.com/ws/<symbol>@kline_<interval>` - K线实时数据流
   - 支持代理连接
   - 有心跳机制（20秒 ping）

**三、Execution 可下的订单类型**
`internal/execution/binance/executor.go` 支持以下订单类型:
1. **MARKET** - 市价单（开仓/平仓主要使用）
2. **LIMIT** - 限价单（需要指定价格和 TimeInForce）
3. **STOP_MARKET** - 市价止损单（需要 stopPrice，支持 closePosition=true 全平）
4. **STOP_LIMIT** - 限价止损单
5. **TAKE_PROFIT** - 止盈单
6. **TRAILING_STOP_MARKET** - 跟踪止损市价单（需要 callbackRate）

**订单参数支持**:
- 单向持仓模式 (`PositionSide=BOTH`)
- 杠杆设置 (1-125倍)
- 只减仓标志 (`reduceOnly`)
- 平掉全部仓位 (`closePosition`)
- 数量计算：支持通过 `usdt_amount` 元数据自动计算数量
- 止损单支持 `stopPrice`
- 跟踪止损支持 `callbackRate` (0.1-10，单位%)

**四、Manager 功能** (`internal/position/manager.go`)
1. **信号处理** (`ProcessSignal`):
   - 检测反向信号并先平仓
   - 处理开仓信号（OpenLong/OpenShort）
   - 处理加仓信号（AddLong/AddShort）- 限制只能加仓1次，且需在10分钟内
   - 处理平仓信号（CloseLong/CloseShort）
   - 加仓需满足策略的 `add_position_eligible=1.0` 元数据

2. **持仓管理**:
   - 维护持仓状态 (`PositionState`)，包含最高盈利、止损价格、止损单ID、加仓次数等
   - 持仓为0时自动删除持仓记录
   - 支持多个交易对同时持仓

3. **风险控制** (`CheckRisk`):
   - 最大持仓数检查（MaxOpenPositions）
   - 杠杆范围验证 (1-125)
   - 订单数量合法性检查

4. **仓位计算** (`CalculatePositionSize`):
   - 根据账户余额和配置的百分比计算 USDT 金额
   - 开仓使用 `OpenPercent`
   - 加仓使用 `AddPercent`
   - 受 `MaxPositionSize` 限制

5. **止损机制**:
   - 固定止损：0.6% 距离入场价
   - 跟踪止盈：3个级别
     - 1.8%+ 盈利：0.68% 回撤止盈
     - 1.0%+ 盈利：0.55% 回撤止盈
     - 0.5%+ 盈利：0.4% 回撤止盈

6. **订单创建**:
   - `createOpenOrder`: 创建市价开仓单，自动计算数量
   - `createAddOrder`: 创建加仓单（逻辑同开仓）
   - `createCloseOrder`: 创建市价平仓单，使用持仓的全部数量

**五、数据流**
```
WebSocket K线数据 → SQLite存储 → Strategy.OnKline() → TradingSignal
→ Manager.ProcessSignal() → Order → Executor.PlaceOrder() → Binance API
→ Order Response → Manager.UpdatePosition() → PositionState更新
```

####