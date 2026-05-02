## https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/trade/websocket-api 内容

下单 (TRADE)
接口描述
下单

方式
order.place

请求
order.place

{
"id": "3f7df6e3-2df4-44b9-9919-d2f38f90a99a",
"method": "order.place",
"params": {
"apiKey": "HMOchcfii9ZRZnhjp2XjGXhsOBd6msAhKz9joQaWwZ7arcJTlD2hGPHQj1lGdTjR",
"positionSide": "BOTH",
"price": 43187.00,
"quantity": 0.1,
"side": "BUY",
"symbol": "BTCUSDT",
"timeInForce": "GTC",
"timestamp": 1702555533821,
"type": "LIMIT",
"signature": "0f04368b2d22aafd0ggc8809ea34297eff602272917b5f01267db4efbc1c9422"
}
}

请求权重
0

请求参数
名称	类型	是否必需	描述
symbol	STRING	YES	交易对
side	ENUM	YES	买卖方向 SELL, BUY
positionSide	ENUM	NO	持仓方向，单向持仓模式下非必填，默认且仅可填BOTH;在双向持仓模式下必填,且仅可选择 LONG 或 SHORT
type	ENUM	YES	订单类型 LIMIT, MARKET, STOP, TAKE_PROFIT, STOP_MARKET, TAKE_PROFIT_MARKET, TRAILING_STOP_MARKET
reduceOnly	STRING	NO	true, false; 非双开模式下默认false；双开模式下不接受此参数； 使用closePosition不支持此参数。
quantity	DECIMAL	NO	下单数量,使用closePosition不支持此参数。
price	DECIMAL	NO	委托价格
newClientOrderId	STRING	NO	用户自定义的订单号，不可以重复出现在挂单中。如空缺系统会自动赋值。必须满足正则规则 ^[\.A-Z\:/a-z0-9_-]{1,36}$
stopPrice	DECIMAL	NO	触发价, 仅 STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET 需要此参数
closePosition	STRING	NO	true, false；触发后全部平仓，仅支持STOP_MARKET和TAKE_PROFIT_MARKET；不与quantity合用；自带只平仓效果，不与reduceOnly 合用
activationPrice	DECIMAL	NO	追踪止损激活价格，仅TRAILING_STOP_MARKET 需要此参数, 默认为下单当前市场价格(支持不同workingType)
callbackRate	DECIMAL	NO	追踪止损回调比例，可取值范围[0.1, 10],其中 1代表1% ,仅TRAILING_STOP_MARKET 需要此参数
timeInForce	ENUM	NO	有效方法
workingType	ENUM	NO	stopPrice 触发类型: MARK_PRICE(标记价格), CONTRACT_PRICE(合约最新价). 默认 CONTRACT_PRICE
priceProtect	STRING	NO	条件单触发保护："TRUE","FALSE", 默认"FALSE". 仅 STOP, STOP_MARKET, TAKE_PROFIT, TAKE_PROFIT_MARKET 需要此参数
newOrderRespType	ENUM	NO	"ACK", "RESULT", 默认 "ACK"
priceMatch	ENUM	NO	OPPONENT/ OPPONENT_5/ OPPONENT_10/ OPPONENT_20/QUEUE/ QUEUE_5/ QUEUE_10/ QUEUE_20；不能与price同时传
selfTradePreventionMode	ENUM	NO	NONE / EXPIRE_TAKER/ EXPIRE_MAKER/ EXPIRE_BOTH； 默认NONE
goodTillDate	LONG	NO	TIF为GTD时订单的自动取消时间， 当timeInforce为GTD时必传；传入的时间戳仅保留秒级精度，毫秒级部分会被自动忽略，时间戳需大于当前时间+600s且小于253402300799000
recvWindow	LONG	NO
timestamp	LONG	YES
根据 order type的不同，某些参数强制要求，具体如下:

Type	强制要求的参数
LIMIT	timeInForce, quantity, price或priceMatch
MARKET	quantity
STOP, TAKE_PROFIT	quantity, stopPrice
STOP_MARKET, TAKE_PROFIT_MARKET	stopPrice, price或priceMatch
TRAILING_STOP_MARKET	callbackRate
条件单的触发必须:

