package cache

import (
	"context"
	"sync"
	"testing"

	"goQuant/internal/core"
	"goQuant/internal/logger"
)

func init() {
	// Initialize logger for tests
	logConfig := logger.DefaultConfig()
	logConfig.Level = "error"          // Only show errors in tests
	logConfig.OutputPath = "/dev/null" // Don't write logs in tests
	logger.Init(logConfig)
}

// MockExecutor 模拟执行器
type MockExecutor struct {
	account   *core.Account
	positions []*core.Position
	orders    []*core.Order
}

func (m *MockExecutor) GetAccount(ctx context.Context) (*core.Account, error) {
	return m.account, nil
}

func (m *MockExecutor) GetPositions(ctx context.Context) ([]*core.Position, error) {
	return m.positions, nil
}

func (m *MockExecutor) GetOpenOrders(ctx context.Context, symbol string) ([]*core.Order, error) {
	if symbol == "" {
		return m.orders, nil
	}
	filtered := make([]*core.Order, 0)
	for _, order := range m.orders {
		if order.Symbol == symbol {
			filtered = append(filtered, order)
		}
	}
	return filtered, nil
}

// TestNewAccountCache 测试缓存创建
func TestNewAccountCache(t *testing.T) {
	cache := NewAccountCache()
	if cache == nil {
		t.Fatal("NewAccountCache returned nil")
	}
	if cache.positions == nil {
		t.Error("positions map not initialized")
	}
	if cache.orders == nil {
		t.Error("orders map not initialized")
	}
}

// TestInitFromRestAPI 测试从REST API初始化
func TestInitFromRestAPI(t *testing.T) {
	cache := NewAccountCache()

	mockExecutor := &MockExecutor{
		account: &core.Account{
			AvailableBalance: 10000.0,
		},
		positions: []*core.Position{
			{
				Symbol:     "BTCUSDT",
				Side:       core.PositionSideLong,
				Size:       0.5,
				EntryPrice: 50000.0,
			},
			{
				Symbol:     "ETHUSDT",
				Side:       core.PositionSideShort,
				Size:       2.0,
				EntryPrice: 3000.0,
			},
		},
		orders: []*core.Order{
			{
				ID:     "order1",
				Symbol: "BTCUSDT",
				Type:   core.OrderTypeLimit,
				Side:   core.OrderSideBuy,
			},
		},
	}

	ctx := context.Background()
	err := cache.InitFromRestAPI(ctx, mockExecutor)
	if err != nil {
		t.Fatalf("InitFromRestAPI failed: %v", err)
	}

	// 验证余额
	if cache.GetBalance() != 10000.0 {
		t.Errorf("Expected balance 10000.0, got %f", cache.GetBalance())
	}

	// 验证持仓
	if len(cache.positions) != 2 {
		t.Errorf("Expected 2 positions, got %d", len(cache.positions))
	}

	// 验证订单
	if len(cache.orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(cache.orders))
	}
}

// TestBalanceOperations 测试余额操作
func TestBalanceOperations(t *testing.T) {
	cache := NewAccountCache()

	// 初始余额应该为0
	if cache.GetBalance() != 0 {
		t.Errorf("Initial balance should be 0, got %f", cache.GetBalance())
	}

	// 更新余额
	cache.UpdateBalance(5000.0, 1)
	if cache.GetBalance() != 5000.0 {
		t.Errorf("Expected balance 5000.0, got %f", cache.GetBalance())
	}

	// 测试版本控制：旧版本不应更新
	cache.UpdateBalance(3000.0, 0)
	if cache.GetBalance() != 5000.0 {
		t.Errorf("Balance should not be updated by old version, got %f", cache.GetBalance())
	}

	// 新版本应该更新
	cache.UpdateBalance(7000.0, 2)
	if cache.GetBalance() != 7000.0 {
		t.Errorf("Expected balance 7000.0, got %f", cache.GetBalance())
	}
}

