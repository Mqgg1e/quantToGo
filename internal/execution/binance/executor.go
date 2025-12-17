package binance

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"goQuant/internal/cache"
	"goQuant/internal/core"
)

// LiveExecutor 币安期货实盘执行器
type LiveExecutor struct {
	client       *Client
	accountCache *cache.AccountCache // 账户缓存（替代本地缓存）
	wsOrder      *WSOrderClient      // WebSocket 订单客户端
	useWSOrder   bool                // 是否使用 WebSocket 下单
	mu           sync.RWMutex        // 保护非缓存字段的锁
}

// NewLiveExecutor 创建实盘执行器
func NewLiveExecutor(apiKey, secretKey, baseURL string, accountCache *cache.AccountCache) *LiveExecutor {
	return &LiveExecutor{
		client:       NewClient(apiKey, secretKey, baseURL),
		accountCache: accountCache,
		useWSOrder:   false, // 默认使用 REST API
	}
}

// EnableWebSocketOrder 启用 WebSocket 下单
func (e *LiveExecutor) EnableWebSocketOrder(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.wsOrder != nil {
		return nil // 已启用
	}

	wsOrder := NewWSOrderClient(e.client)
	if err := wsOrder.Start(ctx); err != nil {
		return fmt.Errorf("start websocket order: %w", err)
	}

	e.wsOrder = wsOrder
	e.useWSOrder = true
	return nil
}

// DisableWebSocketOrder 禁用 WebSocket 下单
func (e *LiveExecutor) DisableWebSocketOrder() {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.wsOrder != nil {
		e.wsOrder.Stop()
		e.wsOrder = nil
	}
	e.useWSOrder = false
}

// GetClient 获取币安客户端（用于 UserDataStream）
func (e *LiveExecutor) GetClient() *Client {
	return e.client
}

// SetAccountCache 设置账户缓存（用于后期注入）
func (e *LiveExecutor) SetAccountCache(cache *cache.AccountCache) {
	e.accountCache = cache
}

// ========== 实现 core.Executor 接口 ==========

// PlaceOrder 下单
func (e *LiveExecutor) PlaceOrder(ctx context.Context, order *core.Order) (*core.Order, error) {
	// 生成客户端订单ID
	if order.ID == "" {
		order.ID = fmt.Sprintf("go_%d", time.Now().UnixNano())
	}

	// 检查是否使用 WebSocket 下单
	e.mu.RLock()
	useWS := e.useWSOrder && e.wsOrder != nil
	wsOrder := e.wsOrder
	e.mu.RUnlock()

	if useWS {
		return e.placeOrderViaWebSocket(ctx, order, wsOrder)
	}

	// 使用 REST API 下单（原有逻辑）
	return e.placeOrderViaREST(ctx, order)
}

// placeOrderViaWebSocket 通过 WebSocket 下单
func (e *LiveExecutor) placeOrderViaWebSocket(ctx context.Context, order *core.Order, wsOrder *WSOrderClient) (*core.Order, error) {
	// 根据订单类型路由到不同的 WebSocket 方法
	switch order.Type {
	case core.OrderTypeMarket:
		// 市价单
		if order.Metadata != nil {
			if reduceOnly, ok := order.Metadata["reduce_only"].(bool); ok && reduceOnly {
				// 平仓市价单
				return wsOrder.ClosePositionMarket(ctx, order.Symbol, order.Side, order.Quantity)
			}
		}
		// 开仓市价单
		return wsOrder.PlaceMarketOrder(ctx, order.Symbol, order.Side, order.Quantity)

	case core.OrderTypeLimit:
		// 限价单
		timeInForce := "GTC"
		if order.Metadata != nil {
			if tif, ok := order.Metadata["time_in_force"].(string); ok && tif != "" {
				timeInForce = tif
			}
		}
		return wsOrder.PlaceLimitOrder(ctx, order.Symbol, order.Side, order.Quantity, order.Price, timeInForce)

	case core.OrderTypeStopMarket:
		// 止损单
		return wsOrder.PlaceStopLossOrder(ctx, order.Symbol, order.Side, order.StopPrice)

	case core.OrderTypeTrailingStop:
		// 跟踪止损单
		callbackRate := 1.0 // 默认 1%
		activatePrice := 0.0
		quantity := order.Quantity
		if order.Metadata != nil {
			if rate, ok := order.Metadata["callback_rate"].(float64); ok {
				callbackRate = rate
			}
			//if price, ok := order.Metadata["activate_price"].(float64); ok {
			//	activatePrice = price
			//}
		}
		if order.ActivationPrice > 0 {
			activatePrice = order.ActivationPrice
		}
		return wsOrder.PlaceTrailingStopOrder(ctx, order.Symbol, order.Side, quantity, activatePrice, callbackRate)

	default:
		return nil, fmt.Errorf("unsupported order type for WebSocket: %v", order.Type)
	}
}

