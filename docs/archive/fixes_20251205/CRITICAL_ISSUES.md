# 🐛 发现的两个关键问题

## 问题1: Caller显示错误 ⚠️ （次要问题）

### 现象
```json
{"caller":"v2/message_dispatcher.go:80","msg":"Kline received"}
{"caller":"v2/message_dispatcher.go:80","msg":"Signal generated"}
{"caller":"v2/message_dispatcher.go:80","msg":"Position manager error"}
```

所有日志的`caller`都显示为`message_dispatcher.go:80`

### 原因
调用链：`message_dispatcher.go:80` → `goroutine` → `adapter.OnKline()` → `logger.Info()`

因为在goroutine中调用，zap记录的是goroutine启动位置。

### 影响
- 不影响功能
- 只是日志中的`caller`字段不准确
- 依然可以通过`msg`字段区分不同的日志

### 解决方案（可选）
暂时接受这个限制，或者不使用goroutine分发（会影响性能）。

---

## 问题2: API签名错误 ❌ （**严重问题**）

### 现象
```json
{"level":"ERROR","msg":"Position manager error","error":"get account: API error (status 400): {\"code\":-1022,\"msg\":\"Signature for this request is not valid.\"}"}
```

### 影响
**策略虽然生成了信号，但无法下单！**

```
✅ Signal generated: OPEN_LONG @ 91170.9
❌ Position manager error: 无法获取账户信息
❌ 无法计算仓位大小
❌ 无法下单
```

### 原因分析

可能的原因：
1. **系统时间不同步** - 币安要求时间戳误差<5秒
2. **API密钥格式错误** - 有空格或换行
3. **签名算法错误** - 参数顺序或编码问题
4. **测试网URL错误** - URL和密钥不匹配

### 定位问题

从你之前的测试输出看：
```bash
# 这个能成功
curl "https://testnet.binancefuture.com/fapi/v1/time"

# 但带签名的请求失败
API error (status 400): {"code":-1022,"msg":"Signature for this request is not valid."}
```

说明：
- ✅ 网络连接正常
- ✅ 测试网可访问
- ❌ 签名验证失败

### 检查点

#### 1. 检查系统时间
```bash
# 同步系统时间
sudo ntpdate -u time.nist.gov

# 或
sudo timedatectl set-ntp true
```

#### 2. 检查API密钥
确认密钥没有多余的空格或换行：
```bash
# 当前配置中的密钥
api_key: "UdwTGmKnQ0sFqlOumcBQjc08GGHAMh4rCQX10FfuJTmLgpZFoqBbN1oNyjb28s9z"
secret_key: "IvWeC8asG5Pj2eBF8yF2APamO0SuYvAc7USINqDhBeJxu6hv9kuUASbbxTqjKXCi"
```

#### 3. 验证测试网密钥
登录 https://testnet.binancefuture.com/ 确认：
- ✅ API密钥是测试网的（不是主网的）
- ✅ API权限包含"交易"权限
- ✅ 密钥没有过期

#### 4. 检查签名实现

当前签名代码：
```go
// 添加时间戳
params.Set("timestamp", strconv.FormatInt(time.Now().UnixMilli(), 10))
// 生成签名
signature := c.sign(params.Encode())
params.Set("signature", signature)
```

可能的问题：
- `params.Encode()` 的参数顺序
- URL编码方式
- 时间戳精度

### 快速修复建议

#### 方案1: 同步系统时间（最可能）
```bash
sudo ntpdate -u time.nist.gov
# 然后重启程序
./scripts/start-live.sh
```

#### 方案2: 检查币安服务器时间差
```bash
# 获取币安服务器时间
curl https://testnet.binancefuture.com/fapi/v1/time

# 对比本地时间
date +%s%3N
```

如果时间差>5秒，需要同步时间。

#### 方案3: 添加时间窗口参数
修改签名时添加`recvWindow`参数：
```go
params.Set("recvWindow", "60000") // 60秒窗口
params.Set("timestamp", ...)
```

### 验证修复

修复后，日志应该显示：
```json
{"msg":"Signal generated","signal_type":"OPEN_LONG"}
{"msg":"Creating order","usdt_amount":1000}  ← 成功获取账户信息
{"msg":"Order placed","order_id":"xxx"}      ← 成功下单
```

---

## 🎯 总结

| 问题 | 严重性 | 状态 | 影响 |
|------|-------|------|------|
| Caller显示错误 | ⚠️ 次要 | 可接受 | 不影响功能，只是日志字段不准 |
| API签名错误 | ❌ 严重 | **必须修复** | 策略无法下单！ |

**下一步：修复API签名问题，让策略能够真正下单！**

---

## 🔧 立即修复

```bash
# 1. 同步系统时间
sudo ntpdate -u time.nist.gov

# 2. 重启程序
./scripts/start-live.sh

# 3. 观察日志
tail -f logs/session_*/BTCUSDT_1m.log | jq

# 期望看到：
# {"msg":"Signal generated"}
# {"msg":"Order placed"}  ← 这个是关键！
```

