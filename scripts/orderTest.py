# binance_ws_order_examples.py
# Python 3.8+
# 依赖: websockets, requests
# pip install websockets requests

import asyncio
import websockets
import os
import time
import hmac
import hashlib
import json
from urllib.parse import urlparse

# =========================
# === 全部变量（请修改）===
# =========================


PROXY = "http://127.0.0.1:7897"   # <-- 修改为你的代理，或者设为 "" 不使用代理

# 如果你还要通过 REST (requests) 调用，requests 的 proxies dict 示例：
REST_PROXIES = None
if PROXY:
    REST_PROXIES = {"http": PROXY, "https": PROXY}

API_KEY = "UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
API_SECRET = "IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"

USE_TESTNET = True  # True -> testnet; False -> 生产
# testnet websocket endpoint (docs: ws-fapi/v1)
WS_ENDPOINT = "wss://testnet.binancefuture.com/ws-fapi/v1" if USE_TESTNET else "wss://ws-fapi.binance.com/ws-fapi/v1"

# 通用单参数示例（按需改）
SYMBOL = "ETHUSDT"
POSITION_SIDE = "BOTH"  # 单向模式填 BOTH；双向模式要填 LONG/SHORT
RECV_WINDOW = 5000

# =========================
# === 签名与帮助函数  ===
# =========================
def now_ms():
    return int(time.time() * 1000)

def sign_params(params: dict, secret: str) -> str:
    """
    对 params（不包含 signature）按 key 字母序拼接 querystring，然后 HMAC-SHA256。
    返回 signature(hex)。
    """
    items = sorted((k, str(v)) for k, v in params.items())
    qs = "&".join(f"{k}={v}" for k, v in items)
    sig = hmac.new(secret.encode(), qs.encode(), hashlib.sha256).hexdigest()
    return sig

async def send_order(ws, payload: dict):
    """发送 payload（完整 JSON），等待并返回第一个匹配 id 的响应（或超时）"""
    payload_text = json.dumps(payload)
    req_id = payload.get("id")
    start = time.perf_counter()
    await ws.send(payload_text)
    # 等待匹配 id 的响应（超时 10s）
    try:
        while True:
            text = await asyncio.wait_for(ws.recv(), timeout=10)
            msg = json.loads(text)
            if str(msg.get("id")) == str(req_id):
                rtt_ms = (time.perf_counter() - start) * 1000.0
                return msg, rtt_ms
            # 若不是该 id 可短路处理或继续循环（这里继续）
    except asyncio.TimeoutError:
        return {"error": "timeout waiting for response"}, None
    

def configure_proxy_for_websockets(proxy_url: str):
    """
    两种可选方式来让 websockets 使用代理：
    1) 设置环境变量 (wss_proxy / https_proxy)，websockets 会读取系统代理配置；
    2) 直接把 proxy_url 传给 websockets.connect(..., proxy=proxy_url)。

    这里我们同时做两件事：
      - 如果 PROXY 非空，则设置常用的环境变量（兼容库/子进程）
      - 返回 proxy 参数（或 None）以便在 connect() 中直接使用（更明确）
    """
    if not proxy_url:
        return None

    # Set env vars so other libs also use same proxy (optional but convenient)
    # 用 lower-case 环境变量最常见 (也支持大写)。wss_proxy 专门用于 wss://
    os.environ.setdefault("wss_proxy", proxy_url)
    os.environ.setdefault("https_proxy", proxy_url)
    os.environ.setdefault("http_proxy", proxy_url)
    # no_proxy 如果需要可在外部设置
    return proxy_url

# =========================
# === 每种订单类型 示例 ===
# =========================
def build_limit_params(price: str, quantity: str, timeInForce: str = "GTC"):
    # LIMIT: 必需 timeInForce, quantity, price 或 priceMatch
    params = {
        "apiKey": API_KEY,
        "symbol": SYMBOL,
        "positionSide": POSITION_SIDE,
        "type": "LIMIT",
        "side": "SELL",
        # "price": price,
        "priceMatch": "QUEUE_5",
        "quantity": quantity,
        "timeInForce": timeInForce,
        "timestamp": now_ms(),
        "recvWindow": RECV_WINDOW,
    }
    params["signature"] = sign_params({k:v for k,v in params.items() if k!="signature"}, API_SECRET)
    return params

def build_market_params(quantity: str):
    # MARKET: 必需 quantity
    params = {
        "apiKey": API_KEY,
        "symbol": SYMBOL,
        "positionSide": POSITION_SIDE,
        "type": "MARKET",
        "side": "SELL",
        "quantity": quantity,
        "timestamp": now_ms(),
        "recvWindow": RECV_WINDOW,
    }
    params["signature"] = sign_params({k:v for k,v in params.items() if k!="signature"}, API_SECRET)
    return params

def build_stop_params(quantity: str, stopPrice: str, stop_type: str = "STOP"):
    # STOP / TAKE_PROFIT: 必需 quantity, stopPrice
    params = {
        "algoType": "CONDITIONAL",
        "triggerPrice": "3500",
        "price": "3550",
        "apiKey": API_KEY,
        "symbol": SYMBOL,
        "positionSide": POSITION_SIDE,
        "type": "STOP",  # "STOP" 或 "TAKE_PROFIT"
        "side": "BUY",
        "quantity": quantity,
        "stopPrice": stopPrice,
        "timestamp": now_ms(),
        "recvWindow": RECV_WINDOW,
    }
    params["signature"] = sign_params({k:v for k,v in params.items() if k!="signature"}, API_SECRET)
    return params

