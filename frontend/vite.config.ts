import { defineConfig, type Plugin } from 'vite'
import vue from '@vitejs/plugin-vue'
import { writeFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'

// emptyOutDir 会在构建时清空 web/dist（含 go:embed 依赖的占位 .keep），
// 构建结束后把 .keep 补回，避免它被删导致 git 工作区出现“deleted”脏状态。
function keepDistPlaceholder(): Plugin {
  const keepPath = fileURLToPath(new URL('../web/dist/.keep', import.meta.url))
  const content =
    '占位文件：保证 //go:embed all:dist 在未构建前端时仍有匹配项，CI 的 go build/test/vet 不致失败。真实前端产物由 `mise run build` 生成并覆盖。\n'
  return {
    name: 'keep-dist-placeholder',
    closeBundle() {
      writeFileSync(keepPath, content)
    },
  }
}

// 面板后端默认监听 9101，dev 时把 /api 代理过去，前端享受 Vite HMR。
export default defineConfig({
  plugins: [vue(), keepDistPlaceholder()],
  resolve: {
    alias: { '@': fileURLToPath(new URL('./src', import.meta.url)) },
  },
  server: {
    proxy: {
      '/api': { target: 'http://localhost:9101', changeOrigin: true },
    },
  },
  build: {
    // 产物输出到 Go embed 包目录（go:embed 不能引用包外路径，故指进 web/dist）。
    outDir: '../web/dist',
    emptyOutDir: true,
  },
})
