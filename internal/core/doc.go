// Package core 定义了goQuant量化交易框架的核心接口和类型
//
// 该包提供了所有模块的标准接口定义，确保模块间的解耦和可测试性。
//
// 主要接口：
//   - DataProvider: 数据提供者（K线、订单簿）
//   - Strategy: 交易策略
//   - PositionManager: 仓位管理
//   - Executor: 订单执行器（回测/实盘）
//   - Logger, MetricsCollector, Alerter: 可观测性组件
//
// 设计原则：
//  1. 接口优先：所有模块通过接口交互，便于mock和测试
//  2. 明确职责：每个接口只负责一个领域
//  3. 可扩展性：预留扩展点，支持多种实现
//  4. 类型安全：使用明确的类型定义，避免interface{}滥用
//
// 使用示例：
//
//	// 实现策略接口
//	type MyStrategy struct { ... }
//	func (s *MyStrategy) OnKline(kline core.KlineData) (*core.TradingSignal, error) { ... }
//
//	// 注入依赖
//	executor := backtest.NewBacktestExecutor(...)
//	positionMgr := position.NewManager(executor)
//	strategy := strategy.NewMyStrategy()
package core
