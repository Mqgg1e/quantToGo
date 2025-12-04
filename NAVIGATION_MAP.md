# 🗺️ goQuant 项目导航地图

## 快速导航

### 👤 "我是新手，想学习"
1. 阅读: `ARCHITECTURE.md`
2. 查看: `examples/basic/main.go`
3. 运行: `cd examples/basic && go run main.go`
4. 文档: `internal/dataManager/README.md`

### 🏭 "我要用于生产环境"
1. 阅读: `ARCHITECTURE.md` 中的版本对比
2. 查看: `examples/enhanced/main.go`
3. 集成: `cmd/bot/main.go` 作为参考
4. 监控: `internal/dataManager/ENHANCEMENT.md`

### 🔧 "我要进行定制开发"
1. 学习接口: `internal/dataManager/base/processor.go`
2. 查看示例: `examples/basic/main.go`
3. 实现接口: `v2.KlineSubscriber` 或 `v2.BufferedKlineSubscriber`
4. 参考: `internal/dataManager/QUICK_REFERENCE.md`

### 🐛 "我遇到了问题"
1. 查看日志和错误信息
2. 参考: `internal/dataManager/ENHANCEMENT.md` 中的故障排查
3. 检查: 连接配置、代理设置、API 配额
4. 文档: `QUICK_REFERENCE.md` 中的常见问题

## 📁 目录指南

```
goQuant/
│
├── 📄 README.md                          ← 项目概述
├── 📄 ARCHITECTURE.md                    ← 项目架构（必读！）
├── 📄 RESTRUCTURE_SUMMARY.md            ← 重组说明
├── 📄 RESTRUCTURE_VISUAL_SUMMARY.md     ← 可视化总结
│
├── 📂 cmd/                               ← 命令行工具
│   └── bot/
│       └── 📄 main.go                   ← 主程序（v2）
│
├── 📂 examples/                         ← 示例代码
│   ├── basic/
│   │   └── 📄 main.go                  ← v1 示例（推荐初学者）
│   │
│   └── enhanced/
│       └── 📄 main.go                  ← v2 示例（推荐生产）
│
├── 📂 internal/dataManager/            ← 核心包
│   │
│   ├── 📄 doc.go                       ← 包文档
│   ├── 📄 models.go                    ← 数据结构
│   ├── 📄 utils.go                     ← 共享工具
│   ├── 📄 dataFromWS.go                ← WebSocket 基础
│   ├── 📄 klinestore.go                ← 数据库层
│   ├── 📄 dataFromWS_test.go           ← 基础测试
│   │
│   ├── 📚 README.md                    ← 基础文档
│   ├── 📚 ENHANCEMENT.md               ← v2 增强说明
│   ├── 📚 QUICK_REFERENCE.md           ← API 参考
│   ├── 📚 PROCESSOR_COMPARISON.md      ← 版本对比
│   ├── 📚 MULTI_PROCESSOR_GUIDE.md     ← 多处理器指南
│   │
│   ├── 📂 base/                        ← v1 版本（基础）
│   │   ├── 📄 processor.go             ← 单处理器
│   │   ├── 📄 multi_processor.go       ← 多处理器
│   │   ├── 📄 processor_test.go
│   │   ├── 📄 multi_processor_test.go
│   │   └── 📄 *.go (from shared)
│   │
│   └── 📂 v2/                          ← v2 版本（增强）
│       ├── 📄 connection_manager.go    ← 重连管理
│       ├── 📄 completion_checker.go    ← 完整性检查
│       ├── 📄 message_dispatcher.go    ← 消息分发
│       ├── 📄 enhanced_processor.go    ← 增强处理器
│       ├── 📄 enhanced_multi_processor.go
│       ├── 📄 enhanced_test.go
│       └── 📄 *.go (from shared)
│
├── 📂 data/                            ← 数据存储（自动创建）
│   ├── wsdata/                        ← v1 基础版本
│   └── basic_wsdata/                 ← 示例用
│
└── go.mod                              ← 项目配置
```

## 🎯 常用命令

### 编译
```bash
# 编译主程序
go build ./cmd/bot

# 编译示例
go build ./examples/basic
go build ./examples/enhanced

# 编译所有
go build ./...
```

### 测试
```bash
# 测试 v1 版本
go test ./internal/dataManager/base -v

# 测试 v2 版本
go test ./internal/dataManager/v2 -v

# 测试所有
go test ./internal/dataManager/... -v
```

### 运行
```bash
# 运行主程序
cd cmd/bot && go run main.go

# 运行 v1 示例
cd examples/basic && go run main.go

# 运行 v2 示例
cd examples/enhanced && go run main.go
```

## 📖 文档阅读顺序

### 初学者路线
1. **ARCHITECTURE.md** (5 min)
   - 理解项目整体架构
   - 了解 v1 和 v2 的区别

