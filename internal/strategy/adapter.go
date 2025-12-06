package strategy

import (
	"context"
	"fmt"
	"time"

	"goQuant/internal/core"
	v2 "goQuant/internal/dataManager/v2"
	"goQuant/internal/logger"

	"go.uber.org/zap"
)

// Adapter 策略适配器，实现v2.KlineSubscriber接口
// 将DataManager的K线数据流连接到Strategy
type Adapter struct {
	strategy    core.Strategy
	positionMgr core.PositionManager
	executor    core.Executor
	symbol      string
	interval    string

	// 止损止盈检查
	enableRiskCheck bool

	// 日志记录器
	tradingLog   *logger.TradingLogger
	symbolLogger *zap.Logger // 按品种的专用日志
}

// NewAdapter 创建策略适配器
func NewAdapter(
	strategy core.Strategy,
	positionMgr core.PositionManager,
	executor core.Executor,
	symbol, interval string,
) *Adapter {
	// 获取按品种分文件的日志记录器
	symbolLogger := logger.GetSymbolLogger().GetLogger(symbol, interval)

	return &Adapter{
		strategy:        strategy,
		positionMgr:     positionMgr,
		executor:        executor,
		symbol:          symbol,
		interval:        interval,
		enableRiskCheck: true,
		tradingLog:      logger.NewTradingLogger(),
		symbolLogger:    symbolLogger,
	}
}

// ========== 实现 v2.KlineSubscriber 接口 ==========

// OnKline 处理K线数据
func (a *Adapter) OnKline(kline *v2.KlineData) {
	ctx := context.Background()

	// 记录K线接收（同时记录到品种专用日志）
	a.symbolLogger.Info("Kline received",
		zap.Time("open_time", time.UnixMilli(kline.GetStartTime().UnixMilli())),
		zap.Float64("open", kline.GetOpenPrice()),
		zap.Float64("high", kline.GetHighPrice()),
		zap.Float64("low", kline.GetLowPrice()),
		zap.Float64("close", kline.GetClosePrice()),
		zap.Float64("volume", kline.GetVolume()),
	)

	// 1. 将K线传递给策略，生成信号
	signal, err := a.strategy.OnKline(kline)
	if err != nil {
		a.symbolLogger.Error("Strategy error", zap.Error(err))
		return
	}

	// 无操作信号，只做风险检查
	if signal.Type == core.SignalTypeNoAction {
		if a.enableRiskCheck {
			a.checkRiskManagement(ctx, kline.GetClosePrice())
		}
		return
	}

	// 2. 记录信号
	a.symbolLogger.Info("Signal generated",
		zap.String("signal_type", string(signal.Type)),
		zap.Float64("price", signal.Price),
		zap.Float64("confidence", signal.Confidence),
		zap.String("reason", signal.Reason),
	)

	// 3. 仓位管理器处理信号，生成订单
	orders, err := a.positionMgr.ProcessSignal(signal, kline.GetClosePrice())
	if err != nil {
		a.symbolLogger.Error("Position manager error", zap.Error(err))
		return
	}

	if len(orders) == 0 {
		a.symbolLogger.Info("No orders generated")
		return
	}

	// 4. 执行订单
	for _, order := range orders {
		err := a.executeOrder(ctx, order)
		if err != nil {
			a.symbolLogger.Error("Execute order failed",
				zap.String("symbol", order.Symbol),
				zap.Error(err),
			)
		}
	}

	// 5. 更新持仓状态
	a.updatePositions(ctx)
}

// OnError 处理错误
func (a *Adapter) OnError(err error) {
	logger.Error("Data stream error",
		zap.String("adapter", a.Name()),
		zap.Error(err),
	)
}

// Name 返回适配器名称
func (a *Adapter) Name() string {
	return fmt.Sprintf("StrategyAdapter_%s_%s_%s", a.strategy.Name(), a.symbol, a.interval)
}

// ========== 私有方法 ==========

