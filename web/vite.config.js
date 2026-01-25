import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vitejs.dev
export default defineConfig({
  plugins: [vue()], // Обязательный плагин для vue

  server: {
    port: 5173,
    proxy: {
      // Вы продолжаете писать запросы к /api/v1/..., и браузер не блокирует их по политике CORS,
      // так как Vite выступает в роли посредника. Все запросы к API будут перенаправлены на Go-бэкенд
      '/api/v1': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        // Игнорируем проверку SSL сертификатов, так как локально используем HTTP
        secure: false,
        // Чтобы не падать на локальном HTTP
        // ws: true
        // Поддержка WebSocket (если надумаем делать real-time уведомления)
      }
    }
  },

  build: {
    outDir: 'dist',
    emptyOutDir: true,
  }
})
