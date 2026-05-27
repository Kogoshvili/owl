import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

const backendPort = process.env.OWL_BACKEND_PORT ?? '3721'

export default defineConfig({
  plugins: [preact()],
  server: {
    proxy: {
      '/api': {
        target: `http://localhost:${backendPort}`,
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/api/, ''),
      },
    },
  },
})
