package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"goQuant/internal/core"
	"goQuant/internal/logger"

	"go.uber.org/zap"
)

// AccountCache 账户状态缓存
// 维护账户余额、持仓、订单的内存缓存
// 线程安全，支持并发读写
type AccountCache struct {
	mu             sync.RWMutex
	balance        float64                   // 账户余额 (USDT)
	positions      map[string]*core.Position // 持仓 key: symbol
	orders         map[string]*core.Order    // 订单 key: clientOrderID
	lastUpdateTime time.Time                 // 最后更新时间
	updateVersion  int64                     // 更新版本号（防止乱序）
}

// Executor 执行器接口（避免循环依赖）
type Executor interface {
	GetAccount(ctx context.Context) (*core.Account, error)
	GetPositions(ctx context.Context) ([]*core.Position, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]*core.Order, error)
}

// NewAccountCache 创建账户缓存实例
func NewAccountCache() *AccountCache {
	return &AccountCache{
		balance:   0,
		positions: make(map[string]*core.Position),
		orders:    make(map[string]*core.Order),
	}
}

// ========== 初始化方法 ==========

// InitFromRestAPI 从 REST API 全量同步账户信息
// 在启动时或重连后调用，确保缓存与交易所同步
func (c *AccountCache) InitFromRestAPI(ctx context.Context, executor Executor) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	logger.Info("Initializing account cache from REST API...")

	// 1. 获取账户信息（余额）
	account, err := executor.GetAccount(ctx)
	if err != nil {
		return fmt.Errorf("get account: %w", err)
	}
	c.balance = account.AvailableBalance
	logger.Info("Account balance loaded", zap.Float64("balance", c.balance))

	// 2. 获取所有持仓
	positions, err := executor.GetPositions(ctx)
	if err != nil {
		return fmt.Errorf("get positions: %w", err)
	}

	// 清空并重建持仓缓存
	c.positions = make(map[string]*core.Position)
	for _, pos := range positions {
		if pos.Size != 0 { // 只缓存有持仓的
			c.positions[pos.Symbol] = pos
			logger.Info("Position loaded",
				zap.String("symbol", pos.Symbol),
				zap.String("side", string(pos.Side)),
				zap.Float64("size", pos.Size),
				zap.Float64("entry_price", pos.EntryPrice),
			)
		}
	}

	// 3. 获取所有未成交订单
	// 注意：需要遍历所有交易对（或传入空字符串获取全部）
	orders, err := executor.GetOpenOrders(ctx, "")
	if err != nil {
		return fmt.Errorf("get open orders: %w", err)
	}

	// 清空并重建订单缓存
	c.orders = make(map[string]*core.Order)
	for _, order := range orders {
		c.orders[order.ID] = order
		logger.Info("Open order loaded",
			zap.String("symbol", order.Symbol),
			zap.String("order_id", order.ID),
			zap.String("type", string(order.Type)),
			zap.String("side", string(order.Side)),
		)
	}

	c.lastUpdateTime = time.Now()
	c.updateVersion++

	logger.Info("Account cache initialized successfully",
		zap.Float64("balance", c.balance),
		zap.Int("positions", len(c.positions)),
		zap.Int("orders", len(c.orders)),
		zap.Int64("version", c.updateVersion),
	)

	return nil
}

// ========== 余额管理 ==========

// GetBalance 获取账户余额
func (c *AccountCache) GetBalance() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.balance
}

// UpdateBalance 更新账户余额
// version: 更新版本号，用于防止乱序更新
func (c *AccountCache) UpdateBalance(balance float64, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 版本检查：只忽略明显过时的更新（严格小于）
	// 允许同一版本号的多个更新（币安同一交易可能推送多个事件）
	if version < c.updateVersion {
		logger.Debug("Ignoring outdated balance update",
			zap.Int64("current_version", c.updateVersion),
			zap.Int64("update_version", version),
		)
		return
	}

	c.balance = balance
	c.updateVersion = version
	c.lastUpdateTime = time.Now()

	logger.Debug("Balance updated",
		zap.Float64("balance", balance),
		zap.Int64("version", version),
	)
}

// ========== 持仓管理 ==========

// GetPosition 获取指定交易对的持仓
func (c *AccountCache) GetPosition(symbol string) (*core.Position, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	pos, exists := c.positions[symbol]
	if !exists {
		return nil, false
	}

	// 返回副本，避免外部修改
	posCopy := *pos
	return &posCopy, true
}

// GetAllPositions 获取所有持仓
func (c *AccountCache) GetAllPositions() []*core.Position {
	c.mu.RLock()
	defer c.mu.RUnlock()

	positions := make([]*core.Position, 0, len(c.positions))
	for _, pos := range c.positions {
		posCopy := *pos
		positions = append(positions, &posCopy)
	}

	return positions
}

