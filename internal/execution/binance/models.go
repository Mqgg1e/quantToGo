package binance

import (
	"time"

	"goQuant/internal/core"
)

// ========== 币安期货API数据结构 ==========

// FuturesOrderSide 订单方向
type FuturesOrderSide string

const (
	SideBuy  FuturesOrderSide = "BUY"
	SideSell FuturesOrderSide = "SELL"
)

// FuturesPositionSide 持仓方向（用于双向持仓模式）
type FuturesPositionSide string

const (
	PositionSideBoth  FuturesPositionSide = "BOTH"  // 单向持仓
	PositionSideLong  FuturesPositionSide = "LONG"  // 多头
	PositionSideShort FuturesPositionSide = "SHORT" // 空头
)

// FuturesOrderType 订单类型
type FuturesOrderType string

const (
	OrderTypeLimit              FuturesOrderType = "LIMIT"                // 限价单
	OrderTypeMarket             FuturesOrderType = "MARKET"               // 市价单
	OrderTypeStop               FuturesOrderType = "STOP"                 // 止损限价单
	OrderTypeStopMarket         FuturesOrderType = "STOP_MARKET"          // 止损市价单
	OrderTypeTakeProfit         FuturesOrderType = "TAKE_PROFIT"          // 止盈限价单
	OrderTypeTakeProfitMarket   FuturesOrderType = "TAKE_PROFIT_MARKET"   // 止盈市价单
	OrderTypeTrailingStopMarket FuturesOrderType = "TRAILING_STOP_MARKET" // 跟踪止损
)

// TimeInForce 订单有效期
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // 成交为止
	TimeInForceIOC TimeInForce = "IOC" // 无法立即成交的部分就撤销
	TimeInForceFOK TimeInForce = "FOK" // 无法全部立即成交就撤销
	TimeInForceGTX TimeInForce = "GTX" // 无法成为挂单方就撤销
)

// WorkingType 条件单触发价格类型
type WorkingType string

const (
	WorkingTypeMark     WorkingType = "MARK_PRICE"     // 标记价格
	WorkingTypeContract WorkingType = "CONTRACT_PRICE" // 最新价格
)

// MarginType 保证金模式
type MarginType string

const (
	MarginTypeIsolated MarginType = "ISOLATED" // 逐仓
	MarginTypeCrossed  MarginType = "CROSSED"  // 全仓
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "NEW"
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	OrderStatusFilled          OrderStatus = "FILLED"
	OrderStatusCanceled        OrderStatus = "CANCELED"
	OrderStatusRejected        OrderStatus = "REJECTED"
	OrderStatusExpired         OrderStatus = "EXPIRED"
)

// ========== 请求结构 ==========

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	Symbol           string              `json:"symbol"`                     // 交易对
	Side             FuturesOrderSide    `json:"side"`                       // 买卖方向
	PositionSide     FuturesPositionSide `json:"positionSide,omitempty"`     // 持仓方向
	Type             FuturesOrderType    `json:"type"`                       // 订单类型
	TimeInForce      TimeInForce         `json:"timeInForce,omitempty"`      // 有效方式
	Quantity         string              `json:"quantity,omitempty"`         // 下单数量（数量模式）
	ReduceOnly       bool                `json:"reduceOnly,omitempty"`       // 只减仓
	Price            string              `json:"price,omitempty"`            // 价格
	NewClientOrderId string              `json:"newClientOrderId,omitempty"` // 客户自定义订单ID
	StopPrice        string              `json:"stopPrice,omitempty"`        // 触发价
	ClosePosition    bool                `json:"closePosition,omitempty"`    // 平掉全部仓位（用于STOP_MARKET/TAKE_PROFIT_MARKET）
	ActivationPrice  string              `json:"activationPrice,omitempty"`  // 追踪止损激活价格
	CallbackRate     string              `json:"callbackRate,omitempty"`     // 追踪止损回调比例
	WorkingType      WorkingType         `json:"workingType,omitempty"`      // 条件价格触发类型
	PriceProtect     bool                `json:"priceProtect,omitempty"`     // 价格保护
}

// OrderResponse 订单响应
type OrderResponse struct {
	OrderID       int64               `json:"orderId"`
	Symbol        string              `json:"symbol"`
	Status        OrderStatus         `json:"status"`
	ClientOrderID string              `json:"clientOrderId"`
	Price         string              `json:"price"`
	AvgPrice      string              `json:"avgPrice"`
	OrigQty       string              `json:"origQty"`
	ExecutedQty   string              `json:"executedQty"`
	CumQuote      string              `json:"cumQuote"`
	TimeInForce   TimeInForce         `json:"timeInForce"`
	Type          FuturesOrderType    `json:"type"`
	ReduceOnly    bool                `json:"reduceOnly"`
	Side          FuturesOrderSide    `json:"side"`
	PositionSide  FuturesPositionSide `json:"positionSide"`
	StopPrice     string              `json:"stopPrice"`
	WorkingType   WorkingType         `json:"workingType"`
	UpdateTime    int64               `json:"updateTime"`
}

