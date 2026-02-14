import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      },
      '/metrics': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  optimizeDeps: {
    include: ['monaco-editor']
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    chunkSizeWarningLimit: 1000,
    rollupOptions: {
      output: {
        manualChunks(id) {
          // Vue 核心库
          if (id.includes('node_modules/vue/') || id.includes('node_modules/@vue/') || id.includes('node_modules/pinia/') || id.includes('node_modules/vue-router/')) {
            return 'vue-vendor'
          }
          // Element Plus UI 组件库
          if (id.includes('node_modules/element-plus/') || id.includes('node_modules/@element-plus/')) {
            return 'element-plus'
          }
          // Monaco Editor 编辑器
          if (id.includes('node_modules/monaco-editor/')) {
            return 'monaco-editor'
          }
          // ECharts 图表库
          if (id.includes('node_modules/echarts/')) {
            return 'echarts'
          }
          // Axios HTTP 客户端
          if (id.includes('node_modules/axios/')) {
            return 'axios'
          }
        }
      }
    }
  }
})
