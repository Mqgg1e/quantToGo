# 架构设计文档

## 系统架构

### 整体架构图

```
┌─────────────────────────────────────────────────────────────┐
│                      Trading Bot Application                 │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌────────────┐    ┌────────────┐    ┌────────────┐       │
│  │  Data      │───▶│  Strategy  │───▶│  Position  │       │
│  │  Manager   │    │  Engine    │    │  Manager   │       │
│  └────────────┘    └────────────┘    └────────────┘       │
│         │                                      │            │
│         │                                      ▼            │
│         │                              ┌────────────┐      │
│         │                              │  Executor  │      │
│         │                              └────────────┘      │
│         │                                      │            │
└─────────┼──────────────────────────────────────┼───────────┘
          │                                      │
          ▼                                      ▼
  ┌───────────────┐                    ┌───────────────┐
  │ Binance       │                    │ Binance       │
  │ WebSocket     │                    │ REST API      │
  │ (K线数据)      │                    │ (交易执行)     │
  └───────────────┘                    └───────────────┘
```

## 核心模块

### 1. Data Manager (数据管理)
**位置**: `internal/dataManager/v2/`

**职责**:
- WebSocket实时K线订阅
- 数据完整性检查和补全
- SQLite数据存储
- 多订阅者消息分发

**关键文件**:
- `enhanced_multi_processor.go` - 多交易对处理器
- `connection_manager.go` - 连接管理和重连
- `completion_checker.go` - 数据完整性检查
- `message_dispatcher.go` - 消息分发

### 2. Strategy Engine (策略引擎)
**位置**: `internal/strategy/`

**职责**:
- 技术指标计算
- 交易信号生成
- 策略逻辑实现

**关键文件**:
- `macd_ema_strategy.go` - MACD+EMA策略实现
- `indicators.go` - 技术指标库
- `signal.go` - 信号类型和工具
- `adapter.go` - 策略适配器

### 3. Position Manager (仓位管理)
**位置**: `internal/position/`

**职责**:
- 信号转订单
- 资金分配计算
- 风险控制
- 持仓状态跟踪

**关键文件**:
- `manager.go` - 仓位管理器

### 4. Executor (执行器)
**位置**: `internal/execution/binance/`

**职责**:
- 币安API调用
- 订单执行
- 账户查询
- 持仓查询

**关键文件**:
- `executor.go` - 实盘执行器
- `client.go` - API客户端
- `models.go` - 数据结构

### 5. Core (核心接口)
**位置**: `internal/core/`

**职责**:
- 定义所有模块的标准接口
- 确保模块解耦

**关键文件**:
- `interfaces.go` - 接口定义

## 数据流

### 实时交易流程

```
1. WebSocket接收K线
   ↓
2. DataManager解析并存储
   ↓
3. 分发给Strategy Adapter
   ↓
4. Strategy计算指标
   ↓
5. 生成交易信号
   ↓
6. Position Manager处理信号
   ↓
7. 计算仓位大小
   ↓
8. 生成订单
   ↓
9. Executor执行订单
   ↓
10. 更新持仓状态
```

### 风险控制流程

```
每次K线更新时:
1. 更新持仓当前价格
   ↓
2. 计算未实现盈亏
   ↓
3. 检查固定止损 (0.6%)
   ├─ 触发 → 立即平仓
   └─ 未触发
       ↓
4. 检查跟踪止盈
   ├─ Level 1-4判断
   ├─ 触发 → 平仓
   └─ 未触发
       ↓
5. 检查反向信号
   ├─ 有 → 先平仓再开反向
   └─ 无 → 继续持仓
```

## 配置驱动

所有参数通过配置文件控制，无需修改代码：

```yaml
strategy:
  parameters:
    macd_fast: 16      # ← 修改这里即可调整MACD参数
    
position:
  default_leverage: 5   # ← 修改杠杆
  position_sizing:
    open_percent: 0.20  # ← 修改开仓比例
```

## 设计原则

1. **模块化**: 每个模块独立，通过接口通信
2. **可配置**: 所有参数可通过配置文件调整
3. **可测试**: 接口设计便于mock和单元测试
4. **可扩展**: 易于添加新策略、新交易所

## 扩展点

### 添加新策略
实现 `core.Strategy` 接口:
```go
type MyStrategy struct{}

func (s *MyStrategy) OnKline(kline core.KlineData) (*core.TradingSignal, error) {
    // 你的策略逻辑
}
```

### 添加新交易所
实现 `core.Executor` 接口:
```go
type OtherExchangeExecutor struct{}

func (e *OtherExchangeExecutor) PlaceOrder(ctx context.Context, order *core.Order) (*core.Order, error) {
    // 调用其他交易所API
}
```

### 添加新技术指标
在 `indicators.go` 中添加:
```go
type MyIndicator struct {
    // 状态
}

func (i *MyIndicator) Update(price float64) float64 {
    // 计算逻辑
}
```

---

**下一步**: 查看 [策略实现文档](03-STRATEGY.md) 了解具体策略逻辑

