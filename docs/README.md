# goQuant 文档索引

本项目的所有文档说明。

---

## 📚 核心文档

### 用户文档
- **[USER_GUIDE.md](USER_GUIDE.md)** - 完整的用户使用指南
  - 快速开始
  - 配置说明
  - 日志系统使用
  - 模式切换
  - 策略说明
  - 常见问题

- **[01-QUICK_START.md](01-QUICK_START.md)** - 快速入门指南（简化版）

### 开发文档
- **[API_REFERENCE.md](API_REFERENCE.md)** - 完整的API参考文档
  - 所有模块的函数签名
  - 参数和返回值说明
  - 使用示例

- **[02-ARCHITECTURE.md](02-ARCHITECTURE.md)** - 系统架构说明
  - 模块划分
  - 数据流
  - 接口设计

- **[03-STRATEGY.md](03-STRATEGY.md)** - 策略实现说明
  - 技术指标
  - 信号生成
  - 仓位管理

### 修改记录
- **[CHANGELOG.md](CHANGELOG.md)** - 所有重要修改记录
  - 2025-12-06: 仓位管理完善
  - 2025-12-05: 订单数量修复
  - 2025-12-05: API签名修复
  - 2025-12-05: 策略逻辑修复

---

## 📁 归档文档

### 修复过程记录
- **[archive/fixes_20251205/](archive/fixes_20251205/)** - 2025-12-05 详细修复过程
  - STRATEGY_FIX_SUMMARY.md - 策略修复总结
  - VERIFICATION_GUIDE.md - 验证指南
  - FINAL_FIX_SUMMARY.md - 最终修复总结

### 历史文档
- **[archive/](archive/)** - 历史实现文档
  - LOGGING_GUIDE.md - 日志使用指南（已合并到 USER_GUIDE.md）
  - MODE_SWITCH_GUIDE.md - 模式切换指南（已合并到 USER_GUIDE.md）
  - SYMBOL_LOGGING_GUIDE.md - 品种日志指南（已合并到 USER_GUIDE.md）

---

## 🗺️ 快速导航

### 我想...

#### 快速开始使用
→ 阅读 [USER_GUIDE.md](USER_GUIDE.md) 的"快速开始"部分

#### 了解系统架构
→ 阅读 [02-ARCHITECTURE.md](02-ARCHITECTURE.md)

#### 修改或扩展代码
→ 阅读 [API_REFERENCE.md](API_REFERENCE.md)

#### 了解策略逻辑
→ 阅读 [03-STRATEGY.md](03-STRATEGY.md) 或 [USER_GUIDE.md](USER_GUIDE.md) 的"策略说明"部分

#### 查看日志
→ 阅读 [USER_GUIDE.md](USER_GUIDE.md) 的"日志系统"部分

#### 切换运行模式
→ 阅读 [USER_GUIDE.md](USER_GUIDE.md) 的"模式切换"部分

#### 排查问题
→ 阅读 [USER_GUIDE.md](USER_GUIDE.md) 的"常见问题"部分

#### 查看最新修改
→ 阅读 [CHANGELOG.md](CHANGELOG.md)

---

## 📖 文档结构说明

```
docs/
├── README.md                    # 本索引文档
├── USER_GUIDE.md               # ⭐ 用户完整指南
├── CHANGELOG.md                # ⭐ 修改记录
├── API_REFERENCE.md            # ⭐ API参考
├── 01-QUICK_START.md           # 快速开始
├── 02-ARCHITECTURE.md          # 架构说明
├── 03-STRATEGY.md              # 策略说明
├── plansAndProgress.md         # 开发进度跟踪
├── archive/                    # 归档文档
│   ├── fixes_20251205/         # 修复过程记录
│   ├── LOGGING_GUIDE.md        # 已合并
│   ├── MODE_SWITCH_GUIDE.md    # 已合并
│   └── SYMBOL_LOGGING_GUIDE.md # 已合并
└── 文档整理说明.md              # 文档整理记录
```

---

## ⭐ 推荐阅读顺序

### 新用户
1. [01-QUICK_START.md](01-QUICK_START.md) - 5分钟快速入门
2. [USER_GUIDE.md](USER_GUIDE.md) - 详细使用说明
3. [03-STRATEGY.md](03-STRATEGY.md) - 了解策略逻辑

### 开发者
1. [02-ARCHITECTURE.md](02-ARCHITECTURE.md) - 理解系统架构
2. [API_REFERENCE.md](API_REFERENCE.md) - 熟悉API接口
3. [CHANGELOG.md](CHANGELOG.md) - 了解最新修改

### 运维人员
1. [USER_GUIDE.md](USER_GUIDE.md) 的"监控和维护"部分
2. [USER_GUIDE.md](USER_GUIDE.md) 的"常见问题"部分
3. [CHANGELOG.md](CHANGELOG.md) - 跟踪系统变更

---

## 🔄 文档更新

### 最近更新
- **2025-12-06**: 整理文档结构，合并所有 GUIDE 到 USER_GUIDE.md
- **2025-12-06**: 创建完整的 CHANGELOG.md
- **2025-12-05**: 创建 API_REFERENCE.md

### 维护原则
1. **USER_GUIDE.md** - 包含所有用户使用相关的内容
2. **CHANGELOG.md** - 记录所有代码修改和问题修复
3. **API_REFERENCE.md** - 仅包含API接口文档
4. **临时文档** - 修复过程文档归档到 `archive/fixes_YYYYMMDD/`

---

**文档维护者**: AI Assistant  
**最后更新**: 2025-12-06

