# 代码注释补充计划

本文档列出所有需要补充完整注释的代码文件。

---

## 已完成

### ✅ Core 模块
- `internal/core/interfaces.go` - 接口定义完整，包含详细注释
- `internal/core/doc.go` - 包文档完整

---

## 需要补充的模块

### 1. Config 配置模块

**文件**: `internal/config/config.go`

需要补充的函数注释：
- [x] `LoadConfig(configPath string) (*Config, error)`
- [ ] 所有Config结构体字段的注释

### 2. DataManager 数据管理模块

#### 2.1 V2版本（当前使用）

**文件**: `internal/dataManager/v2/enhanced_multi_processor.go`
- [ ] `NewEnhancedMultiProcessor(...) (*EnhancedMultiProcessor, error)`
- [ ] `Start(ctx context.Context) error`
- [ ] `Subscribe(...) error`
- [ ] `GetHistoricalKlines(...) ([]core.KlineData, error)`
- [ ] `Stop() error`

**文件**: `internal/dataManager/v2/klinestore.go`
- [ ] `NewKlineStore(dbPath string) (*KlineStore, error)`
- [ ] `SaveKline(kline *KlineData) error`
- [ ] `GetKlines(...) ([]*KlineData, error)`
- [ ] `GetLatestKline() (*KlineData, error)`
- [ ] `Close() error`

**文件**: `internal/dataManager/v2/connection_manager.go`
- [ ] 所有公开方法

**文件**: `internal/dataManager/v2/message_dispatcher.go`
- [ ] 所有公开方法

**文件**: `internal/dataManager/v2/completion_checker.go`
- [ ] 所有公开方法

### 3. Strategy 策略模块

**文件**: `internal/strategy/macd_ema_strategy.go`
- [ ] `NewMACDEMAStrategy(symbol, interval string) *MACDEMAStrategy`
- [ ] `OnKline(kline core.KlineData) (*core.TradingSignal, error)`
- [ ] `Warmup(historicalKlines []core.KlineData) error`
- [ ] `GetRequiredWarmupPeriods() int`
- [ ] `Reset() error`
- [ ] `checkScenario1(...) *core.TradingSignal`
- [ ] `checkScenario2(...) *core.TradingSignal`

**文件**: `internal/strategy/indicators.go`
- [ ] `NewMACD(fast, slow, signal int) *MACD`
- [ ] `(m *MACD) Update(price float64)`
- [ ] `(m *MACD) Values() (dif, dea, macd float64)`
- [ ] `NewEMA(period int) *EMA`
- [ ] `(e *EMA) Update(price float64) float64`
- [ ] `NewVWAP(period int) *VWAP`
- [ ] `(v *VWAP) Update(price, volume float64) float64`
- [ ] `DetectCross(...) CrossType`
- [ ] `DetectTrend(...) (TrendType, float64)`

**文件**: `internal/strategy/adapter.go`
- [ ] `NewAdapter(...) *Adapter`
- [ ] `OnKline(kline *v2.KlineData)`
- [ ] `OnError(err error)`
- [ ] `Name() string`

**文件**: `internal/strategy/signal.go`
- [ ] `NewSignal(...) *core.TradingSignal`
- [ ] `NoActionSignal(symbol string) *core.TradingSignal`

### 4. Position 仓位管理模块

**文件**: `internal/position/manager.go`
- [ ] `NewManager(cfg *config.PositionConfig, executor core.Executor) *Manager`
- [ ] `ProcessSignal(...) ([]*core.Order, error)`
- [ ] `UpdatePosition(position *core.Position) error`
- [ ] `GetPosition(symbol string) (*core.Position, error)`
- [ ] `GetAllPositions() ([]*core.Position, error)`
- [ ] `CheckRisk(order *core.Order) error`
- [ ] `CalculatePositionSize(...) (float64, error)`
- [ ] `CheckStopLoss(...) (bool, *core.Order)`
- [ ] `CheckTrailingStop(...) (bool, *core.Order)`
- [ ] 所有私有辅助方法

### 5. Execution 执行模块

