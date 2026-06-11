import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

// 后台前端默认通过 /api/admin 访问同源后端接口；本地开发时可按需要把 target 改成实际 Go 服务地址。
export default defineConfig({
  plugins: [react()],
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:8080',
        changeOrigin: true,
      },
    },
  },
});
