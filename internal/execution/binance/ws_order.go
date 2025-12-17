package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"goQuant/internal/core"
	"goQuant/internal/logger"
)

// WSOrderClient WebSocket 订单客户端
type WSOrderClient struct {
	client     *Client
	conn       *websocket.Conn
	mu         sync.RWMutex
	stopCh     chan struct{}
	stopOnce   sync.Once
	responseCh map[string]chan *WSOrderResponse
	responseMu sync.RWMutex
	isTestnet  bool
}

// WSOrderRequest WebSocket 订单请求
type WSOrderRequest struct {
	ID     string                 `json:"id"`
	Method string                 `json:"method"`
	Params map[string]interface{} `json:"params"`
}

// WSOrderResponse WebSocket 订单响应
type WSOrderResponse struct {
	ID         string                 `json:"id"`
	Status     int                    `json:"status"`
	Result     map[string]interface{} `json:"result,omitempty"`
	Error      *WSOrderError          `json:"error,omitempty"`
	RateLimits []RateLimit            `json:"rateLimits,omitempty"`
}

// WSOrderError WebSocket 订单错误
type WSOrderError struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

// RateLimit 限流信息
type RateLimit struct {
	RateLimitType string `json:"rateLimitType"`
	Interval      string `json:"interval"`
	IntervalNum   int    `json:"intervalNum"`
	Limit         int    `json:"limit"`
	Count         int    `json:"count"`
}

// NewWSOrderClient 创建 WebSocket 订单客户端
func NewWSOrderClient(client *Client) *WSOrderClient {
	isTestnet := false
	if client.baseURL == "https://testnet.binancefuture.com" {
		isTestnet = true
	}

	return &WSOrderClient{
		client:     client,
		responseCh: make(map[string]chan *WSOrderResponse),
		stopCh:     make(chan struct{}),
		isTestnet:  isTestnet,
	}
}

// Start 启动 WebSocket 连接
func (w *WSOrderClient) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.conn != nil {
		return fmt.Errorf("websocket already connected")
	}

	// 获取 WebSocket URL
	wsURL := w.getWebSocketURL()

	// 连接 WebSocket
	dialer := w.client.GetWebSocketDialer()
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}

	w.conn = conn
	logger.Info("WebSocket 订单连接已建立", zap.String("url", wsURL))

	// 启动消息处理和心跳
	go w.readLoop()
	go w.pingLoop()

	return nil
}

// Stop 停止 WebSocket 连接
func (w *WSOrderClient) Stop() {
	w.stopOnce.Do(func() {
		w.mu.Lock()
		defer w.mu.Unlock()

		close(w.stopCh)

		if w.conn != nil {
			_ = w.conn.Close()
			w.conn = nil
		}
	})
}

// getWebSocketURL 获取 WebSocket URL
//
//	func (w *WSOrderClient) getWebSocketURL() string {
//		if w.isTestnet {
//			return "wss://stream.binancefuture.com/ws-fapi/v1"
//		}
//		return "wss://fstream.binance.com/ws-fapi/v1"
//	}
func (w *WSOrderClient) getWebSocketURL() string {
	if w.isTestnet {
		return "wss://testnet.binancefuture.com/ws-fapi/v1"
	}
	return "wss://ws-fapi.binance.com/ws-fapi/v1"
}

// readLoop 读取消息循环
func (w *WSOrderClient) readLoop() {
	for {
		select {
		case <-w.stopCh:
			return
		default:
			w.mu.RLock()
			conn := w.conn
			w.mu.RUnlock()

			if conn == nil {
				return
			}

			_, message, err := conn.ReadMessage()
			if err != nil {
				logger.Error("WebSocket 读取消息失败", zap.Error(err))
				// 触发重连
				go w.reconnect()
				return
			}

			// 解析响应
			var response WSOrderResponse
			if err := json.Unmarshal(message, &response); err != nil {
				logger.Error("解析 WebSocket 响应失败", zap.Error(err), zap.String("message", string(message)))
				continue
			}

			// 发送到对应的响应通道
			w.responseMu.RLock()
			ch, ok := w.responseCh[response.ID]
			w.responseMu.RUnlock()

			if ok {
				select {
				case ch <- &response:
				case <-time.After(5 * time.Second):
					logger.Warn("发送响应到通道超时", zap.String("id", response.ID))
				}
			}
		}
	}
}

// pingLoop 心跳循环
func (w *WSOrderClient) pingLoop() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.mu.RLock()
			conn := w.conn
			w.mu.RUnlock()

			if conn != nil {
				if err := conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
					logger.Error("发送 ping 失败", zap.Error(err))
				}
			}
		}
	}
}

// reconnect 重连
func (w *WSOrderClient) reconnect() {
	w.mu.Lock()
	if w.conn != nil {
		_ = w.conn.Close()
		w.conn = nil
	}
	w.mu.Unlock()

	logger.Info("尝试重新连接 WebSocket 订单...")

	for i := 0; i < 5; i++ {
		time.Sleep(time.Duration(i+1) * time.Second)

		if err := w.Start(context.Background()); err != nil {
			logger.Error("重连失败", zap.Int("attempt", i+1), zap.Int("max", 5), zap.Error(err))
			continue
		}

		logger.Info("WebSocket 订单重连成功")
		return
	}

	logger.Error("WebSocket 订单重连失败，已达到最大重试次数")
}

