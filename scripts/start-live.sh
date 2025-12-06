#!/bin/bash

# goQuant 实盘交易启动脚本

set -e

echo "========================================="
echo "  goQuant Live Trading Bot"
echo "========================================="
echo ""

# 检查环境变量
#if [ -z "$BINANCE_API_KEY" ]; then
#    echo "❌ Error: BINANCE_API_KEY not set"
#    echo "Please set environment variables:"
#    echo "  export BINANCE_API_KEY='your_api_key'"
#    echo "  export BINANCE_SECRET_KEY='your_secret_key'"
#    exit 1
#fi
#
#if [ -z "$BINANCE_SECRET_KEY" ]; then
#    echo "❌ Error: BINANCE_SECRET_KEY not set"
#    exit 1
#fi
#
#echo "✅ API keys found"
#echo ""

# 检查配置文件
if [ ! -f "config/config.yaml" ]; then
    echo "❌ Error: config/config.yaml not found"
    echo "Please copy config/config.example.yaml to config/config.yaml"
    exit 1
fi

echo "✅ Config file found"
echo ""

# 编译
echo "🔨 Building..."
go build -o bin/live-trading cmd/live-trading/main.go

if [ $? -eq 0 ]; then
    echo "✅ Build successful"
    echo ""
else
    echo "❌ Build failed"
    exit 1
fi

# 运行
echo "🚀 Starting trading bot..."
echo ""

./bin/live-trading

