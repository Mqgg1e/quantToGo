# goQuant K-Line 数据处理 - 项目结构重组

## 📁 项目架构概览

```
goQuant/
├── cmd/bot/                      # 主程序（使用增强版本 v2）
│   └── main.go                  # 使用 v2 EnhancedMultiKlineProcessor
│
├── examples/                     # 使用示例
│   ├── basic/                   # 基础版本示例（v1）
│   │   └── main.go             # 展示基础 MultiKlineProcessor 用法
│   │
│   └── enhanced/                # 增强版本示例（v2）
│       └── main.go             # 展示增强功能（重连、补全、分发）
│
└── internal/dataManager/        # 主包（re-exports 两个版本）
    ├── doc.go                  # 包文档
    ├── models.go               # 共享数据结构 (KlineData)
    ├── utils.go                # 共享工具函数
    ├── dataFromWS.go           # WebSocket 基础订阅
    ├── klinestore.go           # SQLite 存储层
    │
    ├── base/                   # 基础版本 v1
    │   ├── dataFromWS.go       # 基础 WebSocket 处理
    │   ├── processor.go        # 单处理器
    │   ├── multi_processor.go  # 多处理器
    │   ├── klinestore.go       # 存储层
    │   ├── processor_test.go   # 单处理器测试
    │   └── multi_processor_test.go
    │
    └── v2/                     # 增强版本 v2（生产级）
        ├── connection_manager.go        # 自动重连机制
        ├── completion_checker.go        # 完整性检查与REST补全
        ├── message_dispatcher.go        # 多订阅者消息分发
        ├── enhanced_processor.go        # 增强单处理器
        ├── enhanced_multi_processor.go  # 增强多处理器
        ├── enhanced_test.go            # 增强功能测试
        └── utils.go                    # v2 工具函数
```

## 🚀 快速开始

### 方案 1：使用基础版本（v1）

适合学习和简单场景：

```bash
# 运行基础示例
cd examples/basic
go run main.go
```

**基础版本特性：**
- WebSocket 实时数据订阅
- 多交易对/周期支持
- SQLite 分库存储
- 简单稳定

### 方案 2：使用增强版本（v2）- **推荐生产环境**

适合生产环境和复杂场景：

```bash
# 运行增强示例
cd examples/enhanced
go run main.go
```

**增强版本特性：**
- ✅ 自动 WebSocket 重连（指数退避）
- ✅ 24h 断线自动恢复
- ✅ 网络波动抵抗
- ✅ 数据完整性检查
- ✅ REST API 自动补全
- ✅ 多订阅者消息分发
- ✅ 性能统计监控

### 方案 3：主程序（cmd/bot）

```bash
cd cmd/bot
go run main.go
```

主程序默认使用 **增强版本 v2**。

## 📚 API 使用指南

### 基础版本（v1）导入

```go
import base "goQuant/internal/dataManager/base"

// 创建多处理器
processor, _ := base.NewMultiKlineProcessor("/path/to/db")

// 订阅 K 线
msgCh, errCh, closeFn := base.SubscribeKlines(ctx, "BTCUSDT", "1m", proxyURL)
defer closeFn()

// 处理消息流
processor.ProcessStream(ctx, msgCh, errCh)
```

### 增强版本（v2）导入

```go
import v2 "goQuant/internal/dataManager/v2"

// 创建增强处理器（支持重连、补全、分发）
processor, _ := v2.NewEnhancedMultiKlineProcessor("/path/to/db", proxyURL)

// 启动订阅（自动处理重连）
processor.StartSubscription(ctx, "BTCUSDT", "1m")

// 注册自定义订阅者
processor.Subscribe("BTCUSDT", "1m", mySubscriber)

// 查看统计信息
processor.PrintAllStats()
```

## 🔄 版本对比

| 功能 | v1 (基础) | v2 (增强) |
|------|---------|---------|
| WebSocket 订阅 | ✅ | ✅ |
| 多数据库 | ✅ | ✅ |
| 自动重连 | ❌ | ✅ |
| 完整性检查 | ❌ | ✅ |
| REST 补全 | ❌ | ✅ |
| 多订阅者 | ❌ | ✅ |
| 性能监控 | ❌ | ✅ |
| 生产就绪 | ⚠️ | ✅ |

## 🧪 测试

```bash
# 测试基础版本
go test ./internal/dataManager/base -v

# 测试增强版本
go test ./internal/dataManager/v2 -v

# 测试所有
go test ./internal/dataManager/... -v
```

## 📊 目录说明

### `/cmd/bot` - 主程序
- 使用 v2 增强版本
- 展示了如何集成到实际应用
- 包含统计信息输出

### `/examples/basic` - 基础版本示例
- 展示如何使用 v1 版本
- 适合学习和理解基础概念
- 最小依赖，最简单的实现

### `/examples/enhanced` - 增强版本示例
- 展示如何使用 v2 版本
- 包含重连、补全等高级功能
- 生产环境推荐方案

### `/internal/dataManager`
- 核心数据处理包
- `/base` - v1 实现（稳定、简单）
- `/v2` - v2 实现（功能完整、生产级）

## 📖 文档

- `ENHANCEMENT.md` - v2 增强功能详细说明
- `QUICK_REFERENCE.md` - API 快速参考
- `PROCESSOR_COMPARISON.md` - 版本对比详解
- `MULTI_PROCESSOR_GUIDE.md` - 多处理器使用指南

## 🎯 下一步

1. **选择合适的版本**
   - 学习/测试 → 使用 `examples/basic`
   - 生产环境 → 使用 `examples/enhanced` 或 `cmd/bot`

2. **自定义集成**
   ```go
   // 实现 KlineSubscriber 接口
   type MyStrategy struct {}
   
   func (s *MyStrategy) OnKline(kline *KlineData) {
       // 你的策略逻辑
   }
   
   // 注册到处理器
   processor.Subscribe("BTCUSDT", "1m", myStrategy)
   ```

3. **监控和告警**
   - 使用 `GetStats()` 获取实时统计
   - 监控 `ConnectionErrors` 和 `FilledCount`
   - 设置告警阈值

## ⚠️ 重要提示

- **数据库路径**：程序会自动在指定路径创建独立的数据库文件
- **代理配置**：如需代理，传入 `proxyURL` 参数
- **Context 管理**：记得正确处理 Context 的取消和超时
- **信号处理**：建议实现 SIGTERM/SIGINT 处理用于优雅关闭

## 🐛 故障排查

### 连接失败
- 检查网络连接
- 检查代理设置
- 查看日志中的 `Connection error`

### 数据缺失
- 检查 `FilledCount` 统计
- 查看 `Detected missing klines` 日志
- 验证 REST API 配额

### 高内存占用
- 减少缓冲大小（v2 默认 100）
- 定期清理旧数据
- 检查订阅者数量

---

**维护者**: goQuant Team  
**最后更新**: 2025-12-03  
**版本**: v2.0 - 清晰的二元结构
