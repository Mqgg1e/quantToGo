package position

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goQuant/internal/cache"
	"goQuant/internal/config"
	"goQuant/internal/core"
	"goQuant/internal/logger"

	"go.uber.org/zap"
)

// Manager 仓位管理器
// 持仓信息由accountCache维护（通过UserDataStream实时更新）
// Manager只存储策略特定的元数据
type Manager struct {
	config       *config.PositionConfig
	executor     core.Executor
	accountCache *cache.AccountCache
	mu           sync.RWMutex

	// 只存储策略特定的元数据，不存储持仓本身
	metadata map[string]*PositionMetadata
}

// PositionMetadata 策略特定的持仓元数据
//type PositionMetadata struct {
//	HighestProfit      float64   // 最高盈利（用于跟踪止盈）
//	HighestProfitPrice float64   // 最高盈利价格
//	TrailingStopLevel  int       // 跟踪止盈级别
//	AddPositionCount   int       // 加仓次数
//	OpenTime           time.Time // 开仓时间
//}

type PositionMetadata struct {
	AddPositionCount int // 加仓次数（最多2次）
	OpenTime         time.Time
	StopLossOrderID  string // 止损单ID
	TrailingStopID   string // 跟踪止盈单ID

	LastSignalType core.SignalType // 最近一次处理的信号类型
	LastSignalTime time.Time       // 最近一次信号的时间
}

// NewManager 创建仓位管理器
func NewManager(cfg *config.PositionConfig, executor core.Executor, accountCache *cache.AccountCache) *Manager {
	return &Manager{
		config:       cfg,
		executor:     executor,
		accountCache: accountCache,
		metadata:     make(map[string]*PositionMetadata),
	}
}

// ========== 实现 core.PositionManager 接口 ==========

