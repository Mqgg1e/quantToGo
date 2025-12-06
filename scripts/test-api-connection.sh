#!/bin/bash

# 测试 Binance API 连接和签名

echo "🧪 Testing Binance API Connection..."
echo ""

# 1. 测试公开接口（不需要签名）
echo "1️⃣  Testing public endpoint (server time)..."
RESPONSE=$(curl -s "https://testnet.binancefuture.com/fapi/v1/time")
echo "Response: $RESPONSE"
echo ""

# 2. 显示本地时间
echo "2️⃣  Local system time:"
date
date +%s%3N
echo ""

# 3. 测试账户接口（需要签名）
echo "3️⃣  Testing authenticated endpoint (account info)..."
API_KEY="UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
SECRET_KEY="IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"
BASE_URL="https://testnet.binancefuture.com"

# 获取时间戳
TIMESTAMP=$(date +%s%3N)

# 构建查询字符串
QUERY_STRING="recvWindow=60000&timestamp=$TIMESTAMP"

# 生成签名
SIGNATURE=$(echo -n "$QUERY_STRING" | openssl dgst -sha256 -hmac "$SECRET_KEY" | awk '{print $2}')

# 发送请求
echo "Timestamp: $TIMESTAMP"
echo "Query String: $QUERY_STRING"
echo "Signature: $SIGNATURE"
echo ""

RESPONSE=$(curl -s -X GET \
  -H "X-MBX-APIKEY: $API_KEY" \
  "${BASE_URL}/fapi/v2/account?${QUERY_STRING}&signature=${SIGNATURE}")

echo "Response:"
echo "$RESPONSE" | python3 -m json.tool 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"code":-1022'; then
  echo "❌ Signature verification failed!"
  echo ""
  echo "Possible reasons:"
  echo "  1. System time is not synchronized with Binance server"
  echo "  2. API Key or Secret is incorrect"
  echo "  3. recvWindow is too small"
  echo ""
  echo "💡 Try synchronizing system time:"
  echo "  sudo ntpdate -u time.google.com"
  echo "  or"
  echo "  sudo timedatectl set-ntp true"
elif echo "$RESPONSE" | grep -q '"totalWalletBalance"'; then
  echo "✅ API connection successful!"
else
  echo "⚠️  Unexpected response"
fi