// UpdatePosition 更新持仓信息
// 如果持仓为0，自动删除
func (c *AccountCache) UpdatePosition(position *core.Position, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 版本检查：只忽略明显过时的更新（严格小于）
	if version < c.updateVersion {
		logger.Debug("Ignoring outdated position update",
			zap.String("symbol", position.Symbol),
			zap.Int64("current_version", c.updateVersion),
			zap.Int64("update_version", version),
		)
		return
	}

	// 持仓为0，删除
	if position.Size == 0 {
		delete(c.positions, position.Symbol)
		logger.Info("Position closed and removed from cache",
			zap.String("symbol", position.Symbol),
			zap.Int64("version", version),
		)
	} else {
		// 更新或新增
		c.positions[position.Symbol] = position
		logger.Debug("Position updated",
			zap.String("symbol", position.Symbol),
			zap.String("side", string(position.Side)),
			zap.Float64("size", position.Size),
			zap.Float64("entry_price", position.EntryPrice),
			zap.Float64("unrealized_pnl", position.UnrealizedPnL),
			zap.Int64("version", version),
		)
	}

	c.updateVersion = version
	c.lastUpdateTime = time.Now()
}

// DeletePosition 删除持仓
func (c *AccountCache) DeletePosition(symbol string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.positions, symbol)
	c.lastUpdateTime = time.Now()

	logger.Debug("Position deleted from cache", zap.String("symbol", symbol))
}

// HasPosition 检查是否有持仓
func (c *AccountCache) HasPosition(symbol string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	_, exists := c.positions[symbol]
	return exists
}

// ========== 订单管理 ==========

// GetOrder 获取订单
func (c *AccountCache) GetOrder(orderID string) (*core.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	order, exists := c.orders[orderID]
	if !exists {
		return nil, false
	}

	// 返回副本
	orderCopy := *order
	return &orderCopy, true
}

// GetOpenOrders 获取指定交易对的未成交订单
// symbol 为空时返回所有订单
func (c *AccountCache) GetOpenOrders(symbol string) []*core.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()

	orders := make([]*core.Order, 0)
	for _, order := range c.orders {
		if symbol == "" || order.Symbol == symbol {
			orderCopy := *order
			orders = append(orders, &orderCopy)
		}
	}

	return orders
}

// GetAllOrders 获取所有订单
func (c *AccountCache) GetAllOrders() []*core.Order {
	return c.GetOpenOrders("")
}

// UpdateOrder 更新订单信息
// 如果订单已完成（FILLED/CANCELED/REJECTED），自动删除
func (c *AccountCache) UpdateOrder(order *core.Order, version int64) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 版本检查：只忽略明显过时的更新（严格小于）
	if version < c.updateVersion {
		logger.Debug("Ignoring outdated order update",
			zap.String("order_id", order.ID),
			zap.Int64("current_version", c.updateVersion),
			zap.Int64("update_version", version),
		)
		return
	}

	// 检查订单状态
	isFinal := order.Status == core.OrderStatusFilled ||
		order.Status == core.OrderStatusCanceled ||
		order.Status == core.OrderStatusRejected

	if isFinal {
		// 最终状态，删除订单
		delete(c.orders, order.ID)
		logger.Info("Order completed and removed from cache",
			zap.String("order_id", order.ID),
			zap.String("status", string(order.Status)),
			zap.Int64("version", version),
		)
	} else {
		// 更新或新增
		c.orders[order.ID] = order
		logger.Debug("Order updated",
			zap.String("order_id", order.ID),
			zap.String("symbol", order.Symbol),
			zap.String("status", string(order.Status)),
			zap.String("type", string(order.Type)),
			zap.Int64("version", version),
		)
	}

	c.updateVersion = version
	c.lastUpdateTime = time.Now()
}

// DeleteOrder 删除订单
func (c *AccountCache) DeleteOrder(orderID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.orders, orderID)
	c.lastUpdateTime = time.Now()

	logger.Debug("Order deleted from cache", zap.String("order_id", orderID))
}

// ========== 统计信息 ==========

// GetStats 获取缓存统计信息
func (c *AccountCache) GetStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"balance":          c.balance,
		"position_count":   len(c.positions),
		"order_count":      len(c.orders),
		"last_update_time": c.lastUpdateTime,
		"update_version":   c.updateVersion,
	}
}

// GetLastUpdateTime 获取最后更新时间
func (c *AccountCache) GetLastUpdateTime() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUpdateTime
}

// GetVersion 获取当前版本号
func (c *AccountCache) GetVersion() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.updateVersion
}

// Reset 重置缓存（测试用）
func (c *AccountCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.balance = 0
	c.positions = make(map[string]*core.Position)
	c.orders = make(map[string]*core.Order)
	c.lastUpdateTime = time.Time{}
	c.updateVersion = 0

	logger.Info("Account cache reset")
}
