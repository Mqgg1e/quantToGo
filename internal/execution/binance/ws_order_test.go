package binance

import (
	"encoding/json"
	"testing"
	"time"

	"goQuant/internal/core"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewWSOrderClient 测试创建 WebSocket 订单客户端
func TestNewWSOrderClient(t *testing.T) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	assert.NotNil(t, wsOrder)
	assert.NotNil(t, wsOrder.client)
	assert.NotNil(t, wsOrder.responseCh)
	assert.NotNil(t, wsOrder.stopCh)
	assert.True(t, wsOrder.isTestnet)
}

// TestGetWebSocketURL 测试 WebSocket URL 生成
func TestGetWebSocketURL(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "testnet",
			baseURL:  "https://testnet.binancefuture.com",
			expected: "wss://stream.binancefuture.com/ws-fapi/v1",
		},
		{
			name:     "production",
			baseURL:  "https://fapi.binance.com",
			expected: "wss://fstream.binance.com/ws-fapi/v1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient("test_api_key", "test_secret_key", tt.baseURL)
			wsOrder := NewWSOrderClient(client)
			url := wsOrder.getWebSocketURL()
			assert.Equal(t, tt.expected, url)
		})
	}
}

// TestSignParams 测试参数签名
func TestSignParams(t *testing.T) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	params := map[string]interface{}{
		"symbol": "BTCUSDT",
		"side":   "BUY",
		"type":   "MARKET",
	}

	err := wsOrder.signParams(params)
	require.NoError(t, err)

	// 验证必需的字段已添加
	assert.Equal(t, "test_api_key", params["apiKey"])
	assert.Contains(t, params, "timestamp")
	assert.Contains(t, params, "recvWindow")
	assert.Contains(t, params, "signature")

	// 验证时间戳是合理的
	timestamp, ok := params["timestamp"].(int64)
	require.True(t, ok)
	assert.True(t, timestamp > 0)

	// 验证接收窗口
	recvWindow, ok := params["recvWindow"].(int)
	require.True(t, ok)
	assert.Equal(t, 60000, recvWindow)

	// 验证签名是十六进制字符串
	signature, ok := params["signature"].(string)
	require.True(t, ok)
	assert.NotEmpty(t, signature)
	assert.Len(t, signature, 64) // HMAC SHA256 产生 64 个十六进制字符
}

// TestBuildQueryString 测试查询字符串构建
func TestBuildQueryString(t *testing.T) {
	params := map[string]interface{}{
		"symbol":     "BTCUSDT",
		"side":       "BUY",
		"type":       "MARKET",
		"quantity":   "0.001",
		"timestamp":  int64(1234567890),
		"recvWindow": 60000,
	}

	queryString := buildQueryString(params)

	// 验证所有参数都在查询字符串中
	assert.Contains(t, queryString, "symbol=BTCUSDT")
	assert.Contains(t, queryString, "side=BUY")
	assert.Contains(t, queryString, "type=MARKET")
	assert.Contains(t, queryString, "quantity=0.001")
	assert.Contains(t, queryString, "timestamp=1234567890")
	assert.Contains(t, queryString, "recvWindow=60000")
}

// TestParseOrderResponse 测试订单响应解析
func TestParseOrderResponse(t *testing.T) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	// 模拟币安响应
	response := &WSOrderResponse{
		ID:     "test-id",
		Status: 200,
		Result: map[string]interface{}{
			"orderId":       float64(12345678),
			"clientOrderId": "test_order_123",
			"symbol":        "BTCUSDT",
			"side":          "BUY",
			"type":          "MARKET",
			"status":        "FILLED",
			"origQty":       "0.001",
			"executedQty":   "0.001",
			"price":         "50000.00",
			"avgPrice":      "50000.50",
			"stopPrice":     "0.00",
			"updateTime":    float64(time.Now().UnixMilli()),
		},
	}

	order, err := wsOrder.parseOrderResponse(response)
	require.NoError(t, err)
	assert.NotNil(t, order)

	// 验证订单字段
	assert.Equal(t, "test_order_123", order.ID)
	assert.Equal(t, "BTCUSDT", order.Symbol)
	assert.Equal(t, core.OrderTypeMarket, order.Type)
	assert.Equal(t, core.OrderSideBuy, order.Side)
	assert.Equal(t, core.OrderStatusFilled, order.Status)
	assert.Equal(t, 0.001, order.Quantity)
	assert.Equal(t, 0.001, order.FilledQty)
	assert.Equal(t, 50000.0, order.Price)
	assert.Equal(t, 50000.50, order.AvgPrice)

	// 验证元数据
	assert.NotNil(t, order.Metadata)
	exchangeOrderID, ok := order.Metadata["exchange_order_id"].(string)
	require.True(t, ok)
	assert.Equal(t, "12345678", exchangeOrderID)
}

