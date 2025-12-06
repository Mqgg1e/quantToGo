package core

import (
	"context"
	"time"
)

// ========== 数据层接口 ==========

// KlineData K线数据标准接口
type KlineData interface {
	GetSymbol() string
	GetInterval() string
	GetStartTime() time.Time
	GetCloseTime() time.Time
	GetOpenPrice() float64
	GetClosePrice() float64
	GetHighPrice() float64
	GetLowPrice() float64
	GetVolume() float64
	IsClosed() bool
}

// OrderBookData 订单簿数据接口
type OrderBookData interface {
	GetSymbol() string
	GetBids() []PriceLevel // 买盘 (价格从高到低)
	GetAsks() []PriceLevel // 卖盘 (价格从低到高)
	GetUpdateTime() time.Time
}

// PriceLevel 订单簿价格档位
type PriceLevel struct {
	Price    float64
	Quantity float64
}

// DataProvider 数据提供者接口
type DataProvider interface {
	// SubscribeKline 订阅K线数据
	SubscribeKline(ctx context.Context, symbol, interval string) (<-chan KlineData, error)
	
	// SubscribeOrderBook 订阅订单簿数据
	SubscribeOrderBook(ctx context.Context, symbol string, levels int) (<-chan OrderBookData, error)
	
	// GetHistoricalKlines 获取历史K线（用于预热）
	GetHistoricalKlines(symbol, interval string, limit int) ([]KlineData, error)
	
	// Close 关闭数据提供者
	Close() error
}

// ========== 策略层接口 ==========

// SignalType 信号类型
type SignalType string

const (
	SignalTypeOpenLong    SignalType = "OPEN_LONG"    // 开多仓
	SignalTypeOpenShort   SignalType = "OPEN_SHORT"   // 开空仓
	SignalTypeAddLong     SignalType = "ADD_LONG"     // 加多仓
	SignalTypeAddShort    SignalType = "ADD_SHORT"    // 加空仓
	SignalTypeCloseLong   SignalType = "CLOSE_LONG"   // 平多仓
	SignalTypeCloseShort  SignalType = "CLOSE_SHORT"  // 平空仓
	SignalTypeNoAction    SignalType = "NO_ACTION"    // 无操作
)

// TradingSignal 交易信号
type TradingSignal struct {
	Type        SignalType         // 信号类型
	Symbol      string             // 交易对
	Timestamp   time.Time          // 信号生成时间
	Price       float64            // 触发价格
	Metadata    map[string]float64 // 额外元数据（如指标值）
	Reason      string             // 信号原因（用于日志）
	Confidence  float64            // 信号置信度 [0.0, 1.0]
}

// Strategy 策略接口
type Strategy interface {
	// Name 返回策略名称
	Name() string
	
	// OnKline 处理K线数据，返回交易信号
	OnKline(kline KlineData) (*TradingSignal, error)
	
	// OnOrderBook 处理订单簿数据（可选）
	OnOrderBook(orderBook OrderBookData) error
	
	// Warmup 策略预热（加载历史数据计算初始指标）
	Warmup(historicalKlines []KlineData) error
	
	// GetRequiredWarmupPeriods 返回需要预热的K线数量
	GetRequiredWarmupPeriods() int
	
	// Reset 重置策略状态
	Reset() error
}

// ========== 仓位管理接口 ==========

// PositionSide 仓位方向
type PositionSide string

const (
	PositionSideLong  PositionSide = "LONG"
	PositionSideShort PositionSide = "SHORT"
)

// Position 仓位信息
type Position struct {
	Symbol         string       // 交易对
	Side           PositionSide // 仓位方向
	Size           float64      // 仓位数量
	EntryPrice     float64      // 开仓均价
	CurrentPrice   float64      // 当前价格
	Leverage       int          // 杠杆倍数
	MarginMode     MarginMode   // 保证金模式
	UnrealizedPnL  float64      // 未实现盈亏
	UnrealizedPnLPercent float64 // 未实现盈亏百分比
	OpenTime       time.Time    // 开仓时间
}

// MarginMode 保证金模式
type MarginMode string

const (
	MarginModeIsolated MarginMode = "ISOLATED" // 逐仓
	MarginModeCross    MarginMode = "CROSS"    // 全仓
)

// PositionManager 仓位管理接口
type PositionManager interface {
	// ProcessSignal 处理策略信号，返回实际要执行的订单
	ProcessSignal(signal *TradingSignal, currentPrice float64) ([]*Order, error)
	
	// UpdatePosition 更新仓位信息（从执行模块获取）
	UpdatePosition(position *Position) error
	
	// GetPosition 获取当前仓位
	GetPosition(symbol string) (*Position, error)
	
	// GetAllPositions 获取所有仓位
	GetAllPositions() ([]*Position, error)
	
	// CheckRisk 风险检查
	CheckRisk(order *Order) error
	
	// CalculatePositionSize 计算仓位大小
	CalculatePositionSize(signal *TradingSignal, accountBalance float64) (float64, error)
}

// ========== 执行层接口 ==========

// OrderType 订单类型
type OrderType string

const (
	OrderTypeMarket      OrderType = "MARKET"       // 市价单
	OrderTypeLimit       OrderType = "LIMIT"        // 限价单
	OrderTypeStopMarket  OrderType = "STOP_MARKET"  // 市价止损单
	OrderTypeStopLimit   OrderType = "STOP_LIMIT"   // 限价止损单
	OrderTypeTakeProfit  OrderType = "TAKE_PROFIT"  // 止盈单
)

