#!/bin/bash

API_KEY="UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
SECRET_KEY="IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"

# 获取服务器时间
timestamp=$(curl -s 'https://testnet.binancefuture.com/fapi/v1/time' | grep -oP '\d+')

echo "Server timestamp: $timestamp"
echo "Testing API key..."

# 构建查询字符串
query_string="timestamp=$timestamp"

# 生成签名
signature=$(echo -n "$query_string" | openssl dgst -sha256 -hmac "$SECRET_KEY" | awk '{print $2}')

echo "Query: $query_string"
echo "Signature: $signature"

# 测试账户接口
curl -X GET "https://testnet.binancefuture.com/fapi/v2/account?timestamp=$timestamp&signature=$signature" \
  -H "X-MBX-APIKEY: $API_KEY"

echo ""