// ProcessSignal 处理策略信号，生成订单
func (m *Manager) ProcessSignal(signal *core.TradingSignal, currentPrice float64) ([]*core.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	symbol := signal.Symbol
	position, hasPosition := m.accountCache.GetPosition(symbol)

	var orders []*core.Order

	// 情况1: 无持仓 - 只处理开仓信号
	if !hasPosition {
		if signal.Type == core.SignalTypeOpenLong {
			order, err := m.createOpenOrder(signal, currentPrice, core.PositionSideLong)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order)

			m.metadata[symbol] = &PositionMetadata{
				OpenTime:         time.Now(),
				AddPositionCount: 0, // 初始加仓次数为0
			}

			logger.Info("Opening long position",
				zap.String("symbol", symbol),
				zap.Float64("price", currentPrice),
			)
		} else if signal.Type == core.SignalTypeOpenShort {
			order, err := m.createOpenOrder(signal, currentPrice, core.PositionSideShort)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order)

			m.metadata[symbol] = &PositionMetadata{
				OpenTime:         time.Now(),
				AddPositionCount: 0,
				LastSignalType:   signal.Type,
				LastSignalTime:   time.Now(),
			}

			logger.Info("Opening short position",
				zap.String("symbol", symbol),
				zap.Float64("price", currentPrice),
			)
		}

		return orders, nil
	}

	// 情况2: 有持仓
	meta := m.metadata[symbol]
	if meta == nil {
		meta = &PositionMetadata{
			OpenTime:         time.Now(),
			AddPositionCount: 0,
		}
		m.metadata[symbol] = meta
	}

	// ✅ 信号去重：如果相同信号在5分钟内重复，跳过
	signalInterval := 3 * time.Minute // 可根据你的周期调整（如15分钟K线用15分钟）
	if meta.LastSignalType == signal.Type && time.Since(meta.LastSignalTime) < signalInterval {
		logger.Info("Duplicate signal ignored",
			zap.String("symbol", symbol),
			zap.String("signal_type", string(signal.Type)),
			zap.Duration("time_since_last", time.Since(meta.LastSignalTime)),
		)
		return nil, nil
	}

	// 2.1 加仓（最多2次）
	if signal.AddPosition {
		// 检查加仓次数限制
		if meta.AddPositionCount >= 2 {
			logger.Warn("Max add position count reached",
				zap.String("symbol", symbol),
				zap.Int("current_count", meta.AddPositionCount),
				zap.String("add reason", signal.Reason),
			)
			return nil, nil
		}

		// 检查方向一致性
		if (position.Side == core.PositionSideLong && signal.Type == core.SignalTypeOpenLong) ||
			(position.Side == core.PositionSideShort && signal.Type == core.SignalTypeOpenShort) {

			order, err := m.createAddOrder(signal, currentPrice, position.Side)
			if err != nil {
				return nil, err
			}
			orders = append(orders, order)

			// 增加加仓次数
			meta.AddPositionCount++
			meta.LastSignalType = signal.Type // ✅ 更新信号类型
			meta.LastSignalTime = time.Now()

			logger.Info("Adding to position",
				zap.String("symbol", symbol),
				zap.String("side", string(position.Side)),
				zap.Int("add_count", meta.AddPositionCount),
			)

			return orders, nil
		}
	}

	// 2.2 反向信号：平仓 + 开反向仓
	if (position.Side == core.PositionSideLong && signal.Type == core.SignalTypeOpenShort) ||
		(position.Side == core.PositionSideShort && signal.Type == core.SignalTypeOpenLong) {

		// 先取消所有挂单
		if err := m.cancelAllOrders(context.Background(), symbol); err != nil {
			logger.Error("Failed to cancel orders before reverse",
				zap.String("symbol", symbol),
				zap.Error(err),
			)
		}

		// 平仓
		closeOrder := m.createCloseOrder(symbol, position, currentPrice)
		orders = append(orders, closeOrder)

		// 开反向仓
		var newSide core.PositionSide
		if signal.Type == core.SignalTypeOpenShort {
			newSide = core.PositionSideShort
		} else {
			newSide = core.PositionSideLong
		}

		openOrder, err := m.createOpenOrder(signal, currentPrice, newSide)
		if err != nil {
			return nil, err
		}
		orders = append(orders, openOrder)

		// 重置 metadata
		//m.metadata[symbol] = &PositionMetadata{
		//	OpenTime:         time.Now(),
		//	AddPositionCount: 0,
		//}
		meta.AddPositionCount = 0
		meta.LastSignalType = signal.Type // ✅ 更新信号类型
		meta.LastSignalTime = time.Now()

		logger.Info("Reversing position",
			zap.String("symbol", symbol),
			zap.String("from", string(position.Side)),
			zap.String("to", string(newSide)),
		)
	}

	return orders, nil
}

// UpdatePosition 更新持仓元数据（持仓本身由cache管理）
func (m *Manager) UpdatePosition(position *core.Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 持仓清空时删除元数据
	if position.Size == 0 {
		delete(m.metadata, position.Symbol)
		return nil
	}

	meta := m.metadata[position.Symbol]
	if meta == nil {
		meta = &PositionMetadata{
			OpenTime:         time.Now(),
			AddPositionCount: 0,
		}
		m.metadata[position.Symbol] = meta
	}

	return nil
}

// GetPosition 获取持仓（从cache）
func (m *Manager) GetPosition(symbol string) (*core.Position, error) {
	position, exists := m.accountCache.GetPosition(symbol)
	if !exists {
		return nil, nil
	}
	return position, nil
}

// GetAllPositions 获取所有持仓（从cache）
func (m *Manager) GetAllPositions() ([]*core.Position, error) {
	return m.accountCache.GetAllPositions(), nil
}

// CheckRisk 风险检查
func (m *Manager) CheckRisk(order *core.Order) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. 检查最大持仓数
	allPositions := m.accountCache.GetAllPositions()
	if len(allPositions) >= m.config.RiskLimits.MaxOpenPositions {
		return fmt.Errorf("max open positions reached: %d", m.config.RiskLimits.MaxOpenPositions)
	}

	// 2. 检查杠杆范围
	if order.Leverage < 1 || order.Leverage > 125 {
		return fmt.Errorf("invalid leverage: %d", order.Leverage)
	}

	// 3. 检查订单数量
	if order.Quantity <= 0 {
		return fmt.Errorf("invalid quantity: %f", order.Quantity)
	}

	return nil
}