// sendRequest 发送请求并等待响应
func (w *WSOrderClient) sendRequest(ctx context.Context, method string, params map[string]interface{}) (*WSOrderResponse, error) {
	w.mu.RLock()
	conn := w.conn
	w.mu.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("websocket not connected")
	}

	// 生成请求 ID
	requestID := uuid.New().String()

	if orderType, ok := params["type"].(string); ok {
		if orderType == "STOP_MARKET" || orderType == "TAKE_PROFIT_MARKET" || orderType == "TRAILING_STOP_MARKET" {
			method = "algoOrder.place"
			// ✅ 条件单必须添加 algoType
			if _, hasAlgoType := params["algoType"]; !hasAlgoType {
				params["algoType"] = "CONDITIONAL"
			}
		}
	}

	// 添加签名
	if err := w.signParams(params); err != nil {
		return nil, fmt.Errorf("sign params: %w", err)
	}

	// 构建请求
	request := WSOrderRequest{
		ID:     requestID,
		Method: method,
		Params: params,
	}

	// 创建响应通道
	responseCh := make(chan *WSOrderResponse, 1)
	w.responseMu.Lock()
	w.responseCh[requestID] = responseCh
	w.responseMu.Unlock()

	// 清理响应通道
	defer func() {
		w.responseMu.Lock()
		delete(w.responseCh, requestID)
		w.responseMu.Unlock()
		close(responseCh)
	}()

	// 发送请求
	data, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		return nil, fmt.Errorf("write message: %w", err)
	}

	// 等待响应
	select {
	case response := <-responseCh:
		if response.Error != nil {
			return nil, fmt.Errorf("order error [%d]: %s", response.Error.Code, response.Error.Msg)
		}
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(10 * time.Second):
		return nil, fmt.Errorf("request timeout")
	}
}

// signParams 对参数进行签名
func (w *WSOrderClient) signParams(params map[string]interface{}) error {
	// 添加 API Key
	params["apiKey"] = w.client.apiKey

	// 添加时间戳
	params["timestamp"] = time.Now().UnixMilli()

	// 添加接收窗口
	if _, ok := params["recvWindow"]; !ok {
		params["recvWindow"] = 60000
	}

	// 构建查询字符串
	queryString := buildQueryString(params)

	// 生成签名
	h := hmac.New(sha256.New, []byte(w.client.secretKey))
	h.Write([]byte(queryString))
	signature := hex.EncodeToString(h.Sum(nil))

	params["signature"] = signature

	return nil
}

// buildQueryString 构建查询字符串（用于签名）
func buildQueryString(params map[string]interface{}) string {
	values := url.Values{}
	for key, value := range params {
		if key == "signature" {
			continue
		}
		values.Set(key, fmt.Sprintf("%v", value))
	}
	return values.Encode()
}

// ========== 原子订单方法 ==========

// PlaceMarketOrder 市价单（入场）
func (w *WSOrderClient) PlaceMarketOrder(ctx context.Context, symbol string, side core.OrderSide, quantity float64) (*core.Order, error) {
	params := map[string]interface{}{
		"symbol":       symbol,
		"side":         FromOrderSide(side),
		"type":         "MARKET",
		"quantity":     formatQuantity(symbol, quantity),
		"positionSide": "BOTH",
	}

	response, err := w.sendRequest(ctx, "order.place", params)
	if err != nil {
		return nil, err
	}

	return w.parseOrderResponse(response)
}

// PlaceLimitOrder 限价单（入场）
func (w *WSOrderClient) PlaceLimitOrder(ctx context.Context, symbol string, side core.OrderSide, quantity float64, price float64, timeInForce string) (*core.Order, error) {
	params := map[string]interface{}{
		"symbol":      symbol,
		"side":        FromOrderSide(side),
		"type":        "LIMIT",
		"quantity":    formatQuantity(symbol, quantity),
		"price":       formatPrice(symbol, price),
		"timeInForce": timeInForce,
		//"positionSide": "BOTH",
	}

	response, err := w.sendRequest(ctx, "order.place", params)
	if err != nil {
		return nil, err
	}

	return w.parseOrderResponse(response)
}

// ClosePositionMarket 市价单（"手动"出场，市价平掉某币种全部仓位）
func (w *WSOrderClient) ClosePositionMarket(ctx context.Context, symbol string, side core.OrderSide, quantity float64) (*core.Order, error) {
	params := map[string]interface{}{
		"symbol":       symbol,
		"side":         FromOrderSide(side),
		"type":         "MARKET",
		"quantity":     formatQuantity(symbol, quantity),
		"positionSide": "BOTH",
		"reduceOnly":   "true",
	}

	response, err := w.sendRequest(ctx, "order.place", params)
	if err != nil {
		return nil, err
	}

	return w.parseOrderResponse(response)
}

