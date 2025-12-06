# 下一步行动指南

## 当前状态总结

### ✅ 已完成的工作

1. **核心接口定义** (`internal/core/interfaces.go`)
   - 定义了所有模块的标准接口
   - 包括数据、策略、仓位、执行、可观测性等

2. **配置管理系统** (`internal/config/config.go`)
   - 支持YAML配置文件
   - 环境变量覆盖
   - 完整的配置验证
   - 示例配置文件 (`config/config.example.yaml`)

3. **策略模块基础设施** (`internal/strategy/`)
   - 信号类型定义和工具函数 (`signal.go`)
   - 技术指标库 (`indicators.go`): EMA, MACD, VWAP, SMA, ATR, RSI, Bollinger Bands
   - K线缓冲区和趋势检测工具

4. **数据模块接口适配**
   - `v2.KlineData` 实现了 `core.KlineData` 接口
   - 可以无缝传递给策略模块

---

## 🎯 接下来应该做什么

根据实施优先级，建议按以下顺序进行：

### 第一阶段：实现MACD+EMA策略（当前重点）

#### 步骤1：创建策略适配器
**目标：** 将 dataManager/v2 的数据流连接到策略模块

**需要创建：** `internal/strategy/adapter.go`

**功能：**
```go
// StrategyAdapter 将 v2.KlineSubscriber 适配到 Strategy
type StrategyAdapter struct {
    strategy core.Strategy
    symbol   string
    interval string
}

func (a *StrategyAdapter) OnKline(kline *v2.KlineData) {
    signal, err := a.strategy.OnKline(kline)
    // 将信号发送到下游
}
```

#### 步骤2：实现MACD+EMA策略
**需要创建：** `internal/strategy/macd_ema_strategy.go`

**参考：** `testStrategy.md` 中的策略规则

**核心逻辑：**
- 维护MACD(12,26,9), EMA(5,15), VWAP(8)
- 检测交叉信号
- 检测连续趋势
- 生成开仓/加仓/平仓信号

**建议实现结构：**
```go
type MACDEMAStrategy struct {
    // 指标
    macd    *MACD
    ema5    *EMA
    ema15   *EMA
    vwap8   *VWAP
    
    // 历史数据缓冲
    klineBuffer *KlineBuffer
    
    // 交叉历史
    crossHistory []CrossEvent
    
    // 配置参数
    config MACDEMAConfig
}
```

#### 步骤3：编写策略测试
**需要创建：** `internal/strategy/macd_ema_strategy_test.go`

**测试用例：**
- 预热测试（Warmup）
- 金叉/死叉检测测试
- 趋势检测测试
- 完整信号生成测试

---

### 第二阶段：实现仓位管理模块

#### 步骤4：创建仓位管理器
**需要创建：**
- `internal/position/manager.go` - 仓位管理器主体
- `internal/position/models.go` - 仓位数据模型
- `internal/position/risk.go` - 风险控制
- `internal/position/sizing.go` - 仓位计算

**核心功能：**
```go
type Manager struct {
    config     *config.PositionConfig
    executor   core.Executor
    positions  map[string]*core.Position
    
    riskManager   *RiskManager
    sizeCalculator *SizeCalculator
}

func (m *Manager) ProcessSignal(signal *core.TradingSignal, currentPrice float64) ([]*core.Order, error) {
    // 1. 风险检查
    // 2. 计算仓位大小
    // 3. 生成订单
    // 4. 验证订单合法性
}
```

**参考：** `testStrategy.md` 中的资金管理规则
- 首次开仓20%资金
- 加仓40%资金
- 5倍杠杆
- 逐仓模式

---

### 第三阶段：实现回测执行器

#### 步骤5：创建回测执行器
**需要创建：**
- `internal/execution/backtest/executor.go`
- `internal/execution/backtest/account.go`
- `internal/execution/backtest/order_matcher.go`
- `internal/execution/backtest/fee_calculator.go`