// TestPositionOperations 测试持仓操作
func TestPositionOperations(t *testing.T) {
	cache := NewAccountCache()

	// 初始应该没有持仓
	if cache.HasPosition("BTCUSDT") {
		t.Error("Should not have BTCUSDT position initially")
	}

	// 添加持仓
	pos1 := &core.Position{
		Symbol:     "BTCUSDT",
		Side:       core.PositionSideLong,
		Size:       0.5,
		EntryPrice: 50000.0,
	}
	cache.UpdatePosition(pos1, 1)

	// 验证持仓存在
	if !cache.HasPosition("BTCUSDT") {
		t.Error("Should have BTCUSDT position")
	}

	// 获取持仓
	pos, exists := cache.GetPosition("BTCUSDT")
	if !exists {
		t.Error("Position should exist")
	}
	if pos.Size != 0.5 {
		t.Errorf("Expected size 0.5, got %f", pos.Size)
	}

	// 更新持仓
	pos2 := &core.Position{
		Symbol:     "BTCUSDT",
		Side:       core.PositionSideLong,
		Size:       1.0,
		EntryPrice: 51000.0,
	}
	cache.UpdatePosition(pos2, 2)

	pos, _ = cache.GetPosition("BTCUSDT")
	if pos.Size != 1.0 {
		t.Errorf("Expected size 1.0, got %f", pos.Size)
	}

	// 平仓（Size=0）应该删除持仓
	pos3 := &core.Position{
		Symbol:     "BTCUSDT",
		Side:       core.PositionSideLong,
		Size:       0,
		EntryPrice: 51000.0,
	}
	cache.UpdatePosition(pos3, 3)

	if cache.HasPosition("BTCUSDT") {
		t.Error("Position should be deleted when size is 0")
	}
}

// TestOrderOperations 测试订单操作
func TestOrderOperations(t *testing.T) {
	cache := NewAccountCache()

	// 添加订单
	order1 := &core.Order{
		ID:     "order1",
		Symbol: "BTCUSDT",
		Type:   core.OrderTypeLimit,
		Side:   core.OrderSideBuy,
		Status: core.OrderStatusNew,
	}
	cache.UpdateOrder(order1, 1)

	// 获取订单
	order, exists := cache.GetOrder("order1")
	if !exists {
		t.Error("Order should exist")
	}
	if order.Symbol != "BTCUSDT" {
		t.Errorf("Expected symbol BTCUSDT, got %s", order.Symbol)
	}

	// 获取指定交易对的订单
	orders := cache.GetOpenOrders("BTCUSDT")
	if len(orders) != 1 {
		t.Errorf("Expected 1 order, got %d", len(orders))
	}

	// 订单完成应该删除
	order2 := &core.Order{
		ID:     "order1",
		Symbol: "BTCUSDT",
		Type:   core.OrderTypeLimit,
		Side:   core.OrderSideBuy,
		Status: core.OrderStatusFilled,
	}
	cache.UpdateOrder(order2, 2)

	_, exists = cache.GetOrder("order1")
	if exists {
		t.Error("Filled order should be deleted")
	}
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	cache := NewAccountCache()
	var wg sync.WaitGroup

	// 并发更新余额
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(version int) {
			defer wg.Done()
			cache.UpdateBalance(float64(version)*100, int64(version))
		}(i)
	}

	// 并发读取余额
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = cache.GetBalance()
		}()
	}

	wg.Wait()

	// 最终余额应该是版本99的值
	expectedBalance := 99 * 100.0
	if cache.GetBalance() != expectedBalance {
		t.Errorf("Expected balance %f, got %f", expectedBalance, cache.GetBalance())
	}
}

