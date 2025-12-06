package position

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goQuant/internal/config"
	"goQuant/internal/core"
	"goQuant/internal/logger"

	"go.uber.org/zap"
)

// Manager 仓位管理器
type Manager struct {
	config    *config.PositionConfig
	executor  core.Executor
	mu        sync.RWMutex
	positions map[string]*PositionState // key: symbol
}

// PositionState 持仓状态（扩展信息）
type PositionState struct {
	*core.Position
	HighestProfit      float64   // 最高盈利（用于回撤计算）
	HighestProfitPrice float64   // 最高盈利时的价格
	StopLossPrice      float64   // 止损价格
	TrailingStopLevel  int       // 跟踪止盈级别（1-4）
	LastUpdateTime     time.Time // 最后更新时间
}

// NewManager 创建仓位管理器
func NewManager(cfg *config.PositionConfig, executor core.Executor) *Manager {
	return &Manager{
		config:    cfg,
		executor:  executor,
		positions: make(map[string]*PositionState),
	}
}

// ========== 实现 core.PositionManager 接口 ==========

// ProcessSignal 处理策略信号，生成订单
func (m *Manager) ProcessSignal(signal *core.TradingSignal, currentPrice float64) ([]*core.Order, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	symbol := signal.Symbol
	position, hasPosition := m.positions[symbol]

	// 1. 检查反向信号 - 需要先平仓
	if hasPosition {
		needClose := false
		if position.Side == core.PositionSideLong && (signal.Type == core.SignalTypeOpenShort) {
			needClose = true // 持有多单，出现空单信号
		} else if position.Side == core.PositionSideShort && (signal.Type == core.SignalTypeOpenLong) {
			needClose = true // 持有空单，出现多单信号
		}

		if needClose {
			// 先生成平仓订单
			closeOrder, err := m.createCloseOrder(symbol, position, currentPrice)
			if err != nil {
				return nil, fmt.Errorf("create close order: %w", err)
			}

			// 平仓后清除持仓状态
			delete(m.positions, symbol)

			// 再生成新的开仓订单
			openOrder, err := m.createOpenOrder(signal, currentPrice)
			if err != nil {
				return nil, fmt.Errorf("create open order: %w", err)
			}

			return []*core.Order{closeOrder, openOrder}, nil
		}
	}

	// 2. 处理开仓信号
	if signal.Type == core.SignalTypeOpenLong || signal.Type == core.SignalTypeOpenShort {
		if hasPosition {
			// 已有持仓，检查是否为强信号（可加仓）
			// Metadata 只能存储 float64，1.0 表示 true
			strongSignalValue, hasStrongSignal := signal.Metadata["strong_signal"]
			addPositionValue, hasAddEligible := signal.Metadata["add_position_eligible"]

			isStrongSignal := hasStrongSignal && strongSignalValue == 1.0
			canAddPosition := hasAddEligible && addPositionValue == 1.0

			if isStrongSignal && canAddPosition {
				// 检查方向是否一致
				if (signal.Type == core.SignalTypeOpenLong && position.Side == core.PositionSideLong) ||
					(signal.Type == core.SignalTypeOpenShort && position.Side == core.PositionSideShort) {
					// 方向一致且为强信号，执行加仓
					logger.Info("Strong signal detected, adding to position",
						zap.String("symbol", symbol),
						zap.String("signal_type", string(signal.Type)),
						zap.String("position_side", string(position.Side)),
					)

					order, err := m.createAddOrder(signal, currentPrice)
					if err != nil {
						return nil, err
					}
					return []*core.Order{order}, nil
				}
			}

			// 已有持仓但不是强信号或方向不一致，忽略
			return nil, nil
		}

		order, err := m.createOpenOrder(signal, currentPrice)
		if err != nil {
			return nil, err
		}

		return []*core.Order{order}, nil
	}

	// 3. 处理加仓信号
	if signal.Type == core.SignalTypeAddLong || signal.Type == core.SignalTypeAddShort {
		if !hasPosition {
			// 无持仓，无法加仓
			return nil, nil
		}

		// 检查方向是否一致
		if (signal.Type == core.SignalTypeAddLong && position.Side != core.PositionSideLong) ||
			(signal.Type == core.SignalTypeAddShort && position.Side != core.PositionSideShort) {
			// 方向不一致，忽略
			return nil, nil
		}

		order, err := m.createAddOrder(signal, currentPrice)
		if err != nil {
			return nil, err
		}

		return []*core.Order{order}, nil
	}

	// 4. 处理平仓信号
	if signal.Type == core.SignalTypeCloseLong || signal.Type == core.SignalTypeCloseShort {
		if !hasPosition {
			return nil, nil
		}

		order, err := m.createCloseOrder(symbol, position, currentPrice)
		if err != nil {
			return nil, err
		}

		delete(m.positions, symbol)
		return []*core.Order{order}, nil
	}

	return nil, nil
}

