import React from 'react';

interface CyberStyleProps {
  isDark?: boolean;
}

const CyberStyle: React.FC<CyberStyleProps> = ({ isDark = true }) => {
  return (
    <style>{`
      /* 根主题变量定义 */
      :root {
        --theme-bg-gradient: radial-gradient(circle at 50% 50%, #0d001a 0%, #030008 100%);
        --theme-text-color: #00d4ff;
        --theme-accent-color: #ff00ff;
        --theme-card-bg: rgba(5, 5, 8, 0.85);
        --theme-border-color: rgba(0, 212, 255, 0.15);
        --theme-border-hover: rgba(255, 0, 255, 0.6);
        --theme-shadow-hover: rgba(255, 0, 255, 0.15);
        --theme-input-bg: rgba(0, 0, 0, 0.6);
        --theme-input-border: rgba(0, 212, 255, 0.2);
        --theme-thead-bg: rgba(0, 0, 0, 0.8);
        --theme-thead-text: #ff00ff;
        --theme-thead-border: #ff00ff;
        --theme-tbody-border: rgba(0, 212, 255, 0.1);
        --theme-hover-bg: rgba(0, 212, 255, 0.05);
        --theme-sider-bg: rgba(5, 5, 8, 0.9);
        --theme-menu-selected-bg: linear-gradient(90deg, rgba(255, 0, 255, 0.15), rgba(0, 212, 255, 0.05));
        --theme-menu-selected-border: #ff00ff;
        --theme-header-bg: rgba(5, 5, 8, 0.8);
        --theme-header-border: rgba(0, 212, 255, 0.15);
        --theme-btn-primary-bg: transparent;
        --theme-btn-primary-border: #00d4ff;
        --theme-btn-primary-text: #00d4ff;
        --theme-btn-primary-hover-bg: #00d4ff;
        --theme-btn-primary-hover-text: #000;
        --theme-btn-default-border: rgba(0, 212, 255, 0.3);
        --theme-btn-default-text: #00d4ff;
        --theme-btn-default-hover-border: #ff00ff;
        --theme-btn-default-hover-text: #ff00ff;
        --theme-stat-val: #ff00ff;
      }

      /* 亮色模式覆写 */
      .my-theme-light {
        --theme-bg-gradient: radial-gradient(circle at 50% 50%, #f0f4f8 0%, #dbeafe 100%);
        --theme-text-color: #1e3a8a;
        --theme-accent-color: #2563eb;
        --theme-card-bg: rgba(255, 255, 255, 0.7);
        --theme-border-color: rgba(37, 99, 235, 0.15);
        --theme-border-hover: rgba(37, 99, 235, 0.6);
        --theme-shadow-hover: rgba(37, 99, 235, 0.1);
        --theme-input-bg: rgba(255, 255, 255, 0.9);
        --theme-input-border: rgba(37, 99, 235, 0.25);
        --theme-thead-bg: rgba(241, 245, 249, 0.9);
        --theme-thead-text: #1e3a8a;
        --theme-thead-border: #2563eb;
        --theme-tbody-border: rgba(226, 232, 240, 0.8);
        --theme-hover-bg: rgba(37, 99, 235, 0.05);
        --theme-sider-bg: #ffffff;
        --theme-menu-selected-bg: linear-gradient(90deg, rgba(37, 99, 235, 0.12), rgba(37, 99, 235, 0.02));
        --theme-menu-selected-border: #2563eb;
        --theme-header-bg: rgba(255, 255, 255, 0.8);
        --theme-header-border: rgba(37, 99, 235, 0.12);
        --theme-btn-primary-bg: #2563eb;
        --theme-btn-primary-border: #2563eb;
        --theme-btn-primary-text: #ffffff;
        --theme-btn-primary-hover-bg: #1d4ed8;
        --theme-btn-primary-hover-text: #ffffff;
        --theme-btn-default-border: rgba(71, 85, 105, 0.3);
        --theme-btn-default-text: #475569;
        --theme-btn-default-hover-border: #2563eb;
        --theme-btn-default-hover-text: #2563eb;
        --theme-stat-val: #2563eb;
      }

      body {
        background: var(--theme-bg-gradient) !important;
        color: var(--theme-text-color) !important;
        font-family: 'Consolas', 'Fira Code', 'PingFang SC', monospace;
        min-height: 100vh;
        transition: background 0.3s ease, color 0.3s ease;
      }

      .ant-layout {
        background: transparent !important;
      }

      .ant-card, .ant-table, .ant-modal-content, .ant-menu, .ant-menu-sub, .ant-drawer-content {
        background: var(--theme-card-bg) !important;
        backdrop-filter: blur(20px) !important;
        -webkit-backdrop-filter: blur(20px) !important;
        border: 1px solid var(--theme-border-color) !important;
        color: var(--theme-text-color) !important;
        border-radius: 6px !important;
        transition: all 0.3s ease;
      }

      .ant-card-head {
        border-bottom: 1px solid var(--theme-border-color) !important;
      }

      .ant-card:hover {
        border-color: var(--theme-border-hover) !important;
        box-shadow: 0 0 15px var(--theme-shadow-hover) !important;
      }

      .ant-input, .ant-input-password, .ant-select-selector, .ant-input-affix-wrapper {
        background: var(--theme-input-bg) !important;
        border: 1px solid var(--theme-input-border) !important;
        color: var(--theme-text-color) !important;
        border-radius: 4px !important;
        transition: all 0.3s ease;
      }
      .ant-input-affix-wrapper input {
        background: transparent !important;
        color: var(--theme-text-color) !important;
      }
      .ant-input:focus, .ant-input-affix-wrapper-focused, .ant-select-focused .ant-select-selector {
        border-color: var(--theme-accent-color) !important;
        box-shadow: 0 0 8px rgba(255, 0, 255, 0.15) !important;
      }

      .ant-table-thead > tr > th {
        background: var(--theme-thead-bg) !important;
        color: var(--theme-thead-text) !important;
        border-bottom: 2px solid var(--theme-thead-border) !important;
        font-weight: 800;
      }
      .ant-table-tbody > tr > td {
        border-bottom: 1px solid var(--theme-tbody-border) !important;
        color: var(--theme-text-color) !important;
      }
      .ant-table-tbody > tr:hover > td {
        background: var(--theme-hover-bg) !important;
      }
      .ant-table-placeholder {
        background: transparent !important;
      }

      .ant-layout-sider {
        background: var(--theme-sider-bg) !important;
        border-right: 1px solid var(--theme-border-color) !important;
      }
      .ant-menu-item-selected {
        background: var(--theme-menu-selected-bg) !important;
        border-right: 3px solid var(--theme-menu-selected-border) !important;
        color: var(--theme-accent-color) !important;
      }
      .ant-menu-item:hover, .ant-menu-submenu-title:hover {
        color: var(--theme-accent-color) !important;
      }

      .ant-layout-header {
        background: var(--theme-header-bg) !important;
        backdrop-filter: blur(10px) !important;
        border-bottom: 1px solid var(--theme-header-border) !important;
      }

      h1, h2, h3, h4, h5, .ant-typography, .ant-card-head-title, .ant-statistic-title, .ant-descriptions-item-label {
        color: var(--theme-text-color) !important;
      }
      .ant-statistic-content-value, .ant-descriptions-item-content {
        color: var(--theme-stat-val) !important;
      }

      .ant-btn {
        transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1) !important;
        border-radius: 4px !important;
      }
      .ant-btn-primary {
        background: var(--theme-btn-primary-bg) !important;
        border: 1px solid var(--theme-btn-primary-border) !important;
        color: var(--theme-btn-primary-text) !important;
      }
      .ant-btn-primary:hover {
        background: var(--theme-btn-primary-hover-bg) !important;
        color: var(--theme-btn-primary-hover-text) !important;
        box-shadow: 0 0 15px var(--theme-border-hover) !important;
        transform: translateY(-1px);
      }
      .ant-btn-primary:active {
        transform: translateY(1px);
      }
      .ant-btn-default {
        border: 1px solid var(--theme-btn-default-border) !important;
        color: var(--theme-btn-default-text) !important;
      }
      .ant-btn-default:hover {
        border-color: var(--theme-btn-default-hover-border) !important;
        color: var(--theme-btn-default-hover-text) !important;
      }

      .ant-modal-mask {
        backdrop-filter: blur(5px) !important;
      }
      .ant-modal-header {
        background: transparent !important;
      }
      .ant-modal-title {
        color: var(--theme-accent-color) !important;
      }

      .ant-progress-inner {
        background: rgba(120, 120, 120, 0.08) !important;
        border-radius: 4px !important;
      }

      .status-glow {
        box-shadow: 0 0 10px #10b981 !important;
      }
      .status-glow.is-offline {
        background: var(--theme-accent-color) !important;
        box-shadow: 0 0 10px var(--theme-accent-color) !important;
      }

      ::-webkit-scrollbar { width: 6px; }
      ::-webkit-scrollbar-track { background: transparent; }
      ::-webkit-scrollbar-thumb { background: var(--theme-border-color); }
      ::-webkit-scrollbar-thumb:hover { background: var(--theme-accent-color); }
    `}</style>
  );
};

export default CyberStyle;
