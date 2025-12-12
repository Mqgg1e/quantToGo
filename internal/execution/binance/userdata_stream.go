package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"goQuant/internal/cache"
	"goQuant/internal/core"
	"goQuant/internal/logger"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// UserDataStream Binance UserDataStream 客户端
// 通过 WebSocket 实时接收账户更新和订单更新
type UserDataStream struct {
	client         *Client
	accountCache   *cache.AccountCache
	executor       cache.Executor // 用于从REST API同步状态
	wsBaseURL      string         // WebSocket 端点URL
	mu             sync.RWMutex
	listenKey      string
	conn           *websocket.Conn
	stopCh         chan struct{}
	reconnectDelay time.Duration
	maxReconnect   int
	isRunning      bool
}

const (
	// KeepAlive interval (30 minutes, ListenKey expires in 60 minutes)
	keepAliveInterval = 30 * time.Minute

	// Reconnect settings
	initialReconnectDelay = 1 * time.Second
	maxReconnectDelay     = 60 * time.Second
	maxReconnectAttempts  = 100 // Effectively unlimited

	// WebSocket settings
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
	writeWait  = 10 * time.Second
)

// getWebSocketURL 根据 REST API baseURL 获取对应的 WebSocket URL
func getWebSocketURL(baseURL string) string {
	// 测试网
	if baseURL == "https://testnet.binancefuture.com" {
		return "wss://stream.binancefuture.com/ws"
	}
	// 生产环境
	return "wss://fstream.binance.com/ws"
}

// NewUserDataStream 创建 UserDataStream 实例
func NewUserDataStream(client *Client, accountCache *cache.AccountCache, executor cache.Executor) *UserDataStream {
	return &UserDataStream{
		client:         client,
		accountCache:   accountCache,
		executor:       executor,
		wsBaseURL:      getWebSocketURL(client.baseURL),
		stopCh:         make(chan struct{}),
		reconnectDelay: initialReconnectDelay,
		maxReconnect:   maxReconnectAttempts,
		isRunning:      false,
	}
}

// Start 启动 UserDataStream
func (s *UserDataStream) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("UserDataStream is already running")
	}
	s.isRunning = true
	s.mu.Unlock()

	logger.Info("Starting UserDataStream...")

	// 1. 创建 ListenKey
	listenKey, err := s.client.CreateListenKey(ctx)
	if err != nil {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		return fmt.Errorf("create listen key: %w", err)
	}

	s.mu.Lock()
	s.listenKey = listenKey
	s.mu.Unlock()

	logger.Info("ListenKey created", zap.String("listen_key", listenKey[:8]+"..."))

	// 2. 建立 WebSocket 连接
	if err := s.connect(); err != nil {
		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()
		return fmt.Errorf("connect websocket: %w", err)
	}

	// 3. 启动 ListenKey 保活协程
	go s.keepAliveLoop(ctx)

	// 4. 启动 ping 循环（保持 WebSocket 连接）
	go s.pingLoop(ctx)

	// 5. 启动消息读取协程
	go s.readLoop(ctx)

	logger.Info("UserDataStream started successfully")
	return nil
}

// Stop 停止 UserDataStream
func (s *UserDataStream) Stop() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}
	s.isRunning = false
	s.mu.Unlock()

	logger.Info("Stopping UserDataStream...")

	// 发送停止信号
	close(s.stopCh)

	// 关闭 WebSocket 连接
	s.mu.Lock()
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()

	// 关闭 ListenKey
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	s.mu.RLock()
	listenKey := s.listenKey
	s.mu.RUnlock()

	if listenKey != "" {
		if err := s.client.CloseListenKey(ctx, listenKey); err != nil {
			logger.Warn("Failed to close listen key", zap.Error(err))
		} else {
			logger.Info("ListenKey closed")
		}
	}

	logger.Info("UserDataStream stopped")
}

// connect 建立 WebSocket 连接
func (s *UserDataStream) connect() error {
	s.mu.RLock()
	listenKey := s.listenKey
	wsBaseURL := s.wsBaseURL
	s.mu.RUnlock()

	wsURL := fmt.Sprintf("%s/%s", wsBaseURL, listenKey)

	logger.Info("Connecting to UserDataStream WebSocket",
		zap.String("url", wsBaseURL+"/..."),
		zap.String("listen_key", listenKey[:8]+"..."),
	)

	// 使用客户端的代理配置
	dialer := s.client.GetWebSocketDialer()

	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	// 设置读取超时
	conn.SetReadDeadline(time.Now().Add(pongWait))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	s.mu.Lock()
	s.conn = conn
	s.mu.Unlock()

	logger.Info("WebSocket connected successfully")
	return nil
}

