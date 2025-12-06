### 我正在构建一个替代freqtrade 的量化系统，具体架构如下

数据模块 ---(市场数据)---> 策略模块 ---(仓位信号)---> 仓位管理 <---（当前信息）-- | --（实际下单）--->执行模块

+日志模块

```json
{
  "nodes": [
    {"id":"data_module","label":"数据模块","type":"source","emits_telemetry": true, "telemetry_topics":["telemetry.metrics","telemetry.logs"]},
    {"id":"strategy_module","label":"策略模块","type":"logic","emits_telemetry": true, "telemetry_topics":["telemetry.logs","telemetry.traces"]},
    {"id":"position_manager","label":"仓位管理","type":"state_manager","emits_telemetry": true, "telemetry_topics":["telemetry.metrics","telemetry.alerts","telemetry.logs"]},
    {"id":"execution_module","label":"执行模块","type":"action","emits_telemetry": true, "telemetry_topics":["telemetry.logs","telemetry.traces","telemetry.metrics"]},
    {
      "id":"observability",
      "label":"监控与日志模块",
      "type":"observability",
      "role":"collector",
      "consumes_topics":["telemetry.metrics","telemetry.logs","telemetry.traces","telemetry.alerts"]
    }
  ],
  "edges": [
    {"from":"data_module","to":"strategy_module","channel":"市场数据"},
    {"from":"data_module","to":"position_manager","channel":"市场数据"},
    {"from":"strategy_module","to":"position_manager","channel":"仓位信号"},
    {"from":"position_manager","to":"execution_module","channel":"实际下单"},
    {"from":"execution_module","to":"position_manager","channel":"当前信息"},

    /* 显式观测边（可选，便于图示） */
    {"from":"data_module","to":"observability","channel":"metrics/logs"},
    {"from":"strategy_module","to":"observability","channel":"logs/traces"},
    {"from":"position_manager","to":"observability","channel":"metrics/alerts/logs"},
    {"from":"execution_module","to":"observability","channel":"logs/traces/metrics"}
  ],
  "telemetry_schema": {
    "log": {"fields":["timestamp","level","component_id","message","context"]},
    "metric": {"fields":["timestamp","metric_name","value","labels","component_id"]},
    "trace": {"fields":["trace_id","span_id","parent_span_id","component_id","start","duration","attributes"]},
    "alert": {"fields":["timestamp","severity","component_id","alert_type","details"]}
  },
  "assertions": [
    {"type":"requirement","expr":"every node that has emits_telemetry=true must have at least one telemetry_topic"},
    {"type":"requirement","expr":"observability.consumes_topics contains all telemetry_topics used by nodes"},
    {"type":"requirement","expr":"no node has duplicate id"},
    {"type":"requirement","expr":"position_manager emits telemetry and has alert topic (critical for risk) "}
  ]
}

```

### 每个模块的功能和预期如下

### 1.数据模块 （部分完成）
基本功能：从币安api获取k线数据、维护订单簿等功能，检查正确性并且推送给下游模块或者记录到数据库中

#### 细节描述
1.从币安api获取websocket(<symbol>@kline_<interval>)推送的k线数据（收盘），获取k线数据是最基本功能
+ 可选择是否使用代理
+ 可能接收多个品种和周期的websocket

2.数据校验与清洗
+ 当发现来自websocket有遗漏时需要调用rest api进行补全
+ 尽量避免api超限
+ 确保无论收到k线还是校验k线都应该是严格按时间顺序推送

3.k线数据可以分流，可以选择保存到db数据库或者是推送给下游策略模块或是同时
+ 保存到数据库时只需要k线的指定部分即可（已有）
+ 可以指定数据库目录，品种和周期写在db名字里

4.可选一个获取订单簿的功能，也推送到下游模块
+ <symbol>@depth<levels> OR <symbol>@depth<levels>@500ms OR <symbol>@depth<levels>@100ms，详细参考 https://developers.binance.com/docs/derivatives/usds-margined-futures/websocket-market-streams/Partial-Book-Depth-Streams
+ 订单簿的调用由策略实际情况决定，可以不调用、当k线推送时一起推送或根据固定间隔推送
+ 订单簿推送时也应该保存在db中

5.websocket重连
+ 币安websocket每24h自动断开，需要实现无中断重连
+ 可能因为网络问题websocket连接不上，需要尝试重连

### 2.策略模块
基本功能：实现不同策略的逻辑和仓位管理

#### 细节描述
1.策略和仓位管理与其他模块分开

2.策略只接收k线，最多只向下游输出开仓、加仓、平仓和方向信号
+ 策略中间可以计算各种指标，但是向下游只输出这些信号
+ 可以保存收到的k线和计算的各种指标和信号

3.仓位管理配合配置文件，决定了仓位大小、保证金模式、杠杆倍数等其他信息

4.仓位管理接收k线和从执行处传来的仓位信息，输出是实际会下的单
+ 可能会下一个单，也可能会接连下多个单
+ 有些单有前提关系，所以需要验证函数
+ 因此要提前根据币安api提供的接口定义好输出用的数据结构
  + 这一部分需要进一步调查确定，不可随便开始

5.策略和仓位管理的具体逻辑都是由其各自的文件定义的，但不管怎么写，策略输入只有k线，输出只有开加平和方向，仓位管理只有策略来的信号和k线，输出只有规定好的数据结构

#! 需要指出的是这部分仍然很粗糙，需要实际验证

### 3.执行模块
基本功能：调用策略模块，接收策略模块的信号，执行根据本地数据的模拟盘，根据websocket数据的本地模拟盘或根据websocket数据的实盘，返回现在账户信息、仓位情况和委托情况给仓位管理

#### 细节描述
1.根据币安api允许的下单方式定义数据结构

2.实现本地的模拟交易逻辑，包括手续费，下单，止盈止损，等币安api支持的下单方式，尽量迁移到本地实现
+ 市价单是在k线收盘时入场，限价单是先挂单，然后在价格经过时入场
+ 同样的，为了统一与交易所实盘的api接口，尽量符合其规则

3.使用websocket数据时，需要根据策略情况通过rest api预热数根k线
+ 这一点也同样适用于刚启动或断连时，或者需要websocket但是要先计算指标

4.接受关于账户信息、仓位情况和委托情况给仓位管理的websocket推送，并推送给仓位管理

#! 需要指出的是这部分仍然很粗糙，需要实际验证
#! api功能还没遍历和验证


### 4.日志和监控模块
基本功能：记录每次运行的信息和异常情况，监控异常并且能做出通知

#### 细节描述
1.每次开始运行时，记录时间，品种等详细信息

2.出现rest填补时记下缺失的k线时间，k线顺序失效时也记录

3.websocket断开或重连时记录

4.策略模块接收或发出异常时记录

5.异常终止或手动终止时记录

5.可能会通过sns发送异常警告，但暂时不实现

## 整体要求
1.有很大概率会人工手动测试各个模块，实现的时候注意标注每个函数的功能，以及输入输出的内容和类型，如果有些函数相关性强也要同时标注出前后相关函数

2.留下可能会有gui迭代需求，需要留下重构空间

3.整体方案的细节有待优化，没必要立刻实现全部功能，实现时应该逐步进行

4.注意文档迭代