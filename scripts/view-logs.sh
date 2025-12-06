#!/bin/bash

# 日志查看工具

LOG_FILE="logs/trading.log"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "========================================="
echo "  Trading Log Viewer"
echo "========================================="
echo ""

if [ ! -f "$LOG_FILE" ]; then
    echo "❌ Log file not found: $LOG_FILE"
    exit 1
fi

# 显示菜单
show_menu() {
    echo "Select view mode:"
    echo "  1) All logs (real-time)"
    echo "  2) Kline updates only"
    echo "  3) Signals only"
    echo "  4) Orders only"
    echo "  5) Positions only"
    echo "  6) Errors and warnings"
    echo "  7) Last 50 lines"
    echo "  8) Search by keyword"
    echo "  q) Quit"
    echo ""
}

# 实时查看所有日志
view_all() {
    echo "📊 Viewing all logs (Ctrl+C to stop)..."
    tail -f "$LOG_FILE" | jq -r '. | "\(.time) [\(.level)] \(.msg) \(if .symbol then "[\(.symbol)]" else "" end)"'
}

# 查看K线
view_klines() {
    echo "📈 Viewing kline updates (Ctrl+C to stop)..."
    tail -f "$LOG_FILE" | grep "Kline received" | jq -r '. | "\(.time) \(.symbol) \(.interval) Close: \(.close) Vol: \(.volume)"'
}

# 查看信号
view_signals() {
    echo "🎯 Viewing signals (Ctrl+C to stop)..."
    tail -f "$LOG_FILE" | grep -E "Signal generated|Trading signal" | jq -r '. | "\(.time) [\(.signal_type)] \(.symbol) @ \(.price) - \(.reason // .signal)"'
}

# 查看订单
view_orders() {
    echo "📝 Viewing orders (Ctrl+C to stop)..."
    tail -f "$LOG_FILE" | grep -E "Order event|Order placed" | jq -r '. | "\(.time) \(.symbol) \(.side) \(.quantity) @ \(.price) [\(.status)]"'
}

# 查看持仓
view_positions() {
    echo "💰 Viewing positions (Ctrl+C to stop)..."
    tail -f "$LOG_FILE" | grep "Position update" | jq -r '. | "\(.time) \(.symbol) \(.side) Size: \(.size) Entry: \(.entry_price) PnL: \(.unrealized_pnl)%"'
}

# 查看错误和警告
view_errors() {
    echo "⚠️  Viewing errors and warnings..."
    tail -f "$LOG_FILE" | grep -E '"level":"WARN"|"level":"ERROR"' | jq -r '. | "\(.time) [\(.level)] \(.msg)"'
}

# 查看最后N行
view_last() {
    echo "📜 Last 50 lines..."
    tail -50 "$LOG_FILE" | jq -r '. | "\(.time) [\(.level)] \(.msg)"'
}

# 搜索关键词
search_keyword() {
    echo -n "Enter keyword: "
    read keyword
    echo "🔍 Searching for: $keyword"
    grep -i "$keyword" "$LOG_FILE" | jq -r '. | "\(.time) [\(.level)] \(.msg)"'
}

# 主循环
while true; do
    show_menu
    read -p "Enter choice: " choice
    echo ""

    case $choice in
        1) view_all ;;
        2) view_klines ;;
        3) view_signals ;;
        4) view_orders ;;
        5) view_positions ;;
        6) view_errors ;;
        7) view_last ;;
        8) search_keyword ;;
        q|Q) echo "Goodbye!"; exit 0 ;;
        *) echo "Invalid choice" ;;
    esac

    echo ""
    echo "Press Enter to continue..."
    read
done

