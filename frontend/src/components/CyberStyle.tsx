import React from 'react';

const CyberStyle = () => (
  <style>{`
    /* 赛博朋克底色：深渊黑，带细腻动态暗流光 */
    body {
      background: radial-gradient(circle at 50% 50%, #0d001a 0%, #030008 100%) !important;
      color: #00d4ff !important;
      font-family: 'Consolas', 'Fira Code', 'PingFang SC', monospace;
      min-height: 100vh;
    }

    /* 全局布局透明度适配，防止双重背景色叠底变脏 */
    .ant-layout {
      background: transparent !important;
    }

    /* 统一容器：纯黑玻璃底板 + 极窄霓虹描边 + 赛博方直圆角 */
    .ant-card, .ant-table, .ant-modal-content, .ant-menu, .ant-menu-sub, .ant-drawer-content {
      background: rgba(5, 5, 8, 0.85) !important;
      backdrop-filter: blur(20px) !important;
      -webkit-backdrop-filter: blur(20px) !important;
      border: 1px solid rgba(0, 212, 255, 0.15) !important;
      color: #00d4ff !important;
      border-radius: 4px !important;
    }

    /* 卡片悬停霓虹唤醒 */
    .ant-card:hover {
      border-color: rgba(255, 0, 255, 0.6) !important;
      box-shadow: 0 0 15px rgba(255, 0, 255, 0.15) !important;
    }

    /* 输入框/下拉框：赛博极简暗区 */
    .ant-input, .ant-input-password, .ant-select-selector, .ant-input-affix-wrapper {
      background: rgba(0, 0, 0, 0.6) !important;
      border: 1px solid rgba(0, 212, 255, 0.2) !important;
      color: #00d4ff !important;
      border-radius: 2px !important;
      transition: all 0.3s ease;
    }
    .ant-input-affix-wrapper input {
      background: transparent !important;
      color: #00d4ff !important;
    }
    .ant-input:focus, .ant-input-affix-wrapper-focused, .ant-select-focused .ant-select-selector {
      border-color: #ff00ff !important;
      box-shadow: 0 0 8px rgba(255, 0, 255, 0.3) !important;
    }
    .ant-input::placeholder {
      color: rgba(0, 212, 255, 0.35) !important;
    }

    /* 表格组件赛博朋克深度改造 */
    .ant-table-thead > tr > th {
      background: rgba(0, 0, 0, 0.8) !important;
      color: #ff00ff !important;
      border-bottom: 2px solid #ff00ff !important;
      font-weight: 800;
    }
    .ant-table-tbody > tr > td {
      border-bottom: 1px solid rgba(0, 212, 255, 0.1) !important;
      color: #00d4ff !important;
    }
    .ant-table-tbody > tr:hover > td {
      background: rgba(0, 212, 255, 0.05) !important;
    }
    .ant-table-placeholder {
      background: transparent !important;
    }

    /* 侧边导航栏 */
    .ant-layout-sider {
      background: rgba(5, 5, 8, 0.9) !important;
      border-right: 1px solid rgba(0, 212, 255, 0.15) !important;
    }
    .ant-menu-item-selected {
      background: linear-gradient(90deg, rgba(255, 0, 255, 0.15), rgba(0, 212, 255, 0.05)) !important;
      border-right: 3px solid #ff00ff !important;
      color: #ff00ff !important;
    }
    .ant-menu-item:hover, .ant-menu-submenu-title:hover {
      color: #ff00ff !important;
    }

    /* 顶栏 (Header) 磨砂感强化 */
    .ant-layout-header {
      background: rgba(5, 5, 8, 0.8) !important;
      backdrop-filter: blur(10px) !important;
      border-bottom: 1px solid rgba(0, 212, 255, 0.15) !important;
    }

    /* 所有文字、标题颜色强制亮霓虹化 */
    h1, h2, h3, h4, h5, .ant-typography, .ant-card-head-title, .ant-statistic-title, .ant-descriptions-item-label {
      color: #00d4ff !important;
      text-shadow: 0 0 5px rgba(0, 212, 255, 0.2);
    }
    .ant-statistic-content-value, .ant-descriptions-item-content {
      color: #ff00ff !important;
      text-shadow: 0 0 5px rgba(255, 0, 255, 0.2);
    }

    /* 主操作按钮：霓虹空腔 + 呼吸悬停 */
    .ant-btn {
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
      border-radius: 2px !important;
    }
    .ant-btn-primary {
      background: transparent !important;
      border: 1px solid #00d4ff !important;
      color: #00d4ff !important;
      text-shadow: 0 0 4px #00d4ff;
    }
    .ant-btn-primary:hover {
      background: #00d4ff !important;
      color: #000 !important;
      box-shadow: 0 0 15px #00d4ff !important;
      transform: translateY(-1px);
    }
    .ant-btn-primary:active {
      transform: translateY(1px);
    }
    .ant-btn-default {
      border: 1px solid rgba(0, 212, 255, 0.3) !important;
      color: #00d4ff !important;
    }
    .ant-btn-default:hover {
      border-color: #ff00ff !important;
      color: #ff00ff !important;
      box-shadow: 0 0 10px rgba(255, 0, 255, 0.2) !important;
    }

    /* 模态框与弹窗的适配 */
    .ant-modal-mask {
      backdrop-filter: blur(5px) !important;
    }
    .ant-modal-header {
      background: transparent !important;
    }
    .ant-modal-title {
      color: #ff00ff !important;
    }

    /* 进度条 (Progress) 精细化 */
    .ant-progress-inner {
      background: rgba(255, 255, 255, 0.08) !important;
      border-radius: 2px !important;
    }

    /* 呼吸灯效果覆盖 */
    .status-glow {
      box-shadow: 0 0 10px #10b981 !important;
    }
    .status-glow.is-offline {
      background: #ff00ff !important;
      box-shadow: 0 0 10px #ff00ff !important;
    }

    /* 滚动条赛博化 */
    ::-webkit-scrollbar { width: 6px; }
    ::-webkit-scrollbar-track { background: #050505; }
    ::-webkit-scrollbar-thumb { background: rgba(0, 212, 255, 0.2); }
    ::-webkit-scrollbar-thumb:hover { background: #ff00ff; }
  `}</style>
);

export default CyberStyle;
