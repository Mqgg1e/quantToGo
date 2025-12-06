# 文档整理说明 (2025-12-05)

## AI生成文档归档

以下文档是AI在修复过程中生成的临时文档，已归档到 `docs/archive/fixes_20251205/`：

### 修复相关文档
1. **STRATEGY_FIX_SUMMARY.md** - 策略逻辑修复详细说明
2. **VERIFICATION_GUIDE.md** - 修复验证指南
3. **API_SIGNATURE_ERROR.md** - API签名错误分析（已废弃，问题已解决）
4. **FINAL_FIX_SUMMARY.md** - 最终修复总结

### 实现状态文档（已过时）
这些文档记录了开发过程中的状态，部分内容已过时：

1. **CRITICAL_ISSUES.md** - 关键问题列表（已解决）
2. **IMPLEMENTATION_STATUS.md** - 实现状态（已过时）
3. **STATUS_SUMMARY.md** - 状态总结（已过时）
4. **LOGGING_IMPLEMENTATION.md** - 日志实现说明
5. **LOGGING_READY.md** - 日志系统就绪
6. **LOGGING_SPLIT_BY_SYMBOL.md** - 按品种分日志
7. **SYMBOL_LOGGING_READY.md** - 品种日志就绪
8. **REST_WARMUP_READY.md** - REST预热就绪
9. **NAVIGATION_MAP.md** - 导航地图

### 架构文档（保留）
以下文档保留在根目录：

1. **README.md** - 项目说明（用户文档）
2. **ARCHITECTURE.md** - 架构说明（用户文档）
3. **testStrategy.md** - 策略定义（用户文档）

### 新增文档
1. **docs/API_REFERENCE.md** - 完整的API参考文档

---

## 建议的文档结构

```
goQuant/
├── README.md                          # 项目主文档
├── ARCHITECTURE.md                    # 系统架构说明
├── testStrategy.md                    # 策略定义
├── docs/
│   ├── API_REFERENCE.md              # API 参考文档（新增）
│   ├── 01-QUICK_START.md             # 快速开始
│   ├── 02-ARCHITECTURE.md            # 架构详解
│   ├── 03-STRATEGY.md                # 策略说明
│   ├── LOGGING_GUIDE.md              # 日志使用指南
│   ├── MODE_SWITCH_GUIDE.md          # 模式切换指南
│   ├── SYMBOL_LOGGING_GUIDE.md       # 品种日志指南
│   ├── CHANGELOG.md                  # 变更日志
│   ├── plansAndProgress.md           # 进度跟踪
│   └── archive/
│       └── fixes_20251205/           # 修复过程文档归档
│           ├── STRATEGY_FIX_SUMMARY.md
│           ├── VERIFICATION_GUIDE.md
│           ├── FINAL_FIX_SUMMARY.md
│           └── ... (其他临时文档)
```

---

## 清理操作

可以执行以下命令将根目录的临时文档移到归档目录：

```bash
cd /home/maeda/Documents/projects/goQuant

# 创建归档目录
mkdir -p docs/archive/fixes_20251205

# 移动修复相关文档
mv STRATEGY_FIX_SUMMARY.md docs/archive/fixes_20251205/
mv VERIFICATION_GUIDE.md docs/archive/fixes_20251205/
mv FINAL_FIX_SUMMARY.md docs/archive/fixes_20251205/

# 移动状态文档
mv CRITICAL_ISSUES.md docs/archive/fixes_20251205/
mv IMPLEMENTATION_STATUS.md docs/archive/fixes_20251205/
mv STATUS_SUMMARY.md docs/archive/fixes_20251205/
mv LOGGING_IMPLEMENTATION.md docs/archive/fixes_20251205/
mv LOGGING_READY.md docs/archive/fixes_20251205/
mv LOGGING_SPLIT_BY_SYMBOL.md docs/archive/fixes_20251205/
mv SYMBOL_LOGGING_READY.md docs/archive/fixes_20251205/
mv REST_WARMUP_READY.md docs/archive/fixes_20251205/
mv NAVIGATION_MAP.md docs/archive/fixes_20251205/

# 保留用户文档
# README.md - 保留
# ARCHITECTURE.md - 保留  
# testStrategy.md - 保留
```

---

## 文档说明

### 保留的用户文档
- **README.md**: 项目整体说明，使用说明
- **ARCHITECTURE.md**: 系统架构设计
- **testStrategy.md**: 交易策略定义

### docs/ 目录
- **API_REFERENCE.md**: 完整的函数API文档（新增）
- **01-QUICK_START.md**: 快速开始指南
- **02-ARCHITECTURE.md**: 架构详细说明
- **03-STRATEGY.md**: 策略实现说明
- **各种GUIDE**: 使用指南文档

### 归档文档
- **docs/archive/fixes_20251205/**: 2025-12-05修复过程中生成的临时文档
  - 主要记录了策略逻辑修复和API签名问题的解决过程
  - 仅供参考，不影响当前系统运行

---

**整理日期**: 2025-12-05  
**整理人**: AI Assistant

