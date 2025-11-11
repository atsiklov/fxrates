import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const proxyTarget = env.VITE_PROXY_TARGET || 'http://localhost:8080'
  const port = Number(env.VITE_PORT || 5173)
  const sharedProxy = {
    '/api': {
      target: proxyTarget,
      changeOrigin: true,
    },
  }

  return {
    plugins: [react()],
    server: {
      host: true,
      port,
      proxy: sharedProxy,
    },
    preview: {
      host: true,
      port,
      proxy: sharedProxy,
    },
  }
})
