#!/bin/bash

# WebSocket 订单测试脚本

set -e

echo "=== Building test-ws-order ==="
go build -o bin/test-ws-order cmd/test-ws-order/main.go

echo ""
echo "=== Running WebSocket Order Test ==="
./bin/test-ws-order

