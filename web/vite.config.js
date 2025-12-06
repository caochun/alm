import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  // 生产构建配置
  build: {
    outDir: 'dist',
    assetsDir: 'assets'
  },
  // 开发服务器配置（仅在开发时使用）
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true
      }
    }
  }
})

