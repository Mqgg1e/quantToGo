package v2

import (
	"fmt"
	"sync"
)

// KlineSubscriber 定义K线数据订阅者接口
type KlineSubscriber interface {
	// OnKline 当新的K线到达时被调用
	OnKline(kline *KlineData)
	// OnError 当发生错误时被调用
	OnError(err error)
	// Name 返回订阅者名称（用于日志）
	Name() string
}

// MessageDispatcher 高效的消息分发器，支持多个下游订阅者
type MessageDispatcher struct {
	subscribers map[string]KlineSubscriber // key: subscriber name
	mu          sync.RWMutex
	bufferSize  int
}

// NewMessageDispatcher 创建消息分发器
// bufferSize: 分发给每个订阅者的缓冲大小
func NewMessageDispatcher(bufferSize int) *MessageDispatcher {
	if bufferSize < 1 {
		bufferSize = 10
	}

	return &MessageDispatcher{
		subscribers: make(map[string]KlineSubscriber),
		bufferSize:  bufferSize,
	}
}

// Subscribe 注册一个订阅者
func (md *MessageDispatcher) Subscribe(subscriber KlineSubscriber) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	name := subscriber.Name()
	if _, exists := md.subscribers[name]; exists {
		return fmt.Errorf("subscriber %s already exists", name)
	}

	md.subscribers[name] = subscriber
	fmt.Printf("[MessageDispatcher] ✓ Registered subscriber: %s\n", name)
	return nil
}

// Unsubscribe 注销一个订阅者
func (md *MessageDispatcher) Unsubscribe(subscriberName string) error {
	md.mu.Lock()
	defer md.mu.Unlock()

	if _, exists := md.subscribers[subscriberName]; !exists {
		return fmt.Errorf("subscriber %s not found", subscriberName)
	}

	delete(md.subscribers, subscriberName)
	fmt.Printf("[MessageDispatcher] ✓ Unregistered subscriber: %s\n", subscriberName)
	return nil
}

// Dispatch 将K线数据分发给所有订阅者
// 非阻塞：如果某个订阅者处理缓慢，不会影响其他订阅者
func (md *MessageDispatcher) Dispatch(kline *KlineData) {
	md.mu.RLock()
	subscribers := make([]KlineSubscriber, 0, len(md.subscribers))
	for _, sub := range md.subscribers {
		subscribers = append(subscribers, sub)
	}
	md.mu.RUnlock()

	// 并发分发给所有订阅者
	for _, subscriber := range subscribers {
		go func(sub KlineSubscriber) {
			sub.OnKline(kline)
		}(subscriber)
	}
}

// DispatchError 将错误分发给所有订阅者
func (md *MessageDispatcher) DispatchError(err error) {
	md.mu.RLock()
	subscribers := make([]KlineSubscriber, 0, len(md.subscribers))
	for _, sub := range md.subscribers {
		subscribers = append(subscribers, sub)
	}
	md.mu.RUnlock()

	for _, subscriber := range subscribers {
		go func(sub KlineSubscriber) {
			sub.OnError(err)
		}(subscriber)
	}
}

// GetSubscriberCount 获取当前订阅者数量
func (md *MessageDispatcher) GetSubscriberCount() int {
	md.mu.RLock()
	defer md.mu.RUnlock()
	return len(md.subscribers)
}

// ListSubscribers 列出所有订阅者名称
func (md *MessageDispatcher) ListSubscribers() []string {
	md.mu.RLock()
	defer md.mu.RUnlock()

	names := make([]string, 0, len(md.subscribers))
	for name := range md.subscribers {
		names = append(names, name)
	}
	return names
}

// BufferedKlineSubscriber 带缓冲的K线订阅者基类
// 下游可以继承这个类来实现自己的处理逻辑
type BufferedKlineSubscriber struct {
	name    string
	klineCh chan *KlineData
	errCh   chan error
	stopCh  chan struct{}
	wg      sync.WaitGroup
	onKline func(*KlineData)
	onError func(error)
}

// NewBufferedKlineSubscriber 创建缓冲订阅者
func NewBufferedKlineSubscriber(name string, bufferSize int) *BufferedKlineSubscriber {
	return &BufferedKlineSubscriber{
		name:    name,
		klineCh: make(chan *KlineData, bufferSize),
		errCh:   make(chan error, bufferSize),
		stopCh:  make(chan struct{}),
	}
}

// OnKline 接收K线数据（非阻塞）
func (bks *BufferedKlineSubscriber) OnKline(kline *KlineData) {
	select {
	case bks.klineCh <- kline:
	case <-bks.stopCh:
	default:
		// 缓冲满，丢弃数据（防止阻塞）
		fmt.Printf("[%s] Warning: kline buffer full, dropping data\n", bks.name)
	}
}

// OnError 接收错误（非阻塞）
func (bks *BufferedKlineSubscriber) OnError(err error) {
	select {
	case bks.errCh <- err:
	case <-bks.stopCh:
	default:
	}
}

// Name 返回订阅者名称
func (bks *BufferedKlineSubscriber) Name() string {
	return bks.name
}

// Start 启动消费线程
func (bks *BufferedKlineSubscriber) Start(onKline func(*KlineData), onError func(error)) {
	bks.onKline = onKline
	bks.onError = onError

	bks.wg.Add(1)
	go bks.run()
}

// run 消费循环
func (bks *BufferedKlineSubscriber) run() {
	defer bks.wg.Done()

	for {
		select {
		case <-bks.stopCh:
			return

		case kline := <-bks.klineCh:
			if kline != nil && bks.onKline != nil {
				bks.onKline(kline)
			}

		case err := <-bks.errCh:
			if err != nil && bks.onError != nil {
				bks.onError(err)
			}
		}
	}
}

// Stop 停止消费线程
func (bks *BufferedKlineSubscriber) Stop() {
	close(bks.stopCh)
	bks.wg.Wait()

	// 清空缓冲
	close(bks.klineCh)
	close(bks.errCh)
}
