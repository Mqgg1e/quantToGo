#!/bin/bash

echo "========================================="
echo "  goQuant 运行状态检查"
echo "========================================="
echo ""

# 1. 检查进程
echo "📊 1. 进程状态"
if pgrep -f "live-trading" > /dev/null; then
    echo "✅ 程序正在运行"
    ps aux | grep live-trading | grep -v grep
else
    echo "❌ 程序未运行"
fi
echo ""

# 2. 检查数据库
echo "📂 2. 数据库状态"
if [ -d "data/wsdata" ]; then
    echo "✅ 数据目录存在"
    for db in data/wsdata/*.db; do
        if [ -f "$db" ]; then
            echo "  - $(basename $db)"
            count=$(sqlite3 "$db" "SELECT COUNT(*) FROM klines;" 2>/dev/null)
            if [ $? -eq 0 ]; then
                echo "    K线数量: $count"
                latest=$(sqlite3 "$db" "SELECT datetime(MAX(start_time)/1000, 'unixepoch') FROM klines;" 2>/dev/null)
                echo "    最新时间: $latest"
            fi
        fi
    done
else
    echo "⚠️  数据目录不存在"
fi
echo ""

# 3. 检查配置
echo "⚙️  3. 当前配置"
if [ -f "config/config.yaml" ]; then
    echo "  模式: $(grep 'mode:' config/config.yaml | head -1 | awk '{print $2}' | tr -d '"')"
    echo "  URL: $(grep 'base_url:' config/config.yaml | head -1 | awk '{print $2}' | tr -d '"')"
    echo "  测试网: $(grep 'testnet:' config/config.yaml | head -1 | awk '{print $2}')"
else
    echo "❌ 配置文件不存在"
fi
echo ""

# 4. 网络连接
echo "🌐 4. 网络状态"
if curl -s --max-time 3 "https://testnet.binancefuture.com/fapi/v1/time" > /dev/null; then
    echo "✅ 测试网可访问"
else
    echo "⚠️  测试网无法访问（可能需要代理）"
fi
echo ""

# 5. 日志文件
echo "📝 5. 最新日志 (最后10行)"
if [ -f "logs/trading.log" ]; then
    tail -10 logs/trading.log
else
    echo "⚠️  日志文件不存在"
fi

echo ""
echo "========================================="
echo "检查完成"
echo "========================================="