// UpdateRiskOrders 更新风控订单（加仓后调用）
func (m *Manager) UpdateRiskOrders(ctx context.Context, symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	position, exists := m.accountCache.GetPosition(symbol)
	if !exists {
		return fmt.Errorf("position not found: %s", symbol)
	}

	meta := m.metadata[symbol]
	if meta == nil {
		return fmt.Errorf("metadata not found: %s", symbol)
	}

	logger.Info("Updating risk orders after add position",
		zap.String("symbol", symbol),
		zap.Float64("new_size", position.Size),
	)

	// 1. 取消旧的止损单和跟踪止盈单
	if err := m.cancelAllOrders(ctx, symbol); err != nil {
		logger.Error("Failed to cancel old risk orders",
			zap.String("symbol", symbol),
			zap.Error(err),
		)
		// 继续执行，不中断
	}

	// 2. 设置新的止损单和跟踪止盈单（使用最新的仓位数量）
	return m.setRiskOrders(ctx, symbol, position, meta)
}

// setRiskOrders 设置风控订单（内部方法，不加锁）
func (m *Manager) setRiskOrders(ctx context.Context, symbol string, position *core.Position, meta *PositionMetadata) error {
	// 确定平仓方向
	var stopSide core.OrderSide
	if position.Side == core.PositionSideLong {
		stopSide = core.OrderSideSell
	} else {
		stopSide = core.OrderSideBuy
	}

	// 1. 设置固定止损单
	stopLossPrice := m.calculateStopLossPrice(position)
	stopOrder := &core.Order{
		Symbol:    symbol,
		Type:      core.OrderTypeStopMarket,
		Leverage:  m.config.DefaultLeverage,
		Side:      stopSide,
		Quantity:  position.Size, // ✅ 使用最新仓位数量
		StopPrice: stopLossPrice,
		Metadata: map[string]interface{}{
			"stop_price": stopLossPrice,
		},
	}

	logger.Info("Placing updated stop loss order",
		zap.String("symbol", symbol),
		zap.Float64("quantity", position.Size),
		zap.Float64("stop_price", stopLossPrice),
	)

	stopResult, err := m.executor.PlaceOrder(ctx, stopOrder)
	if err != nil {
		return fmt.Errorf("place stop loss failed: %w", err)
	}
	meta.StopLossOrderID = stopResult.ID

	// 2. 设置跟踪止盈单
	activationPrice := m.calculateTrailingActivationPrice(position)
	trailingOrder := &core.Order{
		Symbol:          symbol,
		Type:            core.OrderTypeTrailingStop,
		Leverage:        m.config.DefaultLeverage,
		Side:            stopSide,
		Quantity:        position.Size, // ✅ 使用最新仓位数量
		ActivationPrice: activationPrice,
		Metadata: map[string]interface{}{
			"callback_rate": 0.6,
		},
	}

	logger.Info("Placing updated trailing stop order",
		zap.String("symbol", symbol),
		zap.Float64("quantity", position.Size),
		zap.Float64("activation_price", activationPrice),
	)

	trailingResult, err := m.executor.PlaceOrder(ctx, trailingOrder)
	if err != nil {
		return fmt.Errorf("place trailing stop failed: %w", err)
	}
	meta.TrailingStopID = trailingResult.ID

	logger.Info("Risk orders updated successfully",
		zap.String("symbol", symbol),
		zap.String("stop_order_id", meta.StopLossOrderID),
		zap.String("trailing_order_id", meta.TrailingStopID),
	)

	return nil
}

