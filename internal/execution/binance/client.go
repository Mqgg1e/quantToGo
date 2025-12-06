package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// Client 币安期货API客户端
type Client struct {
	apiKey     string
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

// NewClient 创建币安期货客户端
func NewClient(apiKey, secretKey, baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://fapi.binance.com"
	}

	return &Client{
		apiKey:    apiKey,
		secretKey: secretKey,
		baseURL:   baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ========== 签名与请求方法 ==========

// sign 生成HMAC SHA256签名
func (c *Client) sign(params string) string {
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(params))
	return hex.EncodeToString(h.Sum(nil))
}

// doRequest 发送HTTP请求
func (c *Client) doRequest(ctx context.Context, method, endpoint string, params url.Values, signed bool) ([]byte, error) {
	urlStr := c.baseURL + endpoint

	if signed {
		// 添加接收窗口（默认5000ms，增加容错）
		if params.Get("recvWindow") == "" {
			params.Set("recvWindow", "60000") // 60秒时间窗口
		}

		// 添加时间戳
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		params.Set("timestamp", timestamp)

		// 生成签名
		queryString := params.Encode()
		signature := c.sign(queryString)
		params.Set("signature", signature)
	}

	// 构建完整URL
	if len(params) > 0 {
		urlStr += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// 添加请求头
	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// ========== 交易接口 ==========

// CreateOrder 创建订单
func (c *Client) CreateOrder(ctx context.Context, req *CreateOrderRequest) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", req.Symbol)
	params.Set("side", string(req.Side))
	params.Set("type", string(req.Type))

	if req.PositionSide != "" {
		params.Set("positionSide", string(req.PositionSide))
	}

	if req.Quantity != "" {
		params.Set("quantity", req.Quantity)
	}

	if req.Price != "" {
		params.Set("price", req.Price)
	}

	if req.TimeInForce != "" {
		params.Set("timeInForce", string(req.TimeInForce))
	}

	if req.ReduceOnly {
		params.Set("reduceOnly", "true")
	}

	if req.StopPrice != "" {
		params.Set("stopPrice", req.StopPrice)
	}

	if req.NewClientOrderId != "" {
		params.Set("newClientOrderId", req.NewClientOrderId)
	}

	if req.WorkingType != "" {
		params.Set("workingType", string(req.WorkingType))
	}

	if req.CallbackRate != "" {
		params.Set("callbackRate", req.CallbackRate)
	}

	if req.ActivationPrice != "" {
		params.Set("activationPrice", req.ActivationPrice)
	}

	body, err := c.doRequest(ctx, "POST", "/fapi/v1/order", params, true)
	if err != nil {
		return nil, err
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &orderResp, nil
}

// CancelOrder 撤销订单
func (c *Client) CancelOrder(ctx context.Context, symbol string, orderID int64) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", strconv.FormatInt(orderID, 10))

	_, err := c.doRequest(ctx, "DELETE", "/fapi/v1/order", params, true)
	return err
}

// GetOrder 查询订单
func (c *Client) GetOrder(ctx context.Context, symbol string, orderID int64) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", strconv.FormatInt(orderID, 10))

	body, err := c.doRequest(ctx, "GET", "/fapi/v1/order", params, true)
	if err != nil {
		return nil, err
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(body, &orderResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &orderResp, nil
}

// GetOpenOrders 查询当前挂单
func (c *Client) GetOpenOrders(ctx context.Context, symbol string) ([]*OrderResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}

	body, err := c.doRequest(ctx, "GET", "/fapi/v1/openOrders", params, true)
	if err != nil {
		return nil, err
	}

	var orders []*OrderResponse
	if err := json.Unmarshal(body, &orders); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return orders, nil
}

// ========== 账户接口 ==========

// GetAccount 查询账户信息
func (c *Client) GetAccount(ctx context.Context) (*AccountInfo, error) {
	params := url.Values{}

	body, err := c.doRequest(ctx, "GET", "/fapi/v2/account", params, true)
	if err != nil {
		return nil, err
	}

	var account AccountInfo
	if err := json.Unmarshal(body, &account); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &account, nil
}

// GetPositionRisk 查询持仓信息
func (c *Client) GetPositionRisk(ctx context.Context, symbol string) ([]*PositionRisk, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}

	body, err := c.doRequest(ctx, "GET", "/fapi/v2/positionRisk", params, true)
	if err != nil {
		return nil, err
	}

	var positions []*PositionRisk
	if err := json.Unmarshal(body, &positions); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return positions, nil
}

// ========== 杠杆与保证金模式 ==========

// SetLeverage 设置杠杆倍数
func (c *Client) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("leverage", strconv.Itoa(leverage))

	_, err := c.doRequest(ctx, "POST", "/fapi/v1/leverage", params, true)
	return err
}

// SetMarginType 设置保证金模式
func (c *Client) SetMarginType(ctx context.Context, symbol string, marginType MarginType) error {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("marginType", string(marginType))

	_, err := c.doRequest(ctx, "POST", "/fapi/v1/marginType", params, true)
	return err
}

// ========== 市场数据接口 ==========

// GetOrderBook 获取订单簿
func (c *Client) GetOrderBook(ctx context.Context, symbol string, limit int) (*OrderBook, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	if limit > 0 {
		params.Set("limit", strconv.Itoa(limit))
	}

	body, err := c.doRequest(ctx, "GET", "/fapi/v1/depth", params, false)
	if err != nil {
		return nil, err
	}

	var orderBook OrderBook
	if err := json.Unmarshal(body, &orderBook); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &orderBook, nil
}

// GetMarkPrice 获取标记价格
func (c *Client) GetMarkPrice(ctx context.Context, symbol string) (float64, error) {
	params := url.Values{}
	params.Set("symbol", symbol)

	body, err := c.doRequest(ctx, "GET", "/fapi/v1/premiumIndex", params, false)
	if err != nil {
		return 0, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("unmarshal response: %w", err)
	}

	markPriceStr, ok := result["markPrice"].(string)
	if !ok {
		return 0, fmt.Errorf("mark price not found")
	}

	return parseFloat(markPriceStr)
}

// ========== 辅助方法 ==========

// GetNthPriceFromOrderBook 获取订单簿第N档价格
// side: "buy" 或 "sell"
// n: 档位（1表示第1档，即最优价格）
func (c *Client) GetNthPriceFromOrderBook(ctx context.Context, symbol, side string, n int) (float64, error) {
	orderBook, err := c.GetOrderBook(ctx, symbol, n*2)
	if err != nil {
		return 0, err
	}

	var levels []OrderBookLevel
	if side == "buy" {
		levels = orderBook.Asks // 买入时看卖单
	} else {
		levels = orderBook.Bids // 卖出时看买单
	}

	if len(levels) < n {
		return 0, fmt.Errorf("orderbook depth insufficient: need %d, got %d", n, len(levels))
	}

	return parseFloat(levels[n-1].Price)
}