// readLoop 读取消息循环
func (s *UserDataStream) readLoop(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("Panic in readLoop", zap.Any("panic", r))
		}
	}()

	for {
		select {
		case <-s.stopCh:
			logger.Info("ReadLoop stopped by signal")
			return
		case <-ctx.Done():
			logger.Info("ReadLoop stopped by context")
			return
		default:
		}

		s.mu.RLock()
		conn := s.conn
		s.mu.RUnlock()

		if conn == nil {
			logger.Warn("Connection is nil, attempting reconnect...")
			if err := s.reconnect(ctx); err != nil {
				logger.Error("Reconnect failed", zap.Error(err))
				time.Sleep(s.reconnectDelay)
			}
			continue
		}

		// 读取消息
		_, message, err := conn.ReadMessage()
		if err != nil {
			logger.Error("Failed to read message", zap.Error(err))

			// 尝试重连
			if err := s.reconnect(ctx); err != nil {
				logger.Error("Reconnect failed", zap.Error(err))
				time.Sleep(s.reconnectDelay)
			}
			continue
		}

		// 处理消息
		if err := s.handleMessage(message); err != nil {
			logger.Error("Failed to handle message", zap.Error(err), zap.String("message", string(message)))
		}

		// 重置重连延迟（成功读取消息）
		s.reconnectDelay = initialReconnectDelay
	}
}

// handleMessage 处理收到的消息
func (s *UserDataStream) handleMessage(message []byte) error {
	// 解析基础事件类型
	var baseEvent UserDataEvent
	if err := json.Unmarshal(message, &baseEvent); err != nil {
		return fmt.Errorf("unmarshal base event: %w", err)
	}

	logger.Info("Received UserDataStream event",
		zap.String("event_type", baseEvent.EventType),
		zap.Int64("event_time", baseEvent.EventTime),
		zap.String("raw_message", string(message)),
	)

	// 根据事件类型分发处理
	switch baseEvent.EventType {
	case "ACCOUNT_UPDATE":
		return s.handleAccountUpdate(message)
	case "ORDER_TRADE_UPDATE":
		return s.handleOrderUpdate(message)
	case "MARGIN_CALL":
		return s.handleMarginCall(message)
	case "ACCOUNT_CONFIG_UPDATE":
		return s.handleAccountConfigUpdate(message)
	default:
		logger.Warn("Unknown event type", zap.String("event_type", baseEvent.EventType))
	}

	return nil
}

// handleAccountUpdate 处理账户更新事件
func (s *UserDataStream) handleAccountUpdate(message []byte) error {
	var event AccountUpdateEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("unmarshal account update: %w", err)
	}

	logger.Info("Account update received",
		zap.String("reason", event.AccountUpdate.Reason),
		zap.Int("balance_count", len(event.AccountUpdate.Balances)),
		zap.Int("position_count", len(event.AccountUpdate.Positions)),
	)

	version := event.Transaction // 使用交易时间作为版本号

	// 更新余额（USDT）
	for _, balance := range event.AccountUpdate.Balances {
		if balance.Asset == "USDT" {
			walletBalance, err := parseFloat(balance.WalletBalance)
			if err != nil {
				logger.Error("Failed to parse wallet balance", zap.Error(err))
				continue
			}

			s.accountCache.UpdateBalance(walletBalance, version)
			logger.Info("Balance updated",
				zap.Float64("wallet_balance", walletBalance),
				zap.Int64("version", version),
			)
		}
	}

	// 更新持仓
	for _, posUpdate := range event.AccountUpdate.Positions {
		position, err := s.convertPositionUpdate(&posUpdate)
		if err != nil {
			logger.Error("Failed to convert position", zap.Error(err), zap.String("symbol", posUpdate.Symbol))
			continue
		}

		s.accountCache.UpdatePosition(position, version)

		if position.Size == 0 {
			logger.Info("Position closed",
				zap.String("symbol", position.Symbol),
			)
		} else {
			logger.Info("Position updated",
				zap.String("symbol", position.Symbol),
				zap.String("side", string(position.Side)),
				zap.Float64("size", position.Size),
				zap.Float64("entry_price", position.EntryPrice),
				zap.Float64("unrealized_pnl", position.UnrealizedPnL),
			)
		}
	}

	return nil
}

