# goQuant 量化交易框架

## 📖 文档导航（按阅读顺序）

### 🚀 新手快速上手
1. **[README.md](README.md)** - 项目概览和快速开始
2. **[QUICK_START.md](docs/01-QUICK_START.md)** - 5分钟启动指南

### 📚 核心文档
3. **[架构设计](docs/02-ARCHITECTURE.md)** - 系统架构和模块说明
4. **[策略实现](docs/03-STRATEGY.md)** - MACD+EMA策略详解
5. **[API文档](docs/04-API.md)** - 币安API使用说明

### 🔧 进阶使用
6. **[配置指南](docs/05-CONFIG.md)** - 配置文件详解
7. **[部署运维](docs/06-DEPLOY.md)** - 部署和监控
8. **[常见问题](docs/07-FAQ.md)** - 问题排查

### 📋 开发文档
9. **[开发指南](docs/DEV_GUIDE.md)** - 二次开发说明
10. **[更新日志](docs/CHANGELOG.md)** - 版本历史

---

## 当前项目状态

- ✅ **核心功能**: 100% 完成
- ✅ **文档**: 重新整理中
- ⚠️ **安全**: 发现API密钥泄露，已修复代码

## 立即开始

```bash
# 1. 设置环境变量（重要！）
export BINANCE_API_KEY="your_new_key"
export BINANCE_SECRET_KEY="your_new_secret"

# 2. 配置
cp config/config.example.yaml config/config.yaml

# 3. 启动
./scripts/start-live.sh
```

**⚠️ 安全提醒：请立即重置你的API密钥！**