// executeOrder 执行订单
func (a *Adapter) executeOrder(ctx context.Context, order *core.Order) error {
	// 风险检查（此时数量已经计算好）
	if err := a.positionMgr.CheckRisk(order); err != nil {
		return fmt.Errorf("risk check failed: %w", err)
	}

	// 保存原始订单类型信息（用于判断是否需要设置止损）
	isOpenOrAddOrder := false
	if order.Metadata != nil {
		if signalType, ok := order.Metadata["signal_type"].(core.SignalType); ok {
			if signalType == core.SignalTypeOpenLong || signalType == core.SignalTypeOpenShort {
				isOpenOrAddOrder = true
			}
		}
	}

	// 提交订单
	resultOrder, err := a.executor.PlaceOrder(ctx, order)
	if err != nil {
		return fmt.Errorf("place order failed: %w", err)
	}

	// 记录订单信息（使用品种专用日志）
	a.symbolLogger.Info("Order placed",
		zap.String("order_id", fmt.Sprintf("%s_%d", resultOrder.Symbol, time.Now().Unix())),
		zap.String("side", string(resultOrder.Side)),
		zap.String("type", string(resultOrder.Type)),
		zap.Float64("quantity", resultOrder.Quantity),
		zap.Float64("price", resultOrder.Price),
		zap.String("status", string(resultOrder.Status)),
	)

	// 如果是市价单且是开仓/加仓订单，等待成交后设置止损
	if resultOrder.Type == core.OrderTypeMarket && isOpenOrAddOrder {
		// 等待订单成交（市价单通常很快成交）
		time.Sleep(500 * time.Millisecond)

		// 查询订单状态
		filledOrder, err := a.executor.GetOrder(ctx, resultOrder.Symbol, resultOrder.ID)
		if err != nil {
			a.symbolLogger.Warn("Failed to query order status",
				zap.String("order_id", resultOrder.ID),
				zap.Error(err),
			)
			// 即使查询失败，也尝试设置止损
		}

		// 检查订单是否成交
		if filledOrder != nil && filledOrder.Status == core.OrderStatusFilled {
			a.symbolLogger.Info("Order filled",
				zap.String("order_id", filledOrder.ID),
				zap.Float64("avg_price", filledOrder.AvgPrice),
			)
		}

		// 开仓/加仓订单：设置止损单
		a.symbolLogger.Info("Setting stop loss after order placed",
			zap.String("symbol", resultOrder.Symbol),
		)

		// 先更新持仓，确保仓位管理器有最新数据
		a.updatePositions(ctx)

		// 设置止损单
		if err := a.positionMgr.SetStopLoss(ctx, resultOrder.Symbol); err != nil {
			a.symbolLogger.Error("Failed to set stop loss",
				zap.String("symbol", resultOrder.Symbol),
				zap.Error(err),
			)
		} else {
			a.symbolLogger.Info("Stop loss order set successfully",
				zap.String("symbol", resultOrder.Symbol),
			)
		}
	}

	return nil
}

// updatePositions 更新持仓状态
func (a *Adapter) updatePositions(ctx context.Context) {
	positions, err := a.executor.GetPositions(ctx)
	if err != nil {
		a.symbolLogger.Warn("Failed to get positions", zap.Error(err))
		return
	}

	for _, pos := range positions {
		if pos.Symbol == a.symbol {
			err := a.positionMgr.UpdatePosition(pos)
			if err != nil {
				logger.Warn("Failed to update position",
					zap.String("adapter", a.Name()),
					zap.Error(err),
				)
			} else {
				// 记录仓位更新（使用品种专用日志）
				side := "LONG"
				if pos.Size < 0 {
					side = "SHORT"
				}
				a.symbolLogger.Info("Position update",
					zap.String("side", side),
					zap.Float64("size", pos.Size),
					zap.Float64("entry_price", pos.EntryPrice),
					zap.Float64("current_price", pos.CurrentPrice),
					zap.Float64("unrealized_pnl", pos.UnrealizedPnL),
					zap.Float64("pnl_percent", pos.UnrealizedPnLPercent),
				)
			}
		}
	}
}

// checkRiskManagement 风险管理检查（止损/止盈）
func (a *Adapter) checkRiskManagement(ctx context.Context, currentPrice float64) {
	// 程序内止损监控（作为交易所止损单的后备方案）
	// 如果 SetStopLoss 设置交易所止损单失败，这里可以作为软件层面的保护

	// 类型断言获取具体的 Manager 类型以访问 CheckStopLoss 方法
	type StopLossChecker interface {
		CheckStopLoss(symbol string, currentPrice float64) (bool, *core.Order)
		CheckTrailingStop(symbol string, currentPrice float64) (bool, *core.Order)
	}

	if checker, ok := a.positionMgr.(StopLossChecker); ok {
		// 检查固定止损
		stopLossTriggered, stopLossOrder := checker.CheckStopLoss(a.symbol, currentPrice)
		if stopLossTriggered && stopLossOrder != nil {
			a.symbolLogger.Warn("Stop loss triggered by program monitor",
				zap.Float64("current_price", currentPrice),
			)
			if err := a.executeOrder(ctx, stopLossOrder); err != nil {
				a.symbolLogger.Error("Failed to execute stop loss order",
					zap.Error(err),
				)
			}
		}

		// 检查跟踪止盈
		trailingStopTriggered, trailingStopOrder := checker.CheckTrailingStop(a.symbol, currentPrice)
		if trailingStopTriggered && trailingStopOrder != nil {
			a.symbolLogger.Info("Trailing stop triggered",
				zap.Float64("current_price", currentPrice),
			)
			if err := a.executeOrder(ctx, trailingStopOrder); err != nil {
				a.symbolLogger.Error("Failed to execute trailing stop order",
					zap.Error(err),
				)
			}
		}
	}
}

// SetEnableRiskCheck 设置是否启用风险检查
func (a *Adapter) SetEnableRiskCheck(enable bool) {
	a.enableRiskCheck = enable
}