// placeOrderViaREST 通过 REST API 下单（原有逻辑）
func (e *LiveExecutor) placeOrderViaREST(ctx context.Context, order *core.Order) (*core.Order, error) {
	// 构建请求
	req := &CreateOrderRequest{
		Symbol:           order.Symbol,
		Side:             FromOrderSide(order.Side),
		Type:             FromOrderType(order.Type),
		NewClientOrderId: order.ID,
		PositionSide:     PositionSideBoth, // 单向持仓模式
	}

	// 数量处理
	// 对于 STOP_MARKET/TAKE_PROFIT_MARKET 止损单，如果是平仓，使用 closePosition 而不是 quantity
	isStopMarketCloseAll := (order.Type == core.OrderTypeStopMarket || order.Type == core.OrderTypeTakeProfit) &&
		order.Metadata != nil

	var useClosePosition bool
	if isStopMarketCloseAll {
		if reduceOnly, ok := order.Metadata["reduce_only"].(bool); ok && reduceOnly {
			useClosePosition = true
		}
	}

	if useClosePosition {
		// STOP_MARKET 平仓：使用 closePosition=true，不发送 quantity
		req.ClosePosition = true
	} else {
		// 其他订单类型：正常处理数量
		// 如果订单没有设置数量，尝试从Metadata中的usdt_amount计算
		if order.Quantity == 0 && order.Metadata != nil {
			if usdtAmount, ok := order.Metadata["usdt_amount"].(float64); ok && usdtAmount > 0 {
				// 获取当前价格（市价单需要估算）
				currentPrice := order.Price
				if currentPrice == 0 {
					// 对于市价单，使用最新市场价格估算
					// 简化处理：从持仓风险接口获取标记价格
					positions, err := e.client.GetPositionRisk(ctx, order.Symbol)
					if err == nil && len(positions) > 0 {
						// 解析标记价格
						if markPrice, parseErr := parseFloat(positions[0].MarkPrice); parseErr == nil && markPrice > 0 {
							currentPrice = markPrice
						}
					}
				}

				// 如果仍然没有价格，返回错误
				if currentPrice == 0 {
					return nil, fmt.Errorf("cannot calculate quantity: no price available")
				}

				// 计算数量：数量 = (USDT金额 * 杠杆) / 价格
				leverage := float64(order.Leverage)
				if leverage == 0 {
					leverage = 1
				}
				quantity := (usdtAmount * leverage) / currentPrice
				order.Quantity = quantity
			}
		}

		if order.Quantity > 0 {
			req.Quantity = formatQuantity(order.Symbol, order.Quantity)
		} else {
			return nil, fmt.Errorf("order quantity is zero or not set")
		}
	}

	// 基本风险检查：验证数量和杠杆
	if order.Quantity <= 0 {
		return nil, fmt.Errorf("invalid quantity: %f", order.Quantity)
	}
	if order.Leverage < 1 || order.Leverage > 125 {
		return nil, fmt.Errorf("invalid leverage: %d (must be 1-125)", order.Leverage)
	}

	// 价格（限价单）
	if order.Type == core.OrderTypeLimit && order.Price > 0 {
		req.Price = formatPrice(order.Symbol, order.Price)
		req.TimeInForce = TimeInForceGTC
	}

	// 止损单：STOP_MARKET 只需要 stopPrice
	if order.Type == core.OrderTypeStopMarket && order.StopPrice > 0 {
		req.StopPrice = formatPrice(order.Symbol, order.StopPrice)
		// WorkingType 可能在测试网不支持，先不设置
		// req.WorkingType = WorkingTypeMark
	}

	// 跟踪止损单：TRAILING_STOP_MARKET 需要 callbackRate
	if order.Type == core.OrderTypeTrailingStop && order.Metadata != nil {
		if callbackRate, ok := order.Metadata["callback_rate"].(float64); ok {
			// Binance API 的 callbackRate: 0.1 到 10，其中 1 表示 1%
			// 我们的 callbackRate 是小数（0.5 表示 0.5%）
			req.CallbackRate = fmt.Sprintf("%.2f", callbackRate)
		}

		//if activatePrice, ok := order.Metadata["activate_price"].(float64); ok && activatePrice > 0 {
		//	req.ActivatePrice = formatPrice(order.Symbol, activatePrice) // 使用 ActivatePrice 字段
		//}

		// 检查是否平掉全部仓位
		if closePosition, ok := order.Metadata["close_position"].(bool); ok && closePosition {
			req.ClosePosition = true
			useClosePosition = true // 不设置 quantity
		}
	}

	// 只减仓标志（平仓订单）
	// 注意：使用 closePosition 时不能同时设置 reduceOnly
	if order.Metadata != nil && !useClosePosition {
		if reduceOnly, ok := order.Metadata["reduce_only"].(bool); ok && reduceOnly {
			req.ReduceOnly = true
		}
	}

	// 发送订单
	resp, err := e.client.CreateOrder(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("create order: %w", err)
	}

	// 转换响应
	resultOrder := OrderResponseToOrder(resp)
	resultOrder.CreateTime = time.Now()

	// 添加额外元数据
	if resultOrder.Metadata == nil {
		resultOrder.Metadata = make(map[string]interface{})
	}
	resultOrder.Metadata["leverage"] = order.Leverage
	resultOrder.Metadata["margin_mode"] = order.MarginMode

	// 不再缓存订单，等待 UserDataStream 更新
	// UserDataStream 会实时接收 ORDER_TRADE_UPDATE 事件并更新缓存

	return resultOrder, nil
}