// PositionRisk 持仓信息
type PositionRisk struct {
	Symbol           string `json:"symbol"`
	PositionAmt      string `json:"positionAmt"`      // 持仓数量
	EntryPrice       string `json:"entryPrice"`       // 开仓均价
	MarkPrice        string `json:"markPrice"`        // 标记价格
	UnRealizedProfit string `json:"unRealizedProfit"` // 未实现盈亏
	LiquidationPrice string `json:"liquidationPrice"` // 强平价格
	Leverage         string `json:"leverage"`         // 杠杆倍数
	MarginType       string `json:"marginType"`       // 保证金模式
	IsolatedMargin   string `json:"isolatedMargin"`   // 逐仓保证金
	PositionSide     string `json:"positionSide"`     // 持仓方向
	UpdateTime       int64  `json:"updateTime"`
}

// AccountInfo 账户信息
type AccountInfo struct {
	Assets                  []AccountAsset    `json:"assets"`
	Positions               []AccountPosition `json:"positions"`
	CanDeposit              bool              `json:"canDeposit"`
	CanTrade                bool              `json:"canTrade"`
	CanWithdraw             bool              `json:"canWithdraw"`
	FeeTier                 int               `json:"feeTier"`
	MaxWithdrawAmount       string            `json:"maxWithdrawAmount"`
	TotalInitialMargin      string            `json:"totalInitialMargin"`
	TotalMaintMargin        string            `json:"totalMaintMargin"`
	TotalMarginBalance      string            `json:"totalMarginBalance"`
	TotalCrossUnPnl         string            `json:"totalCrossUnPnl"`
	TotalCrossWalletBalance string            `json:"totalCrossWalletBalance"`
	TotalWalletBalance      string            `json:"totalWalletBalance"`
	AvailableBalance        string            `json:"availableBalance"`
	UpdateTime              int64             `json:"updateTime"`
}

// AccountAsset 账户资产
type AccountAsset struct {
	Asset                  string `json:"asset"`
	WalletBalance          string `json:"walletBalance"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	MarginBalance          string `json:"marginBalance"`
	MaintMargin            string `json:"maintMargin"`
	InitialMargin          string `json:"initialMargin"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	CrossWalletBalance     string `json:"crossWalletBalance"`
	CrossUnPnl             string `json:"crossUnPnl"`
	AvailableBalance       string `json:"availableBalance"`
	MaxWithdrawAmount      string `json:"maxWithdrawAmount"`
}

// AccountPosition 账户持仓
type AccountPosition struct {
	Symbol                 string `json:"symbol"`
	InitialMargin          string `json:"initialMargin"`
	MaintMargin            string `json:"maintMargin"`
	UnrealizedProfit       string `json:"unrealizedProfit"`
	PositionInitialMargin  string `json:"positionInitialMargin"`
	OpenOrderInitialMargin string `json:"openOrderInitialMargin"`
	Leverage               string `json:"leverage"`
	Isolated               bool   `json:"isolated"`
	EntryPrice             string `json:"entryPrice"`
	MaxNotional            string `json:"maxNotional"`
	PositionSide           string `json:"positionSide"`
	PositionAmt            string `json:"positionAmt"`
	UpdateTime             int64  `json:"updateTime"`
}

// LeverageRequest 设置杠杆请求
type LeverageRequest struct {
	Symbol   string `json:"symbol"`
	Leverage int    `json:"leverage"`
}

// MarginTypeRequest 设置保证金模式请求
type MarginTypeRequest struct {
	Symbol     string     `json:"symbol"`
	MarginType MarginType `json:"marginType"`
}

// OrderBookLevel 订单簿价格档位
type OrderBookLevel struct {
	Price    string `json:"0"` // 价格
	Quantity string `json:"1"` // 数量
}

// OrderBook 订单簿
type OrderBook struct {
	LastUpdateId int64            `json:"lastUpdateId"`
	Bids         []OrderBookLevel `json:"bids"` // 买单
	Asks         []OrderBookLevel `json:"asks"` // 卖单
}

// ========== 转换函数：币安结构 → core结构 ==========

// ToOrderStatus 转换订单状态
func ToOrderStatus(status OrderStatus) core.OrderStatus {
	switch status {
	case OrderStatusNew:
		return core.OrderStatusNew
	case OrderStatusPartiallyFilled:
		return core.OrderStatusPartiallyFilled
	case OrderStatusFilled:
		return core.OrderStatusFilled
	case OrderStatusCanceled:
		return core.OrderStatusCanceled
	case OrderStatusRejected:
		return core.OrderStatusRejected
	case OrderStatusExpired:
		return core.OrderStatusExpired
	default:
		return core.OrderStatusNew
	}
}