// TestVersionControl 测试版本控制
func TestVersionControl(t *testing.T) {
	cache := NewAccountCache()

	// 按顺序更新
	cache.UpdateBalance(1000.0, 1)
	cache.UpdateBalance(2000.0, 2)
	cache.UpdateBalance(3000.0, 3)

	if cache.GetBalance() != 3000.0 {
		t.Errorf("Expected balance 3000.0, got %f", cache.GetBalance())
	}

	// 乱序更新（应该被忽略）
	cache.UpdateBalance(1500.0, 2)
	cache.UpdateBalance(1200.0, 1)

	if cache.GetBalance() != 3000.0 {
		t.Errorf("Balance should remain 3000.0, got %f", cache.GetBalance())
	}

	// 版本号应该是3
	if cache.GetVersion() != 3 {
		t.Errorf("Expected version 3, got %d", cache.GetVersion())
	}
}

// TestGetAllPositions 测试获取所有持仓
func TestGetAllPositions(t *testing.T) {
	cache := NewAccountCache()

	// 添加多个持仓
	positions := []*core.Position{
		{Symbol: "BTCUSDT", Side: core.PositionSideLong, Size: 0.5},
		{Symbol: "ETHUSDT", Side: core.PositionSideShort, Size: 2.0},
		{Symbol: "BNBUSDT", Side: core.PositionSideLong, Size: 10.0},
	}

	for i, pos := range positions {
		cache.UpdatePosition(pos, int64(i+1))
	}

	// 获取所有持仓
	allPos := cache.GetAllPositions()
	if len(allPos) != 3 {
		t.Errorf("Expected 3 positions, got %d", len(allPos))
	}
}

// TestGetStats 测试统计信息
func TestGetStats(t *testing.T) {
	cache := NewAccountCache()

	cache.UpdateBalance(5000.0, 1)
	cache.UpdatePosition(&core.Position{Symbol: "BTCUSDT", Side: core.PositionSideLong, Size: 0.5}, 2)
	cache.UpdateOrder(&core.Order{ID: "order1", Symbol: "BTCUSDT", Status: core.OrderStatusNew}, 3)

	stats := cache.GetStats()

	if stats["balance"].(float64) != 5000.0 {
		t.Error("Stats balance incorrect")
	}
	if stats["position_count"].(int) != 1 {
		t.Error("Stats position_count incorrect")
	}
	if stats["order_count"].(int) != 1 {
		t.Error("Stats order_count incorrect")
	}
	if stats["update_version"].(int64) != 3 {
		t.Error("Stats update_version incorrect")
	}
}

// TestReset 测试重置
func TestReset(t *testing.T) {
	cache := NewAccountCache()

	// 添加数据
	cache.UpdateBalance(5000.0, 1)
	cache.UpdatePosition(&core.Position{Symbol: "BTCUSDT", Side: core.PositionSideLong, Size: 0.5}, 2)
	cache.UpdateOrder(&core.Order{ID: "order1", Symbol: "BTCUSDT", Status: core.OrderStatusNew}, 3)

	// 重置
	cache.Reset()

	// 验证所有数据已清空
	if cache.GetBalance() != 0 {
		t.Error("Balance should be 0 after reset")
	}
	if len(cache.GetAllPositions()) != 0 {
		t.Error("Positions should be empty after reset")
	}
	if len(cache.GetAllOrders()) != 0 {
		t.Error("Orders should be empty after reset")
	}
	if cache.GetVersion() != 0 {
		t.Error("Version should be 0 after reset")
	}
}

// BenchmarkGetBalance 测试读取余额性能
func BenchmarkGetBalance(b *testing.B) {
	cache := NewAccountCache()
	cache.UpdateBalance(10000.0, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetBalance()
	}
}

// BenchmarkUpdateBalance 测试更新余额性能
func BenchmarkUpdateBalance(b *testing.B) {
	cache := NewAccountCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.UpdateBalance(float64(i), int64(i))
	}
}

// BenchmarkConcurrentRead 测试并发读取性能
func BenchmarkConcurrentRead(b *testing.B) {
	cache := NewAccountCache()
	cache.UpdateBalance(10000.0, 1)
	cache.UpdatePosition(&core.Position{Symbol: "BTCUSDT", Side: core.PositionSideLong, Size: 0.5}, 2)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.GetBalance()
			cache.GetPosition("BTCUSDT")
		}
	})
}