// PlaceStopLossOrder 止损单（STOP_MARKET，到触发价市价平掉某币种全部仓位）
func (w *WSOrderClient) PlaceStopLossOrder(ctx context.Context, symbol string, side core.OrderSide, triggerPrice float64) (*core.Order, error) {
	params := map[string]interface{}{
		"algoType": "CONDITIONAL",
		"symbol":   symbol,
		"side":     FromOrderSide(side),
		"type":     "STOP_MARKET",
		//"stopPrice":     formatPrice(symbol, triggerPrice),
		"triggerPrice":  formatPrice(symbol, triggerPrice),
		"positionSide":  "BOTH",
		"closePosition": "true",
		"workingType":   "CONTRACT_PRICE",
	}

	response, err := w.sendRequest(ctx, "order.place", params)
	if err != nil {
		return nil, err
	}

	return w.parseOrderResponse(response)
}

// PlaceTrailingStopOrder 跟踪止损单（TRAILING_STOP_MARKET，平掉全部仓位）
func (w *WSOrderClient) PlaceTrailingStopOrder(ctx context.Context, symbol string, side core.OrderSide, quantity float64, activatePrice float64, callbackRate float64) (*core.Order, error) {
	params := map[string]interface{}{
		"algoType":      "CONDITIONAL",
		"symbol":        symbol,
		"side":          FromOrderSide(side),
		"type":          "TRAILING_STOP_MARKET",
		"positionSide":  "BOTH",
		"callbackRate":  fmt.Sprintf("%.2f", callbackRate),
		"quantity":      formatQuantity(symbol, quantity),
		"workingType":   "CONTRACT_PRICE",
		"activatePrice": "0",
	}

	// activatePrice 可选，默认为当前市场价格
	if activatePrice > 0 {
		params["activatePrice"] = formatPrice(symbol, activatePrice)
	}

	response, err := w.sendRequest(ctx, "order.place", params)
	if err != nil {
		return nil, err
	}

	return w.parseOrderResponse(response)
}

// CancelOrder 撤销订单
func (w *WSOrderClient) CancelOrder(ctx context.Context, symbol string, orderID int64) error {
	params := map[string]interface{}{
		"symbol":  symbol,
		"orderId": orderID,
	}

	_, err := w.sendRequest(ctx, "order.cancel", params)
	return err
}

// parseOrderResponse 解析订单响应
func (w *WSOrderClient) parseOrderResponse(response *WSOrderResponse) (*core.Order, error) {
	if response.Result == nil {
		return nil, fmt.Errorf("empty result")
	}

	result := response.Result

	// 解析订单 ID
	orderID, _ := result["orderId"].(float64)
	clientOrderID, _ := result["clientOrderId"].(string)
	symbol, _ := result["symbol"].(string)
	side, _ := result["side"].(string)
	orderType, _ := result["type"].(string)
	status, _ := result["status"].(string)

	// 解析数量和价格
	origQty := 0.0
	if origQtyStr, ok := result["origQty"].(string); ok {
		origQty, _ = strconv.ParseFloat(origQtyStr, 64)
	}
	executedQty := 0.0
	if executedQtyStr, ok := result["executedQty"].(string); ok {
		executedQty, _ = strconv.ParseFloat(executedQtyStr, 64)
	}
	price := 0.0
	if priceStr, ok := result["price"].(string); ok && priceStr != "" && priceStr != "0" && priceStr != "0.00" {
		price, _ = strconv.ParseFloat(priceStr, 64)
	}
	avgPrice := 0.0
	if avgPriceStr, ok := result["avgPrice"].(string); ok && avgPriceStr != "" && avgPriceStr != "0" && avgPriceStr != "0.00" {
		avgPrice, _ = strconv.ParseFloat(avgPriceStr, 64)
	}

	// 解析止损价格
	stopPrice := 0.0
	if stopPriceStr, ok := result["stopPrice"].(string); ok && stopPriceStr != "" && stopPriceStr != "0" && stopPriceStr != "0.00" {
		stopPrice, _ = strconv.ParseFloat(stopPriceStr, 64)
	}

	// 解析更新时间
	updateTime, _ := result["updateTime"].(float64)

	order := &core.Order{
		ID:         clientOrderID,
		Symbol:     symbol,
		Type:       ToOrderType(FuturesOrderType(orderType)),
		Side:       ToOrderSide(FuturesOrderSide(side)),
		Price:      price,
		Quantity:   origQty,
		FilledQty:  executedQty,
		AvgPrice:   avgPrice,
		Status:     ToOrderStatus(OrderStatus(status)),
		StopPrice:  stopPrice,
		CreateTime: time.UnixMilli(int64(updateTime)),
		UpdateTime: time.UnixMilli(int64(updateTime)),
		Metadata: map[string]interface{}{
			"exchange_order_id": strconv.FormatInt(int64(orderID), 10),
		},
	}

	return order, nil
}