// UpdatePosition 更新持仓信息
func (m *Manager) UpdatePosition(position *core.Position) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if position.Size == 0 {
		// 持仓已清空
		delete(m.positions, position.Symbol)
		return nil
	}

	// 获取或创建持仓状态
	state, exists := m.positions[position.Symbol]
	if !exists {
		state = &PositionState{
			Position:           position,
			HighestProfit:      position.UnrealizedPnLPercent,
			HighestProfitPrice: position.CurrentPrice,
			StopLossPrice:      m.calculateStopLossPrice(position),
			TrailingStopLevel:  0,
			LastUpdateTime:     time.Now(),
		}
		m.positions[position.Symbol] = state
	} else {
		// 更新持仓信息
		state.Position = position
		state.LastUpdateTime = time.Now()

		// 更新最高盈利
		if position.UnrealizedPnLPercent > state.HighestProfit {
			state.HighestProfit = position.UnrealizedPnLPercent
			state.HighestProfitPrice = position.CurrentPrice
		}
	}

	return nil
}

// GetPosition 获取持仓
func (m *Manager) GetPosition(symbol string) (*core.Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.positions[symbol]
	if !exists {
		return nil, nil
	}

	return state.Position, nil
}

// GetAllPositions 获取所有持仓
func (m *Manager) GetAllPositions() ([]*core.Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	positions := make([]*core.Position, 0, len(m.positions))
	for _, state := range m.positions {
		positions = append(positions, state.Position)
	}

	return positions, nil
}

