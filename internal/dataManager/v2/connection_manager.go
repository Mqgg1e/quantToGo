package v2

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ConnectionManager 管理WebSocket连接的生命周期，包括重连逻辑
type ConnectionManager struct {
	symbol    string
	interval  string
	proxyURL  string
	maxRetry  int
	backoffFn func(attempt int) time.Duration
}

// NewConnectionManager 创建连接管理器
func NewConnectionManager(symbol, interval, proxyURL string) *ConnectionManager {
	return &ConnectionManager{
		symbol:   symbol,
		interval: interval,
		proxyURL: proxyURL,
		maxRetry: 10,
		// 指数退避策略：1s, 2s, 4s, 8s, ..., max 5m
		backoffFn: func(attempt int) time.Duration {
			seconds := math.Pow(2, float64(attempt))
			maxSeconds := 300.0
			if seconds > maxSeconds {
				seconds = maxSeconds
			}
			return time.Duration(seconds) * time.Second
		},
	}
}

// ConnectWithRetry 尝试连接，支持重连机制
// 返回: msgCh, errCh, closeFn, 连接是否成功
func (cm *ConnectionManager) ConnectWithRetry(ctx context.Context) (<-chan []byte, <-chan error, func(), bool) {
	for attempt := 0; attempt < cm.maxRetry; attempt++ {
		msgCh, errCh, closeFn := SubscribeKlines(ctx, cm.symbol, cm.interval, cm.proxyURL)

		// 快速检查是否成功连接（等待100ms看是否有错误）
		select {
		case <-ctx.Done():
			return msgCh, errCh, closeFn, false
		case err := <-errCh:
			if err == nil {
				// 没有错误，连接成功
				fmt.Printf("[%s %s] ✓ Connected on attempt %d\n", cm.symbol, cm.interval, attempt+1)
				return msgCh, errCh, closeFn, true
			}
			fmt.Printf("[%s %s] Connection failed on attempt %d: %v\n", cm.symbol, cm.interval, attempt+1, err)
			closeFn()
		case <-time.After(100 * time.Millisecond):
			// 100ms内没有错误就认为连接成功
			fmt.Printf("[%s %s] ✓ Connected on attempt %d\n", cm.symbol, cm.interval, attempt+1)
			return msgCh, errCh, closeFn, true
		}

		if attempt < cm.maxRetry-1 {
			backoffDuration := cm.backoffFn(attempt)
			fmt.Printf("[%s %s] Retrying in %v (attempt %d/%d)\n", cm.symbol, cm.interval, backoffDuration, attempt+2, cm.maxRetry)

			select {
			case <-ctx.Done():
				return nil, nil, func() {}, false
			case <-time.After(backoffDuration):
				// 继续重试
			}
		}
	}

	fmt.Printf("[%s %s] ✗ Failed to connect after %d attempts\n", cm.symbol, cm.interval, cm.maxRetry)
	return nil, nil, func() {}, false
}

// MonitorConnection 监控连接状态，如果连接断开则自动重连
// 如果ctx被取消或达到最大重试次数则返回
func (cm *ConnectionManager) MonitorConnection(ctx context.Context, handler func(<-chan []byte, <-chan error)) {
	for {
		msgCh, errCh, closeFn, success := cm.ConnectWithRetry(ctx)
		if !success {
			fmt.Printf("[%s %s] ✗ Failed to establish connection, giving up\n", cm.symbol, cm.interval)
			return
		}

		// 处理消息流，直到连接断开或context取消
		disconnected := cm.handleMessages(ctx, msgCh, errCh, handler)

		closeFn()

		if !disconnected {
			// context被取消，不再重连
			return
		}

		// 连接断开，等待后重连
		backoffDuration := 5 * time.Second
		fmt.Printf("[%s %s] Connection lost, reconnecting in %v\n", cm.symbol, cm.interval, backoffDuration)

		select {
		case <-ctx.Done():
			return
		case <-time.After(backoffDuration):
			// 继续重连循环
		}
	}
}

// handleMessages 处理消息直到连接断开或ctx取消
// 返回true表示连接断开，返回false表示ctx取消
func (cm *ConnectionManager) handleMessages(ctx context.Context, msgCh <-chan []byte, errCh <-chan error, handler func(<-chan []byte, <-chan error)) bool {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	lastMessageTime := time.Now()

	for {
		select {
		case <-ctx.Done():
			return false

		case err := <-errCh:
			if err != nil {
				fmt.Printf("[%s %s] Connection error: %v\n", cm.symbol, cm.interval, err)
				return true // 连接断开
			}
			return true

		case msg := <-msgCh:
			if msg == nil {
				fmt.Printf("[%s %s] Message channel closed\n", cm.symbol, cm.interval)
				return true // 连接断开
			}
			lastMessageTime = time.Now()
			handler(make(<-chan []byte), make(<-chan error)) // 调用处理函数
			// 实际处理由外层调用者通过handler完成

		case <-ticker.C:
			// 检查是否接收到心跳（心跳间隔应该小于30秒）
			timeSinceLastMessage := time.Since(lastMessageTime)
			if timeSinceLastMessage > 45*time.Second {
				fmt.Printf("[%s %s] No message received for %v, treating as disconnected\n", cm.symbol, cm.interval, timeSinceLastMessage)
				return true // 连接断开
			}
		}
	}
}