// CalculatePositionSize 计算仓位大小
func (m *Manager) CalculatePositionSize(signal *core.TradingSignal, accountBalance float64) (float64, error) {
	// 根据信号类型确定资金比例
	var percent float64
	switch signal.Type {
	case core.SignalTypeOpenLong, core.SignalTypeOpenShort:
		percent = m.config.PositionSizing.OpenPercent // 使用配置的开仓比例
	case core.SignalTypeAddLong, core.SignalTypeAddShort:
		percent = m.config.PositionSizing.AddPercent // 使用配置的加仓比例
	default:
		return 0, fmt.Errorf("invalid signal type for position sizing: %s", signal.Type)
	}

	// 计算使用的资金
	usdtAmount := accountBalance * percent * m.config.MaxPositionSize

	// 应用仓位大小限制
	//maxUsdt := accountBalance * m.config.MaxPositionSize
	//if usdtAmount > maxUsdt {
	//	usdtAmount = maxUsdt
	//}

	return usdtAmount, nil
}

// ========== 私有方法 ==========

// createOpenOrder 创建开仓订单
func (m *Manager) createOpenOrder(signal *core.TradingSignal, currentPrice float64, positionSide core.PositionSide) (*core.Order, error) {
	// 从账户缓存获取余额（UserDataStream实时更新）
	accountBalance := m.accountCache.GetBalance()
	if accountBalance == 0 {
		return nil, fmt.Errorf("account balance is zero or not initialized")
	}

	// 计算仓位大小
	usdtAmount, err := m.CalculatePositionSize(signal, accountBalance)
	if err != nil {
		return nil, err
	}

	// 确定订单方向
	var side core.OrderSide
	//var positionSide core.PositionSide
	if signal.Type == core.SignalTypeOpenLong {
		side = core.OrderSideBuy
		//positionSide = core.PositionSideLong
	} else {
		side = core.OrderSideSell
		//positionSide = core.PositionSideShort
	}

	// 记录仓位计算
	logger.Info("Creating order",
		zap.String("symbol", signal.Symbol),
		zap.String("signal_type", string(signal.Type)),
		zap.Float64("account_balance", accountBalance),
		zap.Float64("usdt_amount", usdtAmount),
		zap.Float64("current_price", currentPrice),
		zap.Int("leverage", m.config.DefaultLeverage),
		zap.String("margin_mode", string(m.config.DefaultMarginMode)),
	)

	// 计算订单数量：quantity = (usdt_amount * leverage) / price
	leverage := float64(m.config.DefaultLeverage)
	quantity := (usdtAmount * leverage) / currentPrice

	logger.Info("Calculated order quantity",
		zap.Float64("quantity", quantity),
		zap.Float64("usdt_amount", usdtAmount),
		zap.Float64("leverage", leverage),
		zap.Float64("price", currentPrice),
	)

	// 根据策略要求使用市价单
	order := &core.Order{
		Symbol:     signal.Symbol,
		Type:       core.OrderTypeMarket,
		Side:       side,
		Quantity:   quantity, // 已经计算好数量
		Leverage:   m.config.DefaultLeverage,
		MarginMode: m.config.DefaultMarginMode,
		Metadata: map[string]interface{}{
			"usdt_amount":   usdtAmount,
			"position_side": positionSide,
			"signal_type":   signal.Type,
			"signal_reason": signal.Reason,
		},
	}

	return order, nil
}

// createAddOrder 创建加仓订单
func (m *Manager) createAddOrder(signal *core.TradingSignal, currentPrice float64, positionSide core.PositionSide) (*core.Order, error) {
	// 加仓逻辑与开仓类似，只是比例不同（已在CalculatePositionSize中处理）
	return m.createOpenOrder(signal, currentPrice, positionSide)
}

// createCloseOrder 创建平仓订单
func (m *Manager) createCloseOrder(symbol string, position *core.Position, currentPrice float64) *core.Order {
	// 确定平仓方向
	var side core.OrderSide
	if position.Side == core.PositionSideLong {
		side = core.OrderSideSell
	} else {
		side = core.OrderSideBuy
	}

	// 市价平仓
	order := &core.Order{
		Symbol:     symbol,
		Type:       core.OrderTypeMarket,
		Side:       side,
		Quantity:   position.Size,
		Leverage:   position.Leverage,
		MarginMode: position.MarginMode,
		Metadata: map[string]interface{}{
			"reduce_only":  true,
			"close_reason": "signal_triggered",
		},
	}

	return order
}

