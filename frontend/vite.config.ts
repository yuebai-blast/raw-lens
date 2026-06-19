import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 面板后端默认监听 9090，dev 时把 /api 代理过去，前端享受 Vite HMR。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      '/api': { target: 'http://localhost:9090', changeOrigin: true },
    },
  },
  build: {
    // 产物输出到 Go embed 包目录（go:embed 不能引用包外路径，故指进 web/dist）。
    outDir: '../web/dist',
    emptyOutDir: true,
  },
})
