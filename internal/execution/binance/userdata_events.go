package binance

// ========== UserDataStream 事件类型 ==========
// 参考: https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/user-data-streams

// UserDataEvent 用户数据流基础事件
type UserDataEvent struct {
	EventType string `json:"e"` // 事件类型
	EventTime int64  `json:"E"` // 事件时间
}

// ========== ACCOUNT_UPDATE 事件 ==========

// AccountUpdateEvent 账户更新事件
type AccountUpdateEvent struct {
	EventType     string            `json:"e"` // "ACCOUNT_UPDATE"
	EventTime     int64             `json:"E"` // 事件时间
	Transaction   int64             `json:"T"` // 撮合时间
	AccountUpdate AccountUpdateData `json:"a"` // 账户更新数据
}

// AccountUpdateData 账户更新数据
type AccountUpdateData struct {
	Reason    string           `json:"m"` // 事件推送原因类型: DEPOSIT, WITHDRAW, ORDER, FUNDING_FEE, etc.
	Balances  []BalanceUpdate  `json:"B"` // 余额信息
	Positions []PositionUpdate `json:"P"` // 持仓信息
}

// BalanceUpdate 余额更新
type BalanceUpdate struct {
	Asset              string `json:"a"`  // 资产名称
	WalletBalance      string `json:"wb"` // 钱包余额
	CrossWalletBalance string `json:"cw"` // 全仓余额
	BalanceChange      string `json:"bc"` // 余额变化
}

// PositionUpdate 持仓更新
type PositionUpdate struct {
	Symbol              string `json:"s"`  // 交易对
	PositionAmount      string `json:"pa"` // 持仓数量
	EntryPrice          string `json:"ep"` // 开仓均价
	AccumulatedRealized string `json:"cr"` // 累计实现盈亏
	UnrealizedPnL       string `json:"up"` // 未实现盈亏
	MarginType          string `json:"mt"` // 保证金模式: isolated, cross
	IsolatedWallet      string `json:"iw"` // 逐仓保证金
	PositionSide        string `json:"ps"` // 持仓方向: BOTH, LONG, SHORT
}

// ========== ORDER_TRADE_UPDATE 事件 ==========

// OrderTradeUpdateEvent 订单/交易更新事件
type OrderTradeUpdateEvent struct {
	EventType   string          `json:"e"` // "ORDER_TRADE_UPDATE"
	EventTime   int64           `json:"E"` // 事件时间
	Transaction int64           `json:"T"` // 撮合时间
	Order       OrderUpdateData `json:"o"` // 订单数据
}

// OrderUpdateData 订单更新数据
type OrderUpdateData struct {
	Symbol             string `json:"s"`  // 交易对
	ClientOrderID      string `json:"c"`  // 客户端订单ID
	Side               string `json:"S"`  // 订单方向: BUY, SELL
	OrderType          string `json:"o"`  // 订单类型: MARKET, LIMIT, STOP, etc.
	TimeInForce        string `json:"f"`  // 有效方式: GTC, IOC, FOK, GTX
	OriginalQuantity   string `json:"q"`  // 订单原始数量
	OriginalPrice      string `json:"p"`  // 订单原始价格
	AveragePrice       string `json:"ap"` // 订单平均价格
	StopPrice          string `json:"sp"` // 条件单触发价格
	ExecutionType      string `json:"x"`  // 执行类型: NEW, CANCELED, CALCULATED, EXPIRED, TRADE, etc.
	OrderStatus        string `json:"X"`  // 订单状态: NEW, PARTIALLY_FILLED, FILLED, CANCELED, etc.
	OrderID            int64  `json:"i"`  // 订单ID
	LastFilledQuantity string `json:"l"`  // 最后成交数量
	AccumulatedFilled  string `json:"z"`  // 累计成交数量
	LastFilledPrice    string `json:"L"`  // 最后成交价格
	CommissionAsset    string `json:"N"`  // 手续费资产
	Commission         string `json:"n"`  // 手续费
	TradeTime          int64  `json:"T"`  // 成交时间
	TradeID            int64  `json:"t"`  // 成交ID
	BidsNotional       string `json:"b"`  // 买单净值
	AsksNotional       string `json:"a"`  // 卖单净值
	IsMakerSide        bool   `json:"m"`  // 是否是挂单方
	IsReduceOnly       bool   `json:"R"`  // 是否仅减仓
	WorkingType        string `json:"wt"` // 条件价格触发类型: CONTRACT_PRICE, MARK_PRICE
	OriginalOrderType  string `json:"ot"` // 原始订单类型
	PositionSide       string `json:"ps"` // 持仓方向: BOTH, LONG, SHORT
	ClosePosition      bool   `json:"cp"` // 是否为触发后全部平仓
	ActivationPrice    string `json:"AP"` // 跟踪止损激活价格
	CallbackRate       string `json:"cr"` // 跟踪止损回调比例
	RealizedProfit     string `json:"rp"` // 实现盈亏
}

// ========== MARGIN_CALL 事件（可选） ==========

// MarginCallEvent 追加保证金通知事件
type MarginCallEvent struct {
	EventType string               `json:"e"` // "MARGIN_CALL"
	EventTime int64                `json:"E"` // 事件时间
	Positions []MarginCallPosition `json:"p"` // 持仓
}

// MarginCallPosition 追加保证金持仓
type MarginCallPosition struct {
	Symbol                    string `json:"s"`  // 交易对
	PositionSide              string `json:"ps"` // 持仓方向
	PositionAmount            string `json:"pa"` // 持仓数量
	MarginType                string `json:"mt"` // 保证金类型
	IsolatedWallet            string `json:"iw"` // 逐仓保证金（逐仓模式）
	MarkPrice                 string `json:"mp"` // 标记价格
	UnrealizedPnL             string `json:"up"` // 未实现盈亏
	MaintenanceMarginRequired string `json:"mm"` // 维持保证金
}

// ========== ACCOUNT_CONFIG_UPDATE 事件（可选） ==========

// AccountConfigUpdateEvent 账户配置更新事件
type AccountConfigUpdateEvent struct {
	EventType    string                  `json:"e"`  // "ACCOUNT_CONFIG_UPDATE"
	EventTime    int64                   `json:"E"`  // 事件时间
	Transaction  int64                   `json:"T"`  // 撮合时间
	ConfigUpdate AccountConfigUpdateData `json:"ac"` // 配置更新数据
}

// AccountConfigUpdateData 账户配置更新数据
type AccountConfigUpdateData struct {
	Symbol   string `json:"s"` // 交易对
	Leverage int    `json:"l"` // 杠杆倍数（杠杆变化时）
}