def build_stop_market_params(quantity: str, stopPrice: str, price: str = None, priceMatch: str = None, closePosition: bool = True):
    # STOP_MARKET / TAKE_PROFIT_MARKET: 必需 stopPrice, price 或 priceMatch
    params = {
        "algoType": "CONDITIONAL",
        "triggerPrice": 3500,
        "apiKey": API_KEY,
        "symbol": SYMBOL,
        "positionSide": POSITION_SIDE,
        "type": "STOP_MARKET",  # 或 "TAKE_PROFIT_MARKET"
        "side": "BUY",
        "closePosition": "true",
        # "stopPrice": stopPrice,
        "timestamp": now_ms(),
        "recvWindow": RECV_WINDOW,
    }
    if closePosition:
        # closePosition=true 时不能传 quantity（会平掉所有仓位）
        params["closePosition"] = "true"
    else:
        # 必须传 quantity 和 (price 或 priceMatch)
        params["quantity"] = quantity
        if price is not None:
            params["price"] = price
        elif priceMatch is not None:
            params["priceMatch"] = priceMatch
    params["signature"] = sign_params({k:v for k,v in params.items() if k!="signature"}, API_SECRET)
    return params

def build_trailing_stop_params(quantity: str, callbackRate: str, activationPrice: str = None):
    # TRAILING_STOP_MARKET: 必需 quantity, callbackRate; activationPrice 可选（默认当前市价）
    params = {
        "algoType": "CONDITIONAL",
        "apiKey": API_KEY,
        "symbol": SYMBOL,
        "positionSide": POSITION_SIDE,
        "type": "TRAILING_STOP_MARKET",
        "side": "SELL",
        "activatePrice": activationPrice,
        "quantity": quantity,
        "callbackRate": callbackRate,  # e.g. "1" 表示 1%
        "timestamp": now_ms(),
        "recvWindow": RECV_WINDOW,
    }
    if activationPrice is not None:
        params["activationPrice"] = activationPrice
    params["signature"] = sign_params({k:v for k,v in params.items() if k!="signature"}, API_SECRET)
    return params

# =========================
# === 主程序：建立 WS, 顺序下单示例 ===
# =========================
async def main_send_examples():
    proxy_arg = configure_proxy_for_websockets(PROXY)

    connect_kwargs = {"max_size": None}
    # 如果 proxy_arg 非空, 我们把它传入 connect() 的 proxy 参数（websockets 支持）
    if proxy_arg:
        connect_kwargs["proxy"] = proxy_arg

    # 可选：如果使用 HTTPS proxy 且需要验证 proxy 证书 / TLS 相关配置，可在 connect_kwargs 加入 ssl=...
    async with websockets.connect(WS_ENDPOINT, **connect_kwargs) as ws:
        print("Connected to", WS_ENDPOINT, "via proxy" if PROXY else "direct")

        # LIMIT 示例
        # limit_params = build_limit_params(price="1700.00", quantity="0.01")
        # limit_req = {"id": "req-limit-1", "method": "order.place", "params": limit_params}
        # resp, rtt = await send_order(ws, limit_req)
        # print("LIMIT resp:", resp, "RTT(ms):", rtt)

        # MARKET 示例
        # market_params = build_market_params(quantity="0.01")
        # market_req = {"id": "req-market-1", "method": "order.place", "params": market_params}
        # resp, rtt = await send_order(ws, market_req)
        # print("MARKET resp:", resp, "RTT(ms):", rtt)

        # STOP 示例 (条件单触发价)
        # stop_params = build_stop_params(quantity="0.01", stopPrice="3500.00", stop_type="STOP")
        # stop_req = {"id": "req-stop-1", "method": "algoOrder.place", "params": stop_params}
        # resp, rtt = await send_order(ws, stop_req)
        # print("STOP resp:", resp, "RTT(ms):", rtt)

        # stop_params = build_stop_market_params(quantity=None, stopPrice="3600.00", price="3700")
        # stop_req = {"id": "req-stop-1", "method": "algoOrder.place", "params": stop_params}
        # resp, rtt = await send_order(ws, stop_req)
        # print("STOP resp:", resp, "RTT(ms):", rtt)

        # TRAILING 示例 (跟踪止损)
        trailing_params = build_trailing_stop_params(quantity="0.01", callbackRate="1", activationPrice="2900.00")
        trailing_req = {"id": "req-trailing-1", "method": "algoOrder.place", "params": trailing_params}
        resp, rtt = await send_order(ws, trailing_req)
        print("TRAILING resp:", resp, "RTT(ms):", rtt)

if __name__ == "__main__":
    # 简单帮助信息
    if PROXY:
        print("Proxy enabled:", PROXY)
        # 如果使用 socks 代理，提醒用户安装 python-socks
        if PROXY.startswith("socks"):
            print("Using SOCKS proxy: make sure python-socks[asyncio] is installed:")
            print("  pip install python-socks[asyncio]")
    else:
        print("No proxy configured (PROXY is empty)")

    asyncio.run(main_send_examples())