**核心功能：**
```go
type BacktestExecutor struct {
    account      *Account
    positions    map[string]*core.Position
    openOrders   map[string]*core.Order
    orderHistory []*core.Order
    
    feeConfig    config.FeesConfig
    slippageConfig config.SlippageConfig
}

func (e *BacktestExecutor) PlaceOrder(ctx context.Context, order *core.Order) (*core.Order, error) {
    // 1. 验证订单
    // 2. 检查余额
    // 3. 计算手续费和滑点
    // 4. 模拟成交
    // 5. 更新账户和持仓
}
```

#### 步骤6：实现订单撮合逻辑
**市价单：** K线收盘价成交  
**限价单：** 挂单，价格触及时成交  
**止损/止盈：** 监控价格触发

---

### 第四阶段：端到端集成测试

#### 步骤7：创建回测主程序
**需要创建：** `cmd/backtest/main.go`

**流程：**
```
1. 加载配置
2. 初始化数据提供者（从数据库读取历史K线）
3. 初始化回测执行器
4. 初始化仓位管理器
5. 初始化策略
6. 启动回测循环
7. 输出回测报告
```

#### 步骤8：验证完整数据流
```
历史K线数据 
  → DataProvider 
  → Strategy (MACD+EMA) 
  → TradingSignal 
  → PositionManager 
  → Orders 
  → BacktestExecutor 
  → 更新仓位/账户
  → 循环
```

---

## 📁 建议的开发顺序

### 本周任务清单

- [ ] **Day 1-2**: 实现 `adapter.go` 和 `macd_ema_strategy.go`
- [ ] **Day 3**: 编写策略单元测试
- [ ] **Day 4-5**: 实现仓位管理模块
- [ ] **Day 6-7**: 实现回测执行器基础版
- [ ] **Day 8**: 端到端集成测试

### 下周任务清单

- [ ] 完善回测执行器（限价单、止损止盈）
- [ ] 实现基础日志系统
- [ ] 实现回测报告生成
- [ ] 性能优化

---

## 💡 开发建议

### 1. 测试驱动开发（TDD）
- 先写测试用例，再实现功能
- 每个模块都要有单元测试
- 关键路径要有集成测试

### 2. 增量开发
- 先实现最简单的版本（市价单 + 固定仓位）
- 再逐步添加复杂功能（限价单、止损止盈、动态仓位）

### 3. 文档同步
- 每完成一个模块，更新 `IMPLEMENTATION_PROGRESS.md`
- 添加godoc注释
- 更新README示例

### 4. 代码审查检查点
- 接口是否符合设计？
- 错误处理是否完善？
- 是否有race condition？
- 性能是否可接受？

---

## 🔍 当前可以运行的测试

```bash
# 测试数据模块
cd /home/maeda/Documents/projects/goQuant
go test ./internal/dataManager/v2/... -v

# 验证编译
go build ./cmd/bot/
```

---

## 📚 参考资料位置

- **策略规则:** `testStrategy.md`
- **架构设计:** `plansAndProgressV1.md`
- **实施进度:** `IMPLEMENTATION_PROGRESS.md`
- **数据模块文档:** `internal/dataManager/README.md`
- **配置示例:** `config/config.example.yaml`

---

## 🚀 快速启动下一步

**立即执行：**
```bash
# 1. 创建策略实现文件
touch internal/strategy/adapter.go
touch internal/strategy/macd_ema_strategy.go
touch internal/strategy/macd_ema_strategy_test.go

# 2. 打开编辑器开始实现
```

**从哪里开始：**
建议先实现 `macd_ema_strategy.go`，因为：
1. 所有依赖（指标库）已准备好
2. 接口定义清晰
3. 有明确的策略规则可参考
4. 可以独立测试，无需等待其他模块

---

**准备好开始了吗？回复"开始实现策略"，我将帮你创建MACD+EMA策略的完整实现！**