**文件**: `internal/execution/binance/executor.go`
- [ ] `NewLiveExecutor(...) *LiveExecutor`
- [ ] `PlaceOrder(...) (*core.Order, error)`
- [ ] `CancelOrder(...) error`
- [ ] `GetOrder(...) (*core.Order, error)`
- [ ] `GetOpenOrders(...) ([]*core.Order, error)`
- [ ] `GetAccount(...) (*core.Account, error)`
- [ ] `GetPositions(...) ([]*core.Position, error)`
- [ ] `SetLeverage(...) error`
- [ ] `SetMarginMode(...) error`
- [ ] `Close() error`

**文件**: `internal/execution/binance/client.go`
- [ ] `NewClient(apiKey, secretKey, baseURL string) *Client`
- [ ] `CreateOrder(...) (*OrderResponse, error)`
- [ ] `CancelOrder(...) error`
- [ ] `GetOrder(...) (*OrderResponse, error)`
- [ ] `GetOpenOrders(...) ([]*OrderResponse, error)`
- [ ] `GetAccount(...) (*AccountInfo, error)`
- [ ] `GetPositionRisk(...) ([]*PositionRisk, error)`
- [ ] `SetLeverage(...) error`
- [ ] `SetMarginType(...) error`
- [ ] 私有方法 `sign(params string) string`
- [ ] 私有方法 `doRequest(...) ([]byte, error)`

**文件**: `internal/execution/binance/models.go`
- [ ] 所有结构体字段注释

**文件**: `internal/execution/binance/utils.go`
- [ ] 所有转换函数

### 6. Logger 日志模块

**文件**: `internal/logger/logger.go`
- [ ] `Init(logLevel, logFile string) error`
- [ ] `Debug(msg string, fields ...zap.Field)`
- [ ] `Info(msg string, fields ...zap.Field)`
- [ ] `Warn(msg string, fields ...zap.Field)`
- [ ] `Error(msg string, fields ...zap.Field)`
- [ ] `Fatal(msg string, fields ...zap.Field)`
- [ ] `Sync() error`

**文件**: `internal/logger/symbol_logger.go`
- [ ] `GetSymbolLogger() *SymbolLogger`
- [ ] `InitSession(sessionDir string) error`
- [ ] `GetLogger(symbol, interval string) *zap.Logger`
- [ ] `Close() error`

**文件**: `internal/logger/trading.go`
- [ ] `NewTradingLogger() *TradingLogger`
- [ ] `LogKline(...)`
- [ ] `LogSignal(...)`
- [ ] `LogOrder(...)`
- [ ] `LogPosition(...)`

---

## 注释格式规范

### 函数注释模板

```go
// FunctionName 函数简要说明
// 
// 参数:
//   paramName: type - 参数说明
//   paramName2: type - 参数说明
//
// 返回:
//   returnType - 返回值说明
//   error - 错误说明（如果有）
//
// 示例:
//   result, err := FunctionName("example")
//   if err != nil {
//       // 处理错误
//   }
func FunctionName(param string) (Result, error) {
    // 实现
}
```

### 结构体注释模板

```go
// StructName 结构体简要说明
//
// 详细说明（如果需要）
type StructName struct {
    // FieldName 字段说明
    FieldName string
    
    // AnotherField 另一个字段的说明
    AnotherField int
}
```

### 常量注释模板

```go
// ConstantType 常量类型说明
type ConstantType string

const (
    // ConstantValue 常量值说明
    ConstantValue ConstantType = "VALUE"
    
    // AnotherValue 另一个常量值说明
    AnotherValue ConstantType = "ANOTHER"
)
```

---

## 补充策略

由于代码文件较多，建议分批补充：

### 优先级 1（核心模块）
1. Strategy 策略模块
2. Position 仓位管理模块
3. Execution 执行模块

### 优先级 2（基础设施）
4. Logger 日志模块
5. Config 配置模块

### 优先级 3（数据层）
6. DataManager V2 模块

---

## 自动化工具

可以编写脚本检查缺失的注释：

```bash
#!/bin/bash
# 检查缺少注释的公开函数

find internal -name "*.go" | while read file; do
    echo "检查: $file"
    # 查找没有注释的公开函数
    grep -B 1 "^func [A-Z]" "$file" | grep -v "//" | grep "^func"
done
```

---

**创建日期**: 2025-12-05  
**状态**: 计划中

