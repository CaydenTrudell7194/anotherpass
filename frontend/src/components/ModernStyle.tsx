import React from 'react';

const ModernStyle = () => (
  <style>{`
    /* 动态渐变背景 */
    body {
      background: linear-gradient(125deg, #0f172a, #1e1b4b, #312e81);
      background-size: 400% 400%;
      animation: gradientBG 15s ease infinite;
      min-height: 100vh;
      color: #fff;
    }
    @keyframes gradientBG {
      0% { background-position: 0% 50%; }
      50% { background-position: 100% 50%; }
      100% { background-position: 0% 50%; }
    }

    /* 强制 Antd 组件玻璃化 */
    .ant-layout, .ant-layout-header, .ant-layout-sider, .ant-card, .ant-table, 
    .ant-modal-content, .ant-menu, .ant-tag, .ant-select-dropdown {
      background: rgba(255, 255, 255, 0.05) !important;
      backdrop-filter: blur(15px) !important;
      -webkit-backdrop-filter: blur(15px) !important;
      border: 1px solid rgba(255, 255, 255, 0.1) !important;
      color: #fff !important;
    }

    /* 输入框需要更高透明度以保证可读性 */
    .ant-input, .ant-select-selector {
      background: rgba(255, 255, 255, 0.12) !important;
      backdrop-filter: blur(15px) !important;
      border: 1px solid rgba(255, 255, 255, 0.15) !important;
      color: #fff !important;
    }
    .ant-input::placeholder {
      color: rgba(255, 255, 255, 0.4) !important;
    }

    .ant-input-affix-wrapper {
      background: rgba(255, 255, 255, 0.12) !important;
      backdrop-filter: blur(15px) !important;
      border: 1px solid rgba(255, 255, 255, 0.15) !important;
    }
    .ant-input-affix-wrapper input {
      background: transparent !important;
      border: none !important;
    }

    /* 修复文字颜色 */
    .ant-typography, .ant-table, .ant-card-head-title, .ant-menu-item {
      color: #fff !important;
    }
    
    .ant-table-thead > tr > th {
      background: rgba(255, 255, 255, 0.1) !important;
      color: #fff !important;
    }

    .ant-table-tbody > tr:hover > td {
      background: rgba(255, 255, 255, 0.1) !important;
    }

    /* 按钮交互特效 */
    .ant-btn {
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
    }
    .ant-btn-primary {
      background: linear-gradient(90deg, #4f46e5, #8b5cf6) !important;
      border: none !important;
    }
    .ant-btn:hover {
      transform: translateY(-2px) scale(1.02);
      box-shadow: 0 6px 20px rgba(79, 70, 229, 0.3);
    }
    .ant-btn:active {
      transform: translateY(0);
    }

    /* 滚动条美化 */
    ::-webkit-scrollbar { width: 8px; }
    ::-webkit-scrollbar-thumb { background: rgba(255, 255, 255, 0.2); border-radius: 4px; }
    ::-webkit-scrollbar-thumb:hover { background: rgba(255, 255, 255, 0.4); }
  `}</style>
);

export default ModernStyle;
