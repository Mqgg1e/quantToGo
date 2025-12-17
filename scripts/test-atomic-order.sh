#!/bin/bash

# 原子订单测试脚本

set -e

echo "=== 原子订单测试 ==="
echo ""

# 进入项目根目录
cd "$(dirname "$0")/.."

# 编译
echo "编译测试程序..."
go build -o bin/test-atomic-order cmd/test-atomic-order/main.go
echo "✅ 编译完成"
echo ""

# 运行
echo "启动测试程序..."
echo "------------------------------------------------"
./bin/test-atomic-order