// handleOrderUpdate 处理订单更新事件
func (s *UserDataStream) handleOrderUpdate(message []byte) error {
	var event OrderTradeUpdateEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("unmarshal order update: %w", err)
	}

	logger.Info("Order update received",
		zap.String("symbol", event.Order.Symbol),
		zap.String("client_order_id", event.Order.ClientOrderID),
		zap.String("status", event.Order.OrderStatus),
		zap.String("execution_type", event.Order.ExecutionType),
	)

	version := event.Transaction // 使用交易时间作为版本号

	// 转换订单
	order, err := s.convertOrderUpdate(&event.Order)
	if err != nil {
		logger.Error("Failed to convert order", zap.Error(err))
		return err
	}

	// 更新缓存
	s.accountCache.UpdateOrder(order, version)

	logger.Info("Order cache updated",
		zap.String("order_id", order.ID),
		zap.String("status", string(order.Status)),
		zap.Float64("filled_qty", order.FilledQty),
	)

	return nil
}

// handleMarginCall 处理追加保证金通知
func (s *UserDataStream) handleMarginCall(message []byte) error {
	var event MarginCallEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("unmarshal margin call: %w", err)
	}

	logger.Warn("⚠️  MARGIN CALL received!",
		zap.Int("position_count", len(event.Positions)),
	)

	for _, pos := range event.Positions {
		logger.Warn("Margin call position",
			zap.String("symbol", pos.Symbol),
			zap.String("position_side", pos.PositionSide),
			zap.String("position_amount", pos.PositionAmount),
			zap.String("mark_price", pos.MarkPrice),
			zap.String("unrealized_pnl", pos.UnrealizedPnL),
			zap.String("maintenance_margin", pos.MaintenanceMarginRequired),
		)
	}

	return nil
}

// handleAccountConfigUpdate 处理账户配置更新
func (s *UserDataStream) handleAccountConfigUpdate(message []byte) error {
	var event AccountConfigUpdateEvent
	if err := json.Unmarshal(message, &event); err != nil {
		return fmt.Errorf("unmarshal account config update: %w", err)
	}

	logger.Info("Account config updated",
		zap.String("symbol", event.ConfigUpdate.Symbol),
		zap.Int("leverage", event.ConfigUpdate.Leverage),
	)

	return nil
}

// convertPositionUpdate 转换持仓更新为 core.Position
func (s *UserDataStream) convertPositionUpdate(update *PositionUpdate) (*core.Position, error) {
	posAmount, err := parseFloat(update.PositionAmount)
	if err != nil {
		return nil, fmt.Errorf("parse position amount: %w", err)
	}

	entryPrice, err := parseFloat(update.EntryPrice)
	if err != nil {
		return nil, fmt.Errorf("parse entry price: %w", err)
	}

	unrealizedPnL, err := parseFloat(update.UnrealizedPnL)
	if err != nil {
		return nil, fmt.Errorf("parse unrealized pnl: %w", err)
	}

	// 确定持仓方向
	var side core.PositionSide
	if posAmount > 0 {
		side = core.PositionSideLong
	} else if posAmount < 0 {
		side = core.PositionSideShort
		posAmount = -posAmount // 转为正数
	} else {
		// 持仓为0，返回空持仓
		return &core.Position{
			Symbol: update.Symbol,
			Size:   0,
		}, nil
	}

	// 转换保证金模式
	var marginMode core.MarginMode
	if update.MarginType == "isolated" {
		marginMode = core.MarginModeIsolated
	} else {
		marginMode = core.MarginModeCross
	}

	// 计算未实现盈亏百分比
	unrealizedPnLPercent := 0.0
	if entryPrice > 0 {
		unrealizedPnLPercent = (unrealizedPnL / (entryPrice * posAmount)) * 100
	}

	return &core.Position{
		Symbol:               update.Symbol,
		Side:                 side,
		Size:                 posAmount,
		EntryPrice:           entryPrice,
		CurrentPrice:         0, // 需要从市场数据获取
		Leverage:             0, // 需要从其他接口获取
		MarginMode:           marginMode,
		UnrealizedPnL:        unrealizedPnL,
		UnrealizedPnLPercent: unrealizedPnLPercent,
		OpenTime:             time.Now(), // 使用当前时间，实际开仓时间需要从其他接口获取
	}, nil
}