如果订单参数priceProtect为true:
达到触发价时，MARK_PRICE(标记价格)与CONTRACT_PRICE(合约最新价)之间的价差不能超过改symbol触发保护阈值
触发保护阈值请参考接口GET /fapi/v1/exchangeInfo 返回内容相应symbol中"triggerProtect"字段
STOP, STOP_MARKET 止损单:
买入: 最新合约价格/标记价格高于等于触发价stopPrice
卖出: 最新合约价格/标记价格低于等于触发价stopPrice
TAKE_PROFIT, TAKE_PROFIT_MARKET 止盈单:
买入: 最新合约价格/标记价格低于等于触发价stopPrice
卖出: 最新合约价格/标记价格高于等于触发价stopPrice
TRAILING_STOP_MARKET 跟踪止损单:
买入: 当合约价格/标记价格区间最低价格低于激活价格activationPrice,且最新合约价格/标记价高于等于最低价设定回调幅度。
卖出: 当合约价格/标记价格区间最高价格高于激活价格activationPrice,且最新合约价格/标记价低于等于最高价设定回调幅度。
TRAILING_STOP_MARKET 跟踪止损单如果遇到报错 {"code": -2021, "msg": "Order would immediately trigger."}
表示订单不满足以下条件:

买入: 指定的activationPrice 必须小于 latest price
卖出: 指定的activationPrice 必须大于 latest price
newOrderRespType 如果传 RESULT:

MARKET 订单将直接返回成交结果；
配合使用特殊 timeInForce 的 LIMIT 订单将直接返回成交或过期拒绝结果。
STOP_MARKET, TAKE_PROFIT_MARKET 配合 closePosition=true:

条件单触发依照上述条件单触发逻辑
条件触发后，平掉当时持有所有多头仓位(若为卖单)或当时持有所有空头仓位(若为买单)
不支持 quantity 参数
自带只平仓属性，不支持reduceOnly参数
双开模式下,LONG方向上不支持BUY; SHORT 方向上不支持SELL
响应示例
{
"id": "3f7df6e3-2df4-44b9-9919-d2f38f90a99a",
"status": 200,
"result": {
"orderId": 325078477,
"symbol": "BTCUSDT",
"status": "NEW",
"clientOrderId": "iCXL1BywlBaf2sesNUrVl3",
"price": "43187.00",
"avgPrice": "0.00",
"origQty": "0.100",
"executedQty": "0.000",
"cumQty": "0.000",
"cumQuote": "0.00000",
"timeInForce": "GTC",
"type": "LIMIT",
"reduceOnly": false,
"closePosition": false,
"side": "BUY",
"positionSide": "BOTH",
"stopPrice": "0.00",
"workingType": "CONTRACT_PRICE",
"priceProtect": false,
"origType": "LIMIT",
"priceMatch": "NONE",
"selfTradePreventionMode": "NONE",
"goodTillDate": 0,
"updateTime": 1702555534435
},
"rateLimits": [
{
"rateLimitType": "ORDERS",
"interval": "SECOND",
"intervalNum": 10,
"limit": 300,
"count": 1
},
{
"rateLimitType": "ORDERS",
"interval": "MINUTE",
"intervalNum": 1,
"limit": 1200,
"count": 1
},
{
"rateLimitType": "REQUEST_WEIGHT",
"interval": "MINUTE",
"intervalNum": 1,
"limit": 2400,
"count": 1
}
]
}

## https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/trade/rest-api/New-Algo-Order 内容

下条件单 (TRADE)
接口描述
下条件单

HTTP请求
POST /fapi/v1/algoOrder

请求权重
IP rate limit(x-mbx-used-weight-1m)为0