2. **examples/basic/main.go** (10 min)
   - 查看最简单的使用方式
   - 理解基本流程

3. **internal/dataManager/README.md** (10 min)
   - 详细的功能说明
   - 数据结构和 API

4. **QUICK_REFERENCE.md** (5 min)
   - API 快速参考
   - 常用代码片段

### 中级开发者路线
1. **ARCHITECTURE.md**
2. **PROCESSOR_COMPARISON.md**
   - 理解两个版本的差异
   - 了解什么时候用哪个版本

3. **ENHANCEMENT.md**
   - v2 的所有功能详解
   - 自动重连、数据补全等

4. **examples/enhanced/main.go**
   - 查看 v2 的使用方式

### 高级开发者路线
1. **ARCHITECTURE.md**
2. **MULTI_PROCESSOR_GUIDE.md**
   - 多处理器的高级用法
   - 性能优化

3. **internal/dataManager/v2/** 源代码
   - 学习实现细节
   - 理解自动重连机制

4. **QUICK_REFERENCE.md** - API 参考

## 🔍 按功能查找

### "我想要..."

#### 订阅 K 线数据
- 文件: `examples/basic/main.go`
- 类: `base.SubscribeKlines()`

#### 自动重连
- 文件: `internal/dataManager/v2/connection_manager.go`
- 类: `v2.ConnectionManager`

#### 补全缺失数据
- 文件: `internal/dataManager/v2/completion_checker.go`
- 类: `v2.CompletionChecker`

#### 多个订阅者处理数据
- 文件: `internal/dataManager/v2/message_dispatcher.go`
- 类: `v2.MessageDispatcher`

#### 查询历史数据
- 文件: `internal/dataManager/klinestore.go`
- 方法: `GetKlines()`, `GetKlineCount()`

#### 监控运行统计
- 文件: `internal/dataManager/v2/enhanced_processor.go`
- 方法: `GetStats()`, `PrintStats()`

#### 自定义数据处理
- 接口: `v2.KlineSubscriber`
- 示例: `internal/dataManager/v2/enhanced_test.go` 中的 `TestSubscriber`

## 💡 代码片段快速查找

### 使用 v1 基础版本
→ 参考 `examples/basic/main.go`

### 使用 v2 增强版本
→ 参考 `examples/enhanced/main.go` 或 `cmd/bot/main.go`

### 实现自定义订阅者
→ 参考 `internal/dataManager/v2/enhanced_example.go` 中的 `StrategySubscriber`

### 查询数据库
→ 参考 `internal/dataManager/klinestore.go`

### 处理重连
→ 参考 `internal/dataManager/v2/connection_manager.go`

## 🎓 学习路径

```
初学者
  ↓
阅读 ARCHITECTURE.md
  ↓
运行 examples/basic
  ↓
理解 v1 基础版本
  ↓
↙ ↘
读源码     学 v2
  ↓       ↓
深入    运行 enhanced
理解      示例
  ↓       ↓
  └─→ 生产开发
      (使用 v2)
```

## ⚠️ 常见陷阱

### ❌ 混淆导入路径
```go
// 错误 - 这是旧版本的导入方式
import dfs "goQuant/internal/dataManager"

// 正确
import v2 "goQuant/internal/dataManager/v2"
import base "goQuant/internal/dataManager/base"
```

### ❌ 忘记处理 Context
```go
// 错误 - 没有 Context
msgCh, errCh, _ := SubscribeKlines(nil, ...)

// 正确
ctx := context.Background()
msgCh, errCh, closeFn := SubscribeKlines(ctx, ...)
defer closeFn()
```

### ❌ 忘记关闭资源
```go
// 错误
processor, _ := NewMultiKlineProcessor(dir)
// 程序结束时没有 Close()

// 正确
processor, _ := NewMultiKlineProcessor(dir)
defer processor.Close()
```

## 📞 获取帮助

1. **查看文档**: `ARCHITECTURE.md` 或相关 `.md` 文件
2. **查看示例**: `examples/` 目录
3. **查看源代码**: `internal/dataManager/` 目录
4. **查看测试**: `*_test.go` 文件
5. **查看日志**: 程序输出中的错误信息

## ✅ 快速检查清单

- [ ] 我理解了 v1 和 v2 的区别
- [ ] 我阅读了相关文档
- [ ] 我运行了相应的示例
- [ ] 我理解了导入方式
- [ ] 我知道了如何处理 Context
- [ ] 我知道了如何关闭资源

---

**如有问题，请按以下顺序查找**:
1. 本导航文档
2. ARCHITECTURE.md
3. internal/dataManager/README.md
4. QUICK_REFERENCE.md
5. 相关源代码和测试

**推荐版本**: v2（生产环境）  
**更新时间**: 2025-12-03
