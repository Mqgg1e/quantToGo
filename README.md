# goQuant

> 生产级量化交易框架 - 币安期货实盘交易系统

[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Status](https://img.shields.io/badge/Status-Production%20Ready-success)](.)

## 快速开始

### 测试网模式（推荐新手）

```bash
# 1. 获取测试网API密钥
# 访问: https://testnet.binancefuture.com/

# 2. 配置文件已预设好（使用测试网）
# config/config.yaml 已配置

# 3. 直接启动
./scripts/start-live.sh
```

### 实盘模式（有经验用户）

```bash
# 1. 修改配置文件
vim config/config.yaml
# 将 base_url 改为: https://fapi.binance.com
# 将 testnet 改为: false
# 设置你的实盘API密钥

# 2. 启动
./scripts/start-live.sh
```

**完整教程**: 查看 [STATUS_SUMMARY.md](STATUS_SUMMARY.md)

## 核心功能

- ✅ **MACD+EMA+VWAP** 组合策略
- ✅ **20%/40%** 资金分配 + **5倍**杠杆
- ✅ **0.6%**固定止损 + **三段**跟踪止盈
- ✅ 币安期货API完整封装
- ✅ 实时数据 + 自动重连 + 数据完整性检查

## 文档导航

| 文档 | 说明 |
|------|------|
| [STATUS_SUMMARY.md](STATUS_SUMMARY.md) | ⭐ 运行状态说明 |
| [MODE_SWITCH_GUIDE.md](docs/MODE_SWITCH_GUIDE.md) | ⭐ 模式切换指南 |
| [快速开始](docs/01-QUICK_START.md) | 5分钟上手 |
| [架构设计](docs/02-ARCHITECTURE.md) | 系统架构 |
| [策略详解](docs/03-STRATEGY.md) | 策略规则 |
| [更新日志](docs/CHANGELOG.md) | 版本历史 |

## 常用命令

```bash
# 启动程序
./scripts/start-live.sh

# 查看状态
./scripts/check-status.sh

# 查看K线数据
sqlite3 data/wsdata/BTCUSDT_3m.db "SELECT COUNT(*) FROM klines;"

# 停止程序
pkill live-trading
```

## 架构

```
WebSocket → DataManager → Strategy → Signal → 
PositionManager → Order → Executor → Binance
```

## ⚠️ 重要提醒

1. **API密钥安全**: 
   - ❌ 不要硬编码到代码中
   - ✅ 使用环境变量
   - ✅ 限制API权限（只开交易）

2. **必须先测试**: 
   - ✅ 在测试网测试 (https://testnet.binancefuture.com/)
   - ✅ 小额资金开始 (<1000 USDT)
   - ✅ 监控24小时后再加大资金

3. **风险提示**: 
   - 量化交易有风险
   - 持续监控运行
   - 准备应急方案

## 项目状态

- **版本**: v1.0.0
- **发布**: 2024-12-05
- **状态**: ✅ 生产就绪

## License

MIT

---

**⚡ 开始你的量化交易！**

