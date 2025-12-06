#!/bin/bash

# 文档清理脚本
# 将根目录的临时 MD 文档移动到归档目录

echo "🧹 开始整理文档..."

cd /home/maeda/Documents/projects/goQuant

# 创建归档目录
echo "📁 创建归档目录..."
mkdir -p docs/archive/fixes_20251205

# 移动修复相关文档
echo "📦 移动修复相关文档..."
[ -f STRATEGY_FIX_SUMMARY.md ] && mv STRATEGY_FIX_SUMMARY.md docs/archive/fixes_20251205/
[ -f VERIFICATION_GUIDE.md ] && mv VERIFICATION_GUIDE.md docs/archive/fixes_20251205/
[ -f FINAL_FIX_SUMMARY.md ] && mv FINAL_FIX_SUMMARY.md docs/archive/fixes_20251205/
[ -f API_SIGNATURE_ERROR.md ] && mv API_SIGNATURE_ERROR.md docs/archive/fixes_20251205/

# 移动状态文档
echo "📦 移动状态文档..."
[ -f CRITICAL_ISSUES.md ] && mv CRITICAL_ISSUES.md docs/archive/fixes_20251205/
[ -f IMPLEMENTATION_STATUS.md ] && mv IMPLEMENTATION_STATUS.md docs/archive/fixes_20251205/
[ -f STATUS_SUMMARY.md ] && mv STATUS_SUMMARY.md docs/archive/fixes_20251205/
[ -f LOGGING_IMPLEMENTATION.md ] && mv LOGGING_IMPLEMENTATION.md docs/archive/fixes_20251205/
[ -f LOGGING_READY.md ] && mv LOGGING_READY.md docs/archive/fixes_20251205/
[ -f LOGGING_SPLIT_BY_SYMBOL.md ] && mv LOGGING_SPLIT_BY_SYMBOL.md docs/archive/fixes_20251205/
[ -f SYMBOL_LOGGING_READY.md ] && mv SYMBOL_LOGGING_READY.md docs/archive/fixes_20251205/
[ -f REST_WARMUP_READY.md ] && mv REST_WARMUP_READY.md docs/archive/fixes_20251205/
[ -f NAVIGATION_MAP.md ] && mv NAVIGATION_MAP.md docs/archive/fixes_20251205/

# 列出归档的文件
echo ""
echo "✅ 归档完成！以下文件已移动到 docs/archive/fixes_20251205/:"
ls -1 docs/archive/fixes_20251205/

echo ""
echo "📄 根目录保留的文档:"
ls -1 *.md 2>/dev/null || echo "  (仅保留用户文档)"

echo ""
echo "🎉 文档整理完成！"
echo ""
echo "说明："
echo "  - 用户文档（README.md, ARCHITECTURE.md, testStrategy.md）保留在根目录"
echo "  - AI生成的临时文档已归档到 docs/archive/fixes_20251205/"
echo "  - 新增 API 参考文档：docs/API_REFERENCE.md"
echo "  - 详细说明请查看：docs/DOCUMENT_CLEANUP.md"

