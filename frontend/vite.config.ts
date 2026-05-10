import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import UnoCSS from 'unocss/vite'
import { fileURLToPath, URL } from 'node:url'

// Vite 配置：代理 /api 和 /ws 到后端 push_gateway，避免本地跨域
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = env.VITE_API_BASE || 'http://localhost:8080'
  const wsTarget = (env.VITE_WS_URL || 'ws://localhost:8080').replace(/\/ws$/, '')

  return {
    plugins: [vue(), UnoCSS()],
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
      },
    },
    server: {
      port: 5173,
      proxy: {
        '/api': { target: apiBase, changeOrigin: true },
        '/healthz': { target: apiBase, changeOrigin: true },
        '/ws': { target: wsTarget, ws: true, changeOrigin: true },
      },
    },
    build: {
      target: 'es2022',
      sourcemap: false,
      rollupOptions: {
        output: {
          manualChunks: {
            vendor: ['vue', 'vue-router', 'pinia'],
            antd: ['ant-design-vue'],
            // echarts 等到页面真正 import 后再加回拆包
          },
        },
      },
    },
    // R11: 测试仅保留关键路径（Decimal 展示 / WS 订阅计数）
    test: {
      environment: 'jsdom',
      globals: true,
      include: ['src/**/*.spec.ts'],
    },
  }
})