请求参数
名称	类型	是否必需	描述
algoType	ENUM	YES	仅支持 CONDITIONAL
symbol	STRING	YES	交易对
side	ENUM	YES	买卖方向 SELL, BUY
positionSide	ENUM	NO	持仓方向，单向持仓模式下非必填，默认且仅可填BOTH;在双向持仓模式下必填,且仅可选择 LONG 或 SHORT
type	ENUM	YES	条件订单类型 STOP, TAKE_PROFIT, STOP_MARKET, TAKE_PROFIT_MARKET, TRAILING_STOP_MARKET
timeInForce	ENUM	NO	IOC or GTC or FOK, 默认 GTC
quantity	DECIMAL	NO	下单数量,使用closePosition不支持此参数。
price	DECIMAL	NO	委托价格
triggerPrice	DECIMAL	NO	触发价
workingType	ENUM	NO	触发类型: MARK_PRICE(标记价格), CONTRACT_PRICE(合约最新价). 默认 CONTRACT_PRICE
priceMatch	ENUM	NO	OPPONENT/ OPPONENT_5/ OPPONENT_10/ OPPONENT_20/QUEUE/ QUEUE_5/ QUEUE_10/ QUEUE_20；不能与price同时传
closePosition	STRING	NO	true, false；触发后全部平仓，仅支持STOP_MARKET和TAKE_PROFIT_MARKET；不与quantity合用；自带只平仓效果，不与reduceOnly 合用
priceProtect	STRING	NO	条件单触发保护："TRUE","FALSE", 默认"FALSE".
reduceOnly	STRING	NO	true, false; 非双开模式下默认false；双开模式下不接受此参数； 使用closePosition不支持此参数。
activationPrice	DECIMAL	NO	追踪止损激活价格，仅TRAILING_STOP_MARKET 需要此参数, 默认为下单当前市场价格(支持不同workingType)
callbackRate	DECIMAL	NO	追踪止损回调比例，可取值范围[0.1, 10],其中 1代表1% ,仅TRAILING_STOP_MARKET 需要此参数
clientAlgoId	STRING	NO	用户自定义的条件订单号，不可以重复出现在挂单中。如空缺系统会自动赋值。必须满足正则规则 ^[\.A-Z\:/a-z0-9_-]{1,36}$
newOrderRespType	ENUM	NO	"ACK", "RESULT", 默认 "ACK"
selfTradePreventionMode	ENUM	NO	EXPIRE_TAKER/ EXPIRE_MAKER/ EXPIRE_BOTH； 默认NONE
goodTillDate	LONG	NO	TIF为GTD时订单的自动取消时间， 当timeInforce为GTD时必传；传入的时间戳仅保留秒级精度，毫秒级部分会被自动忽略，时间戳需大于当前时间+600s且小于253402300799000
recvWindow	LONG	NO
timestamp	LONG	YES
条件单的触发必须:

如果订单参数priceProtect为true:
达到触发价时，MARK_PRICE(标记价格)与CONTRACT_PRICE(合约最新价)之间的价差不能超过改symbol触发保护阈值
触发保护阈值请参考接口GET /fapi/v1/exchangeInfo 返回内容相应symbol中"triggerProtect"字段
STOP, STOP_MARKET 止损单:
买入: 最新合约价格/标记价格高于等于触发价stopPrice
卖出: 最新合约价格/标记价格低于等于触发价stopPrice
TAKE_PROFIT, TAKE_PROFIT_MARKET 止盈单:
买入: 最新合约价格/标记价格低于等于触发价stopPrice
卖出: 最新合约价格/标记价格高于等于触发价stopPrice
TRAILING_STOP_MARKET 跟踪止损单:
买入: 当合约价格/标记价格区间最低价格低于激活价格activationPrice,且最新合约价格/标记价高于等于最低价设定回调幅度。
卖出: 当合约价格/标记价格区间最高价格高于激活价格activationPrice,且最新合约价格/标记价低于等于最高价设定回调幅度。
TRAILING_STOP_MARKET 跟踪止损单如果遇到报错 {"code": -2021, "msg": "Order would immediately trigger."}
表示订单不满足以下条件:

买入: 指定的activationPrice 必须小于 latest price
卖出: 指定的activationPrice 必须大于 latest price
STOP_MARKET, TAKE_PROFIT_MARKET 配合 closePosition=true:

条件单触发依照上述条件单触发逻辑
条件触发后，平掉当时持有所有多头仓位(若为卖单)或当时持有所有空头仓位(若为买单)
不支持 quantity 参数
自带只平仓属性，不支持reduceOnly参数
双开模式下,LONG方向上不支持BUY; SHORT 方向上不支持SELL
selfTradePreventionMode 仅在 timeInForce为IOC或GTC或GTD时生效.

