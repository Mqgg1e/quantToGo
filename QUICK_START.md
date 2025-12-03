# 🚀 快速启动指南

## 立即开始（5分钟）

### 1️⃣ 验证安装
```bash
cd /home/maeda/Documents/projects/goQuant

# 验证编译
go build ./internal/dataManager
go build ./cmd/bot

# 运行测试
go test ./internal/dataManager -v
```

✅ 如果看到 `PASS`，表示安装成功！

### 2️⃣ 简单示例
```go
package main

import (
    "context"
    "time"
    dfs "goQuant/internal/dataManager"
)

func main() {
    // 创建处理器
    p, _ := dfs.NewKlineProcessor("./data/klines.db")
    defer p.Close()

    // 订阅K线
    ctx, _ := context.WithTimeout(context.Background(), 1*time.Minute)
    ch, err, close := dfs.SubscribeKlines(ctx, "BTCUSDT", "1m", "")
    defer close()

    // 自动保存
    p.ProcessStream(ctx, ch, err)
    
    // 查询结果
    klines, _ := p.QueryKlines("BTCUSDT", "1m", 5)
    for _, k := range klines {
        println(k.Symbol, k.ClosePrice)
    }
}
```

### 3️⃣ 复制现成的例子
```bash
# cmd/bot/main.go 已经是完整的示例
# 直接运行：
go run ./cmd/bot/main.go
```

---

## 📚 文档导航

| 文档 | 用途 | 何时阅读 |
|------|------|--------|
| **COMPLETION_REPORT.md** | 功能完成报告 | 了解实现情况 |
| **KLINE_STORAGE_SUMMARY.md** | 功能总结 | 快速了解功能 |
| **KLINE_API_REFERENCE.md** | API参考 | 编码时查阅 |
| **INTEGRATION_GUIDE.md** | 集成指南 | 集成到项目时 |
| **internal/dataManager/README.md** | 详细文档 | 深入学习 |

---

## 🎯 常见任务

### 订阅单个交易对
```go
processor, _ := dfs.NewKlineProcessor("./data/klines.db")
msgCh, errCh, closeFn := dfs.SubscribeKlines(ctx, "ETHUSDT", "5m", "")
processor.ProcessStream(ctx, msgCh, errCh)
```

### 查询已保存的K线
```go
klines, _ := processor.QueryKlines("BTCUSDT", "1m", 10)
for _, k := range klines {
    fmt.Printf("Close: %.2f, Volume: %.2f\n", k.ClosePrice, k.BaseVolume)
}
```

### 获取数据统计
```go
count, _ := processor.GetKlineCount("BTCUSDT", "1m")
fmt.Printf("共保存 %d 条K线\n", count)
```

### 手动保存K线
```go
kline := &dfs.KlineData{
    Symbol:     "BTCUSDT",
    Interval:   "1m",
    ClosePrice: 43500.50,
    IsClosed:   true,
    // ... 其他字段
}
processor.store.SaveKline(kline)
```

---

## ✅ 检查清单

- [ ] 已编译通过（`go build ./internal/dataManager`）
- [ ] 测试全部通过（`go test ./internal/dataManager -v`）
- [ ] 可以访问 `./data/klines.db`
- [ ] 已读 `KLINE_API_REFERENCE.md`
- [ ] 已运行 `cmd/bot/main.go` 示例

---

## 🔍 故障排除

### 问题：编译失败
```bash
go mod tidy
go mod download
go clean -cache
```

### 问题：数据库无法创建
```bash
mkdir -p ./data
chmod 755 ./data
```

### 问题：连接WebSocket失败
```go
// 检查网络
// 尝试添加代理
proxyURL := "http://127.0.0.1:7897"
```

### 问题：数据未保存
```go
// 检查 IsClosed 是否为 true
// 检查是否调用了 ProcessStream
// 检查数据库路径权限
```

---

## 💡 提示

### 数据库位置可自定义
```go
// 默认位置
NewKlineProcessor("./data/klines.db")

// 自定义位置
NewKlineProcessor("/tmp/my_klines.db")
NewKlineProcessor("./crypto_data/btc.db")
```

### 支持多个交易对
```go
symbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT"}
for _, symbol := range symbols {
    go func(sym string) {
        msgCh, errCh, closeFn := dfs.SubscribeKlines(ctx, sym, "1m", "")
        defer closeFn()
        processor.ProcessStream(ctx, msgCh, errCh)
    }(symbol)
}
```

### 时间转换
```go
import "time"

// 毫秒 → time.Time
t := time.UnixMilli(kline.CloseTime)

// 格式化输出
formatted := t.Format("2006-01-02 15:04:05")
```

---

## 🎓 学习路径

1. **新手** → 查看 `KLINE_API_REFERENCE.md`
2. **了解** → 阅读 `KLINE_STORAGE_SUMMARY.md`
3. **实践** → 运行 `cmd/bot/main.go` 示例
4. **深入** → 学习 `internal/dataManager/README.md`
5. **集成** → 参考 `INTEGRATION_GUIDE.md`

---

## 📞 需要帮助？

### 查看示例
```bash
# cmd/bot/main.go - 完整的使用示例
# processor_test.go - 各种操作的示例
```

### 查阅文档
```bash
# 快速参考
cat KLINE_API_REFERENCE.md

# 详细文档
cat internal/dataManager/README.md

# 集成指南
cat INTEGRATION_GUIDE.md
```

### 运行测试
```bash
go test ./internal/dataManager -v
```

---

## 🎉 你已准备好开始了！

现在你可以：
- ✅ 实时订阅K线数据
- ✅ 自动保存收盘K线
- ✅ 查询历史K线数据
- ✅ 进行数据分析

**祝你使用愉快！** 🚀
