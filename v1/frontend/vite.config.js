import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '127.0.0.1',
    proxy: {
      '/v1': 'http://127.0.0.1:8879',
      '/ai': 'http://127.0.0.1:8879',
      '/ui': 'http://127.0.0.1:8879',
    },
  },
  build: {
    outDir: './dist',
    emptyOutDir: true,
  },
})
