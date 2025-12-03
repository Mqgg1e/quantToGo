# 项目集成指南

## 新增文件清单

### `internal/dataManager/` 目录中的新文件

| 文件 | 描述 |
|------|------|
| `models.go` | K线数据结构和解析逻辑（44行） |
| `utils.go` | 辅助函数：JSON解析、类型转换（42行） |
| `klinestore.go` | SQLite 数据库操作（212行） |
| `processor.go` | 高级处理器：自动保存收盘K线（92行） |
| `processor_test.go` | 单元测试和示例（187行） |
| `README.md` | 详细文档 |

### 根目录文档

| 文件 | 描述 |
|------|------|
| `KLINE_STORAGE_SUMMARY.md` | 功能实现总结 |
| `KLINE_API_REFERENCE.md` | API快速参考 |

### 修改的文件

| 文件 | 修改 |
|------|------|
| `cmd/bot/main.go` | 添加使用示例程序 |
| `cmd/bot/example_klines.go` | 已删除（合并到main.go） |

## 现有代码影响

✅ **完全兼容** - 所有新增代码都是新的包，不影响现有的 `dataFromWS` 包功能

- 保留了原有的 `SubscribeKlines()` 函数
- 保留了原有的 `dataFromWS_test.go` 测试
- 新增功能完全独立，不改动任何现有逻辑

## 快速集成步骤

### 1. 确保依赖已安装
```bash
go get github.com/gorilla/websocket
go get github.com/mattn/go-sqlite3
```

### 2. 验证编译
```bash
go build ./internal/dataManager
go build ./cmd/bot
```

### 3. 运行测试
```bash
go test ./internal/dataManager -v
```

### 4. 在项目中使用

**在你的代码中导入：**
```go
import "goQuant/internal/dataManager"
```

**创建处理器：**
```go
processor, err := dataManager.NewKlineProcessor("./data/klines.db")
if err != nil {
    log.Fatal(err)
}
defer processor.Close()
```

**订阅和处理K线：**
```go
msgCh, errCh, closeFn := dataManager.SubscribeKlines(ctx, "BTCUSDT", "1m", proxyURL)
defer closeFn()

go processor.ProcessStream(ctx, msgCh, errCh)
```

## 目录结构参考

```
goQuant/
├── go.mod
├── go.sum
├── cmd/
│   └── bot/
│       ├── main.go                    ← 更新
│       └── (example_klines.go删除)
├── internal/
│   └── dataManager/
│       ├── dataFromWS.go              ← 原有
│       ├── dataFromWS_test.go         ← 原有
│       ├── models.go                  ← 新增
│       ├── utils.go                   ← 新增
│       ├── klinestore.go              ← 新增
│       ├── processor.go               ← 新增
│       ├── processor_test.go          ← 新增
│       ├── README.md                  ← 新增
├── data/                              ← 自动创建
│   └── klines.db                      ← 数据库文件
├── KLINE_STORAGE_SUMMARY.md           ← 新增
├── KLINE_API_REFERENCE.md             ← 新增
└── pkg/
    └── (其他代码)
```

## 性能考虑

### 数据库优化
- ✅ 批量操作使用事务
- ✅ 复合索引加速查询
- ✅ UNIQUE 约束防止重复

### 并发处理
- ✅ 线程安全的数据库操作
- ✅ 支持多个交易对同时订阅
- ✅ 每个符号一个 goroutine

### 内存管理
- ✅ 自动关闭数据库连接
- ✅ 消息缓冲大小限制（16）
- ✅ 不保留完整消息在内存中

## 故障排除

### 编译错误
```bash
# 清理构建缓存
go clean -cache

# 重新获取依赖
go mod tidy
go mod download
```

### 数据库锁定
- SQLite 在并发写入时可能锁定
- 解决：每个交易对用单独的处理器或添加写入队列

### WebSocket 连接失败
```go
// 检查代理配置
proxyURL := "http://127.0.0.1:7897"  // 根据需要调整

// 或不使用代理
proxyURL := ""
```

### 数据未保存
- 检查 `IsClosed` 是否为 `true`
- 检查数据库路径权限
- 查看错误日志

## 监控和调试

### 检查数据库大小
```bash
ls -lh ./data/klines.db
```

### 查询已保存的K线数量
```go
count, _ := processor.GetKlineCount("BTCUSDT", "1m")
fmt.Printf("已保存 %d 条K线\n", count)
```

### 查看最新的K线
```go
klines, _ := processor.QueryKlines("BTCUSDT", "1m", 5)
for _, k := range klines {
    fmt.Printf("收盘时间: %s, 价格: %.2f\n", 
        time.UnixMilli(k.CloseTime).Format("2006-01-02 15:04:05"),
        k.ClosePrice)
}
```

## 备份和恢复

### 备份数据库
```bash
cp ./data/klines.db ./data/klines.db.backup
```

### 导出为CSV（使用SQLite命令）
```bash
sqlite3 ./data/klines.db ".mode csv" ".output klines.csv" \
  "SELECT * FROM klines ORDER BY close_time DESC;"
```

## 升级注意事项

- 新增的包完全向后兼容
- 现有代码无需修改
- 可以独立升级新功能或修复bug
- 数据库架构稳定，支持扩展

## 常见集成问题

**Q: 能否改变数据库位置？**
A: 可以，通过 `NewKlineProcessor(customPath)` 指定

**Q: 能否同时使用多个处理器？**
A: 可以，但建议每个处理器用不同的数据库文件

**Q: 能否导出数据？**
A: 目前支持通过 SQLite 查询导出，可扩展导出功能

**Q: 能否自定义要保存的字段？**
A: 可以修改 `KlineStore.SaveKline()` 的 SQL 语句

---

**集成完成！所有功能已测试并可投入使用。**