// SetStopLoss 开仓后立即设置止损单和跟踪止盈单
func (m *Manager) SetStopLoss(ctx context.Context, symbol string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	position, exists := m.accountCache.GetPosition(symbol)
	if !exists {
		return fmt.Errorf("position not found: %s", symbol)
	}

	meta := m.metadata[symbol]
	if meta == nil {
		meta = &PositionMetadata{
			OpenTime:         time.Now(),
			AddPositionCount: 0,
		}
		m.metadata[symbol] = meta
	}

	// 如果已经有止损单和跟踪止盈单，先取消
	if meta.StopLossOrderID != "" || meta.TrailingStopID != "" {
		logger.Info("Cancelling existing risk orders before setting new ones",
			zap.String("symbol", symbol),
		)
		err := m.cancelAllOrders(ctx, symbol)
		if err != nil {
			return err
		}
	}
	//
	//// 1. 设置固定止损单（入场价 ± 0.6%）
	//stopLossPrice := m.calculateStopLossPrice(position)
	//var stopSide core.OrderSide
	//if position.Side == core.PositionSideLong {
	//	stopSide = core.OrderSideSell
	//} else {
	//	stopSide = core.OrderSideBuy
	//}
	//
	//stopOrder := &core.Order{
	//	Symbol:    symbol,
	//	Type:      core.OrderTypeStopMarket,
	//	Leverage:  m.config.DefaultLeverage,
	//	Side:      stopSide,
	//	Quantity:  position.Size,
	//	StopPrice: stopLossPrice,
	//	Metadata: map[string]interface{}{
	//		"stop_price": stopLossPrice,
	//		//"close_position": true,
	//	},
	//}
	//
	//logger.Info("Placing stop loss order",
	//	zap.String("symbol", symbol),
	//	zap.Float64("entry", position.EntryPrice),
	//	zap.Float64("stop_price", stopLossPrice),
	//)
	//
	//stopResult, err := m.executor.PlaceOrder(ctx, stopOrder)
	//if err != nil {
	//	return fmt.Errorf("place stop loss failed: %w", err)
	//}
	//meta.StopLossOrderID = stopResult.ID
	//
	//// 2. 设置跟踪止盈单（激活价变化0.6%，回撤0.6%）
	//activationPrice := m.calculateTrailingActivationPrice(position)
	//
	//trailingOrder := &core.Order{
	//	Symbol:          symbol,
	//	Type:            core.OrderTypeTrailingStop,
	//	Leverage:        m.config.DefaultLeverage,
	//	Side:            stopSide,
	//	Quantity:        position.Size,
	//	ActivationPrice: activationPrice,
	//	Metadata: map[string]interface{}{
	//		"callback_rate": 0.6, // 回撤0.6%
	//		//"activation_price": activationPrice,
	//		//"close_position":   true,
	//	},
	//}
	//
	//logger.Info("Placing trailing stop order",
	//	zap.String("symbol", symbol),
	//	zap.Float64("activation_price", activationPrice),
	//	zap.Float64("callback_rate", 0.6),
	//)
	//
	//trailingResult, err := m.executor.PlaceOrder(ctx, trailingOrder)
	//if err != nil {
	//	return fmt.Errorf("place trailing stop failed: %w", err)
	//}
	//meta.TrailingStopID = trailingResult.ID
	//
	//logger.Info("Risk orders placed successfully",
	//	zap.String("symbol", symbol),
	//	zap.String("stop_order_id", meta.StopLossOrderID),
	//	zap.String("trailing_order_id", meta.TrailingStopID),
	//)

	return m.setRiskOrders(ctx, symbol, position, meta)
}

