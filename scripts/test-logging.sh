#!/bin/bash

echo "🧪 Testing Logging System..."
echo ""

# 创建logs目录
mkdir -p logs

# 编译并运行测试
echo "1️⃣  Compiling test program..."
cd /home/maeda/Documents/projects/goQuant
go build -o bin/test-logging cmd/test-logging/main.go

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"
echo ""

echo "2️⃣  Running logging test..."
./bin/test-logging

echo ""
echo "3️⃣  Checking log file..."

if [ -f "logs/trading.log" ]; then
    echo "✅ Log file created: logs/trading.log"
    echo ""
    echo "📄 Log content:"
    echo "---"
    cat logs/trading.log | jq '.'
    echo "---"
    echo ""
    echo "✅ Logging system test PASSED!"
else
    echo "❌ Log file not found"
    exit 1
fi