响应示例
{
"algoId": 2146760,
"clientAlgoId": "6B2I9XVcJpCjqPAJ4YoFX7",
"algoType": "CONDITIONAL",
"orderType": "TAKE_PROFIT",
"symbol": "BNBUSDT",
"side": "SELL",
"positionSide": "BOTH",
"timeInForce": "GTC",
"quantity": "0.01",
"algoStatus": "NEW",
"triggerPrice": "750.000",
"price": "750.000",
"icebergQuantity": null,
"selfTradePreventionMode": "EXPIRE_MAKER",
"workingType": "CONTRACT_PRICE",
"priceMatch": "NONE",
"closePosition": false,
"priceProtect": false,
"reduceOnly": false,
"activatePrice": "", //TRAILING_STOP_MARKET order
"callbackRate": "",  //TRAILING_STOP_MARKET order
"createTime": 1750485492076,
"updateTime": 1750485492076,
"triggerTime": 0,
"goodTillDate": 0
}


## 部分与币安客服对话如下


如果您在使用API在合约交易中设置止盈、止损或追踪止损订单时遇到问题，并出现诸如“此接口不支持此订单类型。请改用算法订单API接口。”之类的错误，则与ALGO订单的API端点的新更新有关。请确保开始使用此端点：https://developers.binance.com/docs/zh-CN/derivatives/usds-margined-futures/trade/rest-api/New-Algo-Order

请注意，此更新仅适用于止盈、止损和追踪止损订单。其他类型的订单（如限价单和市价单）未更改。

自动翻译
查看原文
07:04
自2025-12-09起，USDⓈ-M合约将把条件委托迁移到算法服务，这将影响以下订单类型：STOP_MARKET/TAKE_PROFIT_MARKET/STOP/TAKE_PROFIT/TRAILING_STOP_MARKET。

REST API条件委托的新端点：

POST fapi/v1/algoOrder: 下达算法委托
DELETE /fapi/v1/algoOrder: 撤销算法委托
DELETE fapi/v1/algoOpenOrders: 撤销所有未完成的算法委托
GET /fapi/v1/algoOrder: 查询算法委托
GET /fapi/v1/openAlgoOrders: 查询算法未完成委托
GET /fapi/v1/allAlgoOrders: 查询算法委托
迁移后，以下端点将阻止订单类型的请求：STOP_MARKET/TAKE_PROFIT_MARKET/STOP/TAKE_PROFIT/TRAILING_STOP_MARKET。将会遇到错误代码-4120 STOP_ORDER_SWITCH_ALGO。

POST /fapi/v1/order
POST /fapi/v1/batchOrders
Websocket用户流更新

新的算法委托事件：ALGO_UPDATE
Websocket API更新

新的算法委托：algoOrder.place
撤销算法委托：algoOrder.cancel
请注意，迁移后：

条件委托触发前没有保证金检查。
GTE_GTC委托不再依赖于相反方向的未完成委托，而是仅依赖于仓位。
订单触发不应有延迟增加。
不支持修改未触发的条件委托。

Effective on 2025-12-09, USDⓈ-M Futures will migrate conditional orders to the Algo Service, which will affect the following order types: STOP_MARKET/TAKE_PROFIT_MARKET/STOP/TAKE_PROFIT/TRAILING_STOP_MARKET.

The new endpoints for conditional orders of REST API :

POST fapi/v1/algoOrder: Place an algo order
DELETE /fapi/v1/algoOrder: Cancel an algo order
DELETE fapi/v1/algoOpenOrders: Cancel all open algo orders
GET /fapi/v1/algoOrder: Query an algo order
GET /fapi/v1/openAlgoOrders: Query algo open order(s)
GET /fapi/v1/allAlgoOrders: Query algo order(s)
The following enpoints will block the requests for order types after the migration: STOP_MARKET/TAKE_PROFIT_MARKET/STOP/TAKE_PROFIT/TRAILING_STOP_MARKET. The error code -4120 STOP_ORDER_SWITCH_ALGO will be encountered.

POST /fapi/v1/order
POST /fapi/v1/batchOrders
Websocket User Stream Update

New algo order event: ALGO_UPDATE
Websocket API Update

New algo order : algoOrder.place
Cancel algo order: algoOrder.cancel
Please note that after the migration:

No margin check before the conditional order gets triggered.
GTE_GTC orders no longer depend on open orders of the opposite side, but rather on positions only.
There should be no latency increase in order triggering.
Modification of untriggered conditional orders is not supported.
