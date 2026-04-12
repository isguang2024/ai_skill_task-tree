import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '127.0.0.1',
    port: 5174,
    strictPort: true,
    proxy: {
      '/v1': 'http://127.0.0.1:8880',
      '/ai': 'http://127.0.0.1:8880',
      '/ui': 'http://127.0.0.1:8880',
    },
  },
  build: {
    outDir: './dist',
    emptyOutDir: true,
  },
})