// CheckRisk 风险检查
func (m *Manager) CheckRisk(order *core.Order) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 1. 检查最大持仓数
	if len(m.positions) >= m.config.RiskLimits.MaxOpenPositions {
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

// CalculatePositionSize 计算仓位大小
func (m *Manager) CalculatePositionSize(signal *core.TradingSignal, accountBalance float64) (float64, error) {
	// 根据信号类型确定资金比例
	var percent float64
	switch signal.Type {
	case core.SignalTypeOpenLong, core.SignalTypeOpenShort:
		percent = 0.20 // 开仓使用20%
	case core.SignalTypeAddLong, core.SignalTypeAddShort:
		percent = 0.40 // 加仓使用40%
	default:
		return 0, fmt.Errorf("invalid signal type for position sizing: %s", signal.Type)
	}

	// 计算使用的资金
	usdtAmount := accountBalance * percent

	// 应用仓位大小限制
	maxUsdt := accountBalance * m.config.MaxPositionSize
	if usdtAmount > maxUsdt {
		usdtAmount = maxUsdt
	}

	return usdtAmount, nil
}

// ========== 私有方法 ==========

// createOpenOrder 创建开仓订单
func (m *Manager) createOpenOrder(signal *core.TradingSignal, currentPrice float64) (*core.Order, error) {
	// 获取账户余额
	ctx := context.Background()
	account, err := m.executor.GetAccount(ctx)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}

	// 计算仓位大小
	usdtAmount, err := m.CalculatePositionSize(signal, account.AvailableBalance)
	if err != nil {
		return nil, err
	}

	// 确定订单方向
	var side core.OrderSide
	var positionSide core.PositionSide
	if signal.Type == core.SignalTypeOpenLong {
		side = core.OrderSideBuy
		positionSide = core.PositionSideLong
	} else {
		side = core.OrderSideSell
		positionSide = core.PositionSideShort
	}

	// 记录仓位计算
	logger.Info("Creating order",
		zap.String("symbol", signal.Symbol),
		zap.String("signal_type", string(signal.Type)),
		zap.Float64("account_balance", account.AvailableBalance),
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
func (m *Manager) createAddOrder(signal *core.TradingSignal, currentPrice float64) (*core.Order, error) {
	// 加仓逻辑与开仓类似，只是比例不同（已在CalculatePositionSize中处理）
	return m.createOpenOrder(signal, currentPrice)
}

// createCloseOrder 创建平仓订单
func (m *Manager) createCloseOrder(symbol string, position *PositionState, currentPrice float64) (*core.Order, error) {
	// 确定平仓方向
	var side core.OrderSide
	if position.Side == core.PositionSideLong {
		side = core.OrderSideSell
	} else {
		side = core.OrderSideBuy
	}

	// 市价平仓
	order := &core.Order{
		Symbol:   symbol,
		Type:     core.OrderTypeMarket,
		Side:     side,
		Quantity: position.Size,
		Metadata: map[string]interface{}{
			"reduce_only":  true,
			"close_reason": "signal_triggered",
		},
	}

	return order, nil
}

// calculateStopLossPrice 计算止损价格
func (m *Manager) calculateStopLossPrice(position *core.Position) float64 {
	stopLossPercent := 0.006 // 0.6%

	if position.Side == core.PositionSideLong {
		// 多单：止损价 = 入场价 * (1 - 0.6%)
		return position.EntryPrice * (1 - stopLossPercent)
	} else {
		// 空单：止损价 = 入场价 * (1 + 0.6%)
		return position.EntryPrice * (1 + stopLossPercent)
	}
}

// CheckStopLoss 检查是否触发止损
func (m *Manager) CheckStopLoss(symbol string, currentPrice float64) (bool, *core.Order) {
	m.mu.RLock()
	state, exists := m.positions[symbol]
	m.mu.RUnlock()

	if !exists {
		return false, nil
	}

	// 检查固定止损
	triggered := false
	if state.Side == core.PositionSideLong && currentPrice <= state.StopLossPrice {
		triggered = true
	} else if state.Side == core.PositionSideShort && currentPrice >= state.StopLossPrice {
		triggered = true
	}

	if triggered {
		order, _ := m.createCloseOrder(symbol, state, currentPrice)
		if order != nil {
			order.Metadata["close_reason"] = "stop_loss"
		}

		m.mu.Lock()
		delete(m.positions, symbol)
		m.mu.Unlock()

		return true, order
	}

	return false, nil
}

// CheckTrailingStop 检查跟踪止盈
func (m *Manager) CheckTrailingStop(symbol string, currentPrice float64) (bool, *core.Order) {
	m.mu.Lock()
	defer m.mu.Unlock()

	state, exists := m.positions[symbol]
	if !exists {
		return false, nil
	}

	profit := state.UnrealizedPnLPercent

	// 确定当前级别和回撤阈值
	var level int
	var callbackPercent float64

	if profit > 4.8 {
		level = 4
		callbackPercent = 0.008 // 0.8%
	} else if profit > 1.8 {
		level = 3
		callbackPercent = 0.0068 // 0.68%
	} else if profit > 1.0 {
		level = 2
		callbackPercent = 0.0055 // 0.55%
	} else if profit > 0.6 {
		level = 1
		callbackPercent = 0.005 // 0.5%
	} else {
		// 盈利不足，不启用跟踪止盈
		return false, nil
	}

	// 如果进入更高级别，取消前一级
	if level > state.TrailingStopLevel {
		state.TrailingStopLevel = level
		state.HighestProfit = profit
		state.HighestProfitPrice = currentPrice
		return false, nil
	}

	// 计算回撤
	drawdown := state.HighestProfit - profit

	// 检查是否触发回撤止盈
	if drawdown >= callbackPercent*100 { // 转换为百分比
		order, _ := m.createCloseOrder(symbol, state, currentPrice)
		if order != nil {
			order.Metadata["close_reason"] = fmt.Sprintf("trailing_stop_level_%d", level)
			order.Metadata["highest_profit"] = state.HighestProfit
			order.Metadata["current_profit"] = profit
		}

		delete(m.positions, symbol)
		return true, order
	}

	return false, nil
}