// convertOrderUpdate 转换订单更新为 core.Order
func (s *UserDataStream) convertOrderUpdate(update *OrderUpdateData) (*core.Order, error) {
	origQty, err := parseFloat(update.OriginalQuantity)
	if err != nil {
		return nil, fmt.Errorf("parse original quantity: %w", err)
	}

	price, _ := parseFloat(update.OriginalPrice)
	avgPrice, _ := parseFloat(update.AveragePrice)
	filledQty, _ := parseFloat(update.AccumulatedFilled)
	stopPrice, _ := parseFloat(update.StopPrice)

	// 转换订单方向
	var side core.OrderSide
	if update.Side == "BUY" {
		side = core.OrderSideBuy
	} else {
		side = core.OrderSideSell
	}

	// 转换订单类型
	orderType := ToOrderType(FuturesOrderType(update.OrderType))

	// 转换订单状态
	orderStatus := ToOrderStatus(OrderStatus(update.OrderStatus))

	order := &core.Order{
		ID:         update.ClientOrderID,
		Symbol:     update.Symbol,
		Type:       orderType,
		Side:       side,
		Quantity:   origQty,
		Price:      price,
		StopPrice:  stopPrice,
		FilledQty:  filledQty,
		AvgPrice:   avgPrice,
		Status:     orderStatus,
		CreateTime: time.Unix(update.TradeTime/1000, 0),
		UpdateTime: time.Unix(update.TradeTime/1000, 0),
		Metadata:   make(map[string]interface{}),
	}

	// 添加元数据
	order.Metadata["binance_order_id"] = update.OrderID
	order.Metadata["execution_type"] = update.ExecutionType
	order.Metadata["reduce_only"] = update.IsReduceOnly
	order.Metadata["close_position"] = update.ClosePosition

	return order, nil
}

// keepAliveLoop ListenKey 保活循环
func (s *UserDataStream) keepAliveLoop(ctx context.Context) {
	ticker := time.NewTicker(keepAliveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			logger.Info("KeepAlive loop stopped")
			return
		case <-ctx.Done():
			logger.Info("KeepAlive loop cancelled")
			return
		case <-ticker.C:
			s.mu.RLock()
			listenKey := s.listenKey
			s.mu.RUnlock()

			if listenKey == "" {
				logger.Warn("ListenKey is empty, skipping keep alive")
				continue
			}

			logger.Debug("Keeping ListenKey alive...")

			keepAliveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := s.client.KeepAliveListenKey(keepAliveCtx, listenKey)
			cancel()

			if err != nil {
				logger.Error("Failed to keep alive ListenKey", zap.Error(err))
				// ListenKey 可能已过期，需要重连
				go s.reconnect(ctx)
			} else {
				logger.Debug("ListenKey kept alive successfully")
			}
		}
	}
}

// pingLoop WebSocket ping 循环（保持连接活跃）
func (s *UserDataStream) pingLoop(ctx context.Context) {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			logger.Info("Ping loop stopped")
			return
		case <-ctx.Done():
			logger.Info("Ping loop cancelled")
			return
		case <-ticker.C:
			s.mu.RLock()
			conn := s.conn
			s.mu.RUnlock()

			if conn == nil {
				continue
			}

			if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				logger.Error("Failed to set write deadline", zap.Error(err))
				continue
			}

			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Error("Failed to send ping", zap.Error(err))
				// Ping 失败，可能需要重连
				go s.reconnect(ctx)
			} else {
				logger.Debug("WebSocket ping sent")
			}
		}
	}
}

// reconnect 重新连接
func (s *UserDataStream) reconnect(ctx context.Context) error {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("UserDataStream is not running")
	}

	// 关闭旧连接
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	s.mu.Unlock()

	logger.Warn("Reconnecting UserDataStream...",
		zap.Duration("delay", s.reconnectDelay),
	)

	// 等待重连延迟
	time.Sleep(s.reconnectDelay)

	// 重新创建 ListenKey
	listenKey, err := s.client.CreateListenKey(ctx)
	if err != nil {
		// 增加重连延迟
		s.reconnectDelay = s.reconnectDelay * 2
		if s.reconnectDelay > maxReconnectDelay {
			s.reconnectDelay = maxReconnectDelay
		}
		return fmt.Errorf("create listen key: %w", err)
	}

	s.mu.Lock()
	s.listenKey = listenKey
	s.mu.Unlock()

	logger.Info("New ListenKey created", zap.String("listen_key", listenKey[:8]+"..."))

	// 重新连接 WebSocket
	if err := s.connect(); err != nil {
		s.reconnectDelay = s.reconnectDelay * 2
		if s.reconnectDelay > maxReconnectDelay {
			s.reconnectDelay = maxReconnectDelay
		}
		return fmt.Errorf("connect websocket: %w", err)
	}

	// 重连成功后，从 REST API 同步状态
	logger.Info("Reconnected successfully, syncing state from REST API...")
	if err := s.accountCache.InitFromRestAPI(ctx, s.executor); err != nil {
		logger.Error("Failed to sync state after reconnect", zap.Error(err))
	}

	// 重置重连延迟
	s.reconnectDelay = initialReconnectDelay

	logger.Info("UserDataStream reconnected successfully")
	return nil
}

// IsRunning 检查是否正在运行
func (s *UserDataStream) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// GetListenKey 获取当前 ListenKey
func (s *UserDataStream) GetListenKey() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.listenKey
}
