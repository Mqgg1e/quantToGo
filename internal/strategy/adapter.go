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

	// 如果是市价单，等待一下再查询状态
	if resultOrder.Type == core.OrderTypeMarket {
		// 简化：实际应该通过WebSocket监听订单状态
		order, _ := a.executor.GetOrder(ctx, resultOrder.Symbol, resultOrder.ID)
		if order != nil && order.Status == core.OrderStatusFilled {
			a.symbolLogger.Info("Order filled",
				zap.String("order_id", order.ID),
				zap.Float64("avg_price", order.AvgPrice),
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
	// 这里需要类型断言，因为core.PositionManager接口没有这些方法
	// 实际使用时需要扩展接口或使用具体类型

	// 示例：假设positionMgr实现了这些方法
	// stopLossTriggered, stopLossOrder := a.positionMgr.CheckStopLoss(a.symbol, currentPrice)
	// if stopLossTriggered && stopLossOrder != nil {
	//     log.Printf("[%s] 🛑 Stop loss triggered!\n", a.Name())
	//     a.executeOrder(ctx, stopLossOrder)
	// }

	// trailingStopTriggered, trailingStopOrder := a.positionMgr.CheckTrailingStop(a.symbol, currentPrice)
	// if trailingStopTriggered && trailingStopOrder != nil {
	//     log.Printf("[%s] 📉 Trailing stop triggered!\n", a.Name())
	//     a.executeOrder(ctx, trailingStopOrder)
	// }
}

// SetEnableRiskCheck 设置是否启用风险检查
func (a *Adapter) SetEnableRiskCheck(enable bool) {
	a.enableRiskCheck = enable
}
