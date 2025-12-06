#!/bin/bash

echo "🔍 检查API签名问题"
echo "================================"
echo ""

# 检查系统时间
echo "1️⃣  系统时间检查:"
date -u
echo ""

# 检查API密钥（不显示完整密钥）
echo "2️⃣  API密钥检查:"
API_KEY="UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
SECRET_KEY="IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"

echo "API Key 长度: ${#API_KEY}"
echo "Secret Key 长度: ${#SECRET_KEY}"
echo "API Key 前10位: ${API_KEY:0:10}..."
echo "Secret Key 前10位: ${SECRET_KEY:0:10}..."
echo ""

# 测试签名
echo "3️⃣  测试签名:"
TIMESTAMP=$(date +%s)000
QUERY_STRING="timestamp=${TIMESTAMP}"
SIGNATURE=$(echo -n "${QUERY_STRING}" | openssl dgst -sha256 -hmac "${SECRET_KEY}" | awk '{print $2}')

echo "Query String: ${QUERY_STRING}"
echo "Signature: ${SIGNATURE}"
echo ""

# 测试API连接
echo "4️⃣  测试API连接:"
curl -s "https://testnet.binancefuture.com/fapi/v1/time" | jq
echo ""

# 测试带签名的请求
echo "5️⃣  测试账户查询（带签名）:"
TIMESTAMP=$(date +%s)000
QUERY_STRING="timestamp=${TIMESTAMP}"
SIGNATURE=$(echo -n "${QUERY_STRING}" | openssl dgst -sha256 -hmac "${SECRET_KEY}" | awk '{print $2}')

curl -s -X GET \
  "https://testnet.binancefuture.com/fapi/v2/account?${QUERY_STRING}&signature=${SIGNATURE}" \
  -H "X-MBX-APIKEY: ${API_KEY}" | jq

echo ""
echo "================================"