// ToOrderType 转换订单类型
func ToOrderType(orderType FuturesOrderType) core.OrderType {
	switch orderType {
	case OrderTypeMarket, OrderTypeStopMarket, OrderTypeTakeProfitMarket:
		return core.OrderTypeMarket
	case OrderTypeLimit, OrderTypeStop, OrderTypeTakeProfit:
		return core.OrderTypeLimit
	default:
		return core.OrderTypeMarket
	}
}

// ToOrderSide 转换订单方向
func ToOrderSide(side FuturesOrderSide) core.OrderSide {
	if side == SideBuy {
		return core.OrderSideBuy
	}
	return core.OrderSideSell
}

// ToMarginMode 转换保证金模式
func ToMarginMode(marginType string) core.MarginMode {
	if marginType == "isolated" || marginType == "ISOLATED" {
		return core.MarginModeIsolated
	}
	return core.MarginModeCross
}

// FromOrderSide core订单方向 → 币安方向
func FromOrderSide(side core.OrderSide) FuturesOrderSide {
	if side == core.OrderSideBuy {
		return SideBuy
	}
	return SideSell
}

// FromOrderType core订单类型 → 币安类型
func FromOrderType(orderType core.OrderType) FuturesOrderType {
	switch orderType {
	case core.OrderTypeMarket:
		return OrderTypeMarket
	case core.OrderTypeLimit:
		return OrderTypeLimit
	case core.OrderTypeStopMarket:
		return OrderTypeStopMarket
	case core.OrderTypeStopLimit:
		return OrderTypeStop // STOP 是限价止损单
	case core.OrderTypeTakeProfit:
		return OrderTypeTakeProfitMarket
	default:
		return OrderTypeMarket
	}
}

// FromMarginMode core保证金模式 → 币安模式
func FromMarginMode(mode core.MarginMode) MarginType {
	if mode == core.MarginModeIsolated {
		return MarginTypeIsolated
	}
	return MarginTypeCrossed
}

// OrderResponseToOrder 币安订单响应 → core.Order
func OrderResponseToOrder(resp *OrderResponse) *core.Order {
	price, _ := parseFloat(resp.Price)
	avgPrice, _ := parseFloat(resp.AvgPrice)
	quantity, _ := parseFloat(resp.OrigQty)
	filledQty, _ := parseFloat(resp.ExecutedQty)

	return &core.Order{
		ID:         resp.ClientOrderID,
		Symbol:     resp.Symbol,
		Type:       ToOrderType(resp.Type),
		Side:       ToOrderSide(resp.Side),
		Price:      price,
		Quantity:   quantity,
		Status:     ToOrderStatus(resp.Status),
		FilledQty:  filledQty,
		AvgPrice:   avgPrice,
		UpdateTime: time.UnixMilli(resp.UpdateTime),
		Metadata: map[string]interface{}{
			"binance_order_id": resp.OrderID,
			"position_side":    resp.PositionSide,
		},
	}
}

// PositionRiskToPosition 币安持仓 → core.Position
func PositionRiskToPosition(pos *PositionRisk) *core.Position {
	posAmt, _ := parseFloat(pos.PositionAmt)
	entryPrice, _ := parseFloat(pos.EntryPrice)
	markPrice, _ := parseFloat(pos.MarkPrice)
	unrealizedPnL, _ := parseFloat(pos.UnRealizedProfit)
	leverage, _ := parseInt(pos.Leverage)

	var side core.PositionSide
	if posAmt > 0 {
		side = core.PositionSideLong
	} else if posAmt < 0 {
		side = core.PositionSideShort
		posAmt = -posAmt // 转为正数
	} else {
		return nil // 无持仓
	}

	var pnlPercent float64
	if entryPrice > 0 {
		pnlPercent = unrealizedPnL / (entryPrice * posAmt) * 100
	}

	return &core.Position{
		Symbol:               pos.Symbol,
		Side:                 side,
		Size:                 posAmt,
		EntryPrice:           entryPrice,
		CurrentPrice:         markPrice,
		Leverage:             leverage,
		MarginMode:           ToMarginMode(pos.MarginType),
		UnrealizedPnL:        unrealizedPnL,
		UnrealizedPnLPercent: pnlPercent,
		OpenTime:             time.UnixMilli(pos.UpdateTime),
	}
}

// AccountInfoToAccount 币安账户 → core.Account
func AccountInfoToAccount(info *AccountInfo) *core.Account {
	var totalBalance, availableBalance, usedMargin, unrealizedPnL float64

	// 查找USDT资产
	for _, asset := range info.Assets {
		if asset.Asset == "USDT" {
			totalBalance, _ = parseFloat(asset.WalletBalance)
			availableBalance, _ = parseFloat(asset.AvailableBalance)
			usedMargin, _ = parseFloat(asset.InitialMargin)
			unrealizedPnL, _ = parseFloat(asset.UnrealizedProfit)
			break
		}
	}

	return &core.Account{
		TotalBalance:     totalBalance,
		AvailableBalance: availableBalance,
		UsedMargin:       usedMargin,
		UnrealizedPnL:    unrealizedPnL,
		UpdateTime:       time.UnixMilli(info.UpdateTime),
	}
}
