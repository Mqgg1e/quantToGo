#!/bin/bash

# WebSocket订单原子操作测试脚本

set -e

echo "🚀 WebSocket订单原子操作测试"
echo "================================"
echo ""

# 检查环境变量
if [ -z "$BINANCE_API_KEY" ] || [ -z "$BINANCE_SECRET_KEY" ]; then
    echo "❌ 错误: 请设置环境变量"
    echo ""
    echo "export BINANCE_API_KEY='your_api_key'"
    echo "export BINANCE_SECRET_KEY='your_secret_key'"
    exit 1
fi

echo "✅ 环境变量已设置"
echo ""

# 显示测试场景
echo "📋 测试场景:"
echo "  1. 开市价单（BTC做多0.001）"
echo "  2. 下止损单 → 撤销"
echo "  3. 下跟踪止损单 → 撤销"
echo "  4. 下限价止盈单 → 加仓 → 修改止盈单 → 平仓 → 撤单"
echo ""

# 警告
echo "⚠️  警告:"
echo "  - 这是实盘测试，会真实下单！"
echo "  - 请确保账户有足够余额"
echo "  - BTC价格约90000，ETH价格约3100"
echo "  - 测试会开0.001 BTC（约90 USDT）的仓位"
echo ""

read -p "确认继续测试? (yes/no): " confirm
if [ "$confirm" != "yes" ]; then
    echo "已取消测试"
    exit 0
fi

echo ""
echo "🏃 开始测试..."
echo ""

# 运行测试
cd /home/maeda/Documents/projects/goQuant
go test -v -timeout 5m ./internal/execution/binance -run TestWSOrderAtomic

echo ""
echo "✅ 测试完成!"