// TestParseOrderResponseWithStopPrice 测试带止损价的订单响应解析
func TestParseOrderResponseWithStopPrice(t *testing.T) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	response := &WSOrderResponse{
		ID:     "test-id",
		Status: 200,
		Result: map[string]interface{}{
			"orderId":       float64(12345678),
			"clientOrderId": "test_order_123",
			"symbol":        "BTCUSDT",
			"side":          "SELL",
			"type":          "STOP_MARKET",
			"status":        "NEW",
			"origQty":       "0",
			"executedQty":   "0",
			"price":         "0.00",
			"avgPrice":      "0.00",
			"stopPrice":     "48000.00",
			"updateTime":    float64(time.Now().UnixMilli()),
		},
	}

	order, err := wsOrder.parseOrderResponse(response)
	require.NoError(t, err)
	assert.NotNil(t, order)

	assert.Equal(t, core.OrderTypeStopMarket, order.Type)
	assert.Equal(t, 48000.0, order.StopPrice)
	assert.Equal(t, core.OrderStatusNew, order.Status)
}

// TestParseOrderResponseError 测试错误响应解析
func TestParseOrderResponseError(t *testing.T) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	// 空结果
	response := &WSOrderResponse{
		ID:     "test-id",
		Status: 200,
		Result: nil,
	}

	order, err := wsOrder.parseOrderResponse(response)
	assert.Error(t, err)
	assert.Nil(t, order)
	assert.Contains(t, err.Error(), "empty result")
}

// TestWSOrderRequest 测试 WebSocket 订单请求结构
func TestWSOrderRequest(t *testing.T) {
	req := WSOrderRequest{
		ID:     "test-id",
		Method: "order.place",
		Params: map[string]interface{}{
			"symbol": "BTCUSDT",
			"side":   "BUY",
			"type":   "MARKET",
		},
	}

	// 序列化为 JSON
	data, err := json.Marshal(req)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// 验证 JSON 结构
	var decoded map[string]interface{}
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "test-id", decoded["id"])
	assert.Equal(t, "order.place", decoded["method"])
	assert.NotNil(t, decoded["params"])
}

// TestWSOrderResponse 测试 WebSocket 订单响应结构
func TestWSOrderResponse(t *testing.T) {
	// 成功响应
	successJSON := `{
		"id": "test-id",
		"status": 200,
		"result": {
			"orderId": 12345678,
			"symbol": "BTCUSDT",
			"status": "NEW"
		}
	}`

	var response WSOrderResponse
	err := json.Unmarshal([]byte(successJSON), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-id", response.ID)
	assert.Equal(t, 200, response.Status)
	assert.NotNil(t, response.Result)
	assert.Nil(t, response.Error)

	// 错误响应
	errorJSON := `{
		"id": "test-id",
		"status": 400,
		"error": {
			"code": -1102,
			"msg": "Mandatory parameter was not sent"
		}
	}`

	var errorResponse WSOrderResponse
	err = json.Unmarshal([]byte(errorJSON), &errorResponse)
	require.NoError(t, err)

	assert.Equal(t, "test-id", errorResponse.ID)
	assert.Equal(t, 400, errorResponse.Status)
	assert.NotNil(t, errorResponse.Error)
	assert.Equal(t, -1102, errorResponse.Error.Code)
	assert.Equal(t, "Mandatory parameter was not sent", errorResponse.Error.Msg)
}

// TestOrderTypeConversion 测试订单类型转换
func TestOrderTypeConversion(t *testing.T) {
	tests := []struct {
		binanceType string
		coreType    core.OrderType
	}{
		{"MARKET", core.OrderTypeMarket},
		{"LIMIT", core.OrderTypeLimit},
		{"STOP_MARKET", core.OrderTypeStopMarket},
		{"TRAILING_STOP_MARKET", core.OrderTypeTrailingStop},
	}

	for _, tt := range tests {
		t.Run(tt.binanceType, func(t *testing.T) {
			coreType := ToOrderType(FuturesOrderType(tt.binanceType))
			assert.Equal(t, tt.coreType, coreType)
		})
	}
}

// TestOrderSideConversion 测试订单方向转换
func TestOrderSideConversion(t *testing.T) {
	tests := []struct {
		binanceSide string
		coreSide    core.OrderSide
	}{
		{"BUY", core.OrderSideBuy},
		{"SELL", core.OrderSideSell},
	}

	for _, tt := range tests {
		t.Run(tt.binanceSide, func(t *testing.T) {
			coreSide := ToOrderSide(FuturesOrderSide(tt.binanceSide))
			assert.Equal(t, tt.coreSide, coreSide)
		})
	}
}

// TestOrderStatusConversion 测试订单状态转换
func TestOrderStatusConversion(t *testing.T) {
	tests := []struct {
		binanceStatus string
		coreStatus    core.OrderStatus
	}{
		{"NEW", core.OrderStatusNew},
		{"FILLED", core.OrderStatusFilled},
		{"PARTIALLY_FILLED", core.OrderStatusPartiallyFilled},
		{"CANCELED", core.OrderStatusCanceled},
		{"EXPIRED", core.OrderStatusExpired},
	}

	for _, tt := range tests {
		t.Run(tt.binanceStatus, func(t *testing.T) {
			coreStatus := ToOrderStatus(OrderStatus(tt.binanceStatus))
			assert.Equal(t, tt.coreStatus, coreStatus)
		})
	}
}

// TestWSOrderClientStop 测试停止连接
func TestWSOrderClientStop(t *testing.T) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	// 停止未启动的客户端不应该panic
	assert.NotPanics(t, func() {
		wsOrder.Stop()
	})
}