// CancelOrder 撤单
func (e *LiveExecutor) CancelOrder(ctx context.Context, symbol, orderID string) error {
	// 从账户缓存获取订单
	order, exists := e.accountCache.GetOrder(orderID)
	if !exists {
		return fmt.Errorf("order not found in cache: %s", orderID)
	}

	binanceOrderID, ok := order.Metadata["binance_order_id"].(int64)
	if !ok {
		return fmt.Errorf("binance order ID not found")
	}

	return e.client.CancelOrder(ctx, symbol, binanceOrderID)
}

// GetOrder 查询订单
func (e *LiveExecutor) GetOrder(ctx context.Context, symbol, orderID string) (*core.Order, error) {
	// 先从账户缓存查询
	order, exists := e.accountCache.GetOrder(orderID)
	if exists {
		// 从API刷新状态（可选，因为UserDataStream会实时更新）
		binanceOrderID, ok := order.Metadata["binance_order_id"].(int64)
		if ok {
			resp, err := e.client.GetOrder(ctx, symbol, binanceOrderID)
			if err == nil {
				updatedOrder := OrderResponseToOrder(resp)
				// 让 UserDataStream 更新缓存，或者这里直接更新
				e.accountCache.UpdateOrder(updatedOrder, time.Now().UnixMilli())
				return updatedOrder, nil
			}
		}
		return order, nil
	}

	// 如果缓存中没有，从API查询
	return nil, fmt.Errorf("order not found: %s", orderID)
}

// GetOpenOrders 获取未成交订单
func (e *LiveExecutor) GetOpenOrders(ctx context.Context, symbol string) ([]*core.Order, error) {
	// 如果指定了 symbol 且缓存中有数据，优先从缓存获取
	if symbol != "" {
		cachedOrders := e.accountCache.GetOpenOrders(symbol)
		if len(cachedOrders) > 0 {
			return cachedOrders, nil
		}
	}

	// 从API获取（用于全量同步或缓存未命中）
	binanceOrders, err := e.client.GetOpenOrders(ctx, symbol)
	if err != nil {
		return nil, err
	}

	orders := make([]*core.Order, 0, len(binanceOrders))
	for _, binanceOrder := range binanceOrders {
		order := OrderResponseToOrder(binanceOrder)
		orders = append(orders, order)
		// 注意：不在这里更新缓存，避免 InitFromRestAPI 中的死锁
		// InitFromRestAPI 会直接设置缓存数据
	}

	return orders, nil
}

// GetAccount 获取账户信息
func (e *LiveExecutor) GetAccount(ctx context.Context) (*core.Account, error) {
	// 如果 accountCache 为 nil，直接调用 API（向后兼容）
	if e.accountCache == nil {
		accountInfo, err := e.client.GetAccount(ctx)
		if err != nil {
			return nil, err
		}
		return AccountInfoToAccount(accountInfo), nil
	}

	// 优先从账户缓存读取（UserDataStream实时更新）
	balance := e.accountCache.GetBalance()
	if balance > 0 {
		// 从缓存构建账户信息
		return &core.Account{
			AvailableBalance: balance,
			TotalBalance:     balance,
			UpdateTime:       time.Now(),
		}, nil
	}

	// 如果缓存为空，从API获取（通常在初始化时）
	accountInfo, err := e.client.GetAccount(ctx)
	if err != nil {
		return nil, err
	}

	account := AccountInfoToAccount(accountInfo)
	// 注意：不在这里更新缓存，避免 InitFromRestAPI 中的死锁
	// InitFromRestAPI 会直接设置缓存数据
	return account, nil
}