// OrderSide 订单方向
type OrderSide string

const (
	OrderSideBuy  OrderSide = "BUY"
	OrderSideSell OrderSide = "SELL"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"              // 新建
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED" // 部分成交
	OrderStatusFilled          OrderStatus = "FILLED"           // 完全成交
	OrderStatusCanceled        OrderStatus = "CANCELED"         // 已取消
	OrderStatusRejected        OrderStatus = "REJECTED"         // 被拒绝
	OrderStatusExpired         OrderStatus = "EXPIRED"          // 已过期
)

// Order 订单结构
type Order struct {
	ID            string       // 订单ID（本地生成或交易所返回）
	Symbol        string       // 交易对
	Type          OrderType    // 订单类型
	Side          OrderSide    // 订单方向
	Price         float64      // 价格（限价单）
	Quantity      float64      // 数量
	StopPrice     float64      // 触发价格（止损/止盈单）
	Status        OrderStatus  // 订单状态
	Leverage      int          // 杠杆倍数
	MarginMode    MarginMode   // 保证金模式
	FilledQty     float64      // 已成交数量
	AvgPrice      float64      // 成交均价
	Commission    float64      // 手续费
	CreateTime    time.Time    // 创建时间
	UpdateTime    time.Time    // 更新时间
	Metadata      map[string]interface{} // 额外信息
}

// Executor 执行器接口（支持回测和实盘）
type Executor interface {
	// PlaceOrder 下单
	PlaceOrder(ctx context.Context, order *Order) (*Order, error)
	
	// CancelOrder 撤单
	CancelOrder(ctx context.Context, symbol, orderID string) error
	
	// GetOrder 查询订单
	GetOrder(ctx context.Context, symbol, orderID string) (*Order, error)
	
	// GetOpenOrders 获取所有未成交订单
	GetOpenOrders(ctx context.Context, symbol string) ([]*Order, error)
	
	// GetAccount 获取账户信息
	GetAccount(ctx context.Context) (*Account, error)
	
	// GetPositions 获取持仓信息
	GetPositions(ctx context.Context) ([]*Position, error)
	
	// SetLeverage 设置杠杆
	SetLeverage(ctx context.Context, symbol string, leverage int) error
	
	// SetMarginMode 设置保证金模式
	SetMarginMode(ctx context.Context, symbol string, mode MarginMode) error
	
	// Close 关闭执行器
	Close() error
}

// Account 账户信息
type Account struct {
	TotalBalance      float64   // 总余额
	AvailableBalance  float64   // 可用余额
	UsedMargin        float64   // 已用保证金
	UnrealizedPnL     float64   // 未实现盈亏
	UpdateTime        time.Time // 更新时间
}

// ========== 日志与监控接口 ==========

// LogLevel 日志级别
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry 日志条目
type LogEntry struct {
	Timestamp   time.Time              // 时间戳
	Level       LogLevel               // 日志级别
	ComponentID string                 // 组件ID（data_module, strategy_module等）
	Message     string                 // 日志消息
	Context     map[string]interface{} // 上下文信息
	Error       error                  // 错误对象（可选）
}

// Logger 日志接口
type Logger interface {
	Debug(componentID, message string, context ...map[string]interface{})
	Info(componentID, message string, context ...map[string]interface{})
	Warn(componentID, message string, context ...map[string]interface{})
	Error(componentID, message string, err error, context ...map[string]interface{})
	Fatal(componentID, message string, err error, context ...map[string]interface{})
	
	// WithComponent 返回带组件ID的子Logger
	WithComponent(componentID string) Logger
	
	// Close 关闭日志器
	Close() error
}

// MetricType 指标类型
type MetricType string

const (
	MetricTypeCounter   MetricType = "COUNTER"   // 计数器
	MetricTypeGauge     MetricType = "GAUGE"     // 仪表盘
	MetricTypeHistogram MetricType = "HISTOGRAM" // 直方图
)

// Metric 监控指标
type Metric struct {
	Timestamp   time.Time              // 时间戳
	Name        string                 // 指标名称
	Type        MetricType             // 指标类型
	Value       float64                // 指标值
	Labels      map[string]string      // 标签
	ComponentID string                 // 组件ID
}

// MetricsCollector 指标收集器接口
type MetricsCollector interface {
	// RecordMetric 记录指标
	RecordMetric(metric *Metric)
	
	// GetMetrics 获取指标（用于导出）
	GetMetrics() []*Metric
	
	// Close 关闭收集器
	Close() error
}

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "INFO"
	AlertSeverityWarning  AlertSeverity = "WARNING"
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

// Alert 告警
type Alert struct {
	Timestamp   time.Time              // 时间戳
	Severity    AlertSeverity          // 严重程度
	ComponentID string                 // 组件ID
	AlertType   string                 // 告警类型
	Details     map[string]interface{} // 详细信息
}

// Alerter 告警接口
type Alerter interface {
	// SendAlert 发送告警
	SendAlert(alert *Alert) error
	
	// RegisterHandler 注册告警处理器
	RegisterHandler(handler AlertHandler)
	
	// Close 关闭告警器
	Close() error
}

// AlertHandler 告警处理器
type AlertHandler interface {
	HandleAlert(alert *Alert) error
}

// ObservabilityHub 可观测性中心（整合日志、指标、告警）
type ObservabilityHub interface {
	Logger() Logger
	Metrics() MetricsCollector
	Alerter() Alerter
	Close() error
}