// BenchmarkSignParams 签名性能基准测试
func BenchmarkSignParams(b *testing.B) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	params := map[string]interface{}{
		"symbol":   "BTCUSDT",
		"side":     "BUY",
		"type":     "MARKET",
		"quantity": "0.001",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 清理上次添加的字段
		delete(params, "apiKey")
		delete(params, "timestamp")
		delete(params, "recvWindow")
		delete(params, "signature")

		err := wsOrder.signParams(params)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseOrderResponse 响应解析性能基准测试
func BenchmarkParseOrderResponse(b *testing.B) {
	client := NewClient("test_api_key", "test_secret_key", "https://testnet.binancefuture.com")
	wsOrder := NewWSOrderClient(client)

	response := &WSOrderResponse{
		ID:     "test-id",
		Status: 200,
		Result: map[string]interface{}{
			"orderId":       float64(12345678),
			"clientOrderId": "test_order_123",
			"symbol":        "BTCUSDT",
			"side":          "BUY",
			"type":          "MARKET",
			"status":        "FILLED",
			"origQty":       "0.001",
			"executedQty":   "0.001",
			"price":         "50000.00",
			"avgPrice":      "50000.50",
			"stopPrice":     "0.00",
			"updateTime":    float64(time.Now().UnixMilli()),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := wsOrder.parseOrderResponse(response)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// MockTest - 由于 WebSocket 连接需要实际网络，这里只测试结构和逻辑
// 实际的连接测试应该在 cmd/test-ws-order 中进行
func TestWSOrderClientCreation(t *testing.T) {
	// 测试创建和基本属性
	client := NewClient("test_key", "test_secret", "https://testnet.binancefuture.com")
	wsClient := NewWSOrderClient(client)

	assert.NotNil(t, wsClient)
	assert.NotNil(t, wsClient.client)
	assert.Equal(t, client, wsClient.client)
	assert.True(t, wsClient.isTestnet)

	// 测试响应通道映射初始化
	assert.NotNil(t, wsClient.responseCh)
	assert.Equal(t, 0, len(wsClient.responseCh))
}

// TestFormatQuantityAndPrice 测试数量和价格格式化
func TestFormatQuantityAndPrice(t *testing.T) {
	tests := []struct {
		name     string
		symbol   string
		value    float64
		expected string
	}{
		{"BTC quantity", "BTCUSDT", 0.001, "0.001"},
		{"ETH quantity", "ETHUSDT", 0.1, "0.100"},
		{"BTC price", "BTCUSDT", 50000.123456, "50000.12"},
		{"ETH price", "ETHUSDT", 3000.456789, "3000.46"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result string
			if tt.name == "BTC quantity" || tt.name == "ETH quantity" {
				result = formatQuantity(tt.symbol, tt.value)
			} else {
				result = formatPrice(tt.symbol, tt.value)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}
