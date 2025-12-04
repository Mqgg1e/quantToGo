// Package dataFromWS is the main entry point for K-line data collection
// It re-exports both the base version and the enhanced version
package dataFromWS

// 基础版本 (v1) - 原始功能
// 仅包含基本的WebSocket订阅和数据存储

// 增强版本 (v2) - 生产级功能
// 包含自动重连、完整性检查、REST补全、多订阅者分发

// 用户可以直接导入所需的子模块:
// - goQuant/internal/dataManager/base    (基础版本)
// - goQuant/internal/dataManager/v2     (增强版本)
//
// 或通过别名导入:
// import base "goQuant/internal/dataManager/base"
// import enhanced "goQuant/internal/dataManager/v2"
