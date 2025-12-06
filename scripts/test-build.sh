#!/bin/bash

cd /home/maeda/Documents/projects/goQuant

echo "🔨 开始编译..."
go build -o bin/live-trading cmd/live-trading/main.go 2>&1

if [ $? -eq 0 ]; then
    echo "✅ 编译成功！"
    echo ""
    echo "📁 生成的文件:"
    ls -lh bin/live-trading
    echo ""
    echo "🚀 可以运行了:"
    echo "   ./scripts/start-live.sh"
    exit 0
else
    echo "❌ 编译失败"
    exit 1
fi

