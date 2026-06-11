import React from 'react';
import ReactDOM from 'react-dom/client';
import 'antd/dist/reset.css';
import { ConfigProvider, theme } from 'antd';
import { BrowserRouter } from 'react-router-dom';
import { AppRouter } from './app/router';
import './styles/global.css';

// 后台前端入口统一注入 Ant Design 主题令牌，让所有 CRUD 页面沿用同一套品牌色、圆角和容器层次。
ReactDOM.createRoot(document.getElementById('root') as HTMLElement).render(
  <React.StrictMode>
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: '#2f5d50',
          colorInfo: '#2f5d50',
          colorSuccess: '#2f7d4a',
          colorWarning: '#d48806',
          colorError: '#cf1322',
          colorBgLayout: '#f3efe7',
          colorBgContainer: '#fffaf2',
          colorBorderSecondary: '#e8dece',
          borderRadius: 14,
          fontFamily: 'PingFang SC, Microsoft YaHei, sans-serif',
          fontSize: 14,
        },
        components: {
          Layout: {
            headerBg: '#fffaf2',
            siderBg: '#f8f1e6',
            bodyBg: '#f3efe7',
          },
          Button: {
            controlHeight: 40,
            borderRadius: 12,
          },
          Card: {
            borderRadiusLG: 18,
          },
          Table: {
            headerBg: '#f6efe3',
            headerColor: '#3b3126',
            rowHoverBg: '#f9f4ea',
          },
          Modal: {
            borderRadiusLG: 18,
          },
          Drawer: {
            colorBgElevated: '#fffdf9',
          },
        },
      }}
    >
      <BrowserRouter>
        <AppRouter />
      </BrowserRouter>
    </ConfigProvider>
  </React.StrictMode>,
);
