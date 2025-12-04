package v2

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// SubscribeKlines 订阅 <symbol>@kline_<interval>
// proxyURL 可以为空，如果不为空则作为 HTTP/HTTPS 代理使用，例如 "http://127.0.0.1:7890"
func SubscribeKlines(ctx context.Context, symbol, interval, proxyURL string) (<-chan []byte, <-chan error, func()) {
	msgCh := make(chan []byte, 16)
	errCh := make(chan error, 4)

	symbol = strings.ToLower(symbol)
	urlStr := fmt.Sprintf("wss://fstream.binance.com/ws/%s@kline_%s", symbol, interval)

	dialer := websocket.Dialer{
		HandshakeTimeout:  5 * time.Second,
		EnableCompression: true,
	}

	// 如果提供了代理，则设置 Proxy
	if proxyURL != "" {
		parsedProxy, err := url.Parse(proxyURL)
		if err != nil {
			go func() {
				errCh <- fmt.Errorf("invalid proxy URL: %w", err)
				close(errCh)
				close(msgCh)
			}()
			return msgCh, errCh, func() {}
		}
		dialer.Proxy = http.ProxyURL(parsedProxy)
	}

	conn, _, err := dialer.Dial(urlStr, nil)
	if err != nil {
		go func() {
			errCh <- fmt.Errorf("dial websocket: %w", err)
			close(errCh)
			close(msgCh)
		}()
		return msgCh, errCh, func() {}
	}

	conn.SetReadLimit(1024 * 1024)
	_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(appData string) error {
		_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	done := make(chan struct{})

	// 读消息 goroutine
	go func() {
		defer func() {
			close(done)
			_ = conn.Close()
			close(msgCh)
			close(errCh)
		}()

		for {
			_, message, rerr := conn.ReadMessage()
			if rerr != nil {
				select {
				case <-ctx.Done():
					return
				default:
				}
				errCh <- fmt.Errorf("read websocket message: %w", rerr)
				return
			}
			select {
			case msgCh <- message:
			case <-ctx.Done():
				return
			}
		}
	}()

	// 心跳 goroutine
	pingTicker := time.NewTicker(20 * time.Second)
	go func() {
		defer pingTicker.Stop()
		for {
			select {
			case <-pingTicker.C:
				_ = conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			case <-ctx.Done():
				return
			case <-done:
				return
			}
		}
	}()

	closeFn := func() {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
		_ = conn.Close()
	}

	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()

	return msgCh, errCh, closeFn
}