// GetPositions 获取持仓信息
func (e *LiveExecutor) GetPositions(ctx context.Context) ([]*core.Position, error) {
	// 优先从账户缓存读取（UserDataStream实时更新）
	cachedPositions := e.accountCache.GetAllPositions()
	if len(cachedPositions) > 0 {
		return cachedPositions, nil
	}

	// 如果缓存为空，从API获取（通常在初始化时）
	positions, err := e.client.GetPositionRisk(ctx, "")
	if err != nil {
		return nil, err
	}

	result := make([]*core.Position, 0)
	for _, pos := range positions {
		corePos := PositionRiskToPosition(pos)
		if corePos != nil && corePos.Size > 0 { // 只返回有持仓的
			result = append(result, corePos)
			// 注意：不在这里更新缓存，避免 InitFromRestAPI 中的死锁
		}
	}

	return result, nil
}

// SetLeverage 设置杠杆
func (e *LiveExecutor) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	return e.client.SetLeverage(ctx, symbol, leverage)
}

// SetMarginMode 设置保证金模式
func (e *LiveExecutor) SetMarginMode(ctx context.Context, symbol string, mode core.MarginMode) error {
	marginType := FromMarginMode(mode)
	return e.client.SetMarginType(ctx, symbol, marginType)
}

// Close 关闭执行器
func (e *LiveExecutor) Close() error {
	// 不再需要清理本地缓存
	// AccountCache 由外部管理生命周期
	return nil
}

// ========== 辅助方法 ==========

// GetPosition 获取指定交易对的持仓
func (e *LiveExecutor) GetPosition(ctx context.Context, symbol string) (*core.Position, error) {
	// 先从账户缓存查询
	pos, exists := e.accountCache.GetPosition(symbol)
	if exists {
		return pos, nil
	}

	// 从API查询
	positions, err := e.client.GetPositionRisk(ctx, symbol)
	if err != nil {
		return nil, err
	}

	for _, pos := range positions {
		corePos := PositionRiskToPosition(pos)
		if corePos != nil && corePos.Size > 0 {
			// 注意：不在这里更新缓存，避免死锁
			return corePos, nil
		}
	}

	return nil, nil // 无持仓
}

// GetOrderBookPrice 获取订单簿第N档价格
func (e *LiveExecutor) GetOrderBookPrice(ctx context.Context, symbol, side string, level int) (float64, error) {
	return e.client.GetNthPriceFromOrderBook(ctx, symbol, side, level)
}

// GetMarkPrice 获取标记价格
func (e *LiveExecutor) GetMarkPrice(ctx context.Context, symbol string) (float64, error) {
	return e.client.GetMarkPrice(ctx, symbol)
}

// PlaceMarketOrder 快捷方法：下市价单
func (e *LiveExecutor) PlaceMarketOrder(ctx context.Context, symbol string, side core.OrderSide, quantity float64) (*core.Order, error) {
	order := &core.Order{
		Symbol:   symbol,
		Type:     core.OrderTypeMarket,
		Side:     side,
		Quantity: quantity,
	}
	return e.PlaceOrder(ctx, order)
}

// PlaceLimitOrder 快捷方法：下限价单
func (e *LiveExecutor) PlaceLimitOrder(ctx context.Context, symbol string, side core.OrderSide, price, quantity float64) (*core.Order, error) {
	order := &core.Order{
		Symbol:   symbol,
		Type:     core.OrderTypeLimit,
		Side:     side,
		Price:    price,
		Quantity: quantity,
	}
	return e.PlaceOrder(ctx, order)
}

// ClosePosition 平仓（市价单）
func (e *LiveExecutor) ClosePosition(ctx context.Context, symbol string) (*core.Order, error) {
	// 获取当前持仓
	position, err := e.GetPosition(ctx, symbol)
	if err != nil {
		return nil, err
	}

	if position == nil || position.Size == 0 {
		return nil, fmt.Errorf("no position to close")
	}

	// 确定平仓方向
	var side core.OrderSide
	if position.Side == core.PositionSideLong {
		side = core.OrderSideSell // 平多
	} else {
		side = core.OrderSideBuy // 平空
	}

	// 下市价平仓单
	order := &core.Order{
		Symbol:   symbol,
		Type:     core.OrderTypeMarket,
		Side:     side,
		Quantity: position.Size,
		Metadata: map[string]interface{}{
			"reduce_only": true,
		},
	}

	return e.PlaceOrder(ctx, order)
}

// CalculateQuantity 根据USDT金额计算数量
func (e *LiveExecutor) CalculateQuantity(ctx context.Context, symbol string, usdtAmount float64, leverage int) (float64, error) {
	// 获取当前标记价格
	markPrice, err := e.GetMarkPrice(ctx, symbol)
	if err != nil {
		return 0, err
	}

	// 计算数量 = (USDT金额 * 杠杆) / 价格
	quantity := (usdtAmount * float64(leverage)) / markPrice

	// 格式化到合适精度
	quantityStr := formatQuantity(symbol, quantity)
	return strconv.ParseFloat(quantityStr, 64)
}
