# goQuant API 文档

本文档包含所有模块的函数签名、参数说明和返回值说明。

---

## 目录

- [1. Core 核心接口](#1-core-核心接口)
- [2. Config 配置模块](#2-config-配置模块)
- [3. DataManager 数据管理模块](#3-datamanager-数据管理模块)
- [4. Strategy 策略模块](#4-strategy-策略模块)
- [5. Position 仓位管理模块](#5-position-仓位管理模块)
- [6. Execution 执行模块](#6-execution-执行模块)
- [7. Logger 日志模块](#7-logger-日志模块)

---

## 1. Core 核心接口

定义位置：`internal/core/interfaces.go`

### 1.1 KlineData 接口

K线数据的标准接口，所有K线实现必须满足此接口。

```go
type KlineData interface {
    // GetSymbol 获取交易对名称
    // 返回: string - 交易对符号，如 "BTCUSDT"
    GetSymbol() string
    
    // GetInterval 获取K线周期
    // 返回: string - 时间周期，如 "1m", "5m", "1h"
    GetInterval() string
    
    // GetStartTime 获取K线开始时间
    // 返回: time.Time - K线开盘时间
    GetStartTime() time.Time
    
    // GetCloseTime 获取K线结束时间
    // 返回: time.Time - K线收盘时间
    GetCloseTime() time.Time
    
    // GetOpenPrice 获取开盘价
    // 返回: float64 - 开盘价格
    GetOpenPrice() float64
    
    // GetClosePrice 获取收盘价
    // 返回: float64 - 收盘价格
    GetClosePrice() float64
    
    // GetHighPrice 获取最高价
    // 返回: float64 - 周期内最高价格
    GetHighPrice() float64
    
    // GetLowPrice 获取最低价
    // 返回: float64 - 周期内最低价格
    GetLowPrice() float64
    
    // GetVolume 获取成交量
    // 返回: float64 - 成交量（基础货币）
    GetVolume() float64
    
    // IsClosed 判断K线是否已完成
    // 返回: bool - true 表示K线已收盘，false 表示仍在进行中
    IsClosed() bool
}
```

### 1.2 DataProvider 接口

数据提供者接口，用于订阅和获取市场数据。

```go
type DataProvider interface {
    // SubscribeKline 订阅K线数据流
    // 参数:
    //   ctx: context.Context - 上下文，用于控制取消和超时
    //   symbol: string - 交易对，如 "BTCUSDT"
    //   interval: string - K线周期，如 "1m", "3m", "1h"
    // 返回:
    //   <-chan KlineData - K线数据通道，持续接收K线更新
    //   error - 订阅失败时的错误信息
    SubscribeKline(ctx context.Context, symbol, interval string) (<-chan KlineData, error)
    
    // SubscribeOrderBook 订阅订单簿数据流
    // 参数:
    //   ctx: context.Context - 上下文
    //   symbol: string - 交易对
    //   levels: int - 订单簿深度，如 5, 10, 20
    // 返回:
    //   <-chan OrderBookData - 订单簿数据通道
    //   error - 订阅失败时的错误信息
    SubscribeOrderBook(ctx context.Context, symbol string, levels int) (<-chan OrderBookData, error)
    
    // GetHistoricalKlines 获取历史K线数据（用于策略预热）
    // 参数:
    //   symbol: string - 交易对
    //   interval: string - K线周期
    //   limit: int - 获取数量，如 100 表示最近100根K线
    // 返回:
    //   []KlineData - K线数据切片，按时间升序排列
    //   error - 获取失败时的错误信息
    GetHistoricalKlines(symbol, interval string, limit int) ([]KlineData, error)
    
    // Close 关闭数据提供者，释放资源
    // 返回:
    //   error - 关闭时的错误信息
    Close() error
}
```

### 1.3 Strategy 接口

策略接口，所有交易策略必须实现此接口。

```go
type Strategy interface {
    // Name 返回策略名称
    // 返回: string - 策略标识符，如 "MACD_EMA_Strategy"
    Name() string
    
    // OnKline 处理K线数据并生成交易信号
    // 参数:
    //   kline: KlineData - K线数据
    // 返回:
    //   *TradingSignal - 生成的交易信号，如果无操作则Type为NO_ACTION
    //   error - 处理过程中的错误
    OnKline(kline KlineData) (*TradingSignal, error)
    
    // OnOrderBook 处理订单簿数据（可选，某些策略可能不使用）
    // 参数:
    //   orderBook: OrderBookData - 订单簿数据
    // 返回:
    //   error - 处理过程中的错误
    OnOrderBook(orderBook OrderBookData) error
    
    // Warmup 策略预热，使用历史数据初始化指标
    // 参数:
    //   historicalKlines: []KlineData - 历史K线数据切片
    // 返回:
    //   error - 预热失败时的错误信息
    Warmup(historicalKlines []KlineData) error
    
    // GetRequiredWarmupPeriods 返回策略预热所需的K线数量
    // 返回: int - 需要的K线根数，如 45（MACD需要26+9+缓冲）
    GetRequiredWarmupPeriods() int
    
    // Reset 重置策略状态，清空所有指标和缓存
    // 返回: error - 重置失败时的错误信息
    Reset() error
}
```

### 1.4 PositionManager 接口

仓位管理接口，处理信号转订单、风险控制等。

```go
type PositionManager interface {
    // ProcessSignal 处理策略信号，决定是否生成订单
    // 参数:
    //   signal: *TradingSignal - 策略生成的交易信号
    //   currentPrice: float64 - 当前市场价格
    // 返回:
    //   []*Order - 要执行的订单列表（可能包含平仓+开仓）
    //   error - 处理失败时的错误信息
    ProcessSignal(signal *TradingSignal, currentPrice float64) ([]*Order, error)
    
    // UpdatePosition 更新持仓信息
    // 参数:
    //   position: *Position - 最新的持仓数据
    // 返回:
    //   error - 更新失败时的错误信息
    UpdatePosition(position *Position) error
    
    // GetPosition 获取指定交易对的当前持仓
    // 参数:
    //   symbol: string - 交易对符号
    // 返回:
    //   *Position - 持仓信息，如果无持仓则返回 nil
    //   error - 查询失败时的错误信息
    GetPosition(symbol string) (*Position, error)
    
    // GetAllPositions 获取所有持仓
    // 返回:
    //   []*Position - 所有持仓列表
    //   error - 查询失败时的错误信息
    GetAllPositions() ([]*Position, error)
    
    // CheckRisk 风险检查，验证订单是否符合风控规则
    // 参数:
    //   order: *Order - 待检查的订单
    // 返回:
    //   error - 风险检查失败时的错误信息（如超过最大持仓数）
    CheckRisk(order *Order) error
    
    // CalculatePositionSize 计算仓位大小
    // 参数:
    //   signal: *TradingSignal - 交易信号
    //   accountBalance: float64 - 账户可用余额（USDT）
    // 返回:
    //   float64 - 计算得到的USDT金额
    //   error - 计算失败时的错误信息
    CalculatePositionSize(signal *TradingSignal, accountBalance float64) (float64, error)
}
```

### 1.5 Executor 接口

执行器接口，用于下单、查询订单和持仓等操作。

```go
type Executor interface {
    // PlaceOrder 提交订单到交易所
    // 参数:
    //   ctx: context.Context - 上下文
    //   order: *Order - 订单详情
    // 返回:
    //   *Order - 更新后的订单（包含交易所返回的ID和状态）
    //   error - 下单失败时的错误信息
    PlaceOrder(ctx context.Context, order *Order) (*Order, error)
    
    // CancelOrder 取消订单
    // 参数:
    //   ctx: context.Context - 上下文
    //   symbol: string - 交易对
    //   orderID: string - 订单ID
    // 返回:
    //   error - 取消失败时的错误信息
    CancelOrder(ctx context.Context, symbol, orderID string) error
    
    // GetOrder 查询订单详情
    // 参数:
    //   ctx: context.Context - 上下文
    //   symbol: string - 交易对
    //   orderID: string - 订单ID
    // 返回:
    //   *Order - 订单详情
    //   error - 查询失败时的错误信息
    GetOrder(ctx context.Context, symbol, orderID string) (*Order, error)
    
    // GetOpenOrders 获取所有未成交订单
    // 参数:
    //   ctx: context.Context - 上下文
    //   symbol: string - 交易对，空字符串表示查询所有
    // 返回:
    //   []*Order - 未成交订单列表
    //   error - 查询失败时的错误信息
    GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)
    
    // GetAccount 获取账户信息
    // 参数:
    //   ctx: context.Context - 上下文
    // 返回:
    //   *Account - 账户余额和保证金信息
    //   error - 查询失败时的错误信息
    GetAccount(ctx context.Context) (*Account, error)
    
    // GetPositions 获取所有持仓
    // 参数:
    //   ctx: context.Context - 上下文
    // 返回:
    //   []*Position - 持仓列表
    //   error - 查询失败时的错误信息
    GetPositions(ctx context.Context) ([]*Position, error)
    
    // SetLeverage 设置杠杆倍数
    // 参数:
    //   ctx: context.Context - 上下文
    //   symbol: string - 交易对
    //   leverage: int - 杠杆倍数（1-125）
    // 返回:
    //   error - 设置失败时的错误信息
    SetLeverage(ctx context.Context, symbol string, leverage int) error
    
    // SetMarginMode 设置保证金模式
    // 参数:
    //   ctx: context.Context - 上下文
    //   symbol: string - 交易对
    //   mode: MarginMode - 保证金模式（ISOLATED 或 CROSS）
    // 返回:
    //   error - 设置失败时的错误信息
    SetMarginMode(ctx context.Context, symbol string, mode MarginMode) error
    
    // Close 关闭执行器，释放资源
    // 返回:
    //   error - 关闭失败时的错误信息
    Close() error
}
```

---

## 2. Config 配置模块

定义位置：`internal/config/config.go`

### 2.1 LoadConfig

```go
// LoadConfig 从YAML文件加载配置
// 参数:
//   configPath: string - 配置文件路径，如 "config/config.yaml"
// 返回:
//   *Config - 解析后的配置对象
//   error - 加载或解析失败时的错误信息
func LoadConfig(configPath string) (*Config, error)
```

### 2.2 Config 结构体

完整的配置结构体，包含所有模块的配置选项。

```go
type Config struct {
    App          AppConfig          // 应用程序配置
    Data         DataConfig         // 数据模块配置
    Strategy     StrategyConfig     // 策略配置
    Position     PositionConfig     // 仓位管理配置
    Execution    ExecutionConfig    // 执行模块配置
    Observability ObservabilityConfig // 日志和监控配置
}
```

---

## 3. DataManager 数据管理模块

定义位置：`internal/dataManager/v2/`

### 3.1 EnhancedMultiProcessor

增强的多交易对数据处理器，支持WebSocket订阅和本地存储。

```go
// NewEnhancedMultiProcessor 创建增强的多交易对处理器
// 参数:
//   config: *SubscriptionConfig - 订阅配置（包含交易对、周期等）
//   proxyURL: string - 代理URL，空字符串表示不使用代理
//   dbDir: string - 数据库存储目录
// 返回:
//   *EnhancedMultiProcessor - 处理器实例
//   error - 创建失败时的错误信息
func NewEnhancedMultiProcessor(config *SubscriptionConfig, proxyURL, dbDir string) (*EnhancedMultiProcessor, error)

// Start 启动数据处理器
// 参数:
//   ctx: context.Context - 上下文，用于控制生命周期
// 返回:
//   error - 启动失败时的错误信息
func (emp *EnhancedMultiProcessor) Start(ctx context.Context) error

// Subscribe 订阅K线数据（实现DataProvider接口）
// 参数:
//   subscriber: KlineSubscriber - 订阅者，实现OnKline和OnError方法
//   symbol: string - 交易对
//   interval: string - K线周期
// 返回:
//   error - 订阅失败时的错误信息
func (emp *EnhancedMultiProcessor) Subscribe(subscriber KlineSubscriber, symbol, interval string) error

// GetHistoricalKlines 获取历史K线（用于策略预热）
// 参数:
//   symbol: string - 交易对
//   interval: string - K线周期
//   limit: int - 获取数量
// 返回:
//   []core.KlineData - K线数据切片
//   error - 获取失败时的错误信息
func (emp *EnhancedMultiProcessor) GetHistoricalKlines(symbol, interval string, limit int) ([]core.KlineData, error)

// Stop 停止数据处理器
// 返回:
//   error - 停止失败时的错误信息
func (emp *EnhancedMultiProcessor) Stop() error
```

### 3.2 KlineStore

K线数据本地存储。

```go
// NewKlineStore 创建K线存储实例
// 参数:
//   dbPath: string - 数据库文件路径
// 返回:
//   *KlineStore - 存储实例
//   error - 创建失败时的错误信息
func NewKlineStore(dbPath string) (*KlineStore, error)

// SaveKline 保存K线数据
// 参数:
//   kline: *KlineData - K线数据
// 返回:
//   error - 保存失败时的错误信息
func (ks *KlineStore) SaveKline(kline *KlineData) error

// GetKlines 查询K线数据
// 参数:
//   startTime: time.Time - 开始时间
//   endTime: time.Time - 结束时间
//   limit: int - 最大返回数量，0表示不限制
// 返回:
//   []*KlineData - K线数据切片
//   error - 查询失败时的错误信息
func (ks *KlineStore) GetKlines(startTime, endTime time.Time, limit int) ([]*KlineData, error)

// GetLatestKline 获取最新的K线
// 返回:
//   *KlineData - 最新K线，如果数据库为空则返回nil
//   error - 查询失败时的错误信息
func (ks *KlineStore) GetLatestKline() (*KlineData, error)

// Close 关闭数据库连接
// 返回:
//   error - 关闭失败时的错误信息
func (ks *KlineStore) Close() error
```

---

## 4. Strategy 策略模块

定义位置：`internal/strategy/`

### 4.1 MACDEMAStrategy

MACD+EMA+VWAP组合策略。

```go
// NewMACDEMAStrategy 创建策略实例
// 参数:
//   symbol: string - 交易对
//   interval: string - K线周期
// 返回:
//   *MACDEMAStrategy - 策略实例
func NewMACDEMAStrategy(symbol, interval string) *MACDEMAStrategy

// OnKline 处理K线数据（实现Strategy接口）
// 参数:
//   kline: core.KlineData - K线数据
// 返回:
//   *core.TradingSignal - 交易信号
//   error - 处理失败时的错误信息
func (s *MACDEMAStrategy) OnKline(kline core.KlineData) (*core.TradingSignal, error)

// Warmup 策略预热（实现Strategy接口）
// 参数:
//   historicalKlines: []core.KlineData - 历史K线数据
// 返回:
//   error - 预热失败时的错误信息
func (s *MACDEMAStrategy) Warmup(historicalKlines []core.KlineData) error

// GetRequiredWarmupPeriods 返回预热所需K线数量（实现Strategy接口）
// 返回: int - 需要45根K线（MACD 26+9+缓冲10）
func (s *MACDEMAStrategy) GetRequiredWarmupPeriods() int

// Reset 重置策略状态（实现Strategy接口）
// 返回: error - 重置失败时的错误信息
func (s *MACDEMAStrategy) Reset() error
```

### 4.2 技术指标

```go
// NewMACD 创建MACD指标
// 参数:
//   fastPeriod: int - 快线周期，如 12
//   slowPeriod: int - 慢线周期，如 26
//   signalPeriod: int - 信号线周期，如 9
// 返回:
//   *MACD - MACD指标实例
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD

// Update 更新MACD指标
// 参数:
//   price: float64 - 最新价格
func (m *MACD) Update(price float64)

// Values 获取MACD值
// 返回:
//   dif: float64 - DIF值（快线-慢线）
//   dea: float64 - DEA值（信号线）
//   macd: float64 - MACD柱状图（DIF-DEA）
func (m *MACD) Values() (dif, dea, macd float64)

// NewEMA 创建EMA指标
// 参数:
//   period: int - 周期，如 5, 15
// 返回:
//   *EMA - EMA指标实例
func NewEMA(period int) *EMA

// Update 更新EMA指标
// 参数:
//   price: float64 - 最新价格
// 返回:
//   float64 - 当前EMA值
func (e *EMA) Update(price float64) float64

// NewVWAP 创建VWAP指标
// 参数:
//   period: int - 周期，如 8
// 返回:
//   *VWAP - VWAP指标实例
func NewVWAP(period int) *VWAP

// Update 更新VWAP指标
// 参数:
//   price: float64 - 最新价格
//   volume: float64 - 成交量
// 返回:
//   float64 - 当前VWAP值
func (v *VWAP) Update(price, volume float64) float64
```

### 4.3 Adapter

策略适配器，连接数据流和策略。

```go
// NewAdapter 创建策略适配器
// 参数:
//   strategy: core.Strategy - 策略实例
//   positionMgr: core.PositionManager - 仓位管理器
//   executor: core.Executor - 执行器
//   symbol: string - 交易对
//   interval: string - K线周期
// 返回:
//   *Adapter - 适配器实例
func NewAdapter(strategy core.Strategy, positionMgr core.PositionManager, executor core.Executor, symbol, interval string) *Adapter

// OnKline 处理K线数据（实现KlineSubscriber接口）
// 参数:
//   kline: *v2.KlineData - K线数据
func (a *Adapter) OnKline(kline *v2.KlineData)

// OnError 处理错误（实现KlineSubscriber接口）
// 参数:
//   err: error - 错误信息
func (a *Adapter) OnError(err error)

// Name 返回适配器名称（实现KlineSubscriber接口）
// 返回: string - 适配器标识符
func (a *Adapter) Name() string
```

---

## 5. Position 仓位管理模块

定义位置：`internal/position/manager.go`

### 5.1 Manager

仓位管理器实现。

```go
// NewManager 创建仓位管理器
// 参数:
//   cfg: *config.PositionConfig - 仓位配置
//   executor: core.Executor - 执行器
// 返回:
//   *Manager - 仓位管理器实例
func NewManager(cfg *config.PositionConfig, executor core.Executor) *Manager

// ProcessSignal 处理策略信号（实现PositionManager接口）
// 参数:
//   signal: *core.TradingSignal - 策略信号
//   currentPrice: float64 - 当前市场价格
// 返回:
//   []*core.Order - 订单列表
//   error - 处理失败时的错误信息
func (m *Manager) ProcessSignal(signal *core.TradingSignal, currentPrice float64) ([]*core.Order, error)

// UpdatePosition 更新持仓信息（实现PositionManager接口）
// 参数:
//   position: *core.Position - 持仓数据
// 返回:
//   error - 更新失败时的错误信息
func (m *Manager) UpdatePosition(position *core.Position) error

// GetPosition 获取持仓（实现PositionManager接口）
// 参数:
//   symbol: string - 交易对
// 返回:
//   *core.Position - 持仓信息，无持仓时返回nil
//   error - 查询失败时的错误信息
func (m *Manager) GetPosition(symbol string) (*core.Position, error)

// GetAllPositions 获取所有持仓（实现PositionManager接口）
// 返回:
//   []*core.Position - 持仓列表
//   error - 查询失败时的错误信息
func (m *Manager) GetAllPositions() ([]*core.Position, error)

// CheckRisk 风险检查（实现PositionManager接口）
// 参数:
//   order: *core.Order - 待检查的订单
// 返回:
//   error - 风险检查失败时的错误信息
func (m *Manager) CheckRisk(order *core.Order) error

// CalculatePositionSize 计算仓位大小（实现PositionManager接口）
// 参数:
//   signal: *core.TradingSignal - 交易信号
//   accountBalance: float64 - 账户余额（USDT）
// 返回:
//   float64 - 使用的USDT金额
//   error - 计算失败时的错误信息
func (m *Manager) CalculatePositionSize(signal *core.TradingSignal, accountBalance float64) (float64, error)

// CheckStopLoss 检查止损
// 参数:
//   symbol: string - 交易对
//   currentPrice: float64 - 当前价格
// 返回:
//   bool - 是否触发止损
//   *core.Order - 止损平仓订单（如果触发）
func (m *Manager) CheckStopLoss(symbol string, currentPrice float64) (bool, *core.Order)

// CheckTrailingStop 检查跟踪止盈
// 参数:
//   symbol: string - 交易对
//   currentPrice: float64 - 当前价格
// 返回:
//   bool - 是否触发止盈
//   *core.Order - 止盈平仓订单（如果触发）
func (m *Manager) CheckTrailingStop(symbol string, currentPrice float64) (bool, *core.Order)
```

---

## 6. Execution 执行模块

定义位置：`internal/execution/binance/`

### 6.1 LiveExecutor

币安期货实盘执行器。

```go
// NewLiveExecutor 创建实盘执行器
// 参数:
//   apiKey: string - Binance API Key
//   secretKey: string - Binance Secret Key
//   baseURL: string - API基础URL（主网或测试网）
// 返回:
//   *LiveExecutor - 执行器实例
func NewLiveExecutor(apiKey, secretKey, baseURL string) *LiveExecutor

// PlaceOrder 下单（实现Executor接口）
// 参数:
//   ctx: context.Context - 上下文
//   order: *core.Order - 订单详情
// 返回:
//   *core.Order - 更新后的订单
//   error - 下单失败时的错误信息
func (e *LiveExecutor) PlaceOrder(ctx context.Context, order *core.Order) (*core.Order, error)

// GetAccount 获取账户信息（实现Executor接口）
// 参数:
//   ctx: context.Context - 上下文
// 返回:
//   *core.Account - 账户信息
//   error - 查询失败时的错误信息
func (e *LiveExecutor) GetAccount(ctx context.Context) (*core.Account, error)

// GetPositions 获取持仓（实现Executor接口）
// 参数:
//   ctx: context.Context - 上下文
// 返回:
//   []*core.Position - 持仓列表
//   error - 查询失败时的错误信息
func (e *LiveExecutor) GetPositions(ctx context.Context) ([]*core.Position, error)
```

### 6.2 Client

币安API客户端。

```go
// NewClient 创建币安期货客户端
// 参数:
//   apiKey: string - API Key
//   secretKey: string - Secret Key
//   baseURL: string - API基础URL
// 返回:
//   *Client - 客户端实例
func NewClient(apiKey, secretKey, baseURL string) *Client

// CreateOrder 创建订单
// 参数:
//   ctx: context.Context - 上下文
//   req: *CreateOrderRequest - 订单请求
// 返回:
//   *OrderResponse - 订单响应
//   error - 创建失败时的错误信息
func (c *Client) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderResponse, error)

// GetAccount 查询账户信息
// 参数:
//   ctx: context.Context - 上下文
// 返回:
//   *AccountInfo - 账户信息
//   error - 查询失败时的错误信息
func (c *Client) GetAccount(ctx context.Context) (*AccountInfo, error)

// GetPositionRisk 查询持仓信息
// 参数:
//   ctx: context.Context - 上下文
//   symbol: string - 交易对，空字符串表示查询所有
// 返回:
//   []*PositionRisk - 持仓列表
//   error - 查询失败时的错误信息
func (c *Client) GetPositionRisk(ctx context.Context, symbol string) ([]*PositionRisk, error)

// SetLeverage 设置杠杆倍数
// 参数:
//   ctx: context.Context - 上下文
//   symbol: string - 交易对
//   leverage: int - 杠杆倍数（1-125）
// 返回:
//   error - 设置失败时的错误信息
func (c *Client) SetLeverage(ctx context.Context, symbol string, leverage int) error

// SetMarginType 设置保证金模式
// 参数:
//   ctx: context.Context - 上下文
//   symbol: string - 交易对
//   marginType: MarginType - 保证金模式（ISOLATED或CROSSED）
// 返回:
//   error - 设置失败时的错误信息
func (c *Client) SetMarginType(ctx context.Context, symbol string, marginType MarginType) error
```

---

## 7. Logger 日志模块

定义位置：`internal/logger/`

### 7.1 全局日志函数

```go
// Init 初始化全局日志系统
// 参数:
//   logLevel: string - 日志级别（"debug", "info", "warn", "error"）
//   logFile: string - 日志文件路径，空字符串表示仅输出到控制台
// 返回:
//   error - 初始化失败时的错误信息
func Init(logLevel, logFile string) error

// Debug 记录调试日志
// 参数:
//   msg: string - 日志消息
//   fields: ...zap.Field - 额外字段
func Debug(msg string, fields ...zap.Field)

// Info 记录信息日志
// 参数:
//   msg: string - 日志消息
//   fields: ...zap.Field - 额外字段
func Info(msg string, fields ...zap.Field)

// Warn 记录警告日志
// 参数:
//   msg: string - 日志消息
//   fields: ...zap.Field - 额外字段
func Warn(msg string, fields ...zap.Field)

// Error 记录错误日志
// 参数:
//   msg: string - 日志消息
//   fields: ...zap.Field - 额外字段
func Error(msg string, fields ...zap.Field)

// Fatal 记录致命错误并退出程序
// 参数:
//   msg: string - 日志消息
//   fields: ...zap.Field - 额外字段
func Fatal(msg string, fields ...zap.Field)
```

### 7.2 SymbolLogger

按交易对分文件的日志记录器。

```go
// GetSymbolLogger 获取全局SymbolLogger实例
// 返回: *SymbolLogger - 单例实例
func GetSymbolLogger() *SymbolLogger

// InitSession 初始化日志会话
// 参数:
//   sessionDir: string - 会话目录路径
// 返回:
//   error - 初始化失败时的错误信息
func (sl *SymbolLogger) InitSession(sessionDir string) error

// GetLogger 获取指定交易对的日志记录器
// 参数:
//   symbol: string - 交易对，如 "BTCUSDT"
//   interval: string - K线周期，如 "1m"
// 返回:
//   *zap.Logger - 专用日志记录器
func (sl *SymbolLogger) GetLogger(symbol, interval string) *zap.Logger

// Close 关闭所有日志文件
// 返回:
//   error - 关闭失败时的错误信息
func (sl *SymbolLogger) Close() error
```

### 7.3 TradingLogger

交易事件专用日志记录器。

```go
// NewTradingLogger 创建交易日志记录器
// 返回: *TradingLogger - 日志记录器实例
func NewTradingLogger() *TradingLogger

// LogKline 记录K线接收
// 参数:
//   symbol: string - 交易对
//   interval: string - 周期
//   kline: core.KlineData - K线数据
func (tl *TradingLogger) LogKline(symbol, interval string, kline core.KlineData)

// LogSignal 记录交易信号
// 参数:
//   symbol: string - 交易对
//   signal: *core.TradingSignal - 交易信号
func (tl *TradingLogger) LogSignal(symbol string, signal *core.TradingSignal)

// LogOrder 记录订单事件
// 参数:
//   order: *core.Order - 订单信息
func (tl *TradingLogger) LogOrder(order *core.Order)

// LogPosition 记录持仓更新
// 参数:
//   position: *core.Position - 持仓信息
func (tl *TradingLogger) LogPosition(position *core.Position)
```

---

## 附录：常见数据类型

### SignalType 信号类型

```go
const (
    SignalTypeOpenLong    = "OPEN_LONG"    // 开多仓
    SignalTypeOpenShort   = "OPEN_SHORT"   // 开空仓
    SignalTypeAddLong     = "ADD_LONG"     // 加多仓（已弃用）
    SignalTypeAddShort    = "ADD_SHORT"    // 加空仓（已弃用）
    SignalTypeCloseLong   = "CLOSE_LONG"   // 平多仓
    SignalTypeCloseShort  = "CLOSE_SHORT"  // 平空仓
    SignalTypeNoAction    = "NO_ACTION"    // 无操作
)
```

### OrderType 订单类型

```go
const (
    OrderTypeMarket      = "MARKET"       // 市价单
    OrderTypeLimit       = "LIMIT"        // 限价单
    OrderTypeStopMarket  = "STOP_MARKET"  // 市价止损单
    OrderTypeStopLimit   = "STOP_LIMIT"   // 限价止损单
    OrderTypeTakeProfit  = "TAKE_PROFIT"  // 止盈单
)
```

### OrderStatus 订单状态

```go
const (
    OrderStatusNew             = "NEW"              // 新建
    OrderStatusPartiallyFilled = "PARTIALLY_FILLED" // 部分成交
    OrderStatusFilled          = "FILLED"           // 完全成交
    OrderStatusCanceled        = "CANCELED"         // 已取消
    OrderStatusRejected        = "REJECTED"         // 被拒绝
    OrderStatusExpired         = "EXPIRED"          // 已过期
)
```

### MarginMode 保证金模式

```go
const (
    MarginModeIsolated = "ISOLATED" // 逐仓
    MarginModeCross    = "CROSS"    // 全仓
)
```

### PositionSide 仓位方向

```go
const (
    PositionSideLong  = "LONG"  // 多仓
    PositionSideShort = "SHORT" // 空仓
)
```

---

## 更新日志

- **2025-12-05**: 初始版本，包含所有核心模块的API文档
- **修复说明**: ADD_LONG/ADD_SHORT 信号类型已弃用，策略现在只生成 OPEN 信号，由仓位管理器决定是否加仓

---

**文档维护者**: AI Assistant  
**最后更新**: 2025-12-05