// cancelAllOrders 取消所有挂单（内部方法，不加锁）
func (m *Manager) cancelAllOrders(ctx context.Context, symbol string) error {
	meta := m.metadata[symbol]
	if meta == nil {
		return nil
	}

	// 取消止损单
	if meta.StopLossOrderID != "" {
		err := m.executor.CancelOrder(ctx, symbol, meta.StopLossOrderID)
		if err != nil {
			logger.Error("Failed to cancel stop loss order",
				zap.String("symbol", symbol),
				zap.String("order_id", meta.StopLossOrderID),
				zap.Error(err),
			)
		} else {
			logger.Info("Cancelled stop loss order", zap.String("symbol", symbol))
			meta.StopLossOrderID = ""
		}
	}

	// 取消跟踪止盈单
	if meta.TrailingStopID != "" {
		err := m.executor.CancelOrder(ctx, symbol, meta.TrailingStopID)
		if err != nil {
			logger.Error("Failed to cancel trailing stop order",
				zap.String("symbol", symbol),
				zap.String("order_id", meta.TrailingStopID),
				zap.Error(err),
			)
		} else {
			logger.Info("Cancelled trailing stop order", zap.String("symbol", symbol))
			meta.TrailingStopID = ""
		}
	}

	return nil
}

func (m *Manager) calculateStopLossPrice(position *core.Position) float64 {
	percent := 0.006 // 0.6%
	if position.Side == core.PositionSideLong {
		return position.EntryPrice * (1 - percent)
	}
	return position.EntryPrice * (1 + percent)
}

func (m *Manager) calculateTrailingActivationPrice(position *core.Position) float64 {
	percent := 0.006 // 0.6%
	if position.Side == core.PositionSideLong {
		return position.EntryPrice * (1 + percent)
	}
	return position.EntryPrice * (1 - percent)
}

// setTrailingStopOrder 设置跟踪止盈单
//func (m *Manager) setTrailingStopOrder(symbol string, position *core.Position, callbackRate float64) error {
//	ctx := context.Background()
//
//	// 确定平仓方向
//	var side core.OrderSide
//	if position.Side == core.PositionSideLong {
//		side = core.OrderSideSell
//	} else {
//		side = core.OrderSideBuy
//	}
//
//	// 创建 TRAILING_STOP_MARKET 订单
//	// 注意：Binance 的 callbackRate 需要乘以100（0.5% = 0.5）
//	order := &core.Order{
//		Symbol:   symbol,
//		Type:     core.OrderTypeTrailingStop,
//		Side:     side,
//		Quantity: position.Size,
//		Metadata: map[string]interface{}{
//			"callback_rate":  callbackRate, // 0.5, 0.55, 0.68
//			"close_position": true,         // 平掉全部仓位
//			"order_type":     "trailing_stop",
//		},
//	}
//
//	logger.Info("Setting trailing stop order",
//		zap.String("symbol", symbol),
//		zap.String("side", string(side)),
//		zap.Float64("quantity", order.Quantity),
//		zap.Float64("callback_rate", callbackRate),
//	)
//
//	// 提交订单
//	_, err := m.executor.PlaceOrder(ctx, order)
//	if err != nil {
//		return fmt.Errorf("place trailing stop order failed: %w", err)
//	}
//
//	logger.Info("Trailing stop order placed successfully",
//		zap.String("symbol", symbol),
//		zap.Float64("callback_rate", callbackRate),
//	)
//
//	return nil
//}

// SetStopLoss 设置止损单（开仓或加仓后调用）
//func (m *Manager) SetStopLoss(ctx context.Context, symbol string) error {
//	position, exists := m.accountCache.GetPosition(symbol)
//	if !exists {
//		return fmt.Errorf("position not found for symbol: %s", symbol)
//	}
//
//	logger.Debug("SetStopLoss called",
//		zap.String("symbol", symbol),
//		zap.Float64("size", position.Size),
//	)
//
//	// 通用Manager使用价格检查方式进行止损，不设置交易所止损单
//	// 止损由CheckStopLoss方法在价格更新时检查并触发
//
//	return nil
//}
