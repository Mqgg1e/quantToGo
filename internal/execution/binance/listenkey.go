package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ========== ListenKey 管理 ==========
// UserDataStream 需要 ListenKey 来建立连接
// ListenKey 有效期60分钟，需要定期保活

// ListenKeyResponse ListenKey响应
type ListenKeyResponse struct {
	ListenKey string `json:"listenKey"`
}

// CreateListenKey 创建 ListenKey
// POST /fapi/v1/listenKey
func (c *Client) CreateListenKey(ctx context.Context) (string, error) {
	params := url.Values{}

	body, err := c.doRequest(ctx, http.MethodPost, "/fapi/v1/listenKey", params, false)
	if err != nil {
		return "", fmt.Errorf("create listen key: %w", err)
	}

	var resp ListenKeyResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.ListenKey, nil
}

// KeepAliveListenKey 保持 ListenKey 有效
// PUT /fapi/v1/listenKey
// 建议每30分钟调用一次，ListenKey有效期60分钟
func (c *Client) KeepAliveListenKey(ctx context.Context, listenKey string) error {
	params := url.Values{}
	params.Set("listenKey", listenKey)

	_, err := c.doRequest(ctx, http.MethodPut, "/fapi/v1/listenKey", params, false)
	if err != nil {
		return fmt.Errorf("keep alive listen key: %w", err)
	}

	return nil
}

// CloseListenKey 关闭 ListenKey
// DELETE /fapi/v1/listenKey
func (c *Client) CloseListenKey(ctx context.Context, listenKey string) error {
	params := url.Values{}
	params.Set("listenKey", listenKey)

	_, err := c.doRequest(ctx, http.MethodDelete, "/fapi/v1/listenKey", params, false)
	if err != nil {
		return fmt.Errorf("close listen key: %w", err)
	}

	return nil
}
