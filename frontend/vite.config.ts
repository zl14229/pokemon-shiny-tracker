import vue from '@vitejs/plugin-vue'
import { defineConfig } from 'vite'

export default defineConfig(() => {
  const value = process.env.SHINY_API_PORT ?? '8080'
  if (!/^\d+$/.test(value) || Number(value) < 1 || Number(value) > 65535) {
    throw new Error('SHINY_API_PORT must be an integer from 1 to 65535')
  }
  const proxy = { '/api/v1': { target: `http://127.0.0.1:${Number(value)}`, changeOrigin: false } }
  return {
    plugins: [vue()],
    server: { host: '127.0.0.1', port: 5173, strictPort: true, proxy },
    preview: { host: '127.0.0.1', port: 4173, strictPort: true, proxy },
  }
})
