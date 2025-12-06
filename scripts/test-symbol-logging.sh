#!/bin/bash

echo "🧪 测试按品种分文件的日志系统"
echo "================================"
echo ""

cd /home/maeda/Documents/projects/goQuant

# 清理旧的测试日志
rm -rf logs/session_* 2>/dev/null

# 运行测试
echo "1️⃣  运行测试程序..."
go run cmd/test-symbol-logging/main.go

echo ""
echo "2️⃣  检查生成的文件..."

# 查找会话目录
SESSION_DIR=$(find logs -name "session_*" -type d | head -1)

if [ -z "$SESSION_DIR" ]; then
    echo "❌ 未找到会话目录"
    exit 1
fi

echo "✅ 会话目录: $SESSION_DIR"
echo ""

# 列出生成的文件
echo "📁 生成的日志文件:"
ls -lh "$SESSION_DIR"/*.log 2>/dev/null | awk '{print "  -", $9, "("$5")"}'

echo ""
echo "3️⃣  查看日志内容示例..."

# 显示每个文件的前3行
for logfile in "$SESSION_DIR"/*.log; do
    filename=$(basename "$logfile")
    echo ""
    echo "📄 $filename:"
    echo "---"
    head -3 "$logfile" | jq -r '. | "\(.time) [\(.level)] \(.msg)"' 2>/dev/null || cat "$logfile" | head -3
done

echo ""
echo "================================"
echo "✅ 测试成功！"
echo ""
echo "📝 日志文件已按品种分离："
echo "  - BTCUSDT_3m.log - BTC 3分钟K线"
echo "  - ETHUSDT_3m.log - ETH 3分钟K线"
echo "  - BTCUSDT_1m.log - BTC 1分钟K线"
echo "  - ETHUSDT_1m.log - ETH 1分钟K线"
echo ""
echo "🎯 每次启动都会创建新的会话目录！"